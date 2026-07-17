# Nested Product Diff Paths Work Item

## Status

GitHub issue #60 is the source request. Jira Story `REK-34` and primary
implementation Subtask `REK-35` are linked to this work item. `REK-35` was
read back as `To Do` on 2026-07-16. This item remains `Jira-planning`;
implementation is prohibited until this planning baseline is committed and
`REK-35` is transitioned to `In Progress` and read back.

## Scope

Normalize committed-range and working-tree numstat paths relative to the
repository root supplied by the caller. Add focused nested-product regression
coverage while preserving untracked-file rejection.

## Allowed Paths

- `docs/design`
- `docs/roadmaps`
- `internal/gitdiff`

## Non-Goals

- Broadening scope for `REK-33`.
- Modifying its signed work item or task bundle.
- Changing validation, review, Jira, or publication policy.

## Acceptance Criteria

- Nested-product committed and working diffs use product-relative paths.
- Top-level repository behavior remains correct.
- Untracked files still fail closed.
- The original `REK-33` signed scope passes governed verification.
- Full validation and independent exact-diff gates pass.

## Review Budget

Maximum 2 changed files, 150 changed lines, and one internal/gitdiff path
normalization component. No ADR is needed because this corrects an existing
path-basis defect.

## Validation Evidence

- Focused `internal/gitdiff` tests.
- `make test`.
- `make vet`.
- `make build`.
- `git diff --check`.
- Independent exact-diff reviewer and verifier evidence.
