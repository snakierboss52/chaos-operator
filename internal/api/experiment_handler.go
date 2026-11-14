package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	namegenerator "k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type experimentRequestBase struct {
	Type        string                     `json:"type"`
	Name        string                     `json:"name,omitempty"`
	Namespace   string                     `json:"namespace"`
	Mode        chaosv1alpha1.ChaosMode    `json:"mode"`
	Value       string                     `json:"value,omitempty"`
	AppSelector string                     `json:"appSelector,omitempty"`
	Selector    *chaosv1alpha1.PodSelector `json:"selector,omitempty"`
	Duration    string                     `json:"duration,omitempty"`
}

type podExperimentRequest struct {
	experimentRequestBase `json:",inline"`
	Action                chaosv1alpha1.PodChaosAction      `json:"action"`
	GracePeriod           *int64                            `json:"gracePeriod,omitempty"`
	Force                 bool                              `json:"force,omitempty"`
	ContainerNames        []string                          `json:"containerNames,omitempty"`
	Scheduler             *chaosv1alpha1.SchedulerSpec      `json:"scheduler,omitempty"`
	BlastRadius           *chaosv1alpha1.BlastRadiusControl `json:"blastRadius,omitempty"`
}

type networkExperimentRequest struct {
	experimentRequestBase `json:",inline"`
	Action                chaosv1alpha1.NetworkChaosAction  `json:"action"`
	Direction             chaosv1alpha1.NetworkDirection    `json:"direction,omitempty"`
	Target                *chaosv1alpha1.NetworkTarget      `json:"target,omitempty"`
	Delay                 *chaosv1alpha1.DelaySpec          `json:"delay,omitempty"`
	Loss                  *chaosv1alpha1.LossSpec           `json:"loss,omitempty"`
	Duplicate             *chaosv1alpha1.DuplicateSpec      `json:"duplicate,omitempty"`
	Corrupt               *chaosv1alpha1.CorruptSpec        `json:"corrupt,omitempty"`
	Bandwidth             *chaosv1alpha1.BandwidthSpec      `json:"bandwidth,omitempty"`
	ExternalTargets       []string                          `json:"externalTargets,omitempty"`
	Scheduler             *chaosv1alpha1.SchedulerSpec      `json:"scheduler,omitempty"`
	BlastRadius           *chaosv1alpha1.BlastRadiusControl `json:"blastRadius,omitempty"`
}

type stressExperimentRequest struct {
	experimentRequestBase `json:",inline"`
	Stressors             chaosv1alpha1.StressorsSpec       `json:"stressors"`
	ContainerNames        []string                          `json:"containerNames,omitempty"`
	Scheduler             *chaosv1alpha1.SchedulerSpec      `json:"scheduler,omitempty"`
	BlastRadius           *chaosv1alpha1.BlastRadiusControl `json:"blastRadius,omitempty"`
}

type httpExperimentRequest struct {
	experimentRequestBase `json:",inline"`
	Target                chaosv1alpha1.HTTPTarget          `json:"target"`
	Port                  int32                             `json:"port"`
	Path                  string                            `json:"path,omitempty"`
	Method                string                            `json:"method,omitempty"`
	Abort                 *chaosv1alpha1.HTTPAbortSpec      `json:"abort,omitempty"`
	Delay                 *chaosv1alpha1.HTTPDelaySpec      `json:"delay,omitempty"`
	Replace               *chaosv1alpha1.HTTPReplaceSpec    `json:"replace,omitempty"`
	Patch                 *chaosv1alpha1.HTTPPatchSpec      `json:"patch,omitempty"`
	Scheduler             *chaosv1alpha1.SchedulerSpec      `json:"scheduler,omitempty"`
	BlastRadius           *chaosv1alpha1.BlastRadiusControl `json:"blastRadius,omitempty"`
}

