package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkChaosSpec defines the desired state of NetworkChaos
type NetworkChaosSpec struct {
	// Mode defines the mode to select target pods
	// +kubebuilder:validation:Required
	Mode ChaosMode `json:"mode"`

	// Value provides a parameter for the mode
	// +optional
	Value string `json:"value,omitempty"`

	// Selector defines how to select target pods
	// +kubebuilder:validation:Required
	Selector PodSelector `json:"selector"`

	// Action defines the network chaos action
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=delay;loss;duplicate;corrupt;partition;bandwidth
	Action NetworkChaosAction `json:"action"`

	// Duration defines how long the chaos will last
	// +optional
	Duration string `json:"duration,omitempty"`

	// Direction defines the traffic direction to apply chaos
	// +kubebuilder:validation:Enum=from;to;both
	// +optional
	Direction NetworkDirection `json:"direction,omitempty"`

	// Target defines the target of the network chaos
	// If not specified, all network traffic is affected
	// +optional
	Target *NetworkTarget `json:"target,omitempty"`

	// Delay specifies the delay parameters
	// Only applicable when action is 'delay'
	// +optional
	Delay *DelaySpec `json:"delay,omitempty"`

	// Loss specifies the packet loss parameters
	// Only applicable when action is 'loss'
	// +optional
	Loss *LossSpec `json:"loss,omitempty"`

	// Duplicate specifies the packet duplication parameters
	// Only applicable when action is 'duplicate'
	// +optional
	Duplicate *DuplicateSpec `json:"duplicate,omitempty"`

	// Corrupt specifies the packet corruption parameters
	// Only applicable when action is 'corrupt'
	// +optional
	Corrupt *CorruptSpec `json:"corrupt,omitempty"`

	// Bandwidth specifies the bandwidth limitation parameters
	// Only applicable when action is 'bandwidth'
	// +optional
	Bandwidth *BandwidthSpec `json:"bandwidth,omitempty"`

	// ExternalTargets defines external hosts/IPs to apply chaos to
	// +optional
	ExternalTargets []string `json:"externalTargets,omitempty"`

	// Scheduler defines when to run the chaos experiment
	// +optional
	Scheduler *SchedulerSpec `json:"scheduler,omitempty"`

	// BlastRadius defines limits for the chaos experiment
	// +optional
	BlastRadius *BlastRadiusControl `json:"blastRadius,omitempty"`
}

// NetworkChaosAction defines the action type for NetworkChaos
type NetworkChaosAction string

const (
	// NetworkDelayAction adds network delay
	NetworkDelayAction NetworkChaosAction = "delay"
	// NetworkLossAction causes packet loss
	NetworkLossAction NetworkChaosAction = "loss"
	// NetworkDuplicateAction duplicates packets
	NetworkDuplicateAction NetworkChaosAction = "duplicate"
	// NetworkCorruptAction corrupts packets
	NetworkCorruptAction NetworkChaosAction = "corrupt"
	// NetworkPartitionAction creates network partition
	NetworkPartitionAction NetworkChaosAction = "partition"
	// NetworkBandwidthAction limits bandwidth
	NetworkBandwidthAction NetworkChaosAction = "bandwidth"
)

// NetworkDirection defines the direction of network traffic
type NetworkDirection string

const (
	// NetworkDirectionFrom applies to outgoing traffic
	NetworkDirectionFrom NetworkDirection = "from"
	// NetworkDirectionTo applies to incoming traffic
	NetworkDirectionTo NetworkDirection = "to"
	// NetworkDirectionBoth applies to both directions
	NetworkDirectionBoth NetworkDirection = "both"
)

// NetworkTarget defines the target of network chaos
type NetworkTarget struct {
	// Selector defines how to select target pods
	// +optional
	Selector *PodSelector `json:"selector,omitempty"`

	// Mode defines the mode to select target pods
	// +optional
	Mode ChaosMode `json:"mode,omitempty"`

	// Value provides a parameter for the mode
	// +optional
	Value string `json:"value,omitempty"`
}

// DelaySpec defines delay parameters
type DelaySpec struct {
	// Latency defines the latency to add (e.g., "100ms", "1s")
	// +kubebuilder:validation:Required
	Latency string `json:"latency"`

	// Correlation defines the correlation between successive delay values (0-100)
	// +optional
	Correlation string `json:"correlation,omitempty"`

	// Jitter defines the variation in latency (e.g., "10ms")
	// +optional
	Jitter string `json:"jitter,omitempty"`
}

// LossSpec defines packet loss parameters
type LossSpec struct {
	// Loss defines the percentage of packets to lose (0-100)
	// +kubebuilder:validation:Required
	Loss string `json:"loss"`

	// Correlation defines the correlation between successive lost packets (0-100)
	// +optional
	Correlation string `json:"correlation,omitempty"`
}

// DuplicateSpec defines packet duplication parameters
type DuplicateSpec struct {
	// Duplicate defines the percentage of packets to duplicate (0-100)
	// +kubebuilder:validation:Required
	Duplicate string `json:"duplicate"`

	// Correlation defines the correlation between successive duplicated packets (0-100)
	// +optional
	Correlation string `json:"correlation,omitempty"`
}

// CorruptSpec defines packet corruption parameters
type CorruptSpec struct {
	// Corrupt defines the percentage of packets to corrupt (0-100)
	// +kubebuilder:validation:Required
	Corrupt string `json:"corrupt"`

	// Correlation defines the correlation between successive corrupted packets (0-100)
	// +optional
	Correlation string `json:"correlation,omitempty"`
}

// BandwidthSpec defines bandwidth limitation parameters
type BandwidthSpec struct {
	// Rate defines the bandwidth rate limit (e.g., "1mbps", "100kbps")
	// +kubebuilder:validation:Required
	Rate string `json:"rate"`

	// Limit defines the buffer size (bytes)
	// +optional
	Limit uint32 `json:"limit,omitempty"`

	// Buffer defines the maximum size of the bucket (bytes)
	// +optional
	Buffer uint32 `json:"buffer,omitempty"`
}

// NetworkChaosStatus defines the observed state of NetworkChaos
type NetworkChaosStatus struct {
	ChaosStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=nc
// +kubebuilder:printcolumn:name="Action",type=string,JSONPath=`.spec.action`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NetworkChaos is the Schema for the networkchaos API
type NetworkChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkChaosSpec   `json:"spec,omitempty"`
	Status NetworkChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkChaosList contains a list of NetworkChaos
type NetworkChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkChaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkChaos{}, &NetworkChaosList{})
}
