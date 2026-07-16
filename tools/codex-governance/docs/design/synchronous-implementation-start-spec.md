# Synchronous Implementation Start Specification

## Technical Design

Retain ownership of the headless Codex adapter inside `implementation start`
until the launched command reaches a terminal status. Reconcile that status in
the same invocation and save the implementation run only after it records
`implementation-complete` or `escalated`.

Successful execution must retain the validated result reference. Failed
execution must retain the escalated state, surface a nonzero CLI result, and
identify the private stdout and stderr diagnostic files without embedding
their contents in public output.

The wait must not introduce redispatch or retry behavior. It may use a bounded
poll or an explicit adapter wait contract, but it must avoid a race between
process completion and terminal reconciliation.

## Allowed Paths

Implementation is limited to `docs/design`, `docs/roadmaps`,
`internal/implementation`, and `internal/cli`.

## Review Budget

The total review budget is 6 changed files, 450 changed lines, and synchronous process lifecycle, terminal state and diagnostics.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "synchronous-implementation-start",
      "phase": "Phase 1",
      "change_class": "standard",
      "dependencies": [],
      "allowed_paths": [
        "docs/design",
        "docs/roadmaps",
        "internal/implementation",
        "internal/cli"
      ],
      "review_budget": {
        "max_changed_files": 6,
        "max_changed_lines": 450,
        "components": [
          "synchronous process lifecycle",
          "terminal state and diagnostics"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Block the start invocation until the launched child reaches a terminal
  adapter status, without dispatching another child.
- Reconcile and persist successful execution as `implementation-complete` with
  its readable result reference before returning success.
- Reconcile and persist failed execution as `escalated`; return nonzero and
  report private diagnostic paths before exiting.
- Add deterministic adapter and CLI regression coverage for both terminal
  outcomes, then run the repository validation commands.

## Architecture Decision

No ADR needed: this is a bounded reliability correction. Persistent ownership,
restart recovery, and durable process identity belong to future feature
enhancement GitHub issue #55.
