package controllers

import (
	"context"
	"fmt"
	"strings"
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
	NetworkChaosFinalizer = "chaos.engineering.io/networkchaos-finalizer"
	annotNetRules         = "chaos.engineering.io/net-rules"
	annotNetContainer     = "chaos.engineering.io/net-container"
	annotNetWaveInterval  = "chaos.engineering.io/wave-interval"
	annotNetLastWave      = "chaos.engineering.io/last-wave-time"
)

// cleanNetScript removes all tc qdiscs and flushes iptables INPUT/OUTPUT chains.
const cleanNetScript = `tc qdisc del dev eth0 root 2>/dev/null; iptables -F INPUT 2>/dev/null; iptables -F OUTPUT 2>/dev/null; echo recovered`

// NetworkChaosReconciler reconciles a NetworkChaos object
type NetworkChaosReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Recorder   record.EventRecorder
	Reporter   *infrastructure.ResultReporter
}

// +kubebuilder:rbac:groups=chaos.engineering.io,resources=networkchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=networkchaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=networkchaos/finalizers,verbs=update

func (r *NetworkChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	log := r.Log.WithValues("networkchaos", req.NamespacedName)

	var nc chaosv1alpha1.NetworkChaos
	if err := r.Get(ctx, req.NamespacedName, &nc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NetworkChaos")
		infrastructure.ChaosExecutionErrors.WithLabelValues("NetworkChaos", req.Namespace, "fetch_error").Inc()
		return ctrl.Result{}, err
	}

	defer func() {
		infrastructure.ChaosReconciliationDuration.WithLabelValues("NetworkChaos", nc.Namespace).Observe(time.Since(start).Seconds())
	}()

	if !nc.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &nc)
	}

	if !controllerutil.ContainsFinalizer(&nc, NetworkChaosFinalizer) {
		controllerutil.AddFinalizer(&nc, NetworkChaosFinalizer)
		if err := r.Update(ctx, &nc); err != nil {
			return ctrl.Result{}, err
		}
	}

	if nc.Status.ObservedGeneration != nc.Generation {
		nc.Status.ObservedGeneration = nc.Generation
		if err := r.Status().Update(ctx, &nc); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch nc.Status.Phase {
	case "":
		return r.initializeChaos(ctx, &nc)
	case chaosv1alpha1.ChaosPhasePending:
		return r.executeChaos(ctx, &nc)
	case chaosv1alpha1.ChaosPhaseRunning:
		return r.monitorChaos(ctx, &nc)
	default:
		return ctrl.Result{}, nil
	}
}

func (r *NetworkChaosReconciler) initializeChaos(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("networkchaos", nc.Name)
	log.Info("Initializing NetworkChaos")

	infrastructure.ChaosExperimentsTotal.WithLabelValues("NetworkChaos", nc.Namespace, string(nc.Spec.Action)).Inc()
	infrastructure.ChaosExperimentsActive.WithLabelValues("NetworkChaos", nc.Namespace, "Pending").Inc()

	nc.Status.Phase = chaosv1alpha1.ChaosPhasePending
	now := metav1.Now()
	nc.Status.StartTime = &now
	nc.Status.Conditions = append(nc.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: nc.Generation,
		LastTransitionTime: now,
		Reason:             "Initialized",
		Message:            "NetworkChaos initialized",
	})

	if err := r.Status().Update(ctx, nc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Eventf(nc, corev1.EventTypeNormal, "Initialized", "NetworkChaos experiment initialized with action %s", nc.Spec.Action)
	return ctrl.Result{Requeue: true}, nil
}

func (r *NetworkChaosReconciler) executeChaos(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("networkchaos", nc.Name)
	log.Info("Executing NetworkChaos", "action", nc.Spec.Action)

	affected, err := r.applyToTargets(ctx, nc)
	if err != nil {
		r.Recorder.Eventf(nc, corev1.EventTypeWarning, "ExecutionFailed", "Failed to execute chaos: %v", err)
		return r.updateFailedStatus(ctx, nc, err.Error())
	}
	if affected == 0 {
		return r.updateCompletedStatus(ctx, nc, "No targets matched the selector")
	}

	r.Recorder.Eventf(nc, corev1.EventTypeNormal, "Executing", "Network rules applied to %d pod(s) with action %s", affected, nc.Spec.Action)
	r.stampWaveTime(ctx, nc)

	nc.Status.Phase = chaosv1alpha1.ChaosPhaseRunning
	infrastructure.ChaosExperimentsActive.WithLabelValues("NetworkChaos", nc.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsActive.WithLabelValues("NetworkChaos", nc.Namespace, "Running").Inc()

	now := metav1.Now()
	nc.Status.Conditions = append(nc.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeExecuting,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: nc.Generation,
		LastTransitionTime: now,
		Reason:             "Executing",
		Message:            fmt.Sprintf("Network rules applied to %d pod(s)", affected),
	})

	if err := r.Status().Update(ctx, nc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.nextRequeue(nc)}, nil
}

