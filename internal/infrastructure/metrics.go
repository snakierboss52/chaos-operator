package infrastructure

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ChaosExperimentsTotal tracks total number of chaos experiments
	ChaosExperimentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaos_experiments_total",
			Help: "Total number of chaos experiments created",
		},
		[]string{"type", "namespace", "action"},
	)

	// ChaosExperimentsActive tracks currently active experiments
	ChaosExperimentsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaos_experiments_active",
			Help: "Number of currently active chaos experiments",
		},
		[]string{"type", "namespace", "phase"},
	)

	// ChaosExperimentsDuration tracks duration of experiments
	ChaosExperimentsDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chaos_experiments_duration_seconds",
			Help:    "Duration of chaos experiments in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 13), // 1s to ~68min
		},
		[]string{"type", "namespace", "action"},
	)

	// ChaosTargetsAffected tracks number of targets affected
	ChaosTargetsAffected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaos_targets_affected_total",
			Help: "Total number of targets (pods) affected by chaos",
		},
		[]string{"type", "namespace", "action"},
	)

	// ChaosExperimentsStatus tracks experiment status changes
	ChaosExperimentsStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaos_experiments_status_total",
			Help: "Total number of experiments by final status",
		},
		[]string{"type", "namespace", "status"},
	)

	// ChaosBlastRadiusLimit tracks when blast radius limits are hit
	ChaosBlastRadiusLimit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaos_blast_radius_limit_total",
			Help: "Number of times blast radius limits prevented full execution",
		},
		[]string{"type", "namespace"},
	)

	// ChaosExecutionErrors tracks execution errors
	ChaosExecutionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chaos_execution_errors_total",
			Help: "Total number of chaos execution errors",
		},
		[]string{"type", "namespace", "error_type"},
	)

	// ChaosReconciliationDuration tracks reconciliation loop duration
	ChaosReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "chaos_reconciliation_duration_seconds",
			Help:    "Duration of chaos reconciliation loops",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"type", "namespace"},
	)

	// ChaosExperimentInfo is a persisted-resource view rebuilt from CR status on reconcile.
	ChaosExperimentInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaos_experiment_info",
			Help: "Current chaos experiment resources and their observed state",
		},
		[]string{"type", "namespace", "experiment", "action", "mode", "phase", "reason"},
	)

	// ChaosExperimentRuntime tracks elapsed or completed duration per experiment.
	ChaosExperimentRuntime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaos_experiment_runtime_seconds",
			Help: "Elapsed runtime for active experiments and final runtime for completed experiments",
		},
		[]string{"type", "namespace", "experiment", "action", "phase"},
	)

	// ChaosExperimentAffectedPod identifies pods selected/affected by each experiment.
	ChaosExperimentAffectedPod = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "chaos_experiment_affected_pod",
			Help: "Pods recorded as affected by each chaos experiment",
		},
		[]string{"type", "namespace", "experiment", "pod", "action"},
	)
)

var experimentMetricState = struct {
	sync.Mutex
	info map[string]prometheus.Labels
	pods map[string][]prometheus.Labels
}{info: make(map[string]prometheus.Labels), pods: make(map[string][]prometheus.Labels)}

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		ChaosExperimentsTotal,
		ChaosExperimentsActive,
		ChaosExperimentsDuration,
		ChaosTargetsAffected,
		ChaosExperimentsStatus,
		ChaosBlastRadiusLimit,
		ChaosExecutionErrors,
		ChaosReconciliationDuration,
		ChaosExperimentInfo,
		ChaosExperimentRuntime,
		ChaosExperimentAffectedPod,
	)
}

// ObserveChaosExperiment refreshes gauges from the authoritative Kubernetes resource status.
func ObserveChaosExperiment(experimentType, namespace, name, action, mode, phase, reason string, startedAt, completedAt time.Time, affectedPods []string) {
	key := experimentType + "/" + namespace + "/" + name
	infoLabels := prometheus.Labels{
		"type": experimentType, "namespace": namespace, "experiment": name,
		"action": action, "mode": mode, "phase": phase, "reason": reason,
	}
	runtimeLabels := prometheus.Labels{
		"type": experimentType, "namespace": namespace, "experiment": name,
		"action": action, "phase": phase,
	}
	experimentMetricState.Lock()
	defer experimentMetricState.Unlock()
	if previous, ok := experimentMetricState.info[key]; ok {
		ChaosExperimentInfo.Delete(previous)
		ChaosExperimentRuntime.Delete(prometheus.Labels{
			"type": previous["type"], "namespace": previous["namespace"], "experiment": previous["experiment"],
			"action": previous["action"], "phase": previous["phase"],
		})
		if isActiveExperimentPhase(previous["phase"]) && previous["phase"] != phase {
			ChaosExperimentsActive.WithLabelValues(experimentType, namespace, previous["phase"]).Dec()
		}
	}
	ChaosExperimentInfo.With(infoLabels).Set(1)
	if isActiveExperimentPhase(phase) && (experimentMetricState.info[key] == nil || experimentMetricState.info[key]["phase"] != phase) {
		ChaosExperimentsActive.WithLabelValues(experimentType, namespace, phase).Inc()
	}
	if !startedAt.IsZero() {
		end := time.Now()
		if !completedAt.IsZero() {
			end = completedAt
		}
		ChaosExperimentRuntime.With(runtimeLabels).Set(end.Sub(startedAt).Seconds())
	}
	experimentMetricState.info[key] = infoLabels
	for _, labels := range experimentMetricState.pods[key] {
		ChaosExperimentAffectedPod.Delete(labels)
	}
	updatedPods := make([]prometheus.Labels, 0, len(affectedPods))
	for _, pod := range affectedPods {
		if pod == "" {
			continue
		}
		labels := prometheus.Labels{
			"type": experimentType, "namespace": namespace, "experiment": name,
			"pod": pod, "action": action,
		}
		ChaosExperimentAffectedPod.With(labels).Set(1)
		updatedPods = append(updatedPods, labels)
	}
	experimentMetricState.pods[key] = updatedPods
}

// RemoveChaosExperiment removes per-resource series when its CR enters deletion.
func RemoveChaosExperiment(experimentType, namespace, name string) {
	key := experimentType + "/" + namespace + "/" + name
	experimentMetricState.Lock()
	defer experimentMetricState.Unlock()
	if previous, ok := experimentMetricState.info[key]; ok {
		ChaosExperimentInfo.Delete(previous)
		ChaosExperimentRuntime.Delete(prometheus.Labels{
			"type": previous["type"], "namespace": previous["namespace"], "experiment": previous["experiment"],
			"action": previous["action"], "phase": previous["phase"],
		})
		if isActiveExperimentPhase(previous["phase"]) {
			ChaosExperimentsActive.WithLabelValues(previous["type"], previous["namespace"], previous["phase"]).Dec()
		}
		delete(experimentMetricState.info, key)
	}
	for _, labels := range experimentMetricState.pods[key] {
		ChaosExperimentAffectedPod.Delete(labels)
	}
	delete(experimentMetricState.pods, key)
}

func isActiveExperimentPhase(phase string) bool {
	return phase == "Pending" || phase == "Running"
}
