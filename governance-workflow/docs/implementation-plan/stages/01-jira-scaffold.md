# Stage 1 Prompt: Jira Scaffold

Use this prompt after the master prompt is approved.

## Objective

Create the Jira-backed canonical spec and base scaffold.

## Scope

- Create `docs/governance-toolkit-spec.md` from the supplied canonical spec.
- Initialize the Go module and `cmd/codex-governance/`, `internal/`, and
  `tests/` directory layout.
- Create `governance.yml` with format version, Jira conventions, `generic`
  profile, review-budget policy, CI conventions, and no credentials.
- Create `schemas/jira-work-item.schema.json`.
- Create templates for `AGENTS.md`, `PROJECT_CONTEXT.md`, Jira stories,
  Jira subtasks, review exceptions, ADRs, and future profiles.
- Create `profiles/generic.md` and `docs/future-profiles.md`.

## Do Not Create

Do not create `implementation-packets/`, `verification/evidence/`,
`.codex/summaries/`, Jira adapters, CI integrations, fixtures, or examples.
Existing legacy directories must not be deleted.

## Validation

- Verify scaffold files and template headings.
- Verify `governance.yml` contains no secret-like values.
- Verify the work-item schema is valid JSON.
- Prepare concise Stage 1 Jira handoff text; do not post it without approval.

## Completion

Summarize created files and validation, identify gaps, propose Stage 2, and
wait for approval.
