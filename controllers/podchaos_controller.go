package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	"goland-operator/internal/domain"
	"goland-operator/internal/infrastructure"
)

const (
	PodChaosFinalizer    = "chaos.engineering.io/podchaos-finalizer"
	RequeueAfterDuration = 30 * time.Second

	// annotWaveInterval configures how often a new kill wave fires during Running phase.
	// Value: any Go duration string — e.g. "2m", "90s".
	// When absent the experiment kills once and completes normally.
	annotWaveInterval = "chaos.engineering.io/wave-interval"

	// annotLastWaveTime is written by the controller to track the last kill wave.
	annotLastWaveTime = "chaos.engineering.io/last-wave-time"
)

// PodChaosReconciler reconciles a PodChaos object
type PodChaosReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Recorder   record.EventRecorder
	Reporter   *infrastructure.ResultReporter
}

// +kubebuilder:rbac:groups=chaos.engineering.io,resources=podchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=podchaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=podchaos/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/exec,verbs=create

func (r *PodChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	log := r.Log.WithValues("podchaos", req.NamespacedName)

	var podChaos chaosv1alpha1.PodChaos
	if err := r.Get(ctx, req.NamespacedName, &podChaos); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get PodChaos")
		infrastructure.ChaosExecutionErrors.WithLabelValues("PodChaos", req.Namespace, "fetch_error").Inc()
		return ctrl.Result{}, err
	}

	defer func() {
		infrastructure.ChaosReconciliationDuration.WithLabelValues("PodChaos", podChaos.Namespace).Observe(time.Since(startTime).Seconds())
	}()

	if !podChaos.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &podChaos)
	}

	if !controllerutil.ContainsFinalizer(&podChaos, PodChaosFinalizer) {
		controllerutil.AddFinalizer(&podChaos, PodChaosFinalizer)
		if err := r.Update(ctx, &podChaos); err != nil {
			return ctrl.Result{}, err
		}
	}

	if podChaos.Status.ObservedGeneration != podChaos.Generation {
		podChaos.Status.ObservedGeneration = podChaos.Generation
		if err := r.Status().Update(ctx, &podChaos); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch podChaos.Status.Phase {
	case "":
		return r.initializeChaos(ctx, &podChaos)
	case chaosv1alpha1.ChaosPhasePending:
		return r.executeChaos(ctx, &podChaos)
	case chaosv1alpha1.ChaosPhaseRunning:
		return r.monitorChaos(ctx, &podChaos)
	default:
		return ctrl.Result{}, nil
	}
}

func (r *PodChaosReconciler) initializeChaos(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("podchaos", podChaos.Name)
	log.Info("Initializing PodChaos")

	infrastructure.ChaosExperimentsTotal.WithLabelValues("PodChaos", podChaos.Namespace, string(podChaos.Spec.Action)).Inc()
	infrastructure.ChaosExperimentsActive.WithLabelValues("PodChaos", podChaos.Namespace, "Pending").Inc()

	podChaos.Status.Phase = chaosv1alpha1.ChaosPhasePending
	now := metav1.Now()
	podChaos.Status.StartTime = &now

	podChaos.Status.Conditions = append(podChaos.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: podChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Initialized",
		Message:            "PodChaos initialized successfully",
	})

	if err := r.Status().Update(ctx, podChaos); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(podChaos, corev1.EventTypeNormal, "Initialized", "PodChaos experiment initialized with action %s", podChaos.Spec.Action)
	return ctrl.Result{Requeue: true}, nil
}

func (r *PodChaosReconciler) executeChaos(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("podchaos", podChaos.Name)
	log.Info("Executing PodChaos")

	killed, err := r.killWave(ctx, podChaos)
	if err != nil {
		r.Recorder.Eventf(podChaos, corev1.EventTypeWarning, "ExecutionFailed", "Failed to execute chaos: %v", err)
		return r.updateFailedStatus(ctx, podChaos, fmt.Sprintf("Execution failed: %v", err))
	}
	if killed == 0 {
		return r.updateCompletedStatus(ctx, podChaos, "No targets matched the selector")
	}

	// Build experiment result
	targetResults := make([]chaosv1alpha1.TargetResult, 0, killed)
	for _, podRef := range podChaos.Status.AffectedPods {
		targetResults = append(targetResults, chaosv1alpha1.TargetResult{
			Name:      podRef,
			Success:   true,
			AppliedAt: metav1.Now(),
		})
	}
	podChaos.Status.Result = infrastructure.BuildResult(targetResults, false)

	r.Recorder.Eventf(podChaos, corev1.EventTypeNormal, "Executing", "Chaos applied to %d pod(s) with action %s", killed, podChaos.Spec.Action)

	// Mark last wave time so monitorChaos can schedule the next one
	r.stampWaveTime(ctx, podChaos)

	podChaos.Status.Phase = chaosv1alpha1.ChaosPhaseRunning
	infrastructure.ChaosExperimentsActive.WithLabelValues("PodChaos", podChaos.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsActive.WithLabelValues("PodChaos", podChaos.Namespace, "Running").Inc()

	now := metav1.Now()
	podChaos.Status.Conditions = append(podChaos.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeExecuting,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: podChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Executing",
		Message:            fmt.Sprintf("Wave 1: killed %d pod(s)", killed),
	})

	if err := r.Status().Update(ctx, podChaos); err != nil {
		return ctrl.Result{}, err
	}

	// If wave-interval is configured, requeue after the interval so the next
	// wave fires. Otherwise requeue after the full duration (one-shot mode).
	return ctrl.Result{RequeueAfter: r.nextRequeue(podChaos)}, nil
}

