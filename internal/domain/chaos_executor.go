package domain

import (
	"context"
	"fmt"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// ChaosExecutor defines the interface for executing chaos experiments
// This interface follows the Strategy Pattern, allowing different implementations
// for each type of chaos
type ChaosExecutor interface {
	// Execute applies the chaos to the target pods
	Execute(ctx context.Context, targets []corev1.Pod) error

	// Recover removes the chaos from the target pods
	Recover(ctx context.Context, targets []corev1.Pod) error

	// Validate checks if the chaos can be applied
	Validate(ctx context.Context) error

	// GetType returns the type of chaos executor
	GetType() string
}

// PodChaosExecutor executes PodChaos experiments
type PodChaosExecutor struct {
	Spec   *v1alpha1.PodChaosSpec
	client PodOperations
}

// NetworkChaosExecutor executes NetworkChaos experiments
type NetworkChaosExecutor struct {
	Spec   *v1alpha1.NetworkChaosSpec
	client NetworkOperations
}

// StressChaosExecutor executes StressChaos experiments
type StressChaosExecutor struct {
	Spec   *v1alpha1.StressChaosSpec
	client StressOperations
}

// HTTPChaosExecutor executes HTTPChaos experiments
type HTTPChaosExecutor struct {
	Spec   *v1alpha1.HTTPChaosSpec
	client HTTPOperations
}

// PodOperations defines operations for pod manipulation
type PodOperations interface {
	KillPod(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error
	FailPod(ctx context.Context, pod corev1.Pod) error
	KillContainer(ctx context.Context, pod corev1.Pod, containerName string) error
}

// NetworkOperations defines operations for network manipulation
type NetworkOperations interface {
	ApplyDelay(ctx context.Context, pod corev1.Pod, spec *v1alpha1.DelaySpec) error
	ApplyLoss(ctx context.Context, pod corev1.Pod, spec *v1alpha1.LossSpec) error
	ApplyPartition(ctx context.Context, pod corev1.Pod, targets []string) error
	RemoveNetworkRules(ctx context.Context, pod corev1.Pod) error
}

// StressOperations defines operations for stress injection
type StressOperations interface {
	InjectCPUStress(ctx context.Context, pod corev1.Pod, workers int, load *int) error
	InjectMemoryStress(ctx context.Context, pod corev1.Pod, workers int, size string) error
	RemoveStress(ctx context.Context, pod corev1.Pod) error
}

// HTTPOperations defines operations for HTTP fault injection
type HTTPOperations interface {
	InjectAbort(ctx context.Context, pod corev1.Pod, spec *v1alpha1.HTTPAbortSpec) error
	InjectDelay(ctx context.Context, pod corev1.Pod, spec *v1alpha1.HTTPDelaySpec) error
	RemoveHTTPRules(ctx context.Context, pod corev1.Pod) error
}

// NewNetworkChaosExecutor creates a new NetworkChaosExecutor
func NewNetworkChaosExecutor(spec *v1alpha1.NetworkChaosSpec, client NetworkOperations) *NetworkChaosExecutor {
	return &NetworkChaosExecutor{
		Spec:   spec,
		client: client,
	}
}

// Execute implements ChaosExecutor interface for NetworkChaos
func (e *NetworkChaosExecutor) Execute(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		switch e.Spec.Action {
		case v1alpha1.NetworkDelayAction:
			if e.Spec.Delay != nil {
				if err := e.client.ApplyDelay(ctx, pod, e.Spec.Delay); err != nil {
					return err
				}
			}
		case v1alpha1.NetworkLossAction:
			if e.Spec.Loss != nil {
				if err := e.client.ApplyLoss(ctx, pod, e.Spec.Loss); err != nil {
					return err
				}
			}
		case v1alpha1.NetworkPartitionAction:
			if err := e.client.ApplyPartition(ctx, pod, e.Spec.ExternalTargets); err != nil {
				return err
			}
		}
	}
	return nil
}

// Recover implements ChaosExecutor interface for NetworkChaos
func (e *NetworkChaosExecutor) Recover(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		if err := e.client.RemoveNetworkRules(ctx, pod); err != nil {
			return err
		}
	}
	return nil
}

// Validate implements ChaosExecutor interface for NetworkChaos
func (e *NetworkChaosExecutor) Validate(ctx context.Context) error {
	return nil
}

// GetType implements ChaosExecutor interface for NetworkChaos
func (e *NetworkChaosExecutor) GetType() string {
	return "NetworkChaos"
}

