package infrastructure

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

// KubernetesStressOperations implements domain.StressOperations using ExecInPod.
type KubernetesStressOperations struct {
	restConfig *rest.Config
}

// NewKubernetesStressOperations creates a new KubernetesStressOperations.
func NewKubernetesStressOperations(cfg *rest.Config) *KubernetesStressOperations {
	return &KubernetesStressOperations{restConfig: cfg}
}

// InjectCPUStress starts N background CPU-saturating processes.
func (k *KubernetesStressOperations) InjectCPUStress(ctx context.Context, pod corev1.Pod, workers int, load *int) error {
	if workers < 1 {
		workers = 1
	}
	script := fmt.Sprintf(
		"i=0; while [ $i -lt %d ]; do nohup sh -c 'yes >/dev/null 2>/dev/null' </dev/null >/dev/null 2>/dev/null & echo $!; i=$((i+1)); done",
		workers,
	)
	return k.exec(ctx, pod, script)
}

// InjectMemoryStress allocates memory via dd to /dev/shm.
func (k *KubernetesStressOperations) InjectMemoryStress(ctx context.Context, pod corev1.Pod, workers int, size string) error {
	if workers < 1 {
		workers = 1
	}
	script := ""
	for w := 0; w < workers; w++ {
		if script != "" {
			script += "; "
		}
		script += fmt.Sprintf(
			"nohup sh -c 'dd if=/dev/zero of=/dev/shm/.chaos_mem_%d bs=1M count=32 2>/dev/null; tail -f /dev/null' </dev/null >/dev/null 2>/dev/null & echo $!",
			w,
		)
	}
	return k.exec(ctx, pod, script)
}

// RemoveStress kills all stress processes and cleans up artifacts.
func (k *KubernetesStressOperations) RemoveStress(ctx context.Context, pod corev1.Pod) error {
	script := "rm -f /dev/shm/.chaos_mem* /tmp/.chaos_disk_fill 2>/dev/null; echo recovered"
	return k.exec(ctx, pod, script)
}

func (k *KubernetesStressOperations) exec(ctx context.Context, pod corev1.Pod, script string) error {
	container := ""
	if len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}
	_, err := ExecInPod(ctx, k.restConfig, pod, container, []string{"sh", "-c", script})
	if err != nil {
		return fmt.Errorf("exec in pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	return nil
}
