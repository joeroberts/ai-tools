# Approved Ticket-Plan Contract Roadmap

## Status

Work-item draft for GitHub issue #38. Execute phases sequentially. Each Jira
Subtask must be transitioned to `In Progress` and read back before its phase
begins.

## Phase 1: Contract Schema

Slice ID: `contract-schema`. No implementation dependency.

Define the ADR, strict source identity registry, role bindings, versioned
contract schema, owner-only persistence, and schema fixtures. Exit when malformed
or aliased sources and contracts fail deterministically and a valid three-source
contract round-trips.

## Phase 2: Unified Contract Validation

Slice ID: `unified-contract-validation`. Depends on `contract-schema`.

Make manager generation, workflow binding, CLI validation, and standalone
validation consume the same contract-aware validator. Remove temporary trace
authority and define explicit legacy-version behavior. Exit when generation and
standalone validation agree on valid and invalid fixtures.

## Phase 3: Declared Slice Coverage

Slice ID: `declared-slice-coverage`. Depends on
`unified-contract-validation`.

Extract the declared slice manifest before manager dispatch and enforce exact
cardinality, order, dependencies, budgets, and canonical path arrays. Exit when
omitted finalization, duplicate or extra slices, reordering, and aggregated paths
all fail before plan acceptance.

## Validation Gates

Every phase runs focused tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Each governed commit requires independent exact-diff
reviewer and verifier evidence from distinct executors plus a passing
`make review-gate`.

After Phase 3, run an end-to-end three-source planning fixture and confirm that
the manager catalog renders each source body once while trace records retain the
correct PRD, specification, and roadmap bindings.