func (r *NetworkChaosReconciler) monitorChaos(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("networkchaos", nc.Name)

	// Check total duration
	if nc.Status.StartTime != nil && nc.Spec.Duration != "" {
		if d, err := time.ParseDuration(nc.Spec.Duration); err == nil {
			if time.Since(nc.Status.StartTime.Time) >= d {
				log.Info("Duration elapsed — recovering network rules")
				r.Recorder.Event(nc, corev1.EventTypeNormal, "DurationElapsed", "Experiment duration elapsed, recovering network rules")
				r.recoverAll(ctx, nc)
				return r.updateCompletedStatus(ctx, nc, "Duration elapsed — network rules removed")
			}
		}
	}

	// Wave interval: refresh tc rules on pods that may have restarted
	waveStr, hasWave := nc.Annotations[annotNetWaveInterval]
	if !hasWave || waveStr == "" {
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	waveInterval, err := time.ParseDuration(waveStr)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	if last := nc.Annotations[annotNetLastWave]; last != "" {
		if t, err := time.Parse(time.RFC3339, last); err == nil {
			if time.Since(t) < waveInterval {
				return ctrl.Result{RequeueAfter: time.Until(t.Add(waveInterval)) + time.Second}, nil
			}
		}
	}

	log.Info("Refreshing network rules (wave)")
	r.applyToTargets(ctx, nc) //nolint:errcheck
	r.stampWaveTime(ctx, nc)
	return ctrl.Result{RequeueAfter: waveInterval}, nil
}

// applyToTargets selects pods and execs the tc/iptables script in each one.
func (r *NetworkChaosReconciler) applyToTargets(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) (int, error) {
	log := r.Log.WithValues("networkchaos", nc.Name)

	podLister := infrastructure.NewKubernetesPodLister(r.Client)
	selector := domain.NewTargetSelector(nc.Spec.Mode, nc.Spec.Value, nc.Spec.Selector, nc.Spec.BlastRadius, podLister)

	targets, err := selector.SelectTargets(ctx)
	if err != nil {
		infrastructure.ChaosExecutionErrors.WithLabelValues("NetworkChaos", nc.Namespace, "target_selection_error").Inc()
		return 0, fmt.Errorf("selecting targets: %w", err)
	}

	script := buildNetScript(nc.Spec)
	applied := 0
	podNames := make([]string, 0, len(targets))
	targetResults := make([]chaosv1alpha1.TargetResult, 0, len(targets))

	for i := range targets {
		pod := targets[i]
		container := selectContainer(nil, pod)
		podRef := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		out, execErr := infrastructure.ExecInPod(ctx, r.RestConfig, pod, container, []string{"sh", "-c", script})
		if execErr != nil {
			log.Error(execErr, "Failed to apply network rule", "pod", pod.Name)
			infrastructure.ChaosExecutionErrors.WithLabelValues("NetworkChaos", nc.Namespace, "exec_error").Inc()
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: false, Error: execErr.Error(), AppliedAt: metav1.Now(),
			})
			continue
		}

		result := strings.TrimSpace(out)
		log.Info("Network rule applied", "pod", pod.Name, "action", nc.Spec.Action, "result", result)

		if strings.Contains(result, "applied") {
			infrastructure.ChaosTargetsAffected.WithLabelValues("NetworkChaos", nc.Namespace, string(nc.Spec.Action)).Add(1)
			applied++
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: true, AppliedAt: metav1.Now(),
			})
		} else {
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: false, Error: "rule not applied: " + result, AppliedAt: metav1.Now(),
			})
		}

		// Annotate pod for cleanup
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[annotNetRules] = string(nc.Spec.Action)
		pod.Annotations[annotNetContainer] = container
		r.Update(ctx, &pod) //nolint:errcheck
		podNames = append(podNames, podRef)
	}

	nc.Status.AffectedPods = podNames
	nc.Status.Result = infrastructure.BuildResult(targetResults, false)
	return applied, nil
}

