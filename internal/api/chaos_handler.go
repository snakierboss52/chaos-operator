package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"
)

const maxRequestBodyBytes = 1 << 20

type chaosResource interface {
	client.Object
	runtime.Object
	Default(context.Context, runtime.Object) error
	ValidateCreate(context.Context, runtime.Object) (admission.Warnings, error)
}

type chaosResourceDefinition[T chaosResource] struct {
	kind               string
	path               string
	new                func() T
	selectorNamespaces func(T) *[]string
	otherNamespaces    func(T) []*[]string
	clearStatus        func(T)
}

// NewPodChaosHandler serves create and read endpoints for every supported chaos type.
func NewPodChaosHandler(kubeClient client.Client, reader client.Reader, allowedNamespaces []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedNamespaces))
	for _, namespace := range allowedNamespaces {
		if namespace = strings.TrimSpace(namespace); namespace != "" {
			allowed[namespace] = struct{}{}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/experiments", func(w http.ResponseWriter, r *http.Request) {
		createExperiment(w, r, kubeClient, allowed)
	})
	registerChaosResource(mux, kubeClient, reader, allowed, chaosResourceDefinition[*chaosv1alpha1.PodChaos]{
		kind: "PodChaos", path: "/api/v1/podchaos", new: func() *chaosv1alpha1.PodChaos { return &chaosv1alpha1.PodChaos{} },
		selectorNamespaces: func(r *chaosv1alpha1.PodChaos) *[]string { return &r.Spec.Selector.Namespaces },
		clearStatus:        func(r *chaosv1alpha1.PodChaos) { r.Status = chaosv1alpha1.PodChaosStatus{} },
	})
	registerChaosResource(mux, kubeClient, reader, allowed, chaosResourceDefinition[*chaosv1alpha1.NetworkChaos]{
		kind: "NetworkChaos", path: "/api/v1/networkchaos", new: func() *chaosv1alpha1.NetworkChaos { return &chaosv1alpha1.NetworkChaos{} },
		selectorNamespaces: func(r *chaosv1alpha1.NetworkChaos) *[]string { return &r.Spec.Selector.Namespaces },
		otherNamespaces: func(r *chaosv1alpha1.NetworkChaos) []*[]string {
			if r.Spec.Target != nil && r.Spec.Target.Selector != nil {
				return []*[]string{&r.Spec.Target.Selector.Namespaces}
			}
			return nil
		},
		clearStatus: func(r *chaosv1alpha1.NetworkChaos) { r.Status = chaosv1alpha1.NetworkChaosStatus{} },
	})
	registerChaosResource(mux, kubeClient, reader, allowed, chaosResourceDefinition[*chaosv1alpha1.StressChaos]{
		kind: "StressChaos", path: "/api/v1/stresschaos", new: func() *chaosv1alpha1.StressChaos { return &chaosv1alpha1.StressChaos{} },
		selectorNamespaces: func(r *chaosv1alpha1.StressChaos) *[]string { return &r.Spec.Selector.Namespaces },
		clearStatus:        func(r *chaosv1alpha1.StressChaos) { r.Status = chaosv1alpha1.StressChaosStatus{} },
	})
	registerChaosResource(mux, kubeClient, reader, allowed, chaosResourceDefinition[*chaosv1alpha1.HTTPChaos]{
		kind: "HTTPChaos", path: "/api/v1/httpchaos", new: func() *chaosv1alpha1.HTTPChaos { return &chaosv1alpha1.HTTPChaos{} },
		selectorNamespaces: func(r *chaosv1alpha1.HTTPChaos) *[]string { return &r.Spec.Selector.Namespaces },
		clearStatus:        func(r *chaosv1alpha1.HTTPChaos) { r.Status = chaosv1alpha1.HTTPChaosStatus{} },
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "endpoint not found")
	})
	return mux
}

func registerChaosResource[T chaosResource](mux *http.ServeMux, kubeClient client.Client, reader client.Reader, allowed map[string]struct{}, def chaosResourceDefinition[T]) {
	mux.HandleFunc("POST "+def.path, func(w http.ResponseWriter, r *http.Request) {
		createChaos(w, r, kubeClient, allowed, def)
	})
	mux.HandleFunc("GET "+def.path+"/{namespace}/{name}", func(w http.ResponseWriter, r *http.Request) {
		getChaos(w, r, reader, allowed, def)
	})
}

