# Approved Ticket-Plan Contract Specification

## Source Identity Contract

Each input role records a canonical repository-relative path and SHA-256 digest
from the same verified file descriptor used for parsing. PRD, specification,
and roadmap must have distinct canonical paths and distinct content digests.
Reject repeated paths, symlink aliases, byte-identical aliases, missing roles,
and source drift before manager dispatch.

Store canonical source identities once and bind the three roles to them
explicitly. Render each accepted source body once in the manager catalog. Every
trace retains its role, canonical identity, digest, section, and excerpt.

## Persisted Contract

The strict versioned contract contains source identities and role bindings,
canonical Story content, assignment-owned fields, canonical source-derived
Subtask values, permitted manager-narrative rules, and the complete declared
slice manifest. The generated plan and workflow state bind the contract digest.

Generation and standalone validation call the same contract-aware validator.
Assignment-owned and source-derived values validate by equality with the
contract. Remove the temporary assignment-authority trace mechanism.

Contract artifacts are owner-only and refuse overwrite. Plan, contract, and
workflow versions are explicit. Unsupported combinations use a named migration
or fail with an actionable rejection; they are never silently reinterpreted.

## Allowed Paths

Implementation is limited to `docs/decisions`, `docs/design`, `docs/roadmaps`,
`internal/agentplan`, `internal/cli`, `internal/ticketplan`, and
`testdata/ticket-plans`.

## Review Budget

The total review budget is 26 changed files, 2150 changed lines, and contract schema and ADR, persistence fixtures, contract-aware validation, workflow and CLI integration, declared-slice enforcement, regression fixtures. Each slice must stay within its individual budget.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "contract-schema",
      "phase": "Phase 1",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": [
        "docs/decisions",
        "docs/design",
        "docs/roadmaps",
        "internal/agentplan",
        "internal/ticketplan",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 8,
        "max_changed_lines": 650,
        "components": [
          "contract schema and ADR",
          "persistence fixtures"
        ]
      }
    },
    {
      "id": "unified-contract-validation",
      "phase": "Phase 2",
      "change_class": "high-risk",
      "dependencies": [
        "contract-schema"
      ],
      "allowed_paths": [
        "internal/agentplan",
        "internal/cli",
        "internal/ticketplan",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 800,
        "components": [
          "contract-aware validation",
          "workflow and CLI integration"
        ]
      }
    },
    {
      "id": "declared-slice-coverage",
      "phase": "Phase 3",
      "change_class": "standard",
      "dependencies": [
        "unified-contract-validation"
      ],
      "allowed_paths": [
        "internal/agentplan",
        "internal/ticketplan",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 8,
        "max_changed_lines": 700,
        "components": [
          "declared-slice enforcement",
          "regression fixtures"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Reject unknown contract fields, malformed paths, duplicate slice IDs,
  invalid dependencies, invalid budgets, source aliases, and unsupported
  versions.
- Require generated Subtask cardinality, order, IDs, dependencies, budgets, and
  path arrays to equal the declared manifest.
- Reject omitted, duplicate, reordered, or extra slices and aggregated path
  strings before plan acceptance.
- Bind plan and workflow records to the matching contract digest and verified
  source identities.
- Limit manager remediation to fields explicitly classified as narrative.
- Add fixtures for same-path roles, symlink aliases, identical content under
  different paths, source drift, contract substitution, omitted finalization,
  and aggregated paths.

## Architecture Decision

Use `docs/decisions/0001-persist-ticket-plan-authority-contract.md` for the
persisted authority boundary, versioning, field ownership, and lifecycle.

## Non-Goals

- A parallel source or slice contract.
- Manager-authored evidence as authority.
- Any change to Jira or publication approval policy.
