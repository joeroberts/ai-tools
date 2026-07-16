# Constraint Path Parser Specification

## Technical Design

Parse only Markdown entries under the `Allowed Paths` heading in the approved
specification. Normalize candidate values consistently with the ticket-plan
path validator, reject invalid or duplicate entries, and return a deterministic
sorted path pool.

The parser must accept a repository-relative root file such as `AGENTS.md`, a
root directory such as `testdata`, and nested paths such as
`internal/agentplan`. Paths mentioned in any other specification section are
not allowlist entries.

## Allowed Paths

Implementation is limited to `docs/design`, `internal/agentplan`, and
`testdata/ticket-plans`.

## Review Budget

The total review budget is 4 changed files, 350 changed lines, and constraint-path parsing, focused regression tests.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "allowed-path-parser",
      "phase": "Phase 1",
      "change_class": "standard",
      "dependencies": [],
      "allowed_paths": [
        "docs/design",
        "internal/agentplan",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 4,
        "max_changed_lines": 350,
        "components": [
          "constraint-path parsing",
          "focused regression tests"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Read only the declared Markdown allowlist and preserve valid root-level and
  nested repository-relative paths.
- Reject malformed and duplicate entries before writing a constraints draft.
- Cover the parser boundary with focused deterministic tests.

## Architecture Decision

No ADR needed: this corrects an existing deterministic planning boundary.