// recoverAll removes tc/iptables rules from all pods in AffectedPods.
func (r *NetworkChaosReconciler) recoverAll(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) {
	log := r.Log.WithValues("networkchaos", nc.Name)

	for _, ref := range nc.Status.AffectedPods {
		pod, ok := r.fetchPod(ctx, ref)
		if !ok {
			continue
		}

		container := pod.Annotations[annotNetContainer]
		if container == "" {
			container = selectContainer(nil, *pod)
		}

		out, err := infrastructure.ExecInPod(ctx, r.RestConfig, *pod, container, []string{"sh", "-c", cleanNetScript})
		if err != nil {
			log.Error(err, "Failed to recover network rules", "pod", pod.Name)
		} else {
			log.Info("Network rules removed", "pod", pod.Name, "result", strings.TrimSpace(out))
		}

		if pod.Annotations != nil {
			delete(pod.Annotations, annotNetRules)
			delete(pod.Annotations, annotNetContainer)
			r.Update(ctx, pod) //nolint:errcheck
		}
	}
}

func (r *NetworkChaosReconciler) fetchPod(ctx context.Context, ref string) (*corev1.Pod, bool) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return nil, false
	}
	var pod corev1.Pod
	if err := r.Get(ctx, client.ObjectKey{Namespace: parts[0], Name: parts[1]}, &pod); err != nil {
		return nil, false
	}
	return &pod, true
}

func (r *NetworkChaosReconciler) stampWaveTime(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) {
	if nc.Annotations == nil {
		nc.Annotations = make(map[string]string)
	}
	nc.Annotations[annotNetLastWave] = time.Now().UTC().Format(time.RFC3339)
	r.Update(ctx, nc) //nolint:errcheck
}

func (r *NetworkChaosReconciler) nextRequeue(nc *chaosv1alpha1.NetworkChaos) time.Duration {
	if v, ok := nc.Annotations[annotNetWaveInterval]; ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if nc.Spec.Duration != "" {
		if d, err := time.ParseDuration(nc.Spec.Duration); err == nil {
			return d
		}
	}
	return RequeueAfterDuration
}

