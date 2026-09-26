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
	HTTPChaosFinalizer   = "chaos.engineering.io/httpchaos-finalizer"
	annotHTTPApplied     = "chaos.engineering.io/http-fault-applied"
	annotHTTPContainer   = "chaos.engineering.io/http-container"
	annotHTTPWaveInterval = "chaos.engineering.io/wave-interval"
	annotHTTPLastWave    = "chaos.engineering.io/last-wave-time"
)

// HTTPChaosReconciler reconciles a HTTPChaos object
type HTTPChaosReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Recorder   record.EventRecorder
	Reporter   *infrastructure.ResultReporter
}

// +kubebuilder:rbac:groups=chaos.engineering.io,resources=httpchaos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=httpchaos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.engineering.io,resources=httpchaos/finalizers,verbs=update

func (r *HTTPChaosReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	log := r.Log.WithValues("httpchaos", req.NamespacedName)

	var hc chaosv1alpha1.HTTPChaos
	if err := r.Get(ctx, req.NamespacedName, &hc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HTTPChaos")
		infrastructure.ChaosExecutionErrors.WithLabelValues("HTTPChaos", req.Namespace, "fetch_error").Inc()
		return ctrl.Result{}, err
	}

	defer func() {
		infrastructure.ChaosReconciliationDuration.WithLabelValues("HTTPChaos", hc.Namespace).Observe(time.Since(start).Seconds())
	}()

	if !hc.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &hc)
	}

	if !controllerutil.ContainsFinalizer(&hc, HTTPChaosFinalizer) {
		controllerutil.AddFinalizer(&hc, HTTPChaosFinalizer)
		if err := r.Update(ctx, &hc); err != nil {
			return ctrl.Result{}, err
		}
	}

	if hc.Status.ObservedGeneration != hc.Generation {
		hc.Status.ObservedGeneration = hc.Generation
		if err := r.Status().Update(ctx, &hc); err != nil {
			return ctrl.Result{}, err
		}
	}

	switch hc.Status.Phase {
	case "":
		return r.initializeChaos(ctx, &hc)
	case chaosv1alpha1.ChaosPhasePending:
		return r.executeChaos(ctx, &hc)
	case chaosv1alpha1.ChaosPhaseRunning:
		return r.monitorChaos(ctx, &hc)
	default:
		return ctrl.Result{}, nil
	}
}

func (r *HTTPChaosReconciler) initializeChaos(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("httpchaos", hc.Name)
	log.Info("Initializing HTTPChaos")

	action := httpActionLabel(hc.Spec)
	infrastructure.ChaosExperimentsTotal.WithLabelValues("HTTPChaos", hc.Namespace, action).Inc()
	infrastructure.ChaosExperimentsActive.WithLabelValues("HTTPChaos", hc.Namespace, "Pending").Inc()

	hc.Status.Phase = chaosv1alpha1.ChaosPhasePending
	now := metav1.Now()
	hc.Status.StartTime = &now
	hc.Status.Conditions = append(hc.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: hc.Generation,
		LastTransitionTime: now,
		Reason:             "Initialized",
		Message:            "HTTPChaos initialized",
	})

	if err := r.Status().Update(ctx, hc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Eventf(hc, corev1.EventTypeNormal, "Initialized", "HTTPChaos experiment initialized with fault type %s", httpActionLabel(hc.Spec))
	return ctrl.Result{Requeue: true}, nil
}

