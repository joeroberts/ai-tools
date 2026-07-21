# Roadmap-Impact Contract Roadmap

## Status

`Jira-planning` — GitHub issue [#68](https://github.com/joeroberts/ai-tools/issues/68)
is the backlog source. ADR 0004 and the companion PRD/spec are proposed.
Implementation starts only after a committed plan creates and reads back its
primary Jira Subtask as `In Progress`.

## Sequence

1. **Portable configuration and declaration contract** — schema, work-item and
   ticket-plan impacts, diagnostics, and isolated fixtures.
2. **Migration and adoption assets** — templates, previews, idempotency,
   no-overwrite, and merge-required behavior.
3. **Planning and entry enforcement** — planning, scope/budget/baseline, and
   preflight state checks.
4. **Transition and digest evidence** — preview helper, state transitions, and
   stale/replay rejection.
5. **Commit and publication gates** — completion-transition enforcement.
6. **Finalization and non-interactive check** — finalization gate and stable
   CLI check for later CI adoption.

Each slice is a separate Jira Subtask and reviewable PR, with no more than two
review components. It must pass standard build checks and independent exact-diff
review before publication.

## Boundaries

#19 owns root guidance propagation; #18 authoritative CI; #45 hosted rulesets;
and #48 broader setup UX. This roadmap owns only the local portable contract.
