# Synchronous Implementation Start Work Item

## Status

This work item addresses the immediate headless execution failure discovered
while dispatching `REK-26`. GitHub issue #55 tracks a persistent supervisor as
a future feature enhancement. Jira Story `REK-27` and primary implementation
Subtask `REK-28` are linked to this work item. `REK-28` was read back as
`In Progress` on 2026-07-16. Implementation may begin only after this linked
work-item baseline is committed.

## Scope

Keep `implementation start` alive until its approved `codex exec` process
finishes. Before the command returns, reconcile the adapter outcome and persist
the implementation run in either `implementation-complete` or `escalated`
state with available result and diagnostic references.

## Non-Goals

- Introducing a daemon, service, or persistent process supervisor.
- Allowing detached execution across CLI or host-process restarts.
- Changing approval, Jira, review, publish, or retry policy.
- Resuming or otherwise changing the failed `REK-26` run.

## Acceptance Criteria

- `implementation start` does not return while its `codex exec` child is still
  running.
- Successful execution persists `implementation-complete` and the result
  reference before the command returns.
- Failed execution persists `escalated`, returns a failure status, and reports
  actionable diagnostic locations without losing the child exit failure.
- Focused deterministic tests cover successful and failed child execution.
- `make test`, `make vet`, `make build`, and `git diff --check` pass.

## Review Budget

Maximum 6 changed files, 450 changed lines, and synchronous process lifecycle,
terminal state and diagnostics. No ADR is needed because this is a bounded
reliability correction; GitHub issue #55 reserves the durable supervisor and
recovery architecture for a future feature enhancement.
