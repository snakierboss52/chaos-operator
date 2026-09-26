package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubernetesPodOperations implements domain.PodOperations interface
type KubernetesPodOperations struct {
	client     client.Client
	restConfig *rest.Config
}

// NewKubernetesPodOperations creates a new KubernetesPodOperations
func NewKubernetesPodOperations(client client.Client, restConfig *rest.Config) *KubernetesPodOperations {
	return &KubernetesPodOperations{
		client:     client,
		restConfig: restConfig,
	}
}

// KillPod implements domain.PodOperations
func (k *KubernetesPodOperations) KillPod(ctx context.Context, pod corev1.Pod, gracePeriod *int64) error {
	deleteOptions := &client.DeleteOptions{}
	if gracePeriod != nil {
		deleteOptions.GracePeriodSeconds = gracePeriod
	}

	if err := k.client.Delete(ctx, &pod, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	return nil
}

// FailPod implements domain.PodOperations
func (k *KubernetesPodOperations) FailPod(ctx context.Context, pod corev1.Pod) error {
	return k.KillPod(ctx, pod, nil)
}

// KillContainer implements domain.PodOperations by sending SIGKILL to PID 1 of
// the specified container via exec, which terminates the container process.
func (k *KubernetesPodOperations) KillContainer(ctx context.Context, pod corev1.Pod, containerName string) error {
	if k.restConfig == nil {
		return fmt.Errorf("restConfig is required for container kill operations")
	}

	// Send SIGKILL to PID 1 in the target container, terminating it.
	// The kubelet will restart the container according to the pod's restart policy.
	killCmd := []string{"sh", "-c", "kill -9 1"}
	_, err := ExecInPod(ctx, k.restConfig, pod, containerName, killCmd)
	if err != nil {
		return fmt.Errorf("failed to kill container %s in pod %s/%s: %w", containerName, pod.Namespace, pod.Name, err)
	}

	return nil
}

// ExecInPod runs a command inside a pod container and returns trimmed stdout.
// Uses the Kubernetes SPDY exec subresource via remotecommand.
// A 15-second deadline is applied automatically to prevent the reconciler from hanging
// if background processes in the container keep the exec channel open.
func ExecInPod(ctx context.Context, cfg *rest.Config, pod corev1.Pod, container string, command []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("creating clientset: %w", err)
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("creating SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return "", fmt.Errorf("exec failed (stderr: %s): %w", strings.TrimSpace(stderr.String()), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// KubernetesPodLister implements domain.PodLister interface
type KubernetesPodLister struct {
	client client.Client
}

// NewKubernetesPodLister creates a new KubernetesPodLister
func NewKubernetesPodLister(client client.Client) *KubernetesPodLister {
	return &KubernetesPodLister{
		client: client,
	}
}

// ListPods implements domain.PodLister
func (k *KubernetesPodLister) ListPods(ctx context.Context, selector v1alpha1.PodSelector) ([]corev1.Pod, error) {
	listOpts := []client.ListOption{}

	if len(selector.Namespaces) > 0 {
		listOpts = append(listOpts, client.InNamespace(selector.Namespaces[0]))
	}

	if len(selector.LabelSelectors) > 0 {
		listOpts = append(listOpts, client.MatchingLabels(selector.LabelSelectors))
	}

	if len(selector.FieldSelectors) > 0 {
		for key, value := range selector.FieldSelectors {
			listOpts = append(listOpts, client.MatchingFields{key: value})
		}
	}

	var podList corev1.PodList
	if err := k.client.List(ctx, &podList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	var result []corev1.Pod
	for _, pod := range podList.Items {
		if pod.DeletionTimestamp.IsZero() && matchesPodSelector(pod, selector) {
			result = append(result, pod)
		}
	}

	return result, nil
}

// matchesPodSelector checks if a pod matches additional selector criteria
func matchesPodSelector(pod corev1.Pod, selector v1alpha1.PodSelector) bool {
	if len(selector.Namespaces) > 0 {
		namespaceMatch := false
		for _, ns := range selector.Namespaces {
			if pod.Namespace == ns {
				namespaceMatch = true
				break
			}
		}
		if !namespaceMatch {
			return false
		}
	}

	if len(selector.PodPhaseSelectors) > 0 {
		phaseMatch := false
		for _, phase := range selector.PodPhaseSelectors {
			if string(pod.Status.Phase) == phase {
				phaseMatch = true
				break
			}
		}
		if !phaseMatch {
			return false
		}
	}

	if len(selector.Pods) > 0 {
		podNameMatch := false
		for _, podName := range selector.Pods {
			if pod.Name == podName {
				podNameMatch = true
				break
			}
		}
		if !podNameMatch {
			return false
		}
	}

	return true
}
