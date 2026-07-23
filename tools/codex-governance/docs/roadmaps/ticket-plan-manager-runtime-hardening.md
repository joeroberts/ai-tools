# Ticket-Plan Manager Runtime Hardening Roadmap

## Status

`complete`

## Phase 1: Constraint-Aware Manager Schema

Slice ID: `constraint-aware-manager-schema`. No implementation dependency.

Derive post-assignment finite enums and array bounds from approved constraints.
Align pre-assignment path shaping with root-level support and add focused
fixtures for accepted root paths, aggregated paths, unapproved values, array
bounds, and oversized protocol-like strings.

Exit when all schema fixtures pass without a hosted manager call.

## Phase 2: Supervised Manager Lifecycle

Slice ID: `supervised-manager-lifecycle`. Depends on
`constraint-aware-manager-schema`.

Propagate signal-aware contexts, require configurable timeout and wait-delay
values, persist Codex JSONL and stderr diagnostics, and reconcile controlled
failure states through terminal ledger closure.

Exit when deterministic success, timeout, cancellation, lingering-pipe, and
diagnostic-permission fixtures pass.

## Delivery Order

`supervised-manager-lifecycle` depends on
`constraint-aware-manager-schema`. Complete and validate the schema slice
before beginning lifecycle implementation.

## Separate Architecture Decision

GitHub issue #59 is complete. ADR-0002 retains the post-assignment manager as
a bounded proposal producer, with owner-approved constraints and local
validation remaining authoritative. It defines binding, drift and replay
protection, independent review semantics, migration, rollback, and the
persistent-supervisor boundary. See
`docs/decisions/0002-retain-post-assignment-manager.md`.

## Validation Gates

Run focused package tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Before commit, obtain passing independent exact-diff
reviewer and verifier assessments from distinct external executors.

## Completion Record

GitHub issue #58 is complete. Its backlog, execution, and delivered-diff
evidence remain in their respective GitHub, Jira, and Git/PR/CI records.
