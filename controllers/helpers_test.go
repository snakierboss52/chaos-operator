package controllers

import (
	"testing"

	chaosv1alpha1 "goland-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseSizeMB(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"64MB", 64},
		{"1GB", 1024},
		{"512KB", 1},
		{"256MB", 256},
		{"2GB", 2048},
		{"100", 100},
		{"", 0},
		{"0MB", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSizeMB(tt.input)
			if result != tt.expected {
				t.Errorf("parseSizeMB(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSelectContainer(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main"},
				{Name: "sidecar"},
			},
		},
	}

	t.Run("explicit container names", func(t *testing.T) {
		result := selectContainer([]string{"sidecar"}, pod)
		if result != "sidecar" {
			t.Errorf("expected 'sidecar', got %q", result)
		}
	})

	t.Run("empty container names defaults to first", func(t *testing.T) {
		result := selectContainer(nil, pod)
		if result != "main" {
			t.Errorf("expected 'main', got %q", result)
		}
	})

	t.Run("empty pod containers", func(t *testing.T) {
		emptyPod := corev1.Pod{}
		result := selectContainer(nil, emptyPod)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestActionLabel(t *testing.T) {
	tests := []struct {
		name     string
		spec     chaosv1alpha1.StressorsSpec
		expected string
	}{
		{
			name:     "cpu only",
			spec:     chaosv1alpha1.StressorsSpec{CPU: &chaosv1alpha1.CPUStressor{Workers: 1}},
			expected: "cpu-stress",
		},
		{
			name:     "memory only",
			spec:     chaosv1alpha1.StressorsSpec{Memory: &chaosv1alpha1.MemoryStressor{Workers: 1, Size: "64MB"}},
			expected: "memory-stress",
		},
		{
			name:     "disk only",
			spec:     chaosv1alpha1.StressorsSpec{Disk: &chaosv1alpha1.DiskStressor{Size: "512MB"}},
			expected: "disk-fill",
		},
		{
			name:     "io only",
			spec:     chaosv1alpha1.StressorsSpec{IO: &chaosv1alpha1.IOStressor{Workers: 2}},
			expected: "io-stress",
		},
		{
			name: "combined cpu+memory",
			spec: chaosv1alpha1.StressorsSpec{
				CPU:    &chaosv1alpha1.CPUStressor{Workers: 1},
				Memory: &chaosv1alpha1.MemoryStressor{Workers: 1, Size: "64MB"},
			},
			expected: "combined-stress",
		},
		{
			name: "combined cpu+disk",
			spec: chaosv1alpha1.StressorsSpec{
				CPU:  &chaosv1alpha1.CPUStressor{Workers: 1},
				Disk: &chaosv1alpha1.DiskStressor{Size: "256MB"},
			},
			expected: "combined-stress",
		},
		{
			name:     "empty stressors",
			spec:     chaosv1alpha1.StressorsSpec{},
			expected: "stress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := actionLabel(tt.spec)
			if result != tt.expected {
				t.Errorf("actionLabel() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildStressScript(t *testing.T) {
	t.Run("cpu stress generates yes loop", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			CPU: &chaosv1alpha1.CPUStressor{Workers: 2},
		}
		script := buildStressScript(spec)
		if script == "" {
			t.Error("expected non-empty script")
		}
		if !contains(script, "yes >/dev/null") {
			t.Error("expected 'yes >/dev/null' in CPU stress script")
		}
	})

	t.Run("memory stress generates dd to shm", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			Memory: &chaosv1alpha1.MemoryStressor{Workers: 1, Size: "128MB"},
		}
		script := buildStressScript(spec)
		if !contains(script, "/dev/shm/.chaos_mem") {
			t.Error("expected '/dev/shm/.chaos_mem' in memory stress script")
		}
	})

	t.Run("disk fill generates dd to path", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			Disk: &chaosv1alpha1.DiskStressor{Size: "256MB", Path: "/data"},
		}
		script := buildStressScript(spec)
		if !contains(script, "/data/.chaos_disk_fill") {
			t.Error("expected '/data/.chaos_disk_fill' in disk stress script")
		}
	})

	t.Run("disk fill defaults to /tmp", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			Disk: &chaosv1alpha1.DiskStressor{Size: "64MB"},
		}
		script := buildStressScript(spec)
		if !contains(script, "/tmp/.chaos_disk_fill") {
			t.Error("expected '/tmp/.chaos_disk_fill' in disk stress script")
		}
	})

	t.Run("io stress generates dd loop", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			IO: &chaosv1alpha1.IOStressor{Workers: 2, Size: "8k"},
		}
		script := buildStressScript(spec)
		if !contains(script, "/dev/urandom") {
			t.Error("expected '/dev/urandom' in IO stress script")
		}
		if !contains(script, "bs=8k") {
			t.Error("expected 'bs=8k' in IO stress script")
		}
	})

	t.Run("io stress defaults to 4k block size", func(t *testing.T) {
		spec := chaosv1alpha1.StressorsSpec{
			IO: &chaosv1alpha1.IOStressor{Workers: 1},
		}
		script := buildStressScript(spec)
		if !contains(script, "bs=4k") {
			t.Error("expected 'bs=4k' as default in IO stress script")
		}
	})
}

