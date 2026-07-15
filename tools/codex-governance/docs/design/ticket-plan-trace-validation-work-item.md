# Ticket-Plan Trace Validation Work Item

## Status

GitHub issue #32 is the source request. Jira planning is required before code changes; the resulting implementation Subtask must be `In Progress` before implementation begins.

## Scope

Make ticket-plan validation deterministic after approved constraints are applied. Assignment-controlled fields must be validated only against their approved assignment and its source traceability. Manager-owned fields must retain fail-closed, source-backed validation without depending on formatting or paraphrase.

## Allowed Paths

- `docs/design/`
- `internal/agentplan/`
- `internal/ticketplan/`

## Non-Goals

- Implementing GitHub issue #22 publication behavior.
- Bypassing the ticket-plan approval, reviewer, or verifier gates.
- Weakening traceability requirements for unsupported manager-owned values.

## Acceptance Criteria

- A focused fixture reproduces the prior two-cycle escalation.
- Validation distinguishes assignment-controlled and manager-owned trace fields.
- Unsupported manager-owned fields still fail closed.
- Focused deterministic regression tests cover the corrected contract.

## Review Budget

Maximum 5 changed files, 450 changed lines, and trace-validation policy, assignment application, deterministic tests and documentation. No ADR is needed because this corrects an existing validation contract.
