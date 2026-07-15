# Self-Contained Jira Evidence Work Item

## Status

Jira planning is complete: GitHub issue #26, Story `REK-9`, and primary
Subtask `REK-10` (`In Progress`). This work item is a draft until reviewed and
committed.

## Scope

Make `jira work update` render self-contained, sanitized validation and review
evidence in its Jira preview/comment. It must replace machine-local paths with
structured outcomes from supplied artifacts and fail closed when required
evidence is unavailable.

## Allowed Paths

- `README.md`
- `docs/design/`
- `internal/cli/`
- `internal/jira/`

## Non-Goals

- Posting private prompts, credentials, raw environment data, or raw dumps.
- Changing Jira finalization, board-state enforcement, or model lifecycle.
- Unbounded Jira comments or inferred validation results.

## Acceptance Criteria

- Preview shows rendered command outcomes, reviewer/verifier status, executor
  identities, and exact-diff binding without local artifact paths.
- Rendering redacts local paths and sensitive fields, truncates deterministically,
  and rejects missing or invalid artifacts.
- Focused tests cover passing rendering, redaction, truncation, and rejection.

## Review Budget

Maximum 8 files, 650 changed lines, and 4 components: Jira renderer, CLI,
tests, and documentation. No ADR needed: bounded work-record rendering.