func createExperiment(w http.ResponseWriter, r *http.Request, kubeClient client.Client, allowed map[string]struct{}) {
	if !isSupportedContentType(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "use application/yaml, text/yaml, or application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds 1 MiB")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_body", "could not read request body")
		}
		return
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := yaml.Unmarshal(body, &envelope); err != nil || strings.TrimSpace(envelope.Type) == "" {
		writeError(w, http.StatusBadRequest, "invalid_experiment", "type is required and must be a supported chaos type")
		return
	}

	var resource any
	var def any
	switch envelope.Type {
	case "PodChaos":
		var req podExperimentRequest
		if !decodeExperimentRequest(body, &req, w) {
			return
		}
		selector, err := experimentSelector(req.AppSelector, req.Selector)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_selector", err.Error())
			return
		}
		obj := &chaosv1alpha1.PodChaos{TypeMeta: metav1.TypeMeta{APIVersion: "chaos.engineering.io/v1alpha1", Kind: "PodChaos"}, ObjectMeta: metav1.ObjectMeta{Name: experimentName(req.Name, "podchaos"), Namespace: req.Namespace}, Spec: chaosv1alpha1.PodChaosSpec{Mode: req.Mode, Value: req.Value, Selector: selector, Action: req.Action, Duration: req.Duration, GracePeriod: req.GracePeriod, Force: req.Force, ContainerNames: req.ContainerNames, Scheduler: req.Scheduler, BlastRadius: req.BlastRadius}}
		resource = obj
		def = chaosResourceDefinition[*chaosv1alpha1.PodChaos]{kind: "PodChaos", path: "/api/v1/podchaos", new: func() *chaosv1alpha1.PodChaos { return &chaosv1alpha1.PodChaos{} }, selectorNamespaces: func(r *chaosv1alpha1.PodChaos) *[]string { return &r.Spec.Selector.Namespaces }, clearStatus: func(r *chaosv1alpha1.PodChaos) { r.Status = chaosv1alpha1.PodChaosStatus{} }}
	case "NetworkChaos":
		var req networkExperimentRequest
		if !decodeExperimentRequest(body, &req, w) {
			return
		}
		selector, err := experimentSelector(req.AppSelector, req.Selector)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_selector", err.Error())
			return
		}
		obj := &chaosv1alpha1.NetworkChaos{TypeMeta: metav1.TypeMeta{APIVersion: "chaos.engineering.io/v1alpha1", Kind: "NetworkChaos"}, ObjectMeta: metav1.ObjectMeta{Name: experimentName(req.Name, "networkchaos"), Namespace: req.Namespace}, Spec: chaosv1alpha1.NetworkChaosSpec{Mode: req.Mode, Value: req.Value, Selector: selector, Action: req.Action, Duration: req.Duration, Direction: req.Direction, Target: req.Target, Delay: req.Delay, Loss: req.Loss, Duplicate: req.Duplicate, Corrupt: req.Corrupt, Bandwidth: req.Bandwidth, ExternalTargets: req.ExternalTargets, Scheduler: req.Scheduler, BlastRadius: req.BlastRadius}}
		resource = obj
		def = chaosResourceDefinition[*chaosv1alpha1.NetworkChaos]{kind: "NetworkChaos", path: "/api/v1/networkchaos", new: func() *chaosv1alpha1.NetworkChaos { return &chaosv1alpha1.NetworkChaos{} }, selectorNamespaces: func(r *chaosv1alpha1.NetworkChaos) *[]string { return &r.Spec.Selector.Namespaces }, otherNamespaces: func(r *chaosv1alpha1.NetworkChaos) []*[]string {
			if r.Spec.Target != nil && r.Spec.Target.Selector != nil {
				return []*[]string{&r.Spec.Target.Selector.Namespaces}
			}
			return nil
		}, clearStatus: func(r *chaosv1alpha1.NetworkChaos) { r.Status = chaosv1alpha1.NetworkChaosStatus{} }}
	case "StressChaos":
		var req stressExperimentRequest
		if !decodeExperimentRequest(body, &req, w) {
			return
		}
		selector, err := experimentSelector(req.AppSelector, req.Selector)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_selector", err.Error())
			return
		}
		obj := &chaosv1alpha1.StressChaos{TypeMeta: metav1.TypeMeta{APIVersion: "chaos.engineering.io/v1alpha1", Kind: "StressChaos"}, ObjectMeta: metav1.ObjectMeta{Name: experimentName(req.Name, "stresschaos"), Namespace: req.Namespace}, Spec: chaosv1alpha1.StressChaosSpec{Mode: req.Mode, Value: req.Value, Selector: selector, Stressors: req.Stressors, ContainerNames: req.ContainerNames, Duration: req.Duration, Scheduler: req.Scheduler, BlastRadius: req.BlastRadius}}
		resource = obj
		def = chaosResourceDefinition[*chaosv1alpha1.StressChaos]{kind: "StressChaos", path: "/api/v1/stresschaos", new: func() *chaosv1alpha1.StressChaos { return &chaosv1alpha1.StressChaos{} }, selectorNamespaces: func(r *chaosv1alpha1.StressChaos) *[]string { return &r.Spec.Selector.Namespaces }, clearStatus: func(r *chaosv1alpha1.StressChaos) { r.Status = chaosv1alpha1.StressChaosStatus{} }}
	case "HTTPChaos":
		var req httpExperimentRequest
		if !decodeExperimentRequest(body, &req, w) {
			return
		}
		selector, err := experimentSelector(req.AppSelector, req.Selector)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_selector", err.Error())
			return
		}
		obj := &chaosv1alpha1.HTTPChaos{TypeMeta: metav1.TypeMeta{APIVersion: "chaos.engineering.io/v1alpha1", Kind: "HTTPChaos"}, ObjectMeta: metav1.ObjectMeta{Name: experimentName(req.Name, "httpchaos"), Namespace: req.Namespace}, Spec: chaosv1alpha1.HTTPChaosSpec{Mode: req.Mode, Value: req.Value, Selector: selector, Duration: req.Duration, Target: req.Target, Port: req.Port, Path: req.Path, Method: req.Method, Abort: req.Abort, Delay: req.Delay, Replace: req.Replace, Patch: req.Patch, Scheduler: req.Scheduler, BlastRadius: req.BlastRadius}}
		resource = obj
		def = chaosResourceDefinition[*chaosv1alpha1.HTTPChaos]{kind: "HTTPChaos", path: "/api/v1/httpchaos", new: func() *chaosv1alpha1.HTTPChaos { return &chaosv1alpha1.HTTPChaos{} }, selectorNamespaces: func(r *chaosv1alpha1.HTTPChaos) *[]string { return &r.Spec.Selector.Namespaces }, clearStatus: func(r *chaosv1alpha1.HTTPChaos) { r.Status = chaosv1alpha1.HTTPChaosStatus{} }}
	default:
		writeError(w, http.StatusUnprocessableEntity, "unsupported_type", "type must be PodChaos, NetworkChaos, StressChaos, or HTTPChaos")
		return
	}
	switch obj := resource.(type) {
	case *chaosv1alpha1.PodChaos:
		dispatchBuiltExperiment(w, r, kubeClient, allowed, obj, def.(chaosResourceDefinition[*chaosv1alpha1.PodChaos]))
	case *chaosv1alpha1.NetworkChaos:
		dispatchBuiltExperiment(w, r, kubeClient, allowed, obj, def.(chaosResourceDefinition[*chaosv1alpha1.NetworkChaos]))
	case *chaosv1alpha1.StressChaos:
		dispatchBuiltExperiment(w, r, kubeClient, allowed, obj, def.(chaosResourceDefinition[*chaosv1alpha1.StressChaos]))
	case *chaosv1alpha1.HTTPChaos:
		dispatchBuiltExperiment(w, r, kubeClient, allowed, obj, def.(chaosResourceDefinition[*chaosv1alpha1.HTTPChaos]))
	}
}

