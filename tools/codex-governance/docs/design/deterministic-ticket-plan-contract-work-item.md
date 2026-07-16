# Deterministic Ticket-Plan Contract Work Item

## Status

GitHub issue #36 is the source request. Jira Story `REK-13` and primary
implementation Subtask `REK-14` are linked to this work item. `REK-14` is
`In Progress`; implementation may begin after this linked work-item baseline is
committed.

## Scope

Replace manager-authored plan content as an authority. The planner must derive
assignment-controlled fields and their traceability from approved constraints.
It must copy or canonicalize source-derived acceptance criteria, allowed paths,
review-budget entries, and validation requirements from verified source catalog
entries.

This is a standard correctness hardening of the existing ticket-plan boundary.

The manager may provide only genuinely narrative fields. Each narrative value
must deterministically match verified source content or fail once with an
actionable unsupported-value error. Manager-generated excerpts must never be
authoritative evidence, and formatting or paraphrase must not require a manager
repair-traceability loop.

## Allowed Paths

- `internal/agentplan/`
- `internal/ticketplan/`
- `internal/cli/`
- `testdata/ticket-plans/`
- `docs/design/`

## Non-Goals

- Changing Jira creation, approval, or publication policy.
- Weakening traceability, source verification, or fail-closed validation.
- Retrying the #22 plan generation before the deterministic contract is
  implemented and verified.
- Treating manager-generated excerpts as verified source evidence.

## Acceptance Criteria

- Focused fixtures reproduce the failed #22 manager outputs.
- Assignment-controlled fields override malformed manager values and traces.
- Canonical source-derived fields validate without manager paraphrasing.
- Unsupported manager narrative fails closed with an actionable error.
- The prior two-cycle escalation converges without a second manager attempt.
- A real #22 `jira plan generate` succeeds before any #22 Jira tickets are
  created.

## Review Budget

Maximum 12 changed files, 900 changed lines, and ticket-plan validation. No ADR
is needed because this strengthens an existing deterministic validation boundary
without adding a new architectural component.