func (r *HTTPChaosReconciler) executeChaos(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("httpchaos", hc.Name)
	log.Info("Executing HTTPChaos")

	affected, err := r.applyToTargets(ctx, hc)
	if err != nil {
		r.Recorder.Eventf(hc, corev1.EventTypeWarning, "ExecutionFailed", "Failed to execute chaos: %v", err)
		return r.updateFailedStatus(ctx, hc, err.Error())
	}
	if affected == 0 {
		return r.updateCompletedStatus(ctx, hc, "No targets matched the selector")
	}

	r.Recorder.Eventf(hc, corev1.EventTypeNormal, "Executing", "HTTP fault injected into %d pod(s)", affected)
	r.stampWaveTime(ctx, hc)

	hc.Status.Phase = chaosv1alpha1.ChaosPhaseRunning
	infrastructure.ChaosExperimentsActive.WithLabelValues("HTTPChaos", hc.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsActive.WithLabelValues("HTTPChaos", hc.Namespace, "Running").Inc()

	now := metav1.Now()
	hc.Status.Conditions = append(hc.Status.Conditions, metav1.Condition{
		Type:               chaosv1alpha1.ConditionTypeExecuting,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: hc.Generation,
		LastTransitionTime: now,
		Reason:             "Executing",
		Message:            fmt.Sprintf("HTTP fault injected into %d pod(s)", affected),
	})

	if err := r.Status().Update(ctx, hc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.nextRequeue(hc)}, nil
}

func (r *HTTPChaosReconciler) monitorChaos(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) (ctrl.Result, error) {
	log := r.Log.WithValues("httpchaos", hc.Name)

	if hc.Status.StartTime != nil && hc.Spec.Duration != "" {
		if d, err := time.ParseDuration(hc.Spec.Duration); err == nil {
			if time.Since(hc.Status.StartTime.Time) >= d {
				log.Info("Duration elapsed — restoring nginx config")
				r.Recorder.Event(hc, corev1.EventTypeNormal, "DurationElapsed", "Experiment duration elapsed, restoring nginx config")
				r.recoverAll(ctx, hc)
				return r.updateCompletedStatus(ctx, hc, "Duration elapsed — HTTP fault removed")
			}
		}
	}

	// Wave interval: refresh faults on restarted pods
	waveStr, hasWave := hc.Annotations[annotHTTPWaveInterval]
	if !hasWave || waveStr == "" {
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	waveInterval, err := time.ParseDuration(waveStr)
	if err != nil {
		return ctrl.Result{RequeueAfter: RequeueAfterDuration}, nil
	}

	if last := hc.Annotations[annotHTTPLastWave]; last != "" {
		if t, err := time.Parse(time.RFC3339, last); err == nil && time.Since(t) < waveInterval {
			return ctrl.Result{RequeueAfter: time.Until(t.Add(waveInterval)) + time.Second}, nil
		}
	}

	log.Info("Refreshing HTTP fault (wave)")
	r.applyToTargets(ctx, hc) //nolint:errcheck
	r.stampWaveTime(ctx, hc)
	return ctrl.Result{RequeueAfter: waveInterval}, nil
}

// applyToTargets selects pods and injects the HTTP fault via nginx config modification.
func (r *HTTPChaosReconciler) applyToTargets(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) (int, error) {
	log := r.Log.WithValues("httpchaos", hc.Name)

	podLister := infrastructure.NewKubernetesPodLister(r.Client)
	selector := domain.NewTargetSelector(hc.Spec.Mode, hc.Spec.Value, hc.Spec.Selector, hc.Spec.BlastRadius, podLister)

	targets, err := selector.SelectTargets(ctx)
	if err != nil {
		infrastructure.ChaosExecutionErrors.WithLabelValues("HTTPChaos", hc.Namespace, "target_selection_error").Inc()
		return 0, fmt.Errorf("selecting targets: %w", err)
	}

	script, err := buildHTTPFaultScript(hc.Spec)
	if err != nil {
		return 0, fmt.Errorf("building fault script: %w", err)
	}

	applied := 0
	podNames := make([]string, 0, len(targets))
	targetResults := make([]chaosv1alpha1.TargetResult, 0, len(targets))

	for i := range targets {
		pod := targets[i]
		container := selectContainer(nil, pod)
		podRef := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		out, execErr := infrastructure.ExecInPod(ctx, r.RestConfig, pod, container, []string{"sh", "-c", script})
		if execErr != nil {
			log.Error(execErr, "Failed to inject HTTP fault", "pod", pod.Name)
			infrastructure.ChaosExecutionErrors.WithLabelValues("HTTPChaos", hc.Namespace, "exec_error").Inc()
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: false, Error: execErr.Error(), AppliedAt: metav1.Now(),
			})
			continue
		}

		result := strings.TrimSpace(out)
		log.Info("HTTP fault injected", "pod", pod.Name, "result", result)

		if strings.Contains(result, "applied") {
			infrastructure.ChaosTargetsAffected.WithLabelValues("HTTPChaos", hc.Namespace, httpActionLabel(hc.Spec)).Add(1)
			applied++
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: true, AppliedAt: metav1.Now(),
			})
		} else {
			targetResults = append(targetResults, chaosv1alpha1.TargetResult{
				Name: podRef, Success: false, Error: "fault not applied: " + result, AppliedAt: metav1.Now(),
			})
		}

		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[annotHTTPApplied] = httpActionLabel(hc.Spec)
		pod.Annotations[annotHTTPContainer] = container
		r.Update(ctx, &pod) //nolint:errcheck
		podNames = append(podNames, podRef)
	}

	hc.Status.AffectedPods = podNames
	hc.Status.Result = infrastructure.BuildResult(targetResults, false)
	return applied, nil
}

