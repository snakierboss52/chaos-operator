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

// SetupWebhookWithManager registers the NetworkChaos webhooks with the manager.
func (r *NetworkChaos) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-chaos-engineering-io-v1alpha1-networkchaos,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=networkchaos,verbs=create;update,versions=v1alpha1,name=mnetworkchaos.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-chaos-engineering-io-v1alpha1-networkchaos,mutating=false,failurePolicy=fail,sideEffects=None,groups=chaos.engineering.io,resources=networkchaos,verbs=create;update,versions=v1alpha1,name=vnetworkchaos.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &NetworkChaos{}
var _ webhook.CustomValidator = &NetworkChaos{}

// Default implements webhook.CustomDefaulter.
func (r *NetworkChaos) Default(ctx context.Context, obj runtime.Object) error {
	nc, ok := obj.(*NetworkChaos)
	if !ok {
		return fmt.Errorf("expected NetworkChaos, got %T", obj)
	}

	if nc.Spec.Mode == "" {
		nc.Spec.Mode = OnePodMode
	}

	if nc.Spec.BlastRadius == nil {
		defaultPct := int32(50)
		nc.Spec.BlastRadius = &BlastRadiusControl{
			MaxPercentage: &defaultPct,
		}
	}

	if nc.Spec.Direction == "" {
		nc.Spec.Direction = NetworkDirectionBoth
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator.
func (r *NetworkChaos) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nc, ok := obj.(*NetworkChaos)
	if !ok {
		return nil, fmt.Errorf("expected NetworkChaos, got %T", obj)
	}
	return nil, validateNetworkChaosSpec(&nc.Spec)
}

// ValidateUpdate implements webhook.CustomValidator.
func (r *NetworkChaos) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	nc, ok := newObj.(*NetworkChaos)
	if !ok {
		return nil, fmt.Errorf("expected NetworkChaos, got %T", newObj)
	}
	return nil, validateNetworkChaosSpec(&nc.Spec)
}

// ValidateDelete implements webhook.CustomValidator.
func (r *NetworkChaos) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateNetworkChaosSpec(spec *NetworkChaosSpec) error {
	// Validate that the action-specific spec is provided
	switch spec.Action {
	case NetworkDelayAction:
		if spec.Delay == nil {
			return fmt.Errorf("delay spec is required when action is %q", spec.Action)
		}
	case NetworkLossAction:
		if spec.Loss == nil {
			return fmt.Errorf("loss spec is required when action is %q", spec.Action)
		}
	case NetworkDuplicateAction:
		if spec.Duplicate == nil {
			return fmt.Errorf("duplicate spec is required when action is %q", spec.Action)
		}
	case NetworkCorruptAction:
		if spec.Corrupt == nil {
			return fmt.Errorf("corrupt spec is required when action is %q", spec.Action)
		}
	case NetworkBandwidthAction:
		if spec.Bandwidth == nil {
			return fmt.Errorf("bandwidth spec is required when action is %q", spec.Action)
		}
	case NetworkPartitionAction:
		// No additional spec required
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
