package v1alpha1

import (
	"testing"
)

func TestValidatePodChaosSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    PodChaosSpec
		wantErr bool
	}{
		{
			name:    "valid pod-kill",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: OnePodMode},
			wantErr: false,
		},
		{
			name:    "valid pod-failure",
			spec:    PodChaosSpec{Action: PodFailureAction, Mode: AllMode},
			wantErr: false,
		},
		{
			name: "valid container-kill",
			spec: PodChaosSpec{
				Action:         ContainerKillAction,
				Mode:           OnePodMode,
				ContainerNames: []string{"sidecar"},
			},
			wantErr: false,
		},
		{
			name:    "container-kill without containerNames",
			spec:    PodChaosSpec{Action: ContainerKillAction, Mode: OnePodMode},
			wantErr: true,
		},
		{
			name:    "invalid duration",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: OnePodMode, Duration: "not-a-duration"},
			wantErr: true,
		},
		{
			name:    "valid duration",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: OnePodMode, Duration: "5m30s"},
			wantErr: false,
		},
		{
			name:    "fixed mode without value",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: FixedMode},
			wantErr: true,
		},
		{
			name:    "fixed mode with valid value",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: FixedMode, Value: "3"},
			wantErr: false,
		},
		{
			name:    "fixed-percent mode with invalid value",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: FixedPercentMode, Value: "150"},
			wantErr: true,
		},
		{
			name:    "fixed-percent mode with valid value",
			spec:    PodChaosSpec{Action: PodKillAction, Mode: FixedPercentMode, Value: "50"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePodChaosSpec(&tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePodChaosSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNetworkChaosSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    NetworkChaosSpec
		wantErr bool
	}{
		{
			name:    "valid delay",
			spec:    NetworkChaosSpec{Action: NetworkDelayAction, Mode: OnePodMode, Delay: &DelaySpec{Latency: "100ms"}},
			wantErr: false,
		},
		{
			name:    "delay without spec",
			spec:    NetworkChaosSpec{Action: NetworkDelayAction, Mode: OnePodMode},
			wantErr: true,
		},
		{
			name:    "loss without spec",
			spec:    NetworkChaosSpec{Action: NetworkLossAction, Mode: OnePodMode},
			wantErr: true,
		},
		{
			name:    "valid loss",
			spec:    NetworkChaosSpec{Action: NetworkLossAction, Mode: OnePodMode, Loss: &LossSpec{Loss: "25"}},
			wantErr: false,
		},
		{
			name:    "valid partition (no extra spec needed)",
			spec:    NetworkChaosSpec{Action: NetworkPartitionAction, Mode: OnePodMode},
			wantErr: false,
		},
		{
			name:    "bandwidth without spec",
			spec:    NetworkChaosSpec{Action: NetworkBandwidthAction, Mode: OnePodMode},
			wantErr: true,
		},
		{
			name:    "invalid duration",
			spec:    NetworkChaosSpec{Action: NetworkPartitionAction, Mode: OnePodMode, Duration: "bad"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetworkChaosSpec(&tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNetworkChaosSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStressChaosSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    StressChaosSpec
		wantErr bool
	}{
		{
			name:    "no stressors",
			spec:    StressChaosSpec{Mode: OnePodMode, Stressors: StressorsSpec{}},
			wantErr: true,
		},
		{
			name: "valid cpu stressor",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{CPU: &CPUStressor{Workers: 2}},
			},
			wantErr: false,
		},
		{
			name: "valid disk stressor",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{Disk: &DiskStressor{Size: "512MB"}},
			},
			wantErr: false,
		},
		{
			name: "disk stressor without size",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{Disk: &DiskStressor{}},
			},
			wantErr: true,
		},
		{
			name: "valid io stressor",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{IO: &IOStressor{Workers: 3}},
			},
			wantErr: false,
		},
		{
			name: "io stressor with zero workers",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{IO: &IOStressor{Workers: 0}},
			},
			wantErr: true,
		},
		{
			name: "memory without size",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Stressors: StressorsSpec{Memory: &MemoryStressor{Workers: 1}},
			},
			wantErr: true,
		},
		{
			name: "invalid duration",
			spec: StressChaosSpec{
				Mode:      OnePodMode,
				Duration:  "bad",
				Stressors: StressorsSpec{CPU: &CPUStressor{Workers: 1}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStressChaosSpec(&tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStressChaosSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHTTPChaosSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    HTTPChaosSpec
		wantErr bool
	}{
		{
			name:    "no fault type",
			spec:    HTTPChaosSpec{Port: 80, Mode: OnePodMode, Target: "Request"},
			wantErr: true,
		},
		{
			name: "valid abort",
			spec: HTTPChaosSpec{
				Port: 80, Mode: OnePodMode, Target: "Request",
				Abort: &HTTPAbortSpec{StatusCode: 503},
			},
			wantErr: false,
		},
		{
			name: "invalid abort status code",
			spec: HTTPChaosSpec{
				Port: 80, Mode: OnePodMode, Target: "Request",
				Abort: &HTTPAbortSpec{StatusCode: 999},
			},
			wantErr: true,
		},
		{
			name: "multiple fault types",
			spec: HTTPChaosSpec{
				Port: 80, Mode: OnePodMode, Target: "Request",
				Abort: &HTTPAbortSpec{StatusCode: 500},
				Delay: &HTTPDelaySpec{Latency: "1s"},
			},
			wantErr: true,
		},
		{
			name: "zero port",
			spec: HTTPChaosSpec{
				Port: 0, Mode: OnePodMode, Target: "Request",
				Abort: &HTTPAbortSpec{StatusCode: 500},
			},
			wantErr: true,
		},
		{
			name: "invalid delay latency",
			spec: HTTPChaosSpec{
				Port: 80, Mode: OnePodMode, Target: "Request",
				Delay: &HTTPDelaySpec{Latency: "bad"},
			},
			wantErr: true,
		},
		{
			name: "valid delay",
			spec: HTTPChaosSpec{
				Port: 80, Mode: OnePodMode, Target: "Request",
				Delay: &HTTPDelaySpec{Latency: "500ms"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPChaosSpec(&tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHTTPChaosSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateModeValue(t *testing.T) {
	tests := []struct {
		mode    ChaosMode
		value   string
		wantErr bool
	}{
		{OnePodMode, "", false},
		{AllMode, "", false},
		{FixedMode, "5", false},
		{FixedMode, "", true},
		{FixedMode, "abc", true},
		{FixedMode, "0", true},
		{FixedPercentMode, "50", false},
		{FixedPercentMode, "", true},
		{FixedPercentMode, "101", true},
		{RandomMaxPercentMode, "75", false},
		{RandomMaxPercentMode, "-1", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode)+"_"+tt.value, func(t *testing.T) {
			err := validateModeValue(tt.mode, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModeValue(%s, %q) error = %v, wantErr %v", tt.mode, tt.value, err, tt.wantErr)
			}
		})
	}
}
