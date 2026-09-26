package domain

import (
	"context"
	"fmt"
	"testing"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockPodOperations implements PodOperations for testing.
type mockPodOperations struct {
	killPodCalled      int
	failPodCalled      int
	killContainerCalls []string
	shouldError        bool
}

func (m *mockPodOperations) KillPod(_ context.Context, _ corev1.Pod, _ *int64) error {
	m.killPodCalled++
	if m.shouldError {
		return fmt.Errorf("mock kill error")
	}
	return nil
}

func (m *mockPodOperations) FailPod(_ context.Context, _ corev1.Pod) error {
	m.failPodCalled++
	if m.shouldError {
		return fmt.Errorf("mock fail error")
	}
	return nil
}

func (m *mockPodOperations) KillContainer(_ context.Context, _ corev1.Pod, name string) error {
	m.killContainerCalls = append(m.killContainerCalls, name)
	if m.shouldError {
		return fmt.Errorf("mock container kill error")
	}
	return nil
}

func newTestPod(name string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main"}},
		},
	}
}

func TestPodChaosExecutor_PodKill(t *testing.T) {
	mock := &mockPodOperations{}
	spec := &v1alpha1.PodChaosSpec{Action: v1alpha1.PodKillAction}
	executor := NewPodChaosExecutor(spec, mock)

	targets := []corev1.Pod{newTestPod("pod-1"), newTestPod("pod-2")}
	err := executor.Execute(context.Background(), targets)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.killPodCalled != 2 {
		t.Errorf("expected 2 KillPod calls, got %d", mock.killPodCalled)
	}
}

func TestPodChaosExecutor_PodFailure(t *testing.T) {
	mock := &mockPodOperations{}
	spec := &v1alpha1.PodChaosSpec{Action: v1alpha1.PodFailureAction}
	executor := NewPodChaosExecutor(spec, mock)

	targets := []corev1.Pod{newTestPod("pod-1")}
	err := executor.Execute(context.Background(), targets)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.failPodCalled != 1 {
		t.Errorf("expected 1 FailPod call, got %d", mock.failPodCalled)
	}
}

func TestPodChaosExecutor_ContainerKill(t *testing.T) {
	mock := &mockPodOperations{}
	spec := &v1alpha1.PodChaosSpec{
		Action:         v1alpha1.ContainerKillAction,
		ContainerNames: []string{"sidecar", "helper"},
	}
	executor := NewPodChaosExecutor(spec, mock)

	targets := []corev1.Pod{newTestPod("pod-1")}
	err := executor.Execute(context.Background(), targets)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.killContainerCalls) != 2 {
		t.Errorf("expected 2 KillContainer calls, got %d", len(mock.killContainerCalls))
	}
	if mock.killContainerCalls[0] != "sidecar" || mock.killContainerCalls[1] != "helper" {
		t.Errorf("unexpected container names: %v", mock.killContainerCalls)
	}
}

func TestPodChaosExecutor_EmptyTargets(t *testing.T) {
	mock := &mockPodOperations{}
	spec := &v1alpha1.PodChaosSpec{Action: v1alpha1.PodKillAction}
	executor := NewPodChaosExecutor(spec, mock)

	err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error for empty targets, got: %v", err)
	}
	if mock.killPodCalled != 0 {
		t.Errorf("expected no KillPod calls for empty targets")
	}
}

func TestPodChaosExecutor_ErrorPropagation(t *testing.T) {
	mock := &mockPodOperations{shouldError: true}
	spec := &v1alpha1.PodChaosSpec{Action: v1alpha1.PodKillAction}
	executor := NewPodChaosExecutor(spec, mock)

	targets := []corev1.Pod{newTestPod("pod-1")}
	err := executor.Execute(context.Background(), targets)

	if err == nil {
		t.Error("expected error to propagate from mock")
	}
}

func TestPodChaosExecutor_Recover(t *testing.T) {
	mock := &mockPodOperations{}
	spec := &v1alpha1.PodChaosSpec{Action: v1alpha1.PodKillAction}
	executor := NewPodChaosExecutor(spec, mock)

	err := executor.Recover(context.Background(), []corev1.Pod{newTestPod("pod-1")})
	if err != nil {
		t.Fatalf("expected nil error from Recover (no-op), got: %v", err)
	}
}

func TestPodChaosExecutor_GetType(t *testing.T) {
	spec := &v1alpha1.PodChaosSpec{}
	executor := NewPodChaosExecutor(spec, &mockPodOperations{})
	if executor.GetType() != "PodChaos" {
		t.Errorf("expected 'PodChaos', got %q", executor.GetType())
	}
}

func TestNetworkChaosExecutor_GetType(t *testing.T) {
	spec := &v1alpha1.NetworkChaosSpec{}
	executor := NewNetworkChaosExecutor(spec, nil)
	if executor.GetType() != "NetworkChaos" {
		t.Errorf("expected 'NetworkChaos', got %q", executor.GetType())
	}
}

func TestStressChaosExecutor_Validate(t *testing.T) {
	t.Run("no stressors returns error", func(t *testing.T) {
		spec := &v1alpha1.StressChaosSpec{Stressors: v1alpha1.StressorsSpec{}}
		executor := NewStressChaosExecutor(spec, nil)
		if err := executor.Validate(context.Background()); err == nil {
			t.Error("expected error when no stressors defined")
		}
	})

	t.Run("with cpu stressor passes", func(t *testing.T) {
		spec := &v1alpha1.StressChaosSpec{
			Stressors: v1alpha1.StressorsSpec{CPU: &v1alpha1.CPUStressor{Workers: 1}},
		}
		executor := NewStressChaosExecutor(spec, nil)
		if err := executor.Validate(context.Background()); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestHTTPChaosExecutor_Validate(t *testing.T) {
	t.Run("no fault type returns error", func(t *testing.T) {
		spec := &v1alpha1.HTTPChaosSpec{}
		executor := NewHTTPChaosExecutor(spec, nil)
		if err := executor.Validate(context.Background()); err == nil {
			t.Error("expected error when no fault type defined")
		}
	})

	t.Run("with abort passes", func(t *testing.T) {
		spec := &v1alpha1.HTTPChaosSpec{Abort: &v1alpha1.HTTPAbortSpec{StatusCode: 500}}
		executor := NewHTTPChaosExecutor(spec, nil)
		if err := executor.Validate(context.Background()); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}
