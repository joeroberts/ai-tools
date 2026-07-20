# Descendant Remediation Adoption Phase 3 Roadmap

## Status

`Jira-planning`

This is the executable planning roadmap for the third sequential Jira plan under GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69). Phase 1, the technical-owner prerequisite, and Phase 2 are complete. Jira Story [REK-48](https://rekonlabs.atlassian.net/browse/REK-48) and primary Subtask [REK-49](https://rekonlabs.atlassian.net/browse/REK-49) were created from the approved plan and read back `In Progress`. Implementation remains blocked until this planning baseline is committed and a fresh preflight passes.

## Phase 3: Consume Successor In Publication

Implement exactly two review components:

1. successor publication-view resolution; and
2. publication-boundary consumption.

Exit only when authorization issuance, push, and pull-request creation resolve the complete signed successor record; independently revalidate it at each side-effect boundary; preserve all existing repository-owner authorization and review-evidence gates; and reject revoked, expired, replayed, cross-repository, recovery-ambiguous, or moved-state records before consumption or dispatch.

## Delivery Boundary

This plan creates one Jira Story and one primary implementation Subtask. It does not issue an authorization, push, or create a pull request. Resuming the blocked REK-40 / #67 publication path is a later, separately authorized operation after this phase merges and all normal publication gates pass.

## Dependency Record

```text
#69 Phase 1 complete
  -> #72 technical-owner signer prerequisite complete
  -> #69 Phase 2 complete
  -> Phase 3 successor publication consumption
  -> separately authorize and resume REK-40 / #67 publication
```

## Validation Gates

Run focused deterministic tests, `make test`, `make vet`, `make build`, and `git diff --check`. Require independent reviewer and verifier evidence for the exact diff before commit or publication.

## Completion Rule

Phase 3 becomes `complete` only after its implementation PR merges and its Jira Story/Subtask finalize. The program roadmap can become complete only after the successor transition then successfully unblocks the separately authorized REK-40 publication path. External state never silently changes either roadmap.
