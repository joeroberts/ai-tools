# Roadmap-Impact Contract PRD

## Problem

Issue [#68](https://github.com/joeroberts/ai-tools/issues/68) requires
governed work to explicitly declare and validate its roadmap impact. Existing
roadmap validation checks internal coherence only; it cannot prove that a
governed work item starts, blocks, resumes, or completes a mapped milestone.

## Outcome

Repositories can opt into a portable roadmap-adoption contract. A work item
declares either a required milestone transition or a bounded not-applicable
reason. The CLI rejects missing, malformed, stale, ambiguous, out-of-scope,
cross-repository, or replayed declarations at the applicable lifecycle gate.

## Requirements

- Add repository-relative canonical-roadmap configuration with stable identity,
  supported format, and enforcement mode.
- Add `roadmap_impact` to work-item and ticket-plan contracts.
- Use #51 aggregate/phase semantics as the sole state interpretation.
- Provide non-destructive, idempotent adoption assets and exact previews.
- Bind transitions to work item, repository, roadmap, phase, digests, and type.
- Require roadmap paths in approved scope, budget, and planning baseline.
- Revalidate required impact at planning, preflight, commit, publication, and
  Jira-finalization gates.

## Technical Acceptance Criteria

- A work item without `roadmap_impact` fails deterministic planning validation;
  `not-applicable` without a bounded reason fails.
- Required enforcement rejects absent, ambiguous, absolute, escaping,
  identity-mismatched, unsupported, or machine-specific canonical-roadmap
  configuration with actionable remediation.
- Adoption is idempotent and non-destructive across clean, repeated, and
  user-owned-file/merge-required cases.
- A required transition omitted from allowed paths, review budget, or the
  committed planning baseline fails before dispatch.
- Preflight rejects an applicable work item when its mapped roadmap phase is
  not in the expected active state.
- Stale prior digests, invalid or skipped transitions, replayed evidence,
  cross-repository evidence, and contradictory aggregate state fail closed.
- Completion transitions are required before commit, publication, and Jira
  finalization; non-completing work does not churn roadmap state.
- Two isolated repositories with distinct identities, Jira project keys,
  roadmap paths, and roadmap IDs prove repository neutrality.
- The non-interactive check has deterministic results and stable exit behavior
  suitable for later CI adoption.
- Table-driven tests cover declarations, configuration, all transition types,
  stale and replayed evidence, and every affected lifecycle gate.

## Non-Goals

- Synchronizing roadmap state from GitHub, Jira, commits, branches, or CI.
- Automatically updating Jira or GitHub.
- Changing CI authority (#18), hosted rulesets (#45), root guidance (#19), or
  broader setup UX (#48).

## Delivery

Deliver exactly the six bounded slices in the companion roadmap. The Jira Story
must contain one independently reviewable Subtask for every declared slice, in
the stated dependency order; no slice may be merged into or omitted from
another. Completion requires the GitHub #68 acceptance criteria,
repository-neutral fixtures, and standard build checks.
