package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StressChaosSpec defines the desired state of StressChaos
type StressChaosSpec struct {
	// Mode defines the mode to select target pods
	// +kubebuilder:validation:Required
	Mode ChaosMode `json:"mode"`

	// Value provides a parameter for the mode
	// +optional
	Value string `json:"value,omitempty"`

	// Selector defines how to select target pods
	// +kubebuilder:validation:Required
	Selector PodSelector `json:"selector"`

	// Duration defines how long the chaos will last
	// +optional
	Duration string `json:"duration,omitempty"`

	// Stressors defines the stress parameters
	// +kubebuilder:validation:Required
	Stressors StressorsSpec `json:"stressors"`

	// ContainerNames specifies the containers to inject stress
	// If not specified, stress is injected to the first container
	// +optional
	ContainerNames []string `json:"containerNames,omitempty"`

	// Scheduler defines when to run the chaos experiment
	// +optional
	Scheduler *SchedulerSpec `json:"scheduler,omitempty"`

	// BlastRadius defines limits for the chaos experiment
	// +optional
	BlastRadius *BlastRadiusControl `json:"blastRadius,omitempty"`
}

// StressorsSpec defines the stressors for StressChaos
type StressorsSpec struct {
	// CPU defines CPU stress parameters
	// +optional
	CPU *CPUStressor `json:"cpu,omitempty"`

	// Memory defines memory stress parameters
	// +optional
	Memory *MemoryStressor `json:"memory,omitempty"`

	// Disk defines disk fill stress parameters
	// +optional
	Disk *DiskStressor `json:"disk,omitempty"`

	// IO defines IO stress parameters
	// +optional
	IO *IOStressor `json:"io,omitempty"`
}

// CPUStressor defines CPU stress parameters
type CPUStressor struct {
	// Workers defines the number of CPU stress workers
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Workers int `json:"workers"`

	// Load defines the percentage of CPU load per worker (0-100)
	// If not specified, each worker will use 100% CPU
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	Load *int `json:"load,omitempty"`

	// Options are additional options passed to stress-ng
	// +optional
	Options []string `json:"options,omitempty"`
}

// MemoryStressor defines memory stress parameters
type MemoryStressor struct {
	// Workers defines the number of memory stress workers
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Workers int `json:"workers"`

	// Size defines the amount of memory to allocate per worker
	// Examples: "256MB", "1GB", "50%"
	// +kubebuilder:validation:Required
	Size string `json:"size"`

	// Options are additional options passed to stress-ng
	// +optional
	Options []string `json:"options,omitempty"`

	// OOMScoreAdj adjusts the OOM killer score (-1000 to 1000)
	// Lower values decrease the likelihood of being killed
	// +kubebuilder:validation:Minimum=-1000
	// +kubebuilder:validation:Maximum=1000
	// +optional
	OOMScoreAdj *int `json:"oomScoreAdj,omitempty"`
}

// DiskStressor defines disk fill stress parameters
type DiskStressor struct {
	// Size defines the amount of disk space to fill
	// Examples: "512MB", "1GB"
	// +kubebuilder:validation:Required
	Size string `json:"size"`

	// Path defines the filesystem path to fill
	// Defaults to "/tmp" if not specified
	// +optional
	Path string `json:"path,omitempty"`
}

// IOStressor defines IO stress parameters
type IOStressor struct {
	// Workers defines the number of IO stress workers
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Workers int `json:"workers"`

	// Size defines the block size for IO operations
	// Examples: "4k", "1M"
	// +optional
	Size string `json:"size,omitempty"`
}

// StressChaosStatus defines the observed state of StressChaos
type StressChaosStatus struct {
	ChaosStatus `json:",inline"`

	// StressCommandPid stores the PID of the stress process
	// +optional
	StressCommandPid int `json:"stressCommandPid,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=sc
// +kubebuilder:printcolumn:name="Stressors",type=string,JSONPath=`.spec.stressors`
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// StressChaos is the Schema for the stresschaos API
type StressChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StressChaosSpec   `json:"spec,omitempty"`
	Status StressChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StressChaosList contains a list of StressChaos
type StressChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StressChaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StressChaos{}, &StressChaosList{})
}
