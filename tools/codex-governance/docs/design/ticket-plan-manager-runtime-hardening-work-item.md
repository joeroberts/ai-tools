# Ticket-Plan Manager Runtime Hardening Work Item

## Status

GitHub issue #58 is the source request. Jira Story
[REK-31](https://rekonlabs.atlassian.net/browse/REK-31) contains primary
Subtask [REK-32](https://rekonlabs.atlassian.net/browse/REK-32) and dependent
Subtask [REK-33](https://rekonlabs.atlassian.net/browse/REK-33).

This item is in `Jira-planning`; implementation is prohibited until this
planning baseline is committed and primary Subtask `REK-32` is transitioned
to `In Progress` and read back.

## Jira Execution Contract

- Story: `REK-31`.
- Primary Subtask: `REK-32`, `constraint-aware-manager-schema`.
- Dependent Subtask: `REK-33`, `supervised-manager-lifecycle`.
- Required delivery order: complete and validate `REK-32` before beginning
  `REK-33`.

## Scope

Make post-assignment manager schemas derive finite enums and array bounds from
approved constraints. Align decomposition path shaping with root-level path
support while continuing to reject malformed and aggregated values.

Supervise hosted manager calls with required configurable timeout and
wait-delay values, signal-aware cancellation, documented Codex JSONL output,
owner-only diagnostics, and terminal ledger reconciliation for controlled
failures.

## Allowed Paths

- `docs/design`
- `docs/roadmaps`
- `internal/agentplan`
- `internal/cli`
- `testdata/ticket-plans`

## Non-Goals

- Eliminating the post-assignment manager or reusing the decomposition.
- Persistent or restart-safe supervision.
- Automatic retry or redispatch.
- Jira, approval, review-independence, publication, or merge-policy changes.

## Acceptance Criteria

- Constraint-aware schemas accept exact root-level and nested approved paths
  and reject aggregated, unapproved, oversized, and protocol-like values.
- All manager-output arrays have deterministic finite bounds.
- Manager calls preserve owner-only JSONL, stderr, schema, and result
  diagnostics with actionable references.
- Required timeout and wait-delay configuration fail closed before dispatch.
- Controlled timeout, signal cancellation, and lingering-pipe cases terminate
  without replacement dispatch and close the manager ledger role.
- Focused regression tests cover every reproduced failure class.
- Full repository validation and exact-diff reviewer/verifier gates pass.

## Review Budget

Maximum 12 changed files, 900 changed lines, and constraint-aware manager schema
and fixtures, supervised manager lifecycle and diagnostics. No ADR is needed
because this preserves the accepted manager boundary.

## Validation Evidence

- Focused `internal/agentplan` and `internal/cli` tests.
- `make test`.
- `make vet`.
- `make build`.
- `git diff --check`.
- Independent exact-diff reviewer and verifier evidence from distinct
  executors.
