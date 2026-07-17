# Ticket-Plan Manager Runtime Hardening PRD

## Status

GitHub issue #58 is the source request. This PRD is a work-item draft. Jira
planning, owner approval, and a committed linked work item are required before
implementation.

## Problem

Post-assignment ticket-plan generation sends canonical constraint-owned fields
through a hosted manager schema that rejects root-level paths such as
`AGENTS.md` and `testdata` while accepting aggregated or oversized strings.
Captured manager output demonstrates protocol-like content embedded in path
values that passed the schema and consumed substantially more time and output
than healthy runs.

The manager process also buffers its output without durable progress or usage
diagnostics, has no configured wall-clock deadline, and cannot unwind its
ledger state through normal error handling when the launching CLI is
interrupted.

## Goal

Make post-assignment manager execution constraint-aware, bounded, observable,
and gracefully cancellable without changing manager ownership or Jira policy.

## Product Outcomes

- The hosted manager can emit only paths and other bounded values admitted by
  the approved constraints.
- Root-level paths remain valid without permitting aggregated or protocol-like
  substitutes.
- Owner-only JSONL and stderr diagnostics record manager lifecycle, terminal
  usage when available, and actionable failure context.
- A required positive manager timeout and wait-delay bound prevent indefinite
  execution and lingering I/O waits.
- Graceful CLI interruption cancels the active manager, persists failure
  diagnostics, and closes the execution-ledger role before returning.
- Hard termination and restart-safe supervision remain explicitly outside this
  bounded correction.

## Acceptance Criteria

- Post-assignment schemas derive allowed-value enums and array bounds from the
  approved constraints instead of using the current slash-requiring path
  pattern.
- `AGENTS.md` and `testdata` are accepted; aggregated paths and oversized
  protocol-like strings are rejected before hosted dispatch or by the strict
  output schema.
- Manager invocations use documented Codex JSONL output and preserve owner-only
  stdout, stderr, schema, and final-result diagnostics.
- Generation and decomposition require valid configurable manager timeout and
  wait-delay values.
- `SIGINT`, `SIGTERM`, deadline expiry, and lingering output pipes unwind
  through the normal failure path when the parent process remains alive.
- Timeout and cancellation failures record `failed` and `closed` ledger events
  with diagnostic references.
- Focused deterministic tests cover schema values, bounds, diagnostics,
  timeout, cancellation, wait delay, and ledger closure.

## Non-Goals

- Eliminating the post-assignment manager or reusing the earlier decomposition.
- Introducing a daemon, service, or restart-safe persistent supervisor.
- Guaranteeing cleanup after `SIGKILL`, host termination, or power loss.
- Changing reviewer, verifier, Jira, approval, publication, or merge policy.
