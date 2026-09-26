# Controllers

Place your reconcilers here. Keep them small, testable, and follow the reconcile loop best practices:

- Idempotent operations
- Exponential backoff and requeues
- Finalizers for cleanup
- Event recording and structured logging
- Status updates separated from spec changes
