package controllers

import (
	"context"
	"fmt"
	"strconv"
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
	StressChaosFinalizer   = "chaos.engineering.io/stresschaos-finalizer"
	stressPIDsAnnotation   = "chaos.engineering.io/stress-pids"
	stressContainerAnnot   = "chaos.engineering.io/stress-container"
	stressDiskPathAnnot    = "chaos.engineering.io/stress-disk-path"
)

// StressChaosReconciler reconciles a StressChaos object
type StressChaosReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Recorder   record.EventRecorder
	Reporter   *infrastructure.ResultReporter
}

// +kubebuilder:rbac:groups=chaos.engineering.io,resources=stresschaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=stresschaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=stresschaos/finalizers,verbs=update

func (r *StressChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()
	log := r.Log.WithValues("stresschaos", req.NamespacedName)

	var stressChaos chaosv1alpha1.StressChaos
	if err := r.Get(ctx, req.NamespacedName, &stressChaos); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get StressChaos")
		infrastructure.ChaosExecutionErrors.WithLabelValues("StressChaos", req.Namespace, "fetch_error").Inc()
		return ctrl.Result{}, err
	}

	defer func() {
		duration := time.Since(startTime).Seconds()
		infrastructure.ChaosReconciliationDuration.WithLabelValues("StressChaos", stressChaos.Namespace).Observe(duration)
	}()

	if !stressChaos.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &stressChaos)
	}

	if !controllerutil.ContainsFinalizer(&stressChaos, StressChaosFinalizer) {
		controllerutil.AddFinalizer(&stressChaos, StressChaosFinalizer)
		if err := r.Update(ctx, &stressChaos); err != nil {
			return ctrl.Result{}, err
		}
	}

	if stressChaos.Status.ObservedGeneration != stressChaos.Generation {
		stressChaos.Status.ObservedGeneration = stressChaos.Generation
		if err := r.Status().Update(ctx, &stressChaos); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch stressChaos.Status.Phase {
	case "":
		return r.initializeChaos(ctx, &stressChaos)
	case chaosv1alpha1.ChaosPhasePending:
		return r.executeChaos(ctx, &stressChaos)
	case chaosv1alpha1.ChaosPhaseRunning:
		return r.monitorChaos(ctx, &stressChaos)
	case chaosv1alpha1.ChaosPhaseCompleted, chaosv1alpha1.ChaosPhaseFailed:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func (r *StressChaosReconciler) initializeChaos(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("stresschaos", stressChaos.Name)
	log.Info("Initializing StressChaos")

	action := actionLabel(stressChaos.Spec.Stressors)
	infrastructure.ChaosExperimentsTotal.WithLabelValues("StressChaos", stressChaos.Namespace, action).Inc()
	infrastructure.ChaosExperimentsActive.WithLabelValues("StressChaos", stressChaos.Namespace, "Pending").Inc()

	stressChaos.Status.Phase = chaosv1alpha1.ChaosPhasePending
	now := metav1.Now()
	stressChaos.Status.StartTime = &now

	condition := metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: stressChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Initialized",
		Message:            "StressChaos initialized successfully",
	}
	stressChaos.Status.Conditions = append(stressChaos.Status.Conditions, condition)

	if err := r.Status().Update(ctx, stressChaos); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(stressChaos, corev1.EventTypeNormal, "Initialized", "StressChaos experiment initialized with stressors: %s", action)
	return ctrl.Result{Requeue: true}, nil
}

func (r *StressChaosReconciler) executeChaos(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("stresschaos", stressChaos.Name)
	log.Info("Executing StressChaos")

	podLister := infrastructure.NewKubernetesPodLister(r.Client)
	selector := domain.NewTargetSelector(
		stressChaos.Spec.Mode,
		stressChaos.Spec.Value,
		stressChaos.Spec.Selector,
		stressChaos.Spec.BlastRadius,
		podLister,
	)

	targets, err := selector.SelectTargets(ctx)
	if err != nil {
		log.Error(err, "Failed to select targets")
		infrastructure.ChaosExecutionErrors.WithLabelValues("StressChaos", stressChaos.Namespace, "target_selection_error").Inc()
		r.Recorder.Eventf(stressChaos, corev1.EventTypeWarning, "ExecutionFailed", "Target selection failed: %v", err)
		return r.updateFailedStatus(ctx, stressChaos, fmt.Sprintf("Target selection failed: %v", err))
	}

	if len(targets) == 0 {
		log.Info("No targets selected")
		return r.updateCompletedStatus(ctx, stressChaos, "No targets matched the selector")
	}

	log.Info("Targets selected", "count", len(targets))
	action := actionLabel(stressChaos.Spec.Stressors)
	infrastructure.ChaosTargetsAffected.WithLabelValues("StressChaos", stressChaos.Namespace, action).Add(float64(len(targets)))

	script := buildStressScript(stressChaos.Spec.Stressors)
	podNames := make([]string, 0, len(targets))
	targetResults := make([]chaosv1alpha1.TargetResult, 0, len(targets))

	for i := range targets {
		pod := targets[i]
		containerName := selectContainer(stressChaos.Spec.ContainerNames, pod)
		podRef := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		pids, execErr := infrastructure.ExecInPod(ctx, r.RestConfig, pod, containerName, []string{"sh", "-c", script})
		if execErr != nil {
			log.Error(execErr, "Failed to exec stress script", "pod", pod.Name)
			infrastructure.ChaosExecutionErrors.WithLabelValues("StressChaos", stressChaos.Namespace, "exec_error").Inc()
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: false, Error: execErr.Error(), AppliedAt: metav1.Now(),
			})
			continue
		}

		log.Info("Stress started", "pod", pod.Name, "pids", pids)
		targetResults = append(targetResults, chaosv1alpha1.TargetResult{
			Name: podRef, Success: true, AppliedAt: metav1.Now(),
		})

		// Store PIDs and container in pod annotations for later cleanup
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[stressPIDsAnnotation] = pids
		pod.Annotations[stressContainerAnnot] = containerName
		if stressChaos.Spec.Stressors.Disk != nil {
			diskPath := stressChaos.Spec.Stressors.Disk.Path
			if diskPath == "" {
				diskPath = "/tmp"
			}
			pod.Annotations[stressDiskPathAnnot] = diskPath
		}
		if err := r.Update(ctx, &pod); err != nil {
			log.Error(err, "Failed to annotate pod with stress PIDs", "pod", pod.Name)
		}

		podNames = append(podNames, podRef)
	}

	stressChaos.Status.AffectedPods = podNames
	stressChaos.Status.Result = infrastructure.BuildResult(targetResults, false)
	stressChaos.Status.Phase = chaosv1alpha1.ChaosPhaseRunning
	infrastructure.ChaosExperimentsActive.WithLabelValues("StressChaos", stressChaos.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsActive.WithLabelValues("StressChaos", stressChaos.Namespace, "Running").Inc()

	now := metav1.Now()
	condition := metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeExecuting,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: stressChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Executing",
		Message:            fmt.Sprintf("Stress started on %d pods", len(podNames)),
	}
	stressChaos.Status.Conditions = append(stressChaos.Status.Conditions, condition)

	if err := r.Status().Update(ctx, stressChaos); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(stressChaos, corev1.EventTypeNormal, "Executing", "Stress started on %d pod(s)", len(podNames))

	if stressChaos.Spec.Duration != "" {
		duration, err := time.ParseDuration(stressChaos.Spec.Duration)
		if err != nil {
			duration = RequeueAfterDuration
		}
		return ctrl.Result{RequeueAfter: duration}, nil
	}

	return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
}

