package v1alpha1

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the PodChaos webhooks with the manager.
func (r *PodChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chaos-engineering-io-v1alpha1-podchaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=podchaos,verbs=create;update,versions=v1alpha1,name=mpodchaos.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-chaos-engineering-io-v1alpha1-podchaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=podchaos,verbs=create;update,versions=v1alpha1,name=vpodchaos.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &PodChaos{}
var _ webhook.CustomValidator = &PodChaos{}

// Default implements webhook.CustomDefaulter.
func (r *PodChaos) Default(ctx context.Context, obj runtime.Object) error {
	pc, ok := obj.(*PodChaos)
	if !ok {
		return fmt.Errorf("expected PodChaos, got %T", obj)
	}

	if pc.Spec.Mode == "" {
		pc.Spec.Mode = OnePodMode
	}

	if pc.Spec.BlastRadius == nil {
		defaultPct := int32(50)
		pc.Spec.BlastRadius = &BlastRadiusControl{
			MaxPercentage: &defaultPct,
		}
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator.
func (r *PodChaos) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	pc, ok := obj.(*PodChaos)
	if !ok {
		return nil, fmt.Errorf("expected PodChaos, got %T", obj)
	}
	return nil, validatePodChaosSpec(&pc.Spec)
}

// ValidateUpdate implements webhook.CustomValidator.
func (r *PodChaos) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	pc, ok := newObj.(*PodChaos)
	if !ok {
		return nil, fmt.Errorf("expected PodChaos, got %T", newObj)
	}
	return nil, validatePodChaosSpec(&pc.Spec)
}

// ValidateDelete implements webhook.CustomValidator.
func (r *PodChaos) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validatePodChaosSpec(spec *PodChaosSpec) error {
	// container-kill requires containerNames
	if spec.Action == ContainerKillAction && len(spec.ContainerNames) == 0 {
		return fmt.Errorf("containerNames must not be empty when action is container-kill")
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

// validateModeValue checks that the value field is valid for the given mode.
// Shared by all chaos types.
func validateModeValue(mode ChaosMode, value string) error {
	switch mode {
	case FixedMode:
		if value == "" {
			return fmt.Errorf("value is required when mode is %q", mode)
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("value must be a positive integer when mode is %q, got %q", mode, value)
		}
	case FixedPercentMode, RandomMaxPercentMode:
		if value == "" {
			return fmt.Errorf("value is required when mode is %q", mode)
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 || n > 100 {
			return fmt.Errorf("value must be 0-100 when mode is %q, got %q", mode, value)
		}
	case OnePodMode, AllMode:
		// value is optional
	}
	return nil
}
