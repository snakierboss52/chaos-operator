# Editor Rules for Go Kubernetes Operators

These rules are intended for IDE/editor configuration (GoLand, VS Code, etc.) to maintain consistency, reusability, clean code, and good architecture.

## Language & Formatting
- Use Go 1.22+ toolchain.
- Enable `gofumpt` formatting on save.
- Set max line length guidance to 120 chars (not enforced by gofmt; use linter hints).
- Use 4 spaces indentation for Go; 2 spaces for YAML/JSON/Markdown.

## Linting & Static Analysis
- Enable `golangci-lint` on save or pre-commit with the repo’s `.golangci.yml`.
- Treat warnings as actionable; fix or justify with comments and follow-ups.
- Prefer `context.Context` as the first parameter where applicable.

## Project Structure Conventions
- `cmd/manager`: Entry point only; no business logic.
- `internal/`: Application wiring and internal packages not meant for external reuse.
- `api/`: Versioned CRD types (`<group>/<version>`); keep conversion and defaults.
- `controllers/`: Reconciler implementations; small, testable units.
- `pkg/`: Reusable utilities and libraries; no coupling to controllers.
- `config/`: Manifests and kustomize overlays when using Kubebuilder/Operator SDK.
- `samples/`: Minimal CR examples per API version.

## Coding Practices
- Keep reconcilers idempotent; no side effects on repeated calls.
- Separate concerns: resource building, apply/patch, status updates, event recording.
- Use constants for common names/labels/annotations to reduce duplication.
- Prefer small functions; target cyclomatic complexity < 10.
- Avoid global state. Prefer dependency injection.
- Handle errors explicitly; wrap with context using `%w`.
- Log with structured key/value pairs.

## Testing
- Minimum 80% coverage on business logic. Controllers can be covered by envtest.
- Unit tests in `_test.go` files colocated with implementation.
- Use table-driven tests. Keep tests deterministic and isolated.
- Use fakes/mocks for clients and clock/time when necessary.

## Versioning & APIs
- Follow SemVer for application and Kubernetes API versions.
- Avoid breaking API changes; implement conversions and defaulting.
- Document deprecations with clear timelines.

## Security & Compliance
- Never hardcode secrets. Use Kubernetes Secrets or secret managers.
- Apply RBAC least privilege. Scope permissions per controller.
- Validate and default user input in APIs.

## CI/CD & Automation
- Run `make tidy`, `make lint`, and `make test` in CI.
- Keep `Makefile` targets simple, composable, and documented.
- Commit generated code and manifests only if reproducible.

## Documentation
- Maintain `README.md` and `docs/` alongside code (docs-as-code).
- Add architecture decision records (ADRs) for major decisions.

## Editor-specific
- GoLand: enable `File Watchers` for `gofumpt` or rely on GoLand’s built-in formatter with gofumpt.
- Enable on-save `go mod tidy` or run via `make tidy` frequently.
