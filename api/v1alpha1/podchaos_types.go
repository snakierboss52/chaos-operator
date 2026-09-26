package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodChaosSpec defines the desired state of PodChaos
type PodChaosSpec struct {
	// Mode defines the mode to select target pods
	// +kubebuilder:validation:Required
	Mode ChaosMode `json:"mode"`

	// Value provides a parameter for the mode
	// For 'fixed' mode, this is the number of pods
	// For 'fixed-percent' and 'random-max-percent', this is the percentage
	// +optional
	Value string `json:"value,omitempty"`

	// Selector defines how to select target pods
	// +kubebuilder:validation:Required
	Selector PodSelector `json:"selector"`

	// Action defines the action to take on the target pods
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=pod-kill;pod-failure;container-kill
	Action PodChaosAction `json:"action"`

	// Duration defines how long the chaos will last
	// Examples: "30s", "5m", "1h"
	// +optional
	Duration string `json:"duration,omitempty"`

	// GracePeriod defines the grace period before terminating the pod (in seconds)
	// Only applicable for pod-kill action
	// +optional
	GracePeriod *int64 `json:"gracePeriod,omitempty"`

	// Force indicates whether to force kill the pod without grace period
	// Only applicable for pod-kill action
	// +optional
	Force bool `json:"force,omitempty"`

	// ContainerNames specifies the containers to kill
	// Only applicable for container-kill action
	// +optional
	ContainerNames []string `json:"containerNames,omitempty"`

	// Scheduler defines when to run the chaos experiment
	// +optional
	Scheduler *SchedulerSpec `json:"scheduler,omitempty"`

	// BlastRadius defines limits for the chaos experiment
	// +optional
	BlastRadius *BlastRadiusControl `json:"blastRadius,omitempty"`
}

// PodChaosAction defines the action type for PodChaos
type PodChaosAction string

const (
	// PodKillAction kills the pod
	PodKillAction PodChaosAction = "pod-kill"
	// PodFailureAction makes the pod fail
	PodFailureAction PodChaosAction = "pod-failure"
	// ContainerKillAction kills specific containers
	ContainerKillAction PodChaosAction = "container-kill"
)

// PodChaosStatus defines the observed state of PodChaos
type PodChaosStatus struct {
	ChaosStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pc
// +kubebuilder:printcolumn:name="Action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PodChaos is the Schema for the podchaos API
type PodChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodChaosSpec   `json:"spec,omitempty"`
	Status PodChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodChaosList contains a list of PodChaos
type PodChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodChaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodChaos{}, &PodChaosList{})
}
