# Nested Product Diff Paths PRD

## Status

GitHub issue #60 is the source request. This PRD is a work-item draft. Jira
planning, owner approval, and a committed linked work item are required before
implementation.

## Problem

Governed Git diff collection returns monorepo-root-relative paths even when a
nested product directory is supplied as the repository root. Scope validation
then compares paths such as `tools/codex-governance/internal/agentplan` with
approved product-relative paths such as `internal/agentplan` and rejects valid
changes.

The mismatch blocks governed verification and publication for `REK-33` run
`run-8cfd74604cb1f904`.

## Goal

Make committed-range and working-tree change paths relative to the caller's
supplied repository root without broadening scope or weakening untracked-file
handling.

## Acceptance Criteria

- `WorkingChanges` returns product-relative paths for a nested product root.
- `Changes` uses the same product-relative path basis.
- Untracked files remain a fail-closed validation error.
- A nested-product-root regression covers committed and working changes.
- The original signed `REK-33` bundle verifies without modification after the
  fix is available.
- Focused tests, full tests, vet, build, whitespace, and exact-diff review gates
  pass.

## Non-Goals

- Broadening approved paths.
- Altering signed work items or task bundles.
- Changing Jira, review, publication, or remote-write policy.