func (r *NetworkChaosReconciler) handleDeletion(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(nc, NetworkChaosFinalizer) {
		if nc.Status.Phase == chaosv1alpha1.ChaosPhaseRunning {
			r.Recorder.Event(nc, corev1.EventTypeNormal, "Recovering", "Cleaning up network chaos effects")
			r.recoverAll(ctx, nc)
		}
		controllerutil.RemoveFinalizer(nc, NetworkChaosFinalizer)
		if err := r.Update(ctx, nc); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *NetworkChaosReconciler) updateCompletedStatus(ctx context.Context, nc *chaosv1alpha1.NetworkChaos, msg string) (ctrl.Result, error) {
	nc.Status.Phase = chaosv1alpha1.ChaosPhaseCompleted
	nc.Status.Message = msg
	now := metav1.Now()
	nc.Status.CompletionTime = &now
	infrastructure.ChaosExperimentsActive.WithLabelValues("NetworkChaos", nc.Namespace, "Running").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("NetworkChaos", nc.Namespace, "Completed").Inc()
	if nc.Status.StartTime != nil {
		infrastructure.ChaosExperimentsDuration.WithLabelValues("NetworkChaos", nc.Namespace, string(nc.Spec.Action)).
			Observe(now.Time.Sub(nc.Status.StartTime.Time).Seconds())
	}
	nc.Status.Conditions = append(nc.Status.Conditions, metav1.Condition{
		Type: chaosv1alpha1.ConditionTypeCompleted, Status: metav1.ConditionTrue,
		ObservedGeneration: nc.Generation, LastTransitionTime: now,
		Reason: "Completed", Message: msg,
	})
	if err := r.Status().Update(ctx, nc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(nc, corev1.EventTypeNormal, "Completed", msg)
	r.saveReport(ctx, nc)
	return ctrl.Result{}, nil
}

func (r *NetworkChaosReconciler) updateFailedStatus(ctx context.Context, nc *chaosv1alpha1.NetworkChaos, msg string) (ctrl.Result, error) {
	nc.Status.Phase = chaosv1alpha1.ChaosPhaseFailed
	nc.Status.Message = msg
	now := metav1.Now()
	nc.Status.CompletionTime = &now
	infrastructure.ChaosExperimentsActive.WithLabelValues("NetworkChaos", nc.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("NetworkChaos", nc.Namespace, "Failed").Inc()
	nc.Status.Conditions = append(nc.Status.Conditions, metav1.Condition{
		Type: chaosv1alpha1.ConditionTypeCompleted, Status: metav1.ConditionFalse,
		ObservedGeneration: nc.Generation, LastTransitionTime: now,
		Reason: "Failed", Message: msg,
	})
	if err := r.Status().Update(ctx, nc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(nc, corev1.EventTypeWarning, "Failed", msg)
	r.saveReport(ctx, nc)
	return ctrl.Result{}, nil
}

func (r *NetworkChaosReconciler) saveReport(ctx context.Context, nc *chaosv1alpha1.NetworkChaos) {
	if r.Reporter == nil {
		return
	}
	details := map[string]string{
		"action":    string(nc.Spec.Action),
		"mode":      string(nc.Spec.Mode),
		"direction": string(nc.Spec.Direction),
		"duration":  nc.Spec.Duration,
	}
	if nc.Spec.Value != "" {
		details["value"] = nc.Spec.Value
	}
	if nc.Spec.Delay != nil {
		details["delay.latency"] = nc.Spec.Delay.Latency
		if nc.Spec.Delay.Jitter != "" {
			details["delay.jitter"] = nc.Spec.Delay.Jitter
		}
	}
	if nc.Spec.Loss != nil {
		details["loss.percent"] = nc.Spec.Loss.Loss + "%"
	}
	if err := r.Reporter.CreateSummaryConfigMap(ctx, infrastructure.ReportInput{
		Namespace:      nc.Namespace,
		ExperimentName: nc.Name,
		ChaosType:      "NetworkChaos",
		Phase:          string(nc.Status.Phase),
		StartTime:      nc.Status.StartTime,
		CompletionTime: nc.Status.CompletionTime,
		Result:         resultOrFromPods(nc.Status.Result, nc.Status.AffectedPods),
		AffectedPods:   nc.Status.AffectedPods,
		Message:        nc.Status.Message,
		Details:        details,
	}); err != nil {
		r.Log.Error(err, "Failed to save experiment report", "name", nc.Name)
	}
}

func (r *NetworkChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.NetworkChaos{}).
		Complete(r)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// buildNetScript returns a sh one-liner that installs tc/iptables if needed
// and applies the requested network fault. Outputs "applied" on success.
func buildNetScript(spec chaosv1alpha1.NetworkChaosSpec) string {
	// tc and iptables are expected to be pre-installed in the target image
	// (nginx-chaos:latest). The checks below fail gracefully if missing.
	install := `which tc >/dev/null 2>&1 || { echo "tc not found - use nginx-chaos:latest image"; exit 1; }; true`

	var rule string
	switch spec.Action {
	case chaosv1alpha1.NetworkDelayAction:
		if spec.Delay != nil {
			extra := ""
			if spec.Delay.Jitter != "" {
				extra += " " + spec.Delay.Jitter
			}
			if spec.Delay.Correlation != "" {
				extra += " " + spec.Delay.Correlation + "%"
			}
			rule = fmt.Sprintf("tc qdisc replace dev eth0 root netem delay %s%s", spec.Delay.Latency, extra)
		}

	case chaosv1alpha1.NetworkLossAction:
		if spec.Loss != nil {
			corr := ""
			if spec.Loss.Correlation != "" {
				corr = " " + spec.Loss.Correlation + "%"
			}
			rule = fmt.Sprintf("tc qdisc replace dev eth0 root netem loss %s%%%s", spec.Loss.Loss, corr)
		}

	case chaosv1alpha1.NetworkDuplicateAction:
		if spec.Duplicate != nil {
			corr := ""
			if spec.Duplicate.Correlation != "" {
				corr = " " + spec.Duplicate.Correlation + "%"
			}
			rule = fmt.Sprintf("tc qdisc replace dev eth0 root netem duplicate %s%%%s", spec.Duplicate.Duplicate, corr)
		}

	case chaosv1alpha1.NetworkCorruptAction:
		if spec.Corrupt != nil {
			corr := ""
			if spec.Corrupt.Correlation != "" {
				corr = " " + spec.Corrupt.Correlation + "%"
			}
			rule = fmt.Sprintf("tc qdisc replace dev eth0 root netem corrupt %s%%%s", spec.Corrupt.Corrupt, corr)
		}

	case chaosv1alpha1.NetworkBandwidthAction:
		if spec.Bandwidth != nil {
			limit := uint32(10000)
			if spec.Bandwidth.Limit > 0 {
				limit = spec.Bandwidth.Limit
			}
			buf := uint32(1600)
			if spec.Bandwidth.Buffer > 0 {
				buf = spec.Bandwidth.Buffer
			}
			rule = fmt.Sprintf("tc qdisc replace dev eth0 root tbf rate %s burst %d limit %d", spec.Bandwidth.Rate, buf, limit)
		}

	case chaosv1alpha1.NetworkPartitionAction:
		// Hard partition: drop all inbound and outbound traffic
		rule = "iptables -I INPUT 1 -j DROP; iptables -I OUTPUT 1 -j DROP"
	}

	if rule == "" {
		return "echo 'no rule defined for action'"
	}

	return fmt.Sprintf("%s; %s && echo applied || echo failed", install, rule)
}
