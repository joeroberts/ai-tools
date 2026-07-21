# ADR-0003: Persistent Headless Codex Supervisor

## Status

Accepted on 2026-07-21 for GitHub issue #55 and Jira REK-62/REK-63.

## Context

The current headless adapter retains process state only in its launching CLI
process. A later CLI invocation can observe a PID and a result file, but it
cannot prove process identity, recover durable diagnostics, prevent duplicate
dispatch, or safely distinguish a recycled PID from the original process.

The retained post-assignment manager in ADR-0002 is unrelated: this decision
owns only the lifecycle of an approved headless implementation process.

## Decision

Run headless Codex under a dedicated local supervisor process whose lifecycle
outlives the launching CLI. The launcher starts the supervisor once, records a
durable supervisor record before returning, and never directly owns the Codex
child after that handoff.

Each run has an owner-only directory at `runtime/runs/<run-id>/supervisor`.
It contains one immutable launch record, one atomically replaced current-state
record, an append-only event log, and owner-only stdout, stderr, result, and
exit-status diagnostics. Records use `0600`; their directory uses `0700`.
Existing records are never overwritten except the named current-state file,
which is replaced with a same-directory atomic rename after fsync.

## Identity And State

The launch record binds the run ID, task-bundle digest, source-evidence digest,
worktree path digest, Codex executable identity, child PID, process start
identity, parent supervisor PID, launch time, and diagnostic paths. On systems
where a start identity cannot be read reliably, reconciliation fails closed
rather than treating a PID alone as proof of identity.

Allowed states are `launching`, `running`, `complete`, and `failed`. Terminal
states are immutable. Every transition records its predecessor state, timestamp,
reason, diagnostic references, and an event digest. A terminal result is
published only after the result or failure diagnostic is durable; partial or
empty result files are not terminal evidence.

## Reconciliation And Duplicate Prevention

Before dispatch, the supervisor directory is locked with an owner-only
same-directory lock. A matching non-terminal record prevents a second launch.
A terminal record returns its existing outcome and cannot be retried implicitly.

Reconciliation reads the durable state first. For a non-terminal record it
validates the stored process start identity before checking liveness. A live,
matching process remains `running`. A missing or mismatched process identity is
`failed`, with a stale-identity diagnostic; it is never treated as a new
dispatch opportunity. A durable valid result produces `complete`; a nonzero
exit, malformed result, lost child, or unreadable required diagnostic produces
`failed`.

The CLI may start or reconcile the supervisor, but a restart or lost in-memory
adapter map changes neither state ownership nor the duplicate rule. It reads the
same records and returns the durable outcome.

## Recovery And Rollback

Supervisor startup writes `launching` before starting Codex and writes `running`
only after durable identity capture. Failure between those points is a terminal
`failed` record with a launch diagnostic. Startup scans no global process list;
it recovers only an explicitly requested run directory. Rollback disables new
supervisor dispatch and leaves existing records and diagnostics intact for
reconciliation. It does not kill an unverified PID or silently retry work.

## Required Deterministic Coverage

The implementation must cover launcher exit while the child remains active,
reconciliation after a fresh process map, duplicate start, stale PID/start
identity mismatch, valid terminal result, nonzero child exit, malformed result,
atomic terminal publication, and owner-only diagnostic permissions.

## Consequences

The local runtime gains durable process ownership and recovery evidence without
introducing a network service or changing authorization. A supervisor process
can still fail; its durable records turn that failure into an actionable,
fail-closed state rather than an unsafe retry.
