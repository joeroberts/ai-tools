# Ollama Stop Verification Retry Work Item

## Status

GitHub issue #30, Story `REK-19`, and primary Subtask `REK-20` are linked to
this work item. `REK-20` was transitioned to `In Progress` and read back before
implementation began.

## Scope

Make the bounded post-stop residency verification loop tolerate temporary
connection-refused errors from the configured local Ollama endpoint. After a
successful governed `keep_alive: 0` request, verification may retry while the
model is still resident or while a loopback connection is temporarily refused.
It succeeds only after governed status reports the model unloaded.

This is a standard resilience correction to existing residency verification.

## Allowed Paths

- `docs/design/`
- `internal/ollama/`

## Non-Goals

- Retrying initial policy, installation, or residency requests.
- Retrying load verification or arbitrary HTTP, decoding, policy, remote-host,
  timeout, DNS, or network errors.
- Changing model policy, selection, residency duration, or Make targets.
- Raw Ollama CLI or API access outside the existing governed client boundary.
- Jira writes, publication, merge, deployment, or release automation.

## Acceptance Criteria

- Post-stop verification retries a temporary connection-refused error from a
  configured loopback endpoint within the existing bounded timeout.
- A later governed status response reporting `loaded=false` completes the stop
  successfully.
- Persistent connection refusal fails after the bounded timeout with an
  actionable error that preserves the connection failure.
- Continued `loaded=true` status retains the existing bounded fail-closed
  behavior.
- Load verification and non-loopback or non-connection-refused errors still
  fail immediately.
- Deterministic tests cover transient recovery, persistent refusal, and the
  non-retry boundary without a live Ollama service.
- `make test`, `make vet`, `make build`, and `git diff --check` pass.

## Validation Plan

- Run focused transient-recovery, persistent-refusal, and non-retry tests.
- Run `make test`.
- Run `make vet`.
- Run `make build`.
- Run `git diff --check`.

## Review Budget

Maximum 2 changed files, 220 changed lines, and work-item record, Ollama residency verification.
No ADR is needed because this narrows bounded error handling without adding a
new component or policy boundary.
