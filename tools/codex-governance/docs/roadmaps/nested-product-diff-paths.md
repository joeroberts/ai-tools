# Nested Product Diff Paths Roadmap

## Status

Work-item draft. Begin implementation only after the linked Jira Subtask is
`In Progress` and read back.

## Phase 1: Normalize Diff Paths

Slice ID: `nested-product-diff-path-normalization`. No implementation
dependency.

Use Git's relative diff mode for committed-range and working-tree numstat
collection. Add a nested-product repository fixture that proves the supplied
product root is the path basis and that untracked files remain fail-closed.

Exit when focused tests pass and the original signed `REK-33` task bundle
verifies without modification.

## Validation Gates

Run focused tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Before commit, obtain passing independent exact-diff
reviewer and verifier evidence from distinct executors and run
`make review-gate`.
