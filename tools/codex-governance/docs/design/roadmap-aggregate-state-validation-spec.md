# Roadmap Aggregate-State Validation Specification

## Scope

Change `internal/roadmap` validation and rendering only. Update the canonical
design documentation that defines aggregate/phase failure semantics.

## Delivery slices

1. Document the approved aggregate/phase contract in the canonical design.
2. After that contract is committed, implement validation and rendering changes
   in `internal/roadmap` with focused package and CLI tests. The implementation
   slice depends on the documentation slice.

## Review bounds

| Slice | Allowed paths | Budget |
| --- | --- | --- |
| Documentation | `docs/design/roadmap-aggregate-state-validation-prd.md`, `docs/design/roadmap-aggregate-state-validation-spec.md` | 2 files, 180 lines, 1 component |
| Implementation | `internal/roadmap`, `internal/cli` | 5 files, 500 lines, 2 components |

## Planning assignments

The documentation slice is `Phase 1` with the `trivial` change class. Its
review budget is 2 changed files, 180 changed lines, and the `canonical design
documentation` component. Its allowed paths are
`docs/design/roadmap-aggregate-state-validation-prd.md` and
`docs/design/roadmap-aggregate-state-validation-spec.md`. No ADR is needed:
documents the explicitly approved minimal state contract without a schema or
authority-model change.

The implementation slice is `Phase 2` with the `standard` change class. Its
review budget is 5 changed files, 500 changed lines, and the `roadmap
validation and CLI` components. Its allowed paths are `internal/roadmap` and
`internal/cli`, and it depends on the `documentation` slice. No ADR is needed:
implements the approved minimal validation contract without a schema or
authority-model change.

## Validation rules

1. Preserve existing identity, phase ordering, phase validity, completion
   evidence, and one-active-phase checks.
2. Evaluate aggregate/phase coherence after phase validity is established.
3. Return one actionable diagnostic per violated aggregate rule. Diagnostics
   must name the aggregate status, summarize conflicting phase statuses, and
   state the required correction.
4. `Render` must invoke validation first and return an error for invalid source
   data in every output format.

## Allowed states

- `proposed`: all phases are `pending-approval`.
- `in-progress`: at least one phase is incomplete and none is `blocked`.
- `blocked`: at least one phase is `blocked`.
- `complete`: all phases are `complete`.

## Verification

- Focused table-driven roadmap tests, including each allowed state and every
  aggregate contradiction above.
- CLI tests proving `roadmap check` is nonzero and `roadmap status` rejects an
  inconsistent source file.
- `make test`, `make vet`, `make build`, and `git diff --check`.
