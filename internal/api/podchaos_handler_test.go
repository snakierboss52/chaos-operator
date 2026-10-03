package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	chaosv1alpha1 "goland-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreatePodChaosFromSampleManifest(t *testing.T) {
	sample, err := os.ReadFile("../../samples/podchaos/basic-kill-one.yaml")
	if err != nil {
		t.Fatalf("read sample manifest: %v", err)
	}

	handler, kubeClient := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podchaos", strings.NewReader(string(sample)))
	req.Header.Set("Content-Type", "application/yaml")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"apiVersion":"chaos.engineering.io/v1alpha1"`) || !strings.Contains(response.Body.String(), `"kind":"PodChaos"`) {
		t.Fatalf("expected response TypeMeta, got: %s", response.Body.String())
	}
	if got := response.Header().Get("Location"); got != "/api/v1/podchaos/chaos-demo/basic-kill-one" {
		t.Fatalf("unexpected Location header %q", got)
	}

	created := &chaosv1alpha1.PodChaos{}
	if err := kubeClient.Get(req.Context(), types.NamespacedName{Namespace: "chaos-demo", Name: "basic-kill-one"}, created); err != nil {
		t.Fatalf("get created PodChaos: %v", err)
	}
	if created.Spec.Action != chaosv1alpha1.PodKillAction || created.Spec.Mode != chaosv1alpha1.OnePodMode {
		t.Fatalf("unexpected experiment spec: %#v", created.Spec)
	}
	if created.Spec.BlastRadius == nil || created.Spec.BlastRadius.MaxPercentage == nil || *created.Spec.BlastRadius.MaxPercentage != 50 {
		t.Fatalf("expected webhook default blastRadius.maxPercentage=50, got %#v", created.Spec.BlastRadius)
	}
	if created.Status.Phase != "" {
		t.Fatalf("expected controller-owned status to be empty, got %q", created.Status.Phase)
	}
}

func TestCreatePodChaosRejectsInvalidDuration(t *testing.T) {
	handler, _ := newTestHandler(t)
	manifest := `apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: invalid-duration
  namespace: chaos-demo
spec:
  mode: one
  action: pod-kill
  duration: someday
  selector:
    labelSelectors:
      app: nginx
`

	response := postManifest(handler, manifest)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, response.Code, response.Body.String())
	}
}

func TestCreatePodChaosRejectsNamespaceOutsideAllowlist(t *testing.T) {
	handler, _ := newTestHandler(t)
	manifest := `apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: wrong-namespace
  namespace: default
spec:
  mode: one
  action: pod-kill
  selector:
    labelSelectors:
      app: nginx
`

	response := postManifest(handler, manifest)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.Code, response.Body.String())
	}
}

func TestCreatePodChaosRejectsTargetNamespaceOutsideAllowlist(t *testing.T) {
	handler, _ := newTestHandler(t)
	manifest := `apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: wrong-target-namespace
  namespace: chaos-demo
spec:
  mode: one
  action: pod-kill
  selector:
    namespaces:
      - kube-system
    labelSelectors:
      app: nginx
`

	response := postManifest(handler, manifest)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.Code, response.Body.String())
	}
}

func TestCreatePodChaosDefaultsSelectorNamespace(t *testing.T) {
	handler, kubeClient := newTestHandler(t)
	manifest := `apiVersion: chaos.engineering.io/v1alpha1
kind: PodChaos
metadata:
  name: default-target-namespace
  namespace: chaos-demo
spec:
  mode: one
  action: pod-kill
  selector:
    labelSelectors:
      app: nginx
`

	response := postManifest(handler, manifest)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	created := &chaosv1alpha1.PodChaos{}
	if err := kubeClient.Get(context.Background(), types.NamespacedName{Namespace: "chaos-demo", Name: "default-target-namespace"}, created); err != nil {
		t.Fatalf("get created PodChaos: %v", err)
	}
	if len(created.Spec.Selector.Namespaces) != 1 || created.Spec.Selector.Namespaces[0] != "chaos-demo" {
		t.Fatalf("expected selector namespace to default to chaos-demo, got %#v", created.Spec.Selector.Namespaces)
	}
}

func newTestHandler(t *testing.T) (http.Handler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := chaosv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add chaos API to scheme: %v", err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	return NewPodChaosHandler(kubeClient, kubeClient, []string{"chaos-demo"}), kubeClient
}

func postManifest(handler http.Handler, manifest string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/podchaos", strings.NewReader(manifest))
	req.Header.Set("Content-Type", "application/yaml")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}
