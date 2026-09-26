package app

import (
	"os"
	"testing"
)

func TestStart(t *testing.T) {
	// Start requires a running K8s API server and available ports.
	// Only run in integration environments where KUBECONFIG is set and ports are free.
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test (set INTEGRATION_TEST=true to run)")
	}

	if err := Start("test"); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
}
