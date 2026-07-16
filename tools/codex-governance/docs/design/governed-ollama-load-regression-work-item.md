# Governed Ollama Load Regression Work Item

## Status

GitHub issue #12 is the source request. Its original Story `REK-7` and Subtask
`REK-8` are complete, but the focused successful-load regression test required
by the issue was not delivered. Follow-up Story `REK-17` and primary Subtask
`REK-18` are linked to this work item. `REK-18` was transitioned to
`In Progress` and read back before implementation began.

## Scope

Add deterministic focused coverage for the successful governed model-load
path in `internal/ollama`. The test must prove that load sends no prompt,
requests the configured positive residency duration, and returns successfully
only after governed status reports the allowlisted model as loaded.

This is a standard test-only completion of an existing acceptance criterion.

## Allowed Paths

- `docs/design/`
- `internal/ollama/`

## Non-Goals

- Changing production model lifecycle behavior.
- Implementing transient post-stop status retries from GitHub issue #30.
- Changing policy, model selection, residency duration, Make targets, or agent
  instructions.
- Raw Ollama CLI or API access outside the existing governed test boundary.
- Jira writes, publication, merge, deployment, or release automation.

## Acceptance Criteria

- A focused deterministic test exercises `SetResidency` with `loaded=true`.
- The test rejects any load request containing prompt content.
- The test verifies the request uses the configured positive `keep_alive`
  value and `stream=false`.
- The test verifies installed-model identity before loading and reports a
  loaded model from the governed status endpoint before success.
- Existing stop, policy-rejection, and failed-verification tests remain
  unchanged and passing.
- `make test`, `make vet`, `make build`, and `git diff --check` pass.

## Validation Plan

- Run the focused successful-load test.
- Run `make test`.
- Run `make vet`.
- Run `make build`.
- Run `git diff --check`.

## Review Budget

Maximum 2 changed files, 120 changed lines, and work-item record, Ollama residency tests.
No ADR is needed because this adds missing regression coverage without changing
production behavior.
