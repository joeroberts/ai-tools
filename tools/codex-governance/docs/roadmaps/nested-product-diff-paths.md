# Nested Product Diff Paths Roadmap

## Status

`complete`

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

## Completion Record

GitHub issue #60 is complete. Its backlog, execution, and delivered-diff
evidence remain in their respective GitHub, Jira, and Git/PR/CI records.
