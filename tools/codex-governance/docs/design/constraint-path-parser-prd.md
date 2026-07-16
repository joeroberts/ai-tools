# Constraint Path Parser PRD

## Status

This is the approved product source for the planning-tool fix discovered
while preparing GitHub issue #29. A separate Jira Story and implementation
Subtask are required before code changes.

## Problem

Constraint drafting uses a broad slash-path pattern. It recognizes nested
paths but drops valid root-level allowlist entries such as `AGENTS.md` and
`testdata`, making a compliant downstream ticket plan impossible.

## Goal

Make declared implementation paths deterministic and complete without treating
unrelated prose as path authority.

## Product Outcomes

- Root-level files and directories remain available to approved constraints.
- Existing nested implementation paths remain supported.
- Only the specification's declared allowlist controls downstream path scope.

## Acceptance Criteria

- Valid `AGENTS.md`, `testdata`, and nested paths from `Allowed Paths` are
  included in the path pool.
- Invalid or duplicate allowlist entries fail closed with an actionable error.
- Text outside `Allowed Paths` cannot add a path to the pool.
- Focused tests prevent regression of all three cases.

## Non-Goals

- Changing Jira publication or implementation-preflight policy.
- Relaxing validated-path rules for absolute, traversal, wildcard, or malformed
  entries.
