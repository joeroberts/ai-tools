# Descendant Remediation Adoption Phase 3 PRD

## Status

`Jira-planning`

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the approved backlog source. Phase 1, the technical-owner signer prerequisite, and Phase 2 are merged. Jira Story [REK-48](https://rekonlabs.atlassian.net/browse/REK-48) and primary Subtask [REK-49](https://rekonlabs.atlassian.net/browse/REK-49) were created from the approved plan and read back `In Progress`. No Phase 3 implementation or REK-40 publication may begin until this planning baseline is committed and a fresh implementation preflight passes.

## Problem

Phase 2 can validate and persist one technical-owner-signed adoption record, but publication commands still bind only the immutable predecessor run. A reviewed candidate descendant therefore cannot receive a publication authorization or pass the push and pull-request gates without weakening existing bindings.

## Goal

Make publication authorization issuance, governed push, and pull-request creation consume one verified signed adoption record as the successor view, while preserving the predecessor run, existing repository-owner authorization, review-evidence, target-lineage, one-time-consumption, and fail-closed boundaries.

## Acceptance Criteria

- Consumption verifies the adoption envelope's technical-owner trust, current revocation, issuance and expiry, repository and work-item identity, predecessor-run digest, candidate commit, complete-diff, source, bundle, configuration, guidance, review, deterministic-check, and audit bindings.
- Authorization issuance binds its candidate commit and implementation base to the resolved successor view; no mutable branch or caller-provided successor identity is accepted.
- Push and pull-request creation re-resolve and verify that same record immediately before their remote side effect and reject changed, missing, revoked, expired, replayed, or mismatched records without consuming authorization.
- Existing non-adopted predecessor-run publication behavior remains unchanged; adoption does not grant repository-owner authority or bypass review evidence.
- Focused lifecycle fixtures cover REK-40-shaped recovery and replay, cross-repository records, revoked/expired records, and moved candidate or target state.
- Focused tests, `make test`, `make vet`, `make build`, `git diff --check`, and independent exact-diff reviewer/verifier evidence pass.

## Non-Goals

- Issuing an authorization, pushing, creating a pull request, or otherwise publishing REK-40 / #67 from this implementation task.
- Changing the Phase 2 record schema, persistence protocol, predecessor bytes, or technical-owner decision rights.
- Chained adoption, generalized recovery tooling, broader lifecycle changes, Jira/GitHub writes, merges, releases, or deployment.
