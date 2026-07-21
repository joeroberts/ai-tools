# Roadmap Aggregate-State Validation PRD

## Problem

The structured-roadmap validator accepts contradictory roadmap and phase states.
For example, a roadmap declared `in-progress` can have every phase declared
`complete` and still pass validation.

## Goal

Make `codex-governance roadmap check` and `roadmap status` enforce and expose a
coherent aggregate-state contract without changing roadmap phase contents or
inferring delivery state from external systems.

## Contract

| Roadmap status | Required phase state |
| --- | --- |
| `proposed` | Every phase is `pending-approval`. |
| `in-progress` | At least one phase is incomplete and no phase is `blocked`. |
| `blocked` | At least one phase is `blocked`. |
| `complete` | Every phase is `complete`. |

`blocked` uses the existing phase-level state. This slice does not add a
roadmap-level blocker schema.

## Non-goals

- Automatically rewriting roadmap status.
- Changing phase contents or completion evidence.
- Inferring status from GitHub, Jira, commits, or pull requests.
- Expanding the structured-roadmap schema.

## Acceptance criteria

- Table-driven tests cover every allowed and rejected aggregate/phase state.
- Contradictory combinations fail with the roadmap status, conflicting phase
  states, and an actionable correction.
- Existing valid structured roadmaps continue to pass.
- `roadmap status` rejects invalid state rather than rendering it as healthy.
