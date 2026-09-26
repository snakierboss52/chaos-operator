package infrastructure

import (
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
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
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
)

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
	)
}
