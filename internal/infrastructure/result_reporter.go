package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"goland-operator/api/v1alpha1"
)

// ResultReporter records experiment results and generates summaries.
type ResultReporter struct {
	client client.Client
	log    logr.Logger
}

// NewResultReporter creates a new ResultReporter.
func NewResultReporter(c client.Client, log logr.Logger) *ResultReporter {
	return &ResultReporter{
		client: c,
		log:    log.WithName("result-reporter"),
	}
}

// BuildResult constructs an ExperimentResult from per-target outcomes.
func BuildResult(targetResults []v1alpha1.TargetResult, blastRadiusLimited bool) *v1alpha1.ExperimentResult {
	successful := 0
	failed := 0
	for _, tr := range targetResults {
		if tr.Success {
			successful++
		} else {
			failed++
		}
	}

	return &v1alpha1.ExperimentResult{
		TotalTargets:       len(targetResults),
		SuccessfulTargets:  successful,
		FailedTargets:      failed,
		TargetResults:      targetResults,
		BlastRadiusLimited: blastRadiusLimited,
	}
}

// ReportInput holds all data needed to generate an experiment report.
type ReportInput struct {
	Namespace      string
	ExperimentName string
	ChaosType      string
	Phase          string
	StartTime      *metav1.Time
	CompletionTime *metav1.Time
	Result         *v1alpha1.ExperimentResult
	AffectedPods   []string
	Message        string
	// Details contains type-specific experiment configuration
	Details map[string]string
}

// CreateSummaryConfigMap creates or updates a ConfigMap with the experiment report.
func (r *ResultReporter) CreateSummaryConfigMap(ctx context.Context, input ReportInput) error {
	resultJSON, err := json.MarshalIndent(input.Result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	cmName := fmt.Sprintf("chaos-report-%s", input.ExperimentName)

	var duration string
	if input.StartTime != nil && input.CompletionTime != nil {
		duration = input.CompletionTime.Time.Sub(input.StartTime.Time).Round(time.Second).String()
	}

	data := map[string]string{
		"result.json": string(resultJSON),
		"type":        input.ChaosType,
		"phase":       input.Phase,
		"message":     input.Message,
	}

	if input.StartTime != nil {
		data["startTime"] = input.StartTime.Time.UTC().Format(time.RFC3339)
	}
	if input.CompletionTime != nil {
		data["completionTime"] = input.CompletionTime.Time.UTC().Format(time.RFC3339)
	}
	if duration != "" {
		data["duration"] = duration
	}

	// Affected pods
	if len(input.AffectedPods) > 0 {
		podsJSON, _ := json.Marshal(input.AffectedPods)
		data["affectedPods"] = string(podsJSON)
	}

	// Type-specific details
	for k, v := range input.Details {
		data[k] = v
	}

	// Build summary line
	data["summary"] = fmt.Sprintf("Type: %s, Phase: %s, Targets: %d (ok:%d fail:%d), Duration: %s",
		input.ChaosType, input.Phase,
		input.Result.TotalTargets, input.Result.SuccessfulTargets, input.Result.FailedTargets,
		duration)

	labels := map[string]string{
		"app.kubernetes.io/managed-by":    "chaos-operator",
		"chaos.engineering.io/type":       input.ChaosType,
		"chaos.engineering.io/experiment": input.ExperimentName,
	}

	var existing corev1.ConfigMap
	key := client.ObjectKey{Namespace: input.Namespace, Name: cmName}
	if err := r.client.Get(ctx, key, &existing); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("getting existing ConfigMap: %w", err)
		}
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: input.Namespace,
				Labels:    labels,
			},
			Data: data,
		}
		if err := r.client.Create(ctx, cm); err != nil {
			return fmt.Errorf("creating summary ConfigMap: %w", err)
		}
	} else {
		existing.Labels = labels
		existing.Data = data
		if err := r.client.Update(ctx, &existing); err != nil {
			return fmt.Errorf("updating summary ConfigMap: %w", err)
		}
	}

	r.log.Info("Summary ConfigMap saved",
		"name", cmName,
		"namespace", input.Namespace,
		"phase", input.Phase,
	)

	return nil
}