func (r *StressChaosReconciler) monitorChaos(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("stresschaos", stressChaos.Name)
	log.Info("Monitoring StressChaos")

	if stressChaos.Status.StartTime != nil && stressChaos.Spec.Duration != "" {
		duration, err := time.ParseDuration(stressChaos.Spec.Duration)
		if err == nil && time.Since(stressChaos.Status.StartTime.Time) >= duration {
			log.Info("Duration elapsed, recovering stress")
			r.Recorder.Event(stressChaos, corev1.EventTypeNormal, "DurationElapsed", "Experiment duration elapsed, recovering stress")
			r.recoverStress(ctx, stressChaos)
			return r.updateCompletedStatus(ctx, stressChaos, "Duration elapsed - stress stopped")
		}
	}

	return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
}

// recoverStress kills stress processes in all affected pods via exec.
func (r *StressChaosReconciler) recoverStress(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos) {
	log := r.Log.WithValues("stresschaos", stressChaos.Name)

	for _, podRef := range stressChaos.Status.AffectedPods {
		parts := strings.SplitN(podRef, "/", 2)
		if len(parts) != 2 {
			continue
		}

		var pod corev1.Pod
		if err := r.Get(ctx, client.ObjectKey{Namespace: parts[0], Name: parts[1]}, &pod); err != nil {
			log.Info("Pod not found during recovery (may already be gone)", "pod", podRef)
			continue
		}

		pids := pod.Annotations[stressPIDsAnnotation]
		container := pod.Annotations[stressContainerAnnot]
		if container == "" {
			container = selectContainer(nil, pod)
		}

		if pids != "" {
			pidList := strings.Join(strings.Fields(pids), " ")
			// Build cleanup script: kill processes, remove memory and disk artifacts
			cleanupParts := []string{
				fmt.Sprintf("kill -9 %s 2>/dev/null", pidList),
				"rm -f /dev/shm/.chaos_mem* 2>/dev/null",
			}
			if diskPath := pod.Annotations[stressDiskPathAnnot]; diskPath != "" {
				cleanupParts = append(cleanupParts, fmt.Sprintf("rm -f %s/.chaos_disk_fill 2>/dev/null", diskPath))
			}
			cleanupParts = append(cleanupParts, "echo recovered")
			killScript := strings.Join(cleanupParts, "; ")

			out, err := infrastructure.ExecInPod(ctx, r.RestConfig, pod, container, []string{"sh", "-c", killScript})
			if err != nil {
				log.Error(err, "Failed to kill stress processes", "pod", pod.Name)
			} else {
				log.Info("Stress recovered", "pod", pod.Name, "result", out)
			}
		}

		// Clean annotations
		if pod.Annotations != nil {
			delete(pod.Annotations, stressPIDsAnnotation)
			delete(pod.Annotations, stressContainerAnnot)
			delete(pod.Annotations, stressDiskPathAnnot)
			if err := r.Update(ctx, &pod); err != nil {
				log.Error(err, "Failed to clean pod annotations", "pod", pod.Name)
			}
		}
	}
}