// NewStressChaosExecutor creates a new StressChaosExecutor
func NewStressChaosExecutor(spec *v1alpha1.StressChaosSpec, client StressOperations) *StressChaosExecutor {
	return &StressChaosExecutor{
		Spec:   spec,
		client: client,
	}
}

// Execute implements ChaosExecutor interface for StressChaos
func (e *StressChaosExecutor) Execute(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		if cpu := e.Spec.Stressors.CPU; cpu != nil {
			if err := e.client.InjectCPUStress(ctx, pod, cpu.Workers, cpu.Load); err != nil {
				return err
			}
		}
		if mem := e.Spec.Stressors.Memory; mem != nil {
			if err := e.client.InjectMemoryStress(ctx, pod, mem.Workers, mem.Size); err != nil {
				return err
			}
		}
	}
	return nil
}

// Recover implements ChaosExecutor interface for StressChaos
func (e *StressChaosExecutor) Recover(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		if err := e.client.RemoveStress(ctx, pod); err != nil {
			return err
		}
	}
	return nil
}

// Validate implements ChaosExecutor interface for StressChaos
func (e *StressChaosExecutor) Validate(ctx context.Context) error {
	if e.Spec.Stressors.CPU == nil && e.Spec.Stressors.Memory == nil &&
		e.Spec.Stressors.Disk == nil && e.Spec.Stressors.IO == nil {
		return fmt.Errorf("at least one stressor must be defined")
	}
	return nil
}

// GetType implements ChaosExecutor interface for StressChaos
func (e *StressChaosExecutor) GetType() string {
	return "StressChaos"
}

// NewHTTPChaosExecutor creates a new HTTPChaosExecutor
func NewHTTPChaosExecutor(spec *v1alpha1.HTTPChaosSpec, client HTTPOperations) *HTTPChaosExecutor {
	return &HTTPChaosExecutor{
		Spec:   spec,
		client: client,
	}
}

// Execute implements ChaosExecutor interface for HTTPChaos
func (e *HTTPChaosExecutor) Execute(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		switch {
		case e.Spec.Abort != nil:
			if err := e.client.InjectAbort(ctx, pod, e.Spec.Abort); err != nil {
				return err
			}
		case e.Spec.Delay != nil:
			if err := e.client.InjectDelay(ctx, pod, e.Spec.Delay); err != nil {
				return err
			}
		}
	}
	return nil
}

// Recover implements ChaosExecutor interface for HTTPChaos
func (e *HTTPChaosExecutor) Recover(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		if err := e.client.RemoveHTTPRules(ctx, pod); err != nil {
			return err
		}
	}
	return nil
}

// Validate implements ChaosExecutor interface for HTTPChaos
func (e *HTTPChaosExecutor) Validate(ctx context.Context) error {
	if e.Spec.Abort == nil && e.Spec.Delay == nil && e.Spec.Replace == nil && e.Spec.Patch == nil {
		return fmt.Errorf("at least one fault type must be defined")
	}
	return nil
}

// GetType implements ChaosExecutor interface for HTTPChaos
func (e *HTTPChaosExecutor) GetType() string {
	return "HTTPChaos"
}

// NewPodChaosExecutor creates a new PodChaosExecutor
func NewPodChaosExecutor(spec *v1alpha1.PodChaosSpec, client PodOperations) *PodChaosExecutor {
	return &PodChaosExecutor{
		Spec:   spec,
		client: client,
	}
}

// Execute implements ChaosExecutor interface
func (e *PodChaosExecutor) Execute(ctx context.Context, targets []corev1.Pod) error {
	for _, pod := range targets {
		switch e.Spec.Action {
		case v1alpha1.PodKillAction:
			if err := e.client.KillPod(ctx, pod, e.Spec.GracePeriod); err != nil {
				return err
			}
		case v1alpha1.PodFailureAction:
			if err := e.client.FailPod(ctx, pod); err != nil {
				return err
			}
		case v1alpha1.ContainerKillAction:
			for _, containerName := range e.Spec.ContainerNames {
				if err := e.client.KillContainer(ctx, pod, containerName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Recover implements ChaosExecutor interface
func (e *PodChaosExecutor) Recover(ctx context.Context, targets []corev1.Pod) error {
	// Pod chaos is not recoverable as pods are terminated
	return nil
}

// Validate implements ChaosExecutor interface
func (e *PodChaosExecutor) Validate(ctx context.Context) error {
	// Validation logic would go here
	return nil
}

// GetType implements ChaosExecutor interface
func (e *PodChaosExecutor) GetType() string {
	return "PodChaos"
}
