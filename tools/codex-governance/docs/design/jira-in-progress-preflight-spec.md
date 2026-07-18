# Jira In-Progress Preflight Specification

## Technical Design

The signed offline Jira export must carry the source status for its Story and
primary Subtask. Export parsing and signature validation must reject missing
or malformed status evidence.

Implementation preflight must consume that verified snapshot and fail unless
the primary Subtask status is exactly `In Progress`. The failure happens before
creating a task bundle, run, worktree, or adapter dispatch. Preflight never
changes Jira state.

## Allowed Paths

Implementation is limited to `AGENTS.md`, `docs/design`,
`internal/implementation`, `internal/jira`, and `testdata`.

## Review Budget

The total review budget is 12 changed files, 800 changed lines, signed Jira export status evidence, export validation fixtures, In Progress preflight gate, workflow and preflight fixtures. Each slice must stay within its individual budget and include no more than two components.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "signed-status-evidence",
      "phase": "Phase 1",
      "change_class": "standard",
      "dependencies": [],
      "allowed_paths": [
        "internal/jira",
        "testdata"
      ],
      "review_budget": {
        "max_changed_files": 6,
        "max_changed_lines": 400,
        "components": [
          "signed Jira export status evidence",
          "export validation fixtures"
        ]
      }
    },
    {
      "id": "in-progress-preflight-gate",
      "phase": "Phase 2",
      "change_class": "standard",
      "dependencies": [
        "signed-status-evidence"
      ],
      "allowed_paths": [
        "AGENTS.md",
        "internal/implementation",
        "testdata"
      ],
      "review_budget": {
        "max_changed_files": 6,
        "max_changed_lines": 400,
        "components": [
          "In Progress preflight gate",
          "workflow and preflight fixtures"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Preserve Story and Subtask source status in the signed export contract and
  validate it before accepting the snapshot.
- Enforce the exact `In Progress` primary-Subtask status before all
  implementation side effects.
- Add focused tests for valid status capture, invalid evidence, accepted
  `In Progress`, and rejected statuses with no implementation artifacts.

## Architecture Decision

No ADR needed: this strengthens existing signed-export and implementation
preflight contracts without adding a new architectural component.
