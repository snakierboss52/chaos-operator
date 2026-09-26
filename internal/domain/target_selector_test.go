package domain

import (
	"context"
	"fmt"
	"testing"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mockPodLister implements PodLister for testing.
type mockPodLister struct {
	pods      []corev1.Pod
	returnErr bool
}

func (m *mockPodLister) ListPods(_ context.Context, _ v1alpha1.PodSelector) ([]corev1.Pod, error) {
	if m.returnErr {
		return nil, fmt.Errorf("mock list error")
	}
	return m.pods, nil
}

func makePods(n int) []corev1.Pod {
	pods := make([]corev1.Pod, n)
	for i := range pods {
		pods[i] = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
	}
	return pods
}

func TestTargetSelector_OnePodMode(t *testing.T) {
	lister := &mockPodLister{pods: makePods(5)}
	maxPct := int32(100)
	sel := NewTargetSelector(v1alpha1.OnePodMode, "", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPercentage: &maxPct}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(targets))
	}
}

func TestTargetSelector_AllMode(t *testing.T) {
	lister := &mockPodLister{pods: makePods(3)}
	maxPct := int32(100)
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPercentage: &maxPct}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(targets))
	}
}

func TestTargetSelector_FixedMode(t *testing.T) {
	lister := &mockPodLister{pods: makePods(10)}
	maxPct := int32(100)
	sel := NewTargetSelector(v1alpha1.FixedMode, "3", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPercentage: &maxPct}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(targets))
	}
}

func TestTargetSelector_FixedPercentMode(t *testing.T) {
	lister := &mockPodLister{pods: makePods(10)}
	maxPct := int32(100)
	sel := NewTargetSelector(v1alpha1.FixedPercentMode, "50", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPercentage: &maxPct}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 5 {
		t.Errorf("expected 5 targets (50%% of 10), got %d", len(targets))
	}
}

func TestTargetSelector_BlastRadiusMaxPods(t *testing.T) {
	lister := &mockPodLister{pods: makePods(10)}
	maxPods := int32(2)
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPods: &maxPods}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("expected 2 targets (MaxPods=2), got %d", len(targets))
	}
}

func TestTargetSelector_BlastRadiusMaxPercentage(t *testing.T) {
	lister := &mockPodLister{pods: makePods(10)}
	maxPct := int32(30)
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{},
		&v1alpha1.BlastRadiusControl{MaxPercentage: &maxPct}, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 3 {
		t.Errorf("expected 3 targets (30%% of 10), got %d", len(targets))
	}
}

func TestTargetSelector_DefaultBlastRadius(t *testing.T) {
	lister := &mockPodLister{pods: makePods(10)}
	// No blast radius = default 50%
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{}, nil, lister)

	targets, err := sel.SelectTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 5 {
		t.Errorf("expected 5 targets (default 50%%  of 10), got %d", len(targets))
	}
}

func TestTargetSelector_NoPods(t *testing.T) {
	lister := &mockPodLister{pods: nil}
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{}, nil, lister)

	_, err := sel.SelectTargets(context.Background())
	if err == nil {
		t.Error("expected error when no pods found")
	}
}

func TestTargetSelector_ListError(t *testing.T) {
	lister := &mockPodLister{returnErr: true}
	sel := NewTargetSelector(v1alpha1.AllMode, "", v1alpha1.PodSelector{}, nil, lister)

	_, err := sel.SelectTargets(context.Background())
	if err == nil {
		t.Error("expected error from list failure")
	}
}

func TestTargetSelector_InvalidMode(t *testing.T) {
	lister := &mockPodLister{pods: makePods(1)}
	sel := NewTargetSelector("invalid-mode", "", v1alpha1.PodSelector{}, nil, lister)

	_, err := sel.SelectTargets(context.Background())
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}