func createChaos[T chaosResource](w http.ResponseWriter, r *http.Request, kubeClient client.Client, allowed map[string]struct{}, def chaosResourceDefinition[T]) {
	if !isSupportedContentType(r.Header.Get("Content-Type")) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "use application/yaml, text/yaml, or application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	manifest, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_body", "could not read request body")
		return
	}

	experiment := def.new()
	if err := yaml.UnmarshalStrict(manifest, experiment); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_manifest", fmt.Sprintf("could not decode %s manifest: %v", def.kind, err))
		return
	}
	gvk := experiment.GetObjectKind().GroupVersionKind()
	if gvk.GroupVersion().String() != "" && gvk.GroupVersion().String() != "chaos.engineering.io/v1alpha1" {
		writeError(w, http.StatusBadRequest, "invalid_manifest", "apiVersion must be chaos.engineering.io/v1alpha1")
		return
	}
	if gvk.Kind != "" && gvk.Kind != def.kind {
		writeError(w, http.StatusBadRequest, "invalid_manifest", "kind must be "+def.kind)
		return
	}
	experiment.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "chaos.engineering.io", Version: "v1alpha1", Kind: def.kind})

	if experiment.GetName() == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", "metadata.name is required")
		return
	}
	if !isNamespaceAllowed(experiment.GetNamespace(), allowed) {
		writeError(w, http.StatusForbidden, "namespace_not_allowed", "namespace must be explicitly set to an allowed namespace")
		return
	}
	namespaces := def.selectorNamespaces(experiment)
	if len(*namespaces) == 0 {
		*namespaces = []string{experiment.GetNamespace()}
	}
	if err := validateAllowedSelectorNamespaces(*namespaces, allowed); err != nil {
		writeError(w, err.status, err.code, err.message)
		return
	}
	var otherNamespaces []*[]string
	if def.otherNamespaces != nil {
		otherNamespaces = def.otherNamespaces(experiment)
	}
	for _, other := range otherNamespaces {
		if len(*other) == 0 {
			*other = []string{experiment.GetNamespace()}
		}
		if err := validateAllowedSelectorNamespaces(*other, allowed); err != nil {
			writeError(w, err.status, err.code, err.message)
			return
		}
	}
	if err := experiment.Default(r.Context(), experiment); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		return
	}
	if _, err := experiment.ValidateCreate(r.Context(), experiment); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		return
	}
	def.clearStatus(experiment)
	if err := kubeClient.Create(r.Context(), experiment); err != nil {
		switch {
		case apierrors.IsAlreadyExists(err):
			writeError(w, http.StatusConflict, "already_exists", "a "+def.kind+" with this namespace and name already exists")
		case apierrors.IsInvalid(err):
			writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		case apierrors.IsForbidden(err):
			writeError(w, http.StatusForbidden, "forbidden", "the operator is not allowed to create this experiment")
		default:
			log.Log.Error(err, "Failed to create chaos resource", "kind", def.kind, "namespace", experiment.GetNamespace(), "name", experiment.GetName())
			writeError(w, http.StatusInternalServerError, "create_failed", "could not create "+def.kind+" resource")
		}
		return
	}
	w.Header().Set("Location", fmt.Sprintf("%s/%s/%s", def.path, experiment.GetNamespace(), experiment.GetName()))
	writeJSON(w, http.StatusCreated, experiment)
}

type namespaceValidationError struct {
	status        int
	code, message string
}

func validateAllowedSelectorNamespaces(namespaces []string, allowed map[string]struct{}) *namespaceValidationError {
	if len(namespaces) != 1 {
		return &namespaceValidationError{http.StatusUnprocessableEntity, "invalid_selector", "exactly one selector namespace is currently supported"}
	}
	if !isNamespaceAllowed(namespaces[0], allowed) {
		return &namespaceValidationError{http.StatusForbidden, "target_namespace_not_allowed", "selector namespace is not allowed"}
	}
	return nil
}

func getChaos[T chaosResource](w http.ResponseWriter, r *http.Request, reader client.Reader, allowed map[string]struct{}, def chaosResourceDefinition[T]) {
	namespace, name := r.PathValue("namespace"), r.PathValue("name")
	if !isNamespaceAllowed(namespace, allowed) {
		writeError(w, http.StatusForbidden, "namespace_not_allowed", "namespace is not allowed")
		return
	}
	experiment := def.new()
	if err := reader.Get(r.Context(), types.NamespacedName{Namespace: namespace, Name: name}, experiment); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", def.kind+" resource not found")
		} else {
			writeError(w, http.StatusInternalServerError, "read_failed", "could not read "+def.kind+" resource")
		}
		return
	}
	experiment.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{Group: "chaos.engineering.io", Version: "v1alpha1", Kind: def.kind})
	writeJSON(w, http.StatusOK, experiment)
}

func isSupportedContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch mediaType {
	case "application/yaml", "text/yaml", "application/x-yaml", "application/json":
		return true
	default:
		return false
	}
}

func isNamespaceAllowed(namespace string, allowed map[string]struct{}) bool {
	_, ok := allowed[namespace]
	return namespace != "" && ok
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