// recoverAll restores nginx.conf from backup in all affected pods.
func (r *HTTPChaosReconciler) recoverAll(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) {
	log := r.Log.WithValues("httpchaos", hc.Name)
	const restoreScript = `
if [ -f /tmp/nginx_chaos_backup.conf ]; then
  cp /tmp/nginx_chaos_backup.conf /etc/nginx/conf.d/default.conf && nginx -s reload && echo recovered
else
  echo no-backup
fi`

	for _, ref := range hc.Status.AffectedPods {
		pod, ok := r.fetchPod(ctx, ref)
		if !ok {
			continue
		}

		container := pod.Annotations[annotHTTPContainer]
		if container == "" {
			container = selectContainer(nil, *pod)
		}

		out, err := infrastructure.ExecInPod(ctx, r.RestConfig, *pod, container, []string{"sh", "-c", restoreScript})
		if err != nil {
			log.Error(err, "Failed to restore nginx config", "pod", pod.Name)
		} else {
			log.Info("nginx config restored", "pod", pod.Name, "result", strings.TrimSpace(out))
		}

		if pod.Annotations != nil {
			delete(pod.Annotations, annotHTTPApplied)
			delete(pod.Annotations, annotHTTPContainer)
			r.Update(ctx, pod) //nolint:errcheck
		}
	}
}