func decodeExperimentRequest(body []byte, destination any, w http.ResponseWriter) bool {
	if err := yaml.UnmarshalStrict(body, destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_experiment", fmt.Sprintf("invalid experiment body: %v", err))
		return false
	}
	return true
}

func experimentSelector(app string, selector *chaosv1alpha1.PodSelector) (chaosv1alpha1.PodSelector, error) {
	if selector != nil && strings.TrimSpace(app) != "" {
		return chaosv1alpha1.PodSelector{}, fmt.Errorf("use either appSelector or selector, not both")
	}
	if selector != nil {
		return *selector, nil
	}
	if strings.TrimSpace(app) == "" {
		return chaosv1alpha1.PodSelector{}, fmt.Errorf("appSelector or selector is required")
	}
	return chaosv1alpha1.PodSelector{LabelSelectors: map[string]string{"app": strings.TrimSpace(app)}}, nil
}

func experimentName(name, kind string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return fmt.Sprintf("%s-%s", kind, strings.ToLower(namegenerator.String(8)))
}

func dispatchBuiltExperiment[T chaosResource](w http.ResponseWriter, r *http.Request, kubeClient client.Client, allowed map[string]struct{}, experiment T, def chaosResourceDefinition[T]) {
	body, err := json.Marshal(experiment)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not build Kubernetes experiment")
		return
	}
	request := r.Clone(r.Context())
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	request.Header.Set("Content-Type", "application/json")
	createChaos(w, request, kubeClient, allowed, def)
}
