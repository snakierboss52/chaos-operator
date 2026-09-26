package domain

import (
	"context"
	"fmt"
	"math/rand"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"strconv"
)

// TargetSelector is responsible for selecting target pods based on mode and selector
// This implements the Strategy Pattern for different selection modes
type TargetSelector struct {
	mode           v1alpha1.ChaosMode
	value          string
	selector       v1alpha1.PodSelector
	blastRadius    *v1alpha1.BlastRadiusControl
	podLister      PodLister
}

// PodLister defines the interface for listing pods
type PodLister interface {
	ListPods(ctx context.Context, selector v1alpha1.PodSelector) ([]corev1.Pod, error)
}

// NewTargetSelector creates a new TargetSelector
func NewTargetSelector(
	mode v1alpha1.ChaosMode,
	value string,
	selector v1alpha1.PodSelector,
	blastRadius *v1alpha1.BlastRadiusControl,
	podLister PodLister,
) *TargetSelector {
	return &TargetSelector{
		mode:        mode,
		value:       value,
		selector:    selector,
		blastRadius: blastRadius,
		podLister:   podLister,
	}
}

// SelectTargets selects pods based on the configured mode and selector
func (s *TargetSelector) SelectTargets(ctx context.Context) ([]corev1.Pod, error) {
	// Get all candidate pods
	candidates, err := s.podLister.ListPods(ctx, s.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no pods found matching selector")
	}

	// Select based on mode
	var selected []corev1.Pod
	switch s.mode {
	case v1alpha1.OnePodMode:
		selected = s.selectOne(candidates)
	case v1alpha1.AllMode:
		selected = candidates
	case v1alpha1.FixedMode:
		selected, err = s.selectFixed(candidates)
	case v1alpha1.FixedPercentMode:
		selected, err = s.selectFixedPercent(candidates)
	case v1alpha1.RandomMaxPercentMode:
		selected, err = s.selectRandomMaxPercent(candidates)
	default:
		return nil, fmt.Errorf("unknown chaos mode: %s", s.mode)
	}

	if err != nil {
		return nil, err
	}

	// Apply blast radius controls
	selected = s.applyBlastRadiusControls(selected, len(candidates))

	return selected, nil
}

func (s *TargetSelector) selectOne(candidates []corev1.Pod) []corev1.Pod {
	if len(candidates) == 0 {
		return nil
	}
	// Randomly select one pod
	index := rand.Intn(len(candidates))
	return []corev1.Pod{candidates[index]}
}

func (s *TargetSelector) selectFixed(candidates []corev1.Pod) ([]corev1.Pod, error) {
	count, err := strconv.Atoi(s.value)
	if err != nil {
		return nil, fmt.Errorf("invalid value for fixed mode: %w", err)
	}

	if count > len(candidates) {
		count = len(candidates)
	}

	// Randomly select 'count' pods
	return s.randomSelect(candidates, count), nil
}

func (s *TargetSelector) selectFixedPercent(candidates []corev1.Pod) ([]corev1.Pod, error) {
	percent, err := strconv.Atoi(s.value)
	if err != nil {
		return nil, fmt.Errorf("invalid value for fixed-percent mode: %w", err)
	}

	if percent < 0 || percent > 100 {
		return nil, fmt.Errorf("percentage must be between 0 and 100")
	}

	count := (len(candidates) * percent) / 100
	if count == 0 && percent > 0 {
		count = 1 // At least one pod if percentage > 0
	}

	return s.randomSelect(candidates, count), nil
}

func (s *TargetSelector) selectRandomMaxPercent(candidates []corev1.Pod) ([]corev1.Pod, error) {
	maxPercent, err := strconv.Atoi(s.value)
	if err != nil {
		return nil, fmt.Errorf("invalid value for random-max-percent mode: %w", err)
	}

	if maxPercent < 0 || maxPercent > 100 {
		return nil, fmt.Errorf("percentage must be between 0 and 100")
	}

	// Random percentage between 0 and maxPercent
	randomPercent := rand.Intn(maxPercent + 1)
	count := (len(candidates) * randomPercent) / 100

	return s.randomSelect(candidates, count), nil
}

func (s *TargetSelector) randomSelect(pods []corev1.Pod, count int) []corev1.Pod {
	if count >= len(pods) {
		return pods
	}

	// Fisher-Yates shuffle and take first 'count' elements
	shuffled := make([]corev1.Pod, len(pods))
	copy(shuffled, pods)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

func (s *TargetSelector) applyBlastRadiusControls(selected []corev1.Pod, totalCandidates int) []corev1.Pod {
	if s.blastRadius == nil {
		// Default: limit to 50% of candidates
		maxDefault := totalCandidates / 2
		if len(selected) > maxDefault {
			return selected[:maxDefault]
		}
		return selected
	}

	// Apply MaxPods constraint
	if s.blastRadius.MaxPods != nil && len(selected) > int(*s.blastRadius.MaxPods) {
		return selected[:*s.blastRadius.MaxPods]
	}

	// Apply MaxPercentage constraint
	if s.blastRadius.MaxPercentage != nil {
		maxByPercent := (totalCandidates * int(*s.blastRadius.MaxPercentage)) / 100
		if len(selected) > maxByPercent {
			return selected[:maxByPercent]
		}
	}

	return selected
}
