# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build          # Compiles binary to bin/goland-operator
make run            # Run manager locally (go run ./cmd/manager)

# Test
make test           # Run all tests with coverage (outputs to coverage/cover.out)
go test ./internal/app/...          # Run a specific package's tests
go test -run TestFoo ./controllers/ # Run a single test

# Lint & format
make lint           # golangci-lint (govet, staticcheck, gofumpt, revive, errcheck, gosec, etc.)
make tidy           # go mod tidy

# Code generation (required after modifying api/ types)
make generate       # Regenerate zz_generated.deepcopy.go via controller-gen
make generate-crds  # Regenerate CRD YAML manifests in config/crd/

# Kubernetes deployment
make install-crds   # Apply CRDs to cluster
make deploy         # Deploy operator to chaos-system namespace
make undeploy       # Remove operator from cluster
make docker-build   # Build container image (IMG=goland-operator:latest)
```

## Architecture

This is a **Kubernetes operator** for chaos engineering, built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime). It follows **Clean Architecture**:

```
cmd/manager/        → Entry point (calls internal/app)
api/v1alpha1/       → CRD type definitions (CustomResource structs + deepcopy)
controllers/        → Kubernetes reconcilers (controller-runtime reconcile loop)
internal/app/       → Manager bootstrap (metrics, health probes, leader election)
internal/domain/    → Pure business logic: ChaosExecutor strategy interface + implementations
internal/infrastructure/ → Kubernetes pod operations, Prometheus metrics
pkg/version/        → Version string (set via build flags)
```

**API Group:** `chaos.engineering.io/v1alpha1`

**Chaos types implemented:** `PodChaos`, `NetworkChaos`, `StressChaos`, `HTTPChaos`

### Key design patterns

- **Strategy Pattern** — `ChaosExecutor` interface (`internal/domain/chaos_executor.go`) with separate implementations for each chaos type. Each executor has `Execute`, `Recover`, `Validate`, and `GetType`.
- **Reconciler lifecycle** — Controllers (e.g., `controllers/podchaos_controller.go`) drive experiments through phases: `Pending → Running → Completed/Failed`. Cleanup uses Kubernetes finalizers.
- **Pod selection** — `PodSelector` in `api/v1alpha1/common_types.go` supports namespace, label, field, node selectors, and explicit pod lists. Blast radius is controlled via `BlastRadiusControl` (maxPods / maxPercentage).

### Environment variables (runtime)

| Variable | Default | Description |
|---|---|---|
| `METRICS_ADDR` | `:8080` | Prometheus metrics endpoint |
| `HEALTH_PROBE_ADDR` | `:8081` | `/healthz` and `/readyz` endpoints |
| `ENABLE_LEADER_ELECTION` | `false` | Enable HA leader election |

### After modifying api/ types

Always run `make generate` to regenerate `zz_generated.deepcopy.go`, then `make generate-crds` if CRD schema changed.