func (r *StressChaosReconciler) handleDeletion(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("stresschaos", stressChaos.Name)
	log.Info("Handling deletion")

	if controllerutil.ContainsFinalizer(stressChaos, StressChaosFinalizer) {
		if stressChaos.Status.Phase == chaosv1alpha1.ChaosPhaseRunning {
			r.Recorder.Event(stressChaos, corev1.EventTypeNormal, "Recovering", "Cleaning up stress chaos effects")
			r.recoverStress(ctx, stressChaos)
		}
		controllerutil.RemoveFinalizer(stressChaos, StressChaosFinalizer)
		if err := r.Update(ctx, stressChaos); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *StressChaosReconciler) updateCompletedStatus(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos, message string) (ctrl.Result, error) {
	stressChaos.Status.Phase = chaosv1alpha1.ChaosPhaseCompleted
	stressChaos.Status.Message = message
	now := metav1.Now()
	stressChaos.Status.CompletionTime = &now

	infrastructure.ChaosExperimentsActive.WithLabelValues("StressChaos", stressChaos.Namespace, "Running").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("StressChaos", stressChaos.Namespace, "Completed").Inc()

	action := actionLabel(stressChaos.Spec.Stressors)
	if stressChaos.Status.StartTime != nil {
		duration := now.Time.Sub(stressChaos.Status.StartTime.Time).Seconds()
		infrastructure.ChaosExperimentsDuration.WithLabelValues("StressChaos", stressChaos.Namespace, action).Observe(duration)
	}

	condition := metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeCompleted,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: stressChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Completed",
		Message:            message,
	}
	stressChaos.Status.Conditions = append(stressChaos.Status.Conditions, condition)

	if err := r.Status().Update(ctx, stressChaos); err != nil {
		r.Log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	r.Recorder.Event(stressChaos, corev1.EventTypeNormal, "Completed", message)
	r.saveReport(ctx, stressChaos)
	return ctrl.Result{}, nil
}

func (r *StressChaosReconciler) updateFailedStatus(ctx context.Context, stressChaos *chaosv1alpha1.StressChaos, message string) (ctrl.Result, error) {
	stressChaos.Status.Phase = chaosv1alpha1.ChaosPhaseFailed
	stressChaos.Status.Message = message
	now := metav1.Now()
	stressChaos.Status.CompletionTime = &now

	infrastructure.ChaosExperimentsActive.WithLabelValues("StressChaos", stressChaos.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("StressChaos", stressChaos.Namespace, "Failed").Inc()

	condition := metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeCompleted,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: stressChaos.Generation,
		LastTransitionTime: now,
		Reason:             "Failed",
		Message:            message,
	}
	stressChaos.Status.Conditions = append(stressChaos.Status.Conditions, condition)

	if err := r.Status().Update(ctx, stressChaos); err != nil {
		r.Log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}
	r.Recorder.Event(stressChaos, corev1.EventTypeWarning, "Failed", message)
	r.saveReport(ctx, stressChaos)
	return ctrl.Result{}, nil
}

func (r *StressChaosReconciler) saveReport(ctx context.Context, sc *chaosv1alpha1.StressChaos) {
	if r.Reporter == nil {
		return
	}
	details := map[string]string{
		"mode":     string(sc.Spec.Mode),
		"duration": sc.Spec.Duration,
	}
	if sc.Spec.Value != "" {
		details["value"] = sc.Spec.Value
	}
	if sc.Spec.Stressors.CPU != nil {
		details["cpu.workers"] = strconv.Itoa(sc.Spec.Stressors.CPU.Workers)
		if sc.Spec.Stressors.CPU.Load != nil {
			details["cpu.load"] = strconv.Itoa(*sc.Spec.Stressors.CPU.Load)
		}
	}
	if sc.Spec.Stressors.Memory != nil {
		details["memory.size"] = sc.Spec.Stressors.Memory.Size
		details["memory.workers"] = strconv.Itoa(sc.Spec.Stressors.Memory.Workers)
	}
	if err := r.Reporter.CreateSummaryConfigMap(ctx, infrastructure.ReportInput{
		Namespace:      sc.Namespace,
		ExperimentName: sc.Name,
		ChaosType:      "StressChaos",
		Phase:          string(sc.Status.Phase),
		StartTime:      sc.Status.StartTime,
		CompletionTime: sc.Status.CompletionTime,
		Result:         resultOrFromPods(sc.Status.Result, sc.Status.AffectedPods),
		AffectedPods:   sc.Status.AffectedPods,
		Message:        sc.Status.Message,
		Details:        details,
	}); err != nil {
		r.Log.Error(err, "Failed to save experiment report", "name", sc.Name)
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *StressChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.StressChaos{}).
		Complete(r)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// buildStressScript returns a sh one-liner that starts stress processes fully
// detached from the exec session (all stdio → /dev/null) and prints PIDs to stdout.
// Works with busybox sh (nginx:alpine) — no stress-ng required.
//
// Background processes must redirect ALL stdio so they do not hold any fd of the
// SPDY exec channel open, which would cause StreamWithContext to hang.
func buildStressScript(stressors chaosv1alpha1.StressorsSpec) string {
	var parts []string

	if cpu := stressors.CPU; cpu != nil {
		workers := cpu.Workers
		if workers < 1 {
			workers = 1
		}
		// nohup + full redirect detaches the child from the exec session.
		// Each worker saturates one CPU core (load% requires stress-ng).
		part := fmt.Sprintf(
			"i=0; while [ $i -lt %d ]; do nohup sh -c 'yes >/dev/null 2>/dev/null' </dev/null >/dev/null 2>/dev/null & echo $!; i=$((i+1)); done",
			workers,
		)
		parts = append(parts, part)
	}

	if mem := stressors.Memory; mem != nil {
		sizeMB := parseSizeMB(mem.Size)
		if sizeMB < 1 {
			sizeMB = 32
		}
		for w := 0; w < mem.Workers; w++ {
			// Write zeros to /dev/shm (tmpfs = RAM-backed), then tail -f to keep
			// the process alive so the file stays allocated.
			part := fmt.Sprintf(
				"nohup sh -c 'dd if=/dev/zero of=/dev/shm/.chaos_mem_%d bs=1M count=%d 2>/dev/null; tail -f /dev/null' </dev/null >/dev/null 2>/dev/null & echo $!",
				w, sizeMB,
			)
			parts = append(parts, part)
		}
	}

	if disk := stressors.Disk; disk != nil {
		sizeMB := parseSizeMB(disk.Size)
		if sizeMB < 1 {
			sizeMB = 64
		}
		path := disk.Path
		if path == "" {
			path = "/tmp"
		}
		// Fill disk space with zeros. tail -f keeps the process alive so the
		// file is not cleaned up until we explicitly kill the PID.
		part := fmt.Sprintf(
			"nohup sh -c 'dd if=/dev/zero of=%s/.chaos_disk_fill bs=1M count=%d 2>/dev/null; tail -f /dev/null' </dev/null >/dev/null 2>/dev/null & echo $!",
			path, sizeMB,
		)
		parts = append(parts, part)
	}

	if io := stressors.IO; io != nil {
		workers := io.Workers
		if workers < 1 {
			workers = 1
		}
		bs := "4k"
		if io.Size != "" {
			bs = io.Size
		}
		// Each worker reads from /dev/urandom and writes to /dev/null in a tight loop,
		// generating sustained IO load.
		for w := 0; w < workers; w++ {
			part := fmt.Sprintf(
				"nohup sh -c 'while true; do dd if=/dev/urandom of=/dev/null bs=%s count=1024 2>/dev/null; done' </dev/null >/dev/null 2>/dev/null & echo $!",
				bs,
			)
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, "; ")
}

// parseSizeMB converts strings like "64MB", "1GB", "512KB" to megabytes.
func parseSizeMB(size string) int {
	size = strings.TrimSpace(strings.ToUpper(size))
	for suffix, mult := range map[string]int{"GB": 1024, "MB": 1, "KB": 0} {
		if strings.HasSuffix(size, suffix) {
			n, _ := strconv.Atoi(strings.TrimSuffix(size, suffix))
			if suffix == "KB" {
				return max(1, n/1024)
			}
			return n * mult
		}
	}
	n, _ := strconv.Atoi(size)
	return n
}

// selectContainer returns the container name to exec into.
func selectContainer(containerNames []string, pod corev1.Pod) string {
	if len(containerNames) > 0 {
		return containerNames[0]
	}
	if len(pod.Spec.Containers) > 0 {
		return pod.Spec.Containers[0].Name
	}
	return ""
}

// actionLabel returns the metric action label for a stressors config.
func actionLabel(stressors chaosv1alpha1.StressorsSpec) string {
	hasCPU := stressors.CPU != nil
	hasMem := stressors.Memory != nil
	hasDisk := stressors.Disk != nil
	hasIO := stressors.IO != nil

	count := 0
	if hasCPU {
		count++
	}
	if hasMem {
		count++
	}
	if hasDisk {
		count++
	}
	if hasIO {
		count++
	}

	if count > 1 {
		return "combined-stress"
	}

	switch {
	case hasCPU:
		return "cpu-stress"
	case hasMem:
		return "memory-stress"
	case hasDisk:
		return "disk-fill"
	case hasIO:
		return "io-stress"
	default:
		return "stress"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
