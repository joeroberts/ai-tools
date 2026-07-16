# Constraint Path Parser Work Item

## Status

This work item was discovered while planning GitHub issue #29. Jira Story
`REK-25` and primary implementation Subtask `REK-26` are linked to this work
item. `REK-26` is `In Progress`, and that status was read back from Jira on
2026-07-16. Implementation may begin only after this linked work-item baseline
is committed.

## Scope

Make constraint drafting derive its allowed-path pool from the specification's
`Allowed Paths` section without excluding valid root-level files or
directories. In particular, it must preserve `AGENTS.md` and `testdata` so a
subsequent approved work item can retain those paths in its authoritative
constraint assignment.

## Non-Goals

- Changing #29's signed-export or implementation-preflight behavior.
- Broadening a work item's paths beyond the specification's declared allowlist.
- Weakening path validation or source-digest verification.

## Acceptance Criteria

- Constraint drafting accepts valid nested paths, root-level files, and
  root-level directories declared under `Allowed Paths`.
- Paths appearing outside that section do not enter the allowed-path pool.
- Invalid, duplicate, absolute, traversal, wildcard, or malformed path entries
  are rejected with an actionable error.
- Focused regression tests cover `AGENTS.md`, `testdata`, existing nested
  paths, and non-allowlist text.

## Review Budget

Maximum 4 changed files, 350 changed lines, and constraint-path parsing,
focused regression tests. No ADR is needed because this corrects a planning
parser boundary without adding a new architectural component.
