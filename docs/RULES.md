# Engineering Rules for Go Kubernetes Operators

These rules are tailored to this repository and complement the global cloud software engineering rules.

## Architecture
- Favor clean architecture: keep framework-specific code (controller-runtime) at the edges.
- Controllers should orchestrate; move domain logic to internal/pkg packages.
- Keep APIs (CRDs) versioned and backward compatible. Use conversion webhooks if needed.

## Operators & Controllers
- Single responsibility per controller; split large reconcilers.
- Idempotency is mandatory; ensure reconcile can safely re-run.
- Implement finalizers for external resources. Handle deletion first.
- Use patches over updates to avoid conflicts; prefer server-side apply where applicable.
- Use feature flags for experimental behavior.

## Code Quality
- Enforce linting via `.golangci.yml`.
- Use `gofumpt` for strict formatting.
- Target 80%+ unit test coverage for business logic packages.
- Keep function length small and complexity low; refactor early.

## Observability
- Use structured logging (key/value). Include `namespace`, `name`, `reconcileID`.
- Expose metrics and health probes via `controller-runtime` manager when added.
- Record Kubernetes events for important user-visible actions.

## Reliability
- Implement exponential backoff retries and rate limiting queues as needed.
- Use context deadlines/timeouts for external calls.
- Prefer declarative resource management; diff desired vs actual and apply.

## Security
- RBAC with least privilege per controller. No cluster-admin.
- No secrets in code or logs. Use Kubernetes Secrets/Secret Manager.
- Validate all user input; default missing fields.

## Processes
- PRs require review. Include tests for new logic.
- Keep commits atomic and descriptive. Follow Conventional Commits if possible.
- Update `docs/` and samples with every API change.
