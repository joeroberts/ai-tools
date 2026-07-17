# Nested Product Diff Paths Specification

## Path Contract

`internal/gitdiff` must return slash-separated paths relative to the supplied
repository root for both committed-range and staged/unstaged working changes.
When that root is nested below the Git top level, the monorepo prefix must not
appear in returned `Change.Path` values.

Git's relative diff mode is the canonical path source. Existing numstat
parsing, binary-file handling, and change accounting remain unchanged.

## Untracked Files

`WorkingChanges` continues to query untracked files and fails closed when any
are present. The change does not make untracked files scope-accounted and does
not weaken the existing error.

## Allowed Paths

Implementation is limited to `internal/gitdiff`.

## Review Budget

The total review budget is 2 changed files, 150 changed lines, internal/gitdiff path normalization.
The change has exactly one component and must remain within this budget.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "nested-product-diff-path-normalization",
      "phase": "Phase 1",
      "change_class": "standard",
      "dependencies": [],
      "allowed_paths": [
        "internal/gitdiff"
      ],
      "review_budget": {
        "max_changed_files": 2,
        "max_changed_lines": 150,
        "components": [
          "internal/gitdiff path normalization"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- A nested-product fixture proves `Changes` returns paths relative to the
  supplied product root.
- The same fixture proves `WorkingChanges` returns that path basis.
- A nested-product untracked file still causes `WorkingChanges` to fail.
- Existing top-level repository behavior remains unchanged.
- `REK-33` verification succeeds using its original signed allowed paths and
  task bundle.

## Architecture Decision

No ADR needed: this corrects the existing Git path basis at the validation
boundary without changing scope semantics or adding a component.

## Non-Goals

- Adding path aliases or fallback prefix stripping.
- Modifying validation rules, work-item schemas, or review budgets.
- Changing publication authorization.
