package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

// KubernetesNetworkOperations implements domain.NetworkOperations using ExecInPod.
type KubernetesNetworkOperations struct {
	restConfig *rest.Config
}

// NewKubernetesNetworkOperations creates a new KubernetesNetworkOperations.
func NewKubernetesNetworkOperations(cfg *rest.Config) *KubernetesNetworkOperations {
	return &KubernetesNetworkOperations{restConfig: cfg}
}

// ApplyDelay adds network latency via tc netem.
func (k *KubernetesNetworkOperations) ApplyDelay(ctx context.Context, pod corev1.Pod, spec *v1alpha1.DelaySpec) error {
	extra := ""
	if spec.Jitter != "" {
		extra += " " + spec.Jitter
	}
	if spec.Correlation != "" {
		extra += " " + spec.Correlation + "%"
	}
	cmd := fmt.Sprintf("tc qdisc replace dev eth0 root netem delay %s%s && echo applied", spec.Latency, extra)
	return k.exec(ctx, pod, cmd)
}

// ApplyLoss causes packet loss via tc netem.
func (k *KubernetesNetworkOperations) ApplyLoss(ctx context.Context, pod corev1.Pod, spec *v1alpha1.LossSpec) error {
	corr := ""
	if spec.Correlation != "" {
		corr = " " + spec.Correlation + "%"
	}
	cmd := fmt.Sprintf("tc qdisc replace dev eth0 root netem loss %s%%%s && echo applied", spec.Loss, corr)
	return k.exec(ctx, pod, cmd)
}

// ApplyPartition drops all inbound and outbound traffic via iptables.
func (k *KubernetesNetworkOperations) ApplyPartition(ctx context.Context, pod corev1.Pod, targets []string) error {
	cmd := "iptables -I INPUT 1 -j DROP; iptables -I OUTPUT 1 -j DROP && echo applied"
	return k.exec(ctx, pod, cmd)
}

// RemoveNetworkRules removes all tc qdiscs and flushes iptables chains.
func (k *KubernetesNetworkOperations) RemoveNetworkRules(ctx context.Context, pod corev1.Pod) error {
	cmd := `tc qdisc del dev eth0 root 2>/dev/null; iptables -F INPUT 2>/dev/null; iptables -F OUTPUT 2>/dev/null; echo recovered`
	return k.exec(ctx, pod, cmd)
}

func (k *KubernetesNetworkOperations) exec(ctx context.Context, pod corev1.Pod, script string) error {
	container := ""
	if len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}
	out, err := ExecInPod(ctx, k.restConfig, pod, container, []string{"sh", "-c", script})
	if err != nil {
		return fmt.Errorf("exec in pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	if !strings.Contains(out, "applied") && !strings.Contains(out, "recovered") {
		return fmt.Errorf("unexpected output: %s", out)
	}
	return nil
}
