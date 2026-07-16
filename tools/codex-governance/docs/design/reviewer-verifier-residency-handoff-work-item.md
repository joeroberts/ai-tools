# Reviewer-to-Verifier Residency Handoff Work Item

## Status

GitHub issue #40 is the source request and blocks governed Jira planning for
GitHub issue #29. Jira planning is complete: Story `REK-15` and primary Subtask
`REK-16` (`In Progress`) are linked to this work item. This work-item baseline
must be reviewed and committed before implementation begins.

## Scope

Make ticket-plan orchestration own the fail-closed local-model handoff after
reviewer approval. Persist and close the approved reviewer role, unload the
configured reviewer through the existing governed residency API, verify it is
not resident, load the configured verifier through that same policy boundary,
verify it is resident, and only then invoke the verifier.

Keep reviewer residency across bounded reviewer-remediation cycles. Bind every
residency action to the existing reviewer and verifier runner policies and exact
model identities. Any policy, identity, unload, load, or status-verification
failure must stop before verifier invocation with an actionable error.

Implement the handoff through the existing Go orchestration and governed Ollama
client. Do not introduce a shell wrapper, proxy, raw Ollama call, or background
model manager.

This is a standard hardening of the existing ticket-plan review orchestration.

## Non-Goals

- Weakening reviewer and verifier independence or ticket-plan validation.
- Changing model allowlists, model identities, or qualification records.
- Removing the guarded Make targets or manual residency commands.
- Jira writes, publication, merge, deployment, or release automation.
- A one-run residency exception for GitHub issue #29.

## Acceptance Criteria

- Reviewer approval and role closure precede residency handoff actions.
- Reviewer unload is policy-bound and verified before verifier loading begins.
- Verifier load is policy-bound and verified before verifier invocation.
- Reviewer-remediation cycles retain reviewer residency until approval.
- Every handoff failure stops before verifier invocation with an actionable
  error and closed or explicitly failed agent evidence.
- Deterministic tests cover successful handoff, remediation cycles, policy and
  identity mismatch, unload failure, load failure, and verification failure
  without requiring a live Ollama service.
- Existing guarded Make targets and manual residency workflows remain supported.
- `make test`, `make vet`, `make build`, and `git diff --check` pass.

## Allowed Paths

- `docs/design/`
- `internal/agentplan/`
- `internal/cli/`
- `internal/ollama/`
- `testdata/`

## Review Budget

Maximum 10 changed files, 700 changed lines, and ticket-plan orchestration, Ollama residency, CLI wiring, tests.

Architecture Decision: No ADR needed: this composes existing orchestration and
governed residency boundaries without adding a new architectural component.
