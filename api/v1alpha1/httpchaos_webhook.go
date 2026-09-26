package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the HTTPChaos webhooks with the manager.
func (r *HTTPChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chaos-engineering-io-v1alpha1-httpchaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=httpchaos,verbs=create;update,versions=v1alpha1,name=mhttpchaos.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-chaos-engineering-io-v1alpha1-httpchaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=httpchaos,verbs=create;update,versions=v1alpha1,name=vhttpchaos.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &HTTPChaos{}
var _ webhook.CustomValidator = &HTTPChaos{}

// Default implements webhook.CustomDefaulter.
func (r *HTTPChaos) Default(ctx context.Context, obj runtime.Object) error {
	hc, ok := obj.(*HTTPChaos)
	if !ok {
		return fmt.Errorf("expected HTTPChaos, got %T", obj)
	}

	if hc.Spec.Mode == "" {
		hc.Spec.Mode = OnePodMode
	}

	if hc.Spec.BlastRadius == nil {
		defaultPct := int32(50)
		hc.Spec.BlastRadius = &BlastRadiusControl{
			MaxPercentage: &defaultPct,
		}
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator.
func (r *HTTPChaos) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	hc, ok := obj.(*HTTPChaos)
	if !ok {
		return nil, fmt.Errorf("expected HTTPChaos, got %T", obj)
	}
	return nil, validateHTTPChaosSpec(&hc.Spec)
}

// ValidateUpdate implements webhook.CustomValidator.
func (r *HTTPChaos) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	hc, ok := newObj.(*HTTPChaos)
	if !ok {
		return nil, fmt.Errorf("expected HTTPChaos, got %T", newObj)
	}
	return nil, validateHTTPChaosSpec(&hc.Spec)
}

// ValidateDelete implements webhook.CustomValidator.
func (r *HTTPChaos) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateHTTPChaosSpec(spec *HTTPChaosSpec) error {
	// Port must be positive
	if spec.Port <= 0 {
		return fmt.Errorf("port must be > 0, got %d", spec.Port)
	}

	// Exactly one fault type must be defined
	count := 0
	if spec.Abort != nil {
		count++
	}
	if spec.Delay != nil {
		count++
	}
	if spec.Replace != nil {
		count++
	}
	if spec.Patch != nil {
		count++
	}
	if count == 0 {
		return fmt.Errorf("exactly one fault type (abort, delay, replace, patch) must be defined")
	}
	if count > 1 {
		return fmt.Errorf("only one fault type can be defined at a time, got %d", count)
	}

	// Validate abort status code range
	if spec.Abort != nil {
		if spec.Abort.StatusCode < 100 || spec.Abort.StatusCode > 599 {
			return fmt.Errorf("abort statusCode must be 100-599, got %d", spec.Abort.StatusCode)
		}
	}

	// Validate delay latency format
	if spec.Delay != nil && spec.Delay.Latency != "" {
		if _, err := time.ParseDuration(spec.Delay.Latency); err != nil {
			return fmt.Errorf("invalid delay latency %q: %w", spec.Delay.Latency, err)
		}
	}

	// Validate duration format
	if spec.Duration != "" {
		if _, err := time.ParseDuration(spec.Duration); err != nil {
			return fmt.Errorf("invalid duration %q: %w", spec.Duration, err)
		}
	}

	// Validate value for mode
	if err := validateModeValue(spec.Mode, spec.Value); err != nil {
		return err
	}

	return nil
}