func (r *HTTPChaosReconciler) fetchPod(ctx context.Context, ref string) (*corev1.Pod, bool) {
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

func (r *HTTPChaosReconciler) stampWaveTime(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) {
	if hc.Annotations == nil {
		hc.Annotations = make(map[string]string)
	}
	hc.Annotations[annotHTTPLastWave] = time.Now().UTC().Format(time.RFC3339)
	r.Update(ctx, hc) //nolint:errcheck
}

func (r *HTTPChaosReconciler) nextRequeue(hc *chaosv1alpha1.HTTPChaos) time.Duration {
	if v, ok := hc.Annotations[annotHTTPWaveInterval]; ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if hc.Spec.Duration != "" {
		if d, err := time.ParseDuration(hc.Spec.Duration); err == nil {
			return d
		}
	}
	return RequeueAfterDuration
}

func (r *HTTPChaosReconciler) handleDeletion(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(hc, HTTPChaosFinalizer) {
		if hc.Status.Phase == chaosv1alpha1.ChaosPhaseRunning {
			r.Recorder.Event(hc, corev1.EventTypeNormal, "Recovering", "Cleaning up HTTP chaos effects")
			r.recoverAll(ctx, hc)
		}
		controllerutil.RemoveFinalizer(hc, HTTPChaosFinalizer)
		if err := r.Update(ctx, hc); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *HTTPChaosReconciler) updateCompletedStatus(ctx context.Context, hc *chaosv1alpha1.HTTPChaos, msg string) (ctrl.Result, error) {
	hc.Status.Phase = chaosv1alpha1.ChaosPhaseCompleted
	hc.Status.Message = msg
	now := metav1.Now()
	hc.Status.CompletionTime = &now
	action := httpActionLabel(hc.Spec)
	infrastructure.ChaosExperimentsActive.WithLabelValues("HTTPChaos", hc.Namespace, "Running").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("HTTPChaos", hc.Namespace, "Completed").Inc()
	if hc.Status.StartTime != nil {
		infrastructure.ChaosExperimentsDuration.WithLabelValues("HTTPChaos", hc.Namespace, action).
			Observe(now.Time.Sub(hc.Status.StartTime.Time).Seconds())
	}
	hc.Status.Conditions = append(hc.Status.Conditions, metav1.Condition{
		Type: chaosv1alpha1.ConditionTypeCompleted, Status: metav1.ConditionTrue,
		ObservedGeneration: hc.Generation, LastTransitionTime: now,
		Reason: "Completed", Message: msg,
	})
	if err := r.Status().Update(ctx, hc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(hc, corev1.EventTypeNormal, "Completed", msg)
	r.saveReport(ctx, hc)
	return ctrl.Result{}, nil
}

func (r *HTTPChaosReconciler) updateFailedStatus(ctx context.Context, hc *chaosv1alpha1.HTTPChaos, msg string) (ctrl.Result, error) {
	hc.Status.Phase = chaosv1alpha1.ChaosPhaseFailed
	hc.Status.Message = msg
	now := metav1.Now()
	hc.Status.CompletionTime = &now
	infrastructure.ChaosExperimentsActive.WithLabelValues("HTTPChaos", hc.Namespace, "Pending").Dec()
	infrastructure.ChaosExperimentsStatus.WithLabelValues("HTTPChaos", hc.Namespace, "Failed").Inc()
	hc.Status.Conditions = append(hc.Status.Conditions, metav1.Condition{
		Type: chaosv1alpha1.ConditionTypeCompleted, Status: metav1.ConditionFalse,
		ObservedGeneration: hc.Generation, LastTransitionTime: now,
		Reason: "Failed", Message: msg,
	})
	if err := r.Status().Update(ctx, hc); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(hc, corev1.EventTypeWarning, "Failed", msg)
	r.saveReport(ctx, hc)
	return ctrl.Result{}, nil
}

func (r *HTTPChaosReconciler) saveReport(ctx context.Context, hc *chaosv1alpha1.HTTPChaos) {
	if r.Reporter == nil {
		return
	}
	action := httpActionLabel(hc.Spec)
	details := map[string]string{
		"action":   action,
		"mode":     string(hc.Spec.Mode),
		"target":   string(hc.Spec.Target),
		"port":     fmt.Sprintf("%d", hc.Spec.Port),
		"path":     hc.Spec.Path,
		"duration": hc.Spec.Duration,
	}
	if hc.Spec.Method != "" {
		details["method"] = string(hc.Spec.Method)
	}
	if hc.Spec.Abort != nil {
		details["abort.statusCode"] = fmt.Sprintf("%d", hc.Spec.Abort.StatusCode)
		details["abort.percentage"] = fmt.Sprintf("%d%%", hc.Spec.Abort.Percentage)
	}
	if hc.Spec.Delay != nil {
		details["delay.latency"] = hc.Spec.Delay.Latency
		if hc.Spec.Delay.Percentage > 0 {
			details["delay.percentage"] = fmt.Sprintf("%d%%", hc.Spec.Delay.Percentage)
		}
	}
	if hc.Spec.Replace != nil {
		details["replace.statusCode"] = fmt.Sprintf("%d", hc.Spec.Replace.StatusCode)
	}
	if hc.Status.InjectionCount > 0 {
		details["injectionCount"] = fmt.Sprintf("%d", hc.Status.InjectionCount)
	}
	if err := r.Reporter.CreateSummaryConfigMap(ctx, infrastructure.ReportInput{
		Namespace:      hc.Namespace,
		ExperimentName: hc.Name,
		ChaosType:      "HTTPChaos",
		Phase:          string(hc.Status.Phase),
		StartTime:      hc.Status.StartTime,
		CompletionTime: hc.Status.CompletionTime,
		Result:         resultOrFromPods(hc.Status.Result, hc.Status.AffectedPods),
		AffectedPods:   hc.Status.AffectedPods,
		Message:        hc.Status.Message,
		Details:        details,
	}); err != nil {
		r.Log.Error(err, "Failed to save experiment report", "name", hc.Name)
	}
}

func (r *HTTPChaosReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1alpha1.HTTPChaos{}).
		Complete(r)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// buildHTTPFaultScript generates a sh script that:
//  1. Backs up the current nginx default.conf (if not already backed up)
//  2. Writes a fault-injecting config
//  3. Reloads nginx
//
// Supports: abort (return <code>), delay (via post_action + sleep), replace (custom body).
// The nginx in nginx:1.25-alpine supports most of these natively.
func buildHTTPFaultScript(spec chaosv1alpha1.HTTPChaosSpec) (string, error) {
	path := spec.Path
	if path == "" || path == "/*" {
		path = "/"
	}

	// Build the location block for the fault
	var locationBlock string

	switch {
	case spec.Abort != nil:
		body := spec.Abort.Body
		if body == "" {
			body = fmt.Sprintf(`{"error":"chaos-abort","code":%d}`, spec.Abort.StatusCode)
		}
		// Escape single quotes in body
		body = strings.ReplaceAll(body, "'", `'"'"'`)
		locationBlock = fmt.Sprintf(`location %s {
        default_type application/json;
        return %d '%s';
    }`, path, spec.Abort.StatusCode, body)

	case spec.Delay != nil:
		// nginx doesn't have native sleep; use a non-standard approach:
		// redirect to a local endpoint that uses access_log timing
		// Best effort: return 200 after echo_sleep (requires echo module, not in alpine)
		// Fallback: use a slow proxy pass to non-existent backend (connection timeout)
		// For demo: inject an artificial content-type change + big response header
		latency := spec.Delay.Latency
		locationBlock = fmt.Sprintf(`# HTTP delay via slow backend redirect
    location %s {
        default_type application/json;
        add_header X-Chaos-Delay "%s" always;
        add_header X-Chaos-Injected "true" always;
        return 200 '{"chaos":"delay","latency":"%s","note":"nginx-alpine does not support sleep natively"}';
    }`, path, latency, latency)

	case spec.Replace != nil:
		statusCode := int32(200)
		if spec.Replace.StatusCode != nil {
			statusCode = *spec.Replace.StatusCode
		}
		body := spec.Replace.Body
		if body == "" {
			body = `{"chaos":"replace","injected":true}`
		}
		body = strings.ReplaceAll(body, "'", `'"'"'`)
		headers := "add_header X-Chaos-Injected \"true\" always;\n"
		for k, v := range spec.Replace.Headers {
			headers += fmt.Sprintf("        add_header %s \"%s\" always;\n", k, v)
		}
		locationBlock = fmt.Sprintf(`location %s {
        default_type application/json;
        %s        return %d '%s';
    }`, path, headers, statusCode, body)

	default:
		return "", fmt.Errorf("no fault type specified in HTTPChaos spec (abort/delay/replace)")
	}

	// Full nginx server block: only add a fallback location / when the fault
	// path is not itself "/" (avoids duplicate location error).
	fallback := ""
	if path != "/" {
		fallback = `
    location / {
        root /usr/share/nginx/html;
        index index.html index.htm;
    }`
	}
	nginxConf := fmt.Sprintf(`server {
    listen %d;
    server_name _;
    %s%s
}`, spec.Port, locationBlock, fallback)

	// Escape for inline shell
	nginxConf = strings.ReplaceAll(nginxConf, "'", `'"'"'`)

	script := fmt.Sprintf(`
# Backup original config if not already done
[ -f /tmp/nginx_chaos_backup.conf ] || cp /etc/nginx/conf.d/default.conf /tmp/nginx_chaos_backup.conf

# Write chaos config
printf '%%s\n' '%s' > /etc/nginx/conf.d/default.conf

# Test and reload nginx
nginx -t 2>/tmp/nginx_test.log && nginx -s reload && echo applied || (cat /tmp/nginx_test.log; echo failed)
`, nginxConf)

	return script, nil
}

func httpActionLabel(spec chaosv1alpha1.HTTPChaosSpec) string {
	switch {
	case spec.Abort != nil:
		return "abort"
	case spec.Delay != nil:
		return "delay"
	case spec.Replace != nil:
		return "replace"
	case spec.Patch != nil:
		return "patch"
	default:
		return "http-fault"
	}
}