func TestBuildNetScript(t *testing.T) {
	t.Run("delay action", func(t *testing.T) {
		spec := chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.NetworkDelayAction,
			Delay:  &chaosv1alpha1.DelaySpec{Latency: "100ms", Jitter: "10ms"},
		}
		script := buildNetScript(spec)
		if !contains(script, "netem delay 100ms 10ms") {
			t.Errorf("expected netem delay command, got: %s", script)
		}
	})

	t.Run("loss action", func(t *testing.T) {
		spec := chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.NetworkLossAction,
			Loss:   &chaosv1alpha1.LossSpec{Loss: "25"},
		}
		script := buildNetScript(spec)
		if !contains(script, "netem loss 25%") {
			t.Errorf("expected netem loss command, got: %s", script)
		}
	})

	t.Run("partition action", func(t *testing.T) {
		spec := chaosv1alpha1.NetworkChaosSpec{
			Action: chaosv1alpha1.NetworkPartitionAction,
		}
		script := buildNetScript(spec)
		if !contains(script, "iptables -I INPUT 1 -j DROP") {
			t.Errorf("expected iptables DROP rule, got: %s", script)
		}
	})

	t.Run("bandwidth action", func(t *testing.T) {
		spec := chaosv1alpha1.NetworkChaosSpec{
			Action:    chaosv1alpha1.NetworkBandwidthAction,
			Bandwidth: &chaosv1alpha1.BandwidthSpec{Rate: "1mbps", Limit: 10000, Buffer: 1600},
		}
		script := buildNetScript(spec)
		if !contains(script, "tbf rate 1mbps") {
			t.Errorf("expected tbf rate command, got: %s", script)
		}
	})
}

func TestHTTPActionLabel(t *testing.T) {
	tests := []struct {
		name     string
		spec     chaosv1alpha1.HTTPChaosSpec
		expected string
	}{
		{"abort", chaosv1alpha1.HTTPChaosSpec{Abort: &chaosv1alpha1.HTTPAbortSpec{StatusCode: 500}}, "abort"},
		{"delay", chaosv1alpha1.HTTPChaosSpec{Delay: &chaosv1alpha1.HTTPDelaySpec{Latency: "1s"}}, "delay"},
		{"replace", chaosv1alpha1.HTTPChaosSpec{Replace: &chaosv1alpha1.HTTPReplaceSpec{}}, "replace"},
		{"patch", chaosv1alpha1.HTTPChaosSpec{Patch: &chaosv1alpha1.HTTPPatchSpec{}}, "patch"},
		{"default", chaosv1alpha1.HTTPChaosSpec{}, "http-fault"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpActionLabel(tt.spec)
			if result != tt.expected {
				t.Errorf("httpActionLabel() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildHTTPFaultScript(t *testing.T) {
	t.Run("abort fault", func(t *testing.T) {
		spec := chaosv1alpha1.HTTPChaosSpec{
			Port:  80,
			Abort: &chaosv1alpha1.HTTPAbortSpec{StatusCode: 503},
		}
		script, err := buildHTTPFaultScript(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contains(script, "return 503") {
			t.Errorf("expected 'return 503' in script, got: %s", script)
		}
		if !contains(script, "nginx -t") {
			t.Error("expected nginx test command in script")
		}
	})

	t.Run("delay fault", func(t *testing.T) {
		spec := chaosv1alpha1.HTTPChaosSpec{
			Port:  8080,
			Delay: &chaosv1alpha1.HTTPDelaySpec{Latency: "500ms"},
		}
		script, err := buildHTTPFaultScript(spec)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contains(script, "X-Chaos-Delay") {
			t.Errorf("expected delay header in script, got: %s", script)
		}
	})

	t.Run("no fault type returns error", func(t *testing.T) {
		spec := chaosv1alpha1.HTTPChaosSpec{Port: 80}
		_, err := buildHTTPFaultScript(spec)
		if err == nil {
			t.Error("expected error when no fault type is specified")
		}
	})
}

// Needed by the test since the controllers/stresschaos_controller.go uses
// max(a, b) as a helper.
func TestMax(t *testing.T) {
	_ = metav1.Now() // ensure metav1 import used
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(7, 2) != 7 {
		t.Error("max(7,2) should be 7")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr)) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
