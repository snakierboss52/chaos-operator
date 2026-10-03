package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const maxRequestBodyBytes = 1 << 20

// NewPodChaosHandler creates the HTTP endpoints for submitting and reading
// PodChaos resources. Requests are constrained to the configured namespaces.
func NewPodChaosHandler(kubeClient client.Client, reader client.Reader, allowedNamespaces []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedNamespaces))
	for _, namespace := range allowedNamespaces {
		if namespace = strings.TrimSpace(namespace); namespace != "" {
			allowed[namespace] = struct{}{}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/podchaos", func(w http.ResponseWriter, r *http.Request) {
		createPodChaos(w, r, kubeClient, allowed)
	})
	mux.HandleFunc("GET /api/v1/podchaos/{namespace}/{name}", func(w http.ResponseWriter, r *http.Request) {
		getPodChaos(w, r, reader, allowed)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "endpoint not found")
	})

	return mux
}

func createPodChaos(w http.ResponseWriter, r *http.Request, kubeClient client.Client, allowed map[string]struct{}) {
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

	var experiment chaosv1alpha1.PodChaos
	if err := yaml.UnmarshalStrict(manifest, &experiment); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_manifest", fmt.Sprintf("could not decode PodChaos manifest: %v", err))
		return
	}
	if experiment.APIVersion != "" && experiment.APIVersion != "chaos.engineering.io/v1alpha1" {
		writeError(w, http.StatusBadRequest, "invalid_manifest", "apiVersion must be chaos.engineering.io/v1alpha1")
		return
	}
	if experiment.Kind != "" && experiment.Kind != "PodChaos" {
		writeError(w, http.StatusBadRequest, "invalid_manifest", "kind must be PodChaos")
		return
	}
	experiment.APIVersion = "chaos.engineering.io/v1alpha1"
	experiment.Kind = "PodChaos"

	if experiment.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", "metadata.name is required")
		return
	}
	if !isNamespaceAllowed(experiment.Namespace, allowed) {
		writeError(w, http.StatusForbidden, "namespace_not_allowed", "namespace must be explicitly set to an allowed namespace")
		return
	}
	if len(experiment.Spec.Selector.Namespaces) == 0 {
		experiment.Spec.Selector.Namespaces = []string{experiment.Namespace}
	}
	if len(experiment.Spec.Selector.Namespaces) != 1 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_selector", "exactly one selector namespace is currently supported")
		return
	}
	if !isNamespaceAllowed(experiment.Spec.Selector.Namespaces[0], allowed) {
		writeError(w, http.StatusForbidden, "target_namespace_not_allowed", "selector namespace is not allowed")
		return
	}

	// Apply the same defaults and validation used by the PodChaos admission webhook.
	if err := experiment.Default(r.Context(), &experiment); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		return
	}
	if _, err := experiment.ValidateCreate(r.Context(), &experiment); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		return
	}

	// Status is controller-owned; clients can only submit desired state.
	experiment.Status = chaosv1alpha1.PodChaosStatus{}
	if err := kubeClient.Create(r.Context(), &experiment); err != nil {
		switch {
		case apierrors.IsAlreadyExists(err):
			writeError(w, http.StatusConflict, "already_exists", "a PodChaos with this namespace and name already exists")
		case apierrors.IsInvalid(err):
			writeError(w, http.StatusUnprocessableEntity, "invalid_experiment", err.Error())
		case apierrors.IsForbidden(err):
			writeError(w, http.StatusForbidden, "forbidden", "the operator is not allowed to create this experiment")
		default:
			writeError(w, http.StatusInternalServerError, "create_failed", "could not create PodChaos resource")
		}
		return
	}

	setPodChaosTypeMeta(&experiment)
	w.Header().Set("Location", fmt.Sprintf("/api/v1/podchaos/%s/%s", experiment.Namespace, experiment.Name))
	writeJSON(w, http.StatusCreated, &experiment)
}

func getPodChaos(w http.ResponseWriter, r *http.Request, reader client.Reader, allowed map[string]struct{}) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	if !isNamespaceAllowed(namespace, allowed) {
		writeError(w, http.StatusForbidden, "namespace_not_allowed", "namespace is not allowed")
		return
	}

	var experiment chaosv1alpha1.PodChaos
	if err := reader.Get(r.Context(), types.NamespacedName{Namespace: namespace, Name: name}, &experiment); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "PodChaos resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "read_failed", "could not read PodChaos resource")
		return
	}

	setPodChaosTypeMeta(&experiment)
	writeJSON(w, http.StatusOK, &experiment)
}

func setPodChaosTypeMeta(experiment *chaosv1alpha1.PodChaos) {
	experiment.APIVersion = "chaos.engineering.io/v1alpha1"
	experiment.Kind = "PodChaos"
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
	writeJSON(w, statusCode, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
