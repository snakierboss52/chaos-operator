package controllers

import (
	"time"

	"goland-operator/internal/infrastructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func observeExperimentMetrics(experimentType, namespace, name, action, mode, phase string, conditions []metav1.Condition, startedAt, completedAt *metav1.Time, affectedPods []string) {
	start := time.Time{}
	if startedAt != nil {
		start = startedAt.Time
	}
	completed := time.Time{}
	if completedAt != nil {
		completed = completedAt.Time
	}
	reason := ""
	if len(conditions) > 0 {
		reason = conditions[len(conditions)-1].Reason
	}
	infrastructure.ObserveChaosExperiment(experimentType, namespace, name, action, mode, phase, reason, start, completed, affectedPods)
}
