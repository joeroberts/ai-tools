# Jira In-Progress Preflight PRD

## Status

GitHub issue #29 is the source request. This PRD records the approved
two-slice planning scope. Jira planning and owner approval remain required
before implementation.

## Problem

Implementation preflight currently has no signed Jira evidence for Story or
Subtask lifecycle status. It therefore cannot prove that the primary Subtask
has been transitioned to `In Progress` before creating implementation
artifacts or dispatching an adapter.

## Goal

Require verified Jira lifecycle evidence before a governed implementation run
can begin, while keeping status capture separate from status enforcement.

## Product Outcomes

- Signed offline Jira exports carry the Story and Subtask statuses needed for
  an implementation-entry decision.
- Preflight accepts only a primary Subtask whose verified status is exactly
  `In Progress`.
- Rejected status evidence creates no task bundle, run, worktree, or adapter
  dispatch.
- Repository workflow guidance makes the Jira transition and read-back
  requirement explicit.

## Acceptance Criteria

- Signed exports reject missing or malformed Story and Subtask status
  evidence.
- `To Do`, blocked, stale, missing, and unverifiable primary-Subtask status
  cases fail before implementation side effects.
- Focused tests cover status capture, signature verification, accepted and
  rejected statuses, and absence of side effects.
- `AGENTS.md` requires the primary Subtask transition to `In Progress` and
  Jira read-back before the first implementation edit.

## Non-Goals

- Inferring or silently changing Jira status.
- Performing Jira transitions from implementation preflight.
- Treating a handoff string as authoritative Jira status.
- Expanding GitHub issue #26 or changing Jira evidence rendering.
