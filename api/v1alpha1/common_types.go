package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChaosMode defines the mode to select target pods
// +kubebuilder:validation:Enum=one;all;fixed;fixed-percent;random-max-percent
type ChaosMode string

const (
	// OnePodMode selects exactly one pod
	OnePodMode ChaosMode = "one"
	// AllMode selects all pods that match the selector
	AllMode ChaosMode = "all"
	// FixedMode selects a fixed number of pods
	FixedMode ChaosMode = "fixed"
	// FixedPercentMode selects a fixed percentage of pods
	FixedPercentMode ChaosMode = "fixed-percent"
	// RandomMaxPercentMode selects up to a random percentage of pods
	RandomMaxPercentMode ChaosMode = "random-max-percent"
)

// PodSelector defines how to select target pods
type PodSelector struct {
	// Namespaces is a list of target namespaces
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// LabelSelectors is a map of {key,value} pairs to match labels
	// +optional
	LabelSelectors map[string]string `json:"labelSelectors,omitempty"`

	// FieldSelectors is a map of {key,value} pairs to match fields
	// +optional
	FieldSelectors map[string]string `json:"fieldSelectors,omitempty"`

	// PodPhaseSelectors restricts the phase of pods
	// +optional
	PodPhaseSelectors []string `json:"podPhaseSelectors,omitempty"`

	// NodeSelectors is a map of {key,value} pairs to match node labels
	// +optional
	NodeSelectors map[string]string `json:"nodeSelectors,omitempty"`

	// Pods is a list of pod names to select
	// +optional
	Pods []string `json:"pods,omitempty"`
}

// BlastRadiusControl defines limits for chaos experiments
type BlastRadiusControl struct {
	// MaxPods is the maximum number of pods that can be affected
	// +optional
	MaxPods *int32 `json:"maxPods,omitempty"`

	// MaxPercentage is the maximum percentage of pods that can be affected (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	MaxPercentage *int32 `json:"maxPercentage,omitempty"`
}

// ChaosStatus defines the observed state of a Chaos experiment
type ChaosStatus struct {
	// Phase represents the current phase of the chaos experiment
	// +optional
	Phase ChaosPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the chaos experiment
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// AffectedPods is the list of pods currently affected by the chaos
	// +optional
	AffectedPods []string `json:"affectedPods,omitempty"`

	// StartTime is when the chaos experiment started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the chaos experiment completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration reflects the generation most recently observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Result contains the detailed experiment result
	// +optional
	Result *ExperimentResult `json:"result,omitempty"`
}

// ExperimentResult captures detailed results of a chaos experiment
type ExperimentResult struct {
	// TotalTargets is how many pods were targeted
	TotalTargets int `json:"totalTargets"`

	// SuccessfulTargets is how many pods were successfully affected
	SuccessfulTargets int `json:"successfulTargets"`

	// FailedTargets is how many pods failed to be affected
	FailedTargets int `json:"failedTargets"`

	// TargetResults contains per-pod results
	// +optional
	TargetResults []TargetResult `json:"targetResults,omitempty"`

	// BlastRadiusLimited indicates if blast radius controls reduced the target set
	// +optional
	BlastRadiusLimited bool `json:"blastRadiusLimited,omitempty"`
}

// TargetResult captures the result for a single target pod
type TargetResult struct {
	// Name is the pod name (namespace/name)
	Name string `json:"name"`

	// Success indicates if the chaos was applied successfully
	Success bool `json:"success"`

	// Error contains the error message if the chaos failed
	// +optional
	Error string `json:"error,omitempty"`

	// AppliedAt is when the chaos was applied to this pod
	AppliedAt metav1.Time `json:"appliedAt"`
}

// ChaosPhase defines the phase of a chaos experiment
// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;Stopped
type ChaosPhase string

const (
	// ChaosPhasePending indicates the chaos is waiting to start
	ChaosPhasePending ChaosPhase = "Pending"
	// ChaosPhaseRunning indicates the chaos is currently running
	ChaosPhaseRunning ChaosPhase = "Running"
	// ChaosPhaseCompleted indicates the chaos completed successfully
	ChaosPhaseCompleted ChaosPhase = "Completed"
	// ChaosPhaseFailed indicates the chaos failed
	ChaosPhaseFailed ChaosPhase = "Failed"
	// ChaosPhaseStopped indicates the chaos was manually stopped
	ChaosPhaseStopped ChaosPhase = "Stopped"
)

// SchedulerSpec defines when to run the chaos experiment
type SchedulerSpec struct {
	// Cron defines a cron expression for scheduled chaos
	// +optional
	Cron string `json:"cron,omitempty"`
}

// Condition types for chaos experiments
const (
	// ConditionTypeReady indicates whether the chaos is ready to execute
	ConditionTypeReady = "Ready"
	// ConditionTypeExecuting indicates whether the chaos is currently executing
	ConditionTypeExecuting = "Executing"
	// ConditionTypeCompleted indicates whether the chaos has completed
	ConditionTypeCompleted = "Completed"
)