// monitorChaos handles the Running phase:
//   - Fires repeated kill waves when wave-interval is set.
//   - Completes the experiment when total duration has elapsed.
func (r *PodChaosReconciler) monitorChaos(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("podchaos", podChaos.Name)

	// ── Check total duration ────────────────────────────────────────────────
	if podChaos.Status.StartTime != nil && podChaos.Spec.Duration != "" {
		totalDuration, err := time.ParseDuration(podChaos.Spec.Duration)
		if err == nil && time.Since(podChaos.Status.StartTime.Time) >= totalDuration {
			log.Info("Total duration elapsed — completing experiment")
			r.Recorder.Event(podChaos, corev1.EventTypeNormal, "DurationElapsed", "Experiment total duration elapsed")
			return r.updateCompletedStatus(ctx, podChaos, "Total duration elapsed")
		}
	}

	// ── Wave interval logic ─────────────────────────────────────────────────
	waveIntervalStr, hasWave := podChaos.Annotations[annotWaveInterval]
	if !hasWave || waveIntervalStr == "" {
		// One-shot mode — just wait for duration
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	waveInterval, err := time.ParseDuration(waveIntervalStr)
	if err != nil {
		log.Error(err, "Invalid wave-interval annotation", "value", waveIntervalStr)
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	// Check if it is time to fire the next wave
	lastWaveStr := podChaos.Annotations[annotLastWaveTime]
	if lastWaveStr != "" {
		lastWave, err := time.Parse(time.RFC3339, lastWaveStr)
		if err == nil && time.Since(lastWave) < waveInterval {
			remaining := time.Until(lastWave.Add(waveInterval))
			log.Info("Next wave scheduled", "in", remaining.Round(time.Second))
			return ctrl.Result{RequeueAfter: remaining + time.Second}, nil
		}
	}

	// ── Fire next wave ──────────────────────────────────────────────────────
	log.Info("Firing kill wave")
	killed, err := r.killWave(ctx, podChaos)
	if err != nil {
		log.Error(err, "Wave execution error (continuing)")
		infrastructure.ChaosExecutionErrors.WithLabelValues("PodChaos", podChaos.Namespace, "wave_error").Inc()
	} else {
		log.Info("Wave complete", "pods_killed", killed)
		infrastructure.ChaosTargetsAffected.WithLabelValues("PodChaos", podChaos.Namespace, string(podChaos.Spec.Action)).Add(float64(killed))
		r.Recorder.Eventf(podChaos, corev1.EventTypeNormal, "WaveFired", "Kill wave completed: %d pod(s) affected", killed)
	}

	r.stampWaveTime(ctx, podChaos)
	return ctrl.Result{RequeueAfter: waveInterval}, nil
}

// killWave selects targets and executes the chaos action, returning the number
// of pods actually affected.
func (r *PodChaosReconciler) killWave(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) (int, error) {
	log := r.Log.WithValues("podchaos", podChaos.Name)

	podLister := infrastructure.NewKubernetesPodLister(r.Client)
	podOps := infrastructure.NewKubernetesPodOperations(r.Client, r.RestConfig)
	selector := domain.NewTargetSelector(
		podChaos.Spec.Mode,
		podChaos.Spec.Value,
		podChaos.Spec.Selector,
		podChaos.Spec.BlastRadius,
		podLister,
	)

	targets, err := selector.SelectTargets(ctx)
	if err != nil {
		infrastructure.ChaosExecutionErrors.WithLabelValues("PodChaos", podChaos.Namespace, "target_selection_error").Inc()
		return 0, fmt.Errorf("selecting targets: %w", err)
	}

	if len(targets) == 0 {
		log.Info("No targets selected for this wave (pods may still be restarting)")
		return 0, nil
	}

	log.Info("Targets selected", "count", len(targets))
	infrastructure.ChaosTargetsAffected.WithLabelValues("PodChaos", podChaos.Namespace, string(podChaos.Spec.Action)).Add(float64(len(targets)))

	executor := domain.NewPodChaosExecutor(&podChaos.Spec, podOps)
	if err := executor.Execute(ctx, targets); err != nil {
		infrastructure.ChaosExecutionErrors.WithLabelValues("PodChaos", podChaos.Namespace, "execution_error").Inc()
		return 0, fmt.Errorf("executing chaos: %w", err)
	}

	podNames := make([]string, len(targets))
	for i, p := range targets {
		podNames[i] = fmt.Sprintf("%s/%s", p.Namespace, p.Name)
	}
	podChaos.Status.AffectedPods = podNames
	return len(targets), nil
}

// stampWaveTime writes the current time to the last-wave-time annotation.
func (r *PodChaosReconciler) stampWaveTime(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) {
	if podChaos.Annotations == nil {
		podChaos.Annotations = make(map[string]string)
	}
	podChaos.Annotations[annotLastWaveTime] = time.Now().UTC().Format(time.RFC3339)
	if err := r.Update(ctx, podChaos); err != nil {
		r.Log.Error(err, "Failed to stamp wave time", "podchaos", podChaos.Name)
	}
}

// nextRequeue returns how long to wait before the first monitoring cycle.
func (r *PodChaosReconciler) nextRequeue(podChaos *chaosv1alpha1.PodChaos) time.Duration {
	if v, ok := podChaos.Annotations[annotWaveInterval]; ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if podChaos.Spec.Duration != "" {
		if d, err := time.ParseDuration(podChaos.Spec.Duration); err == nil {
			return d
		}
	}
	return RequeueAfterDuration
}

func (r *PodChaosReconciler) updateCompletedStatus(ctx context.Context, podChaos *chaosv1alpha1.PodChaos, message string) (ctrl.Result, error) {
	podChaos.Status.Phase = chaosv1alpha1.ChaosPhaseCompleted
	podChaos.Status.Message = message
	now := metav1.Now()
	podChaos.Status.CompletionTime = &now

	infrastructure.ChaosExperimentsActive.WithLabelValues("PodChaos", podChaos.Namespace, "Running").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("PodChaos", podChaos.Namespace, "Completed").Inc()

	if podChaos.Status.StartTime != nil {
		infrastructure.ChaosExperimentsDuration.WithLabelValues("PodChaos", podChaos.Namespace, string(podChaos.Spec.Action)).Observe(
			now.Time.Sub(podChaos.Status.StartTime.Time).Seconds(),
		)
	}

	podChaos.Status.Conditions = append(podChaos.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeCompleted,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: podChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Completed",
		Message:            message,
	})

	if err := r.Status().Update(ctx, podChaos); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(podChaos, corev1.EventTypeNormal, "Completed", message)
	r.saveReport(ctx, podChaos)
	return ctrl.Result{}, nil
}

func (r *PodChaosReconciler) updateFailedStatus(ctx context.Context, podChaos *chaosv1alpha1.PodChaos, message string) (ctrl.Result, error) {
	podChaos.Status.Phase = chaosv1alpha1.ChaosPhaseFailed
	podChaos.Status.Message = message
	now := metav1.Now()
	podChaos.Status.CompletionTime = &now

	infrastructure.ChaosExperimentsActive.WithLabelValues("PodChaos", podChaos.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("PodChaos", podChaos.Namespace, "Failed").Inc()

	podChaos.Status.Conditions = append(podChaos.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeCompleted,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: podChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Failed",
		Message:            message,
	})

	if err := r.Status().Update(ctx, podChaos); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(podChaos, corev1.EventTypeWarning, "Failed", message)
	r.saveReport(ctx, podChaos)
	return ctrl.Result{}, nil
}

func (r *PodChaosReconciler) saveReport(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) {
	if r.Reporter == nil {
		return
	}
	details := map[string]string{
		"action":   string(podChaos.Spec.Action),
		"mode":     string(podChaos.Spec.Mode),
		"duration": podChaos.Spec.Duration,
	}
	if podChaos.Spec.Value != "" {
		details["value"] = podChaos.Spec.Value
	}
	if podChaos.Spec.GracePeriod != nil {
		details["gracePeriod"] = fmt.Sprintf("%ds", *podChaos.Spec.GracePeriod)
	}
	if err := r.Reporter.CreateSummaryConfigMap(ctx, infrastructure.ReportInput{
		Namespace:      podChaos.Namespace,
		ExperimentName: podChaos.Name,
		ChaosType:      "PodChaos",
		Phase:          string(podChaos.Status.Phase),
		StartTime:      podChaos.Status.StartTime,
		CompletionTime: podChaos.Status.CompletionTime,
		Result:         resultOrFromPods(podChaos.Status.Result, podChaos.Status.AffectedPods),
		AffectedPods:   podChaos.Status.AffectedPods,
		Message:        podChaos.Status.Message,
		Details:        details,
	}); err != nil {
		r.Log.Error(err, "Failed to save experiment report", "name", podChaos.Name)
	}
}

func (r *PodChaosReconciler) handleDeletion(ctx context.Context, podChaos *chaosv1alpha1.PodChaos) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(podChaos, PodChaosFinalizer) {
		controllerutil.RemoveFinalizer(podChaos, PodChaosFinalizer)
		if err := r.Update(ctx, podChaos); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *PodChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.PodChaos{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// resultOrFromPods returns the existing result or builds one from AffectedPods.
func resultOrFromPods(result *chaosv1alpha1.ExperimentResult, affectedPods []string) *chaosv1alpha1.ExperimentResult {
	if result != nil {
		return result
	}
	targets := make([]chaosv1alpha1.TargetResult, 0, len(affectedPods))
	for _, pod := range affectedPods {
		targets = append(targets, chaosv1alpha1.TargetResult{
			Name:    pod,
			Success: true,
		})
	}
	return infrastructure.BuildResult(targets, false)
}
