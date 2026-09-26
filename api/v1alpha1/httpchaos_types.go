package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HTTPChaosSpec defines the desired state of HTTPChaos
type HTTPChaosSpec struct {
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

	// Target defines whether to target Request or Response
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Request;Response;Both
	Target HTTPTarget `json:"target"`

	// Port defines the HTTP port to intercept
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`

	// Path defines the HTTP path to match (supports wildcards)
	// Examples: "/api/v1/*", "/users"
	// +optional
	Path string `json:"path,omitempty"`

	// Method defines the HTTP method to match
	// +kubebuilder:validation:Enum=GET;POST;PUT;DELETE;PATCH;HEAD;OPTIONS;*
	// +optional
	Method string `json:"method,omitempty"`

	// Abort defines parameters for aborting requests
	// +optional
	Abort *HTTPAbortSpec `json:"abort,omitempty"`

	// Delay defines parameters for delaying requests
	// +optional
	Delay *HTTPDelaySpec `json:"delay,omitempty"`

	// Replace defines parameters for replacing request/response content
	// +optional
	Replace *HTTPReplaceSpec `json:"replace,omitempty"`

	// Patch defines parameters for patching request/response
	// +optional
	Patch *HTTPPatchSpec `json:"patch,omitempty"`

	// Scheduler defines when to run the chaos experiment
	// +optional
	Scheduler *SchedulerSpec `json:"scheduler,omitempty"`

	// BlastRadius defines limits for the chaos experiment
	// +optional
	BlastRadius *BlastRadiusControl `json:"blastRadius,omitempty"`
}

// HTTPTarget defines whether to target request or response
type HTTPTarget string

const (
	// HTTPTargetRequest targets HTTP requests
	HTTPTargetRequest HTTPTarget = "Request"
	// HTTPTargetResponse targets HTTP responses
	HTTPTargetResponse HTTPTarget = "Response"
	// HTTPTargetBoth targets both requests and responses
	HTTPTargetBoth HTTPTarget = "Both"
)

// HTTPAbortSpec defines parameters for aborting HTTP requests
type HTTPAbortSpec struct {
	// StatusCode is the HTTP status code to return
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	StatusCode int32 `json:"statusCode"`

	// Percentage is the percentage of requests to abort (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	Percentage int32 `json:"percentage,omitempty"`

	// Body is the response body to return
	// +optional
	Body string `json:"body,omitempty"`
}

// HTTPDelaySpec defines parameters for delaying HTTP requests
type HTTPDelaySpec struct {
	// Latency defines the delay to add (e.g., "100ms", "2s")
	// +kubebuilder:validation:Required
	Latency string `json:"latency"`

	// Percentage is the percentage of requests to delay (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	Percentage int32 `json:"percentage,omitempty"`

	// Jitter defines the variation in delay (e.g., "10ms")
	// +optional
	Jitter string `json:"jitter,omitempty"`
}

// HTTPReplaceSpec defines parameters for replacing HTTP content
type HTTPReplaceSpec struct {
	// Body replaces the entire body
	// +optional
	Body string `json:"body,omitempty"`

	// Headers replaces specific headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// StatusCode replaces the status code (response only)
	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=599
	// +optional
	StatusCode *int32 `json:"statusCode,omitempty"`

	// Percentage is the percentage of requests to replace (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	Percentage int32 `json:"percentage,omitempty"`
}

// HTTPPatchSpec defines parameters for patching HTTP content
type HTTPPatchSpec struct {
	// Headers adds or modifies headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Queries adds or modifies query parameters
	// +optional
	Queries map[string]string `json:"queries,omitempty"`

	// Body patches the body using JSONPath
	// +optional
	Body *HTTPBodyPatch `json:"body,omitempty"`

	// Percentage is the percentage of requests to patch (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +optional
	Percentage int32 `json:"percentage,omitempty"`
}

// HTTPBodyPatch defines parameters for patching HTTP body
type HTTPBodyPatch struct {
	// Type defines the patch type (json, text)
	// +kubebuilder:validation:Enum=json;text
	Type string `json:"type"`

	// Value is the patch value
	Value string `json:"value"`
}

// HTTPChaosStatus defines the observed state of HTTPChaos
type HTTPChaosStatus struct {
	ChaosStatus `json:",inline"`

	// InjectionCount tracks how many times the fault was injected
	// +optional
	InjectionCount int64 `json:"injectionCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=hc
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.target`
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.port`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HTTPChaos is the Schema for the httpchaos API
type HTTPChaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPChaosSpec   `json:"spec,omitempty"`
	Status HTTPChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HTTPChaosList contains a list of HTTPChaos
type HTTPChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPChaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HTTPChaos{}, &HTTPChaosList{})
}
