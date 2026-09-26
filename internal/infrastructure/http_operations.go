package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"goland-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

// KubernetesHTTPOperations implements domain.HTTPOperations using ExecInPod.
type KubernetesHTTPOperations struct {
	restConfig *rest.Config
}

// NewKubernetesHTTPOperations creates a new KubernetesHTTPOperations.
func NewKubernetesHTTPOperations(cfg *rest.Config) *KubernetesHTTPOperations {
	return &KubernetesHTTPOperations{restConfig: cfg}
}

// InjectAbort modifies nginx config to return an error status code.
func (k *KubernetesHTTPOperations) InjectAbort(ctx context.Context, pod corev1.Pod, spec *v1alpha1.HTTPAbortSpec) error {
	body := spec.Body
	if body == "" {
		body = fmt.Sprintf(`{"error":"chaos-abort","code":%d}`, spec.StatusCode)
	}
	body = strings.ReplaceAll(body, "'", `'"'"'`)

	nginxConf := fmt.Sprintf(`server {
    listen 80;
    server_name _;
    location / {
        default_type application/json;
        return %d '%s';
    }
}`, spec.StatusCode, body)

	return k.applyNginxConfig(ctx, pod, nginxConf)
}

// InjectDelay adds chaos delay headers to nginx responses.
func (k *KubernetesHTTPOperations) InjectDelay(ctx context.Context, pod corev1.Pod, spec *v1alpha1.HTTPDelaySpec) error {
	nginxConf := fmt.Sprintf(`server {
    listen 80;
    server_name _;
    location / {
        default_type application/json;
        add_header X-Chaos-Delay "%s" always;
        add_header X-Chaos-Injected "true" always;
        return 200 '{"chaos":"delay","latency":"%s"}';
    }
}`, spec.Latency, spec.Latency)

	return k.applyNginxConfig(ctx, pod, nginxConf)
}

// RemoveHTTPRules restores the original nginx config from backup.
func (k *KubernetesHTTPOperations) RemoveHTTPRules(ctx context.Context, pod corev1.Pod) error {
	script := `if [ -f /tmp/nginx_chaos_backup.conf ]; then
  cp /tmp/nginx_chaos_backup.conf /etc/nginx/conf.d/default.conf && nginx -s reload && echo recovered
else
  echo no-backup
fi`
	return k.exec(ctx, pod, script)
}

func (k *KubernetesHTTPOperations) applyNginxConfig(ctx context.Context, pod corev1.Pod, nginxConf string) error {
	nginxConf = strings.ReplaceAll(nginxConf, "'", `'"'"'`)
	script := fmt.Sprintf(`[ -f /tmp/nginx_chaos_backup.conf ] || cp /etc/nginx/conf.d/default.conf /tmp/nginx_chaos_backup.conf
printf '%%s\n' '%s' > /etc/nginx/conf.d/default.conf
nginx -t 2>/tmp/nginx_test.log && nginx -s reload && echo applied || (cat /tmp/nginx_test.log; echo failed)`, nginxConf)

	return k.exec(ctx, pod, script)
}

func (k *KubernetesHTTPOperations) exec(ctx context.Context, pod corev1.Pod, script string) error {
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
