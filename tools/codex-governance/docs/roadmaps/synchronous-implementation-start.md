# Synchronous Implementation Start Delivery Roadmap

## Phase 1: Keep Start Alive Through Terminal Reconciliation

Keep the approved `implementation start` invocation alive until `codex exec`
finishes. Reconcile the terminal adapter outcome, persist the resulting run
state and references, return nonzero on child failure, and add deterministic
success and failure coverage.

## Delivery Order

Complete this bounded fix before retrying governed headless implementation for
`REK-26`. Durable detached execution remains a future feature enhancement
tracked by GitHub issue #111.
