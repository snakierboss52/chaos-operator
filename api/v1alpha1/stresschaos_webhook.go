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

// SetupWebhookWithManager registers the StressChaos webhooks with the manager.
func (r *StressChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chaos-engineering-io-v1alpha1-stresschaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=stresschaos,verbs=create;update,versions=v1alpha1,name=mstresschaos.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-chaos-engineering-io-v1alpha1-stresschaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=stresschaos,verbs=create;update,versions=v1alpha1,name=vstresschaos.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &StressChaos{}
var _ webhook.CustomValidator = &StressChaos{}

// Default implements webhook.CustomDefaulter.
func (r *StressChaos) Default(ctx context.Context, obj runtime.Object) error {
	sc, ok := obj.(*StressChaos)
	if !ok {
		return fmt.Errorf("expected StressChaos, got %T", obj)
	}

	if sc.Spec.Mode == "" {
		sc.Spec.Mode = OnePodMode
	}

	if sc.Spec.BlastRadius == nil {
		defaultPct := int32(50)
		sc.Spec.BlastRadius = &BlastRadiusControl{
			MaxPercentage: &defaultPct,
		}
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator.
func (r *StressChaos) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	sc, ok := obj.(*StressChaos)
	if !ok {
		return nil, fmt.Errorf("expected StressChaos, got %T", obj)
	}
	return nil, validateStressChaosSpec(&sc.Spec)
}

// ValidateUpdate implements webhook.CustomValidator.
func (r *StressChaos) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	sc, ok := newObj.(*StressChaos)
	if !ok {
		return nil, fmt.Errorf("expected StressChaos, got %T", newObj)
	}
	return nil, validateStressChaosSpec(&sc.Spec)
}

// ValidateDelete implements webhook.CustomValidator.
func (r *StressChaos) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateStressChaosSpec(spec *StressChaosSpec) error {
	// At least one stressor must be defined
	s := spec.Stressors
	if s.CPU == nil && s.Memory == nil && s.Disk == nil && s.IO == nil {
		return fmt.Errorf("at least one stressor (cpu, memory, disk, io) must be defined")
	}

	// Validate CPU stressor
	if s.CPU != nil && s.CPU.Workers < 1 {
		return fmt.Errorf("cpu workers must be >= 1, got %d", s.CPU.Workers)
	}

	// Validate Memory stressor
	if s.Memory != nil {
		if s.Memory.Workers < 1 {
			return fmt.Errorf("memory workers must be >= 1, got %d", s.Memory.Workers)
		}
		if s.Memory.Size == "" {
			return fmt.Errorf("memory size is required")
		}
	}

	// Validate Disk stressor
	if s.Disk != nil && s.Disk.Size == "" {
		return fmt.Errorf("disk size is required")
	}

	// Validate IO stressor
	if s.IO != nil && s.IO.Workers < 1 {
		return fmt.Errorf("io workers must be >= 1, got %d", s.IO.Workers)
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
