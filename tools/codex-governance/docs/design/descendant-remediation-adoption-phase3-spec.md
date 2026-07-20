# Descendant Remediation Adoption Phase 3 Specification

## Planning Record

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the backlog source. Jira Story [REK-48](https://rekonlabs.atlassian.net/browse/REK-48) and primary Subtask [REK-49](https://rekonlabs.atlassian.net/browse/REK-49) are the approved Phase 3 execution contract and were read back `In Progress`. Planning baseline commit `c3469dd79632c1d688b699b48540bf3c40f6a4f7` and preflight `run-647a35f1589e3d50` authorize the bounded implementation slice.

## Technical Design

The `publication-successor-consumption` slice adds an explicit, read-only successor resolver. It accepts the immutable predecessor run, signed adoption envelope, current trusted-key registry, and candidate worktree. It verifies the Phase 1 record contract and Phase 2 persistence identity, then proves the predecessor-run digest and candidate commit against the exact run and checked-out commit. It produces an immutable publication view; it neither signs nor persists anything.

`implementation issue-publish` must require that resolver when a successor record is supplied. It binds the version-2 repository-owner authorization to the resolved candidate commit and predecessor implementation base. The command continues to read the exact remote target ref, performs no remote mutation, and cannot treat an adoption record as repository-owner authority.

`implementation push` and `implementation create-pr` accept the same record input and resolve it again immediately before authorization consumption and their remote side effect. The candidate in the successor view, authorized commit, repository identity, work item, predecessor run, remote fingerprint, target SHA, and exact-diff review evidence must agree. Any failure leaves the authorization unconsumed. Existing runs without a successor record retain their established publication path unchanged.

Resolution reads the record from the configured owner-local registry using its immutable identity, rejects missing, duplicate/conflicting, symlinked, or non-owner-only state, and treats a removed trusted technical-owner key as revocation. It never repairs registry state or infers a record from Git, Jira, branch names, or commit messages.

## Allowed Paths

Implementation is limited to `internal/implementation`, `internal/cli`, `testdata/implementation`, `docs/design`, and `docs/roadmaps`.

## Review Budget

This phase is limited to 10 changed files, 850 changed lines, and exactly two components: successor publication-view resolution; and publication-boundary consumption.

## Declared Implementation Slice

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "publication-successor-consumption",
      "phase": "Phase 3",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": ["internal/implementation", "internal/cli", "testdata/implementation", "docs/design", "docs/roadmaps"],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 850,
        "components": ["successor publication-view resolution", "publication-boundary consumption"]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Resolver validation is strict and complete: envelope role/trust/revocation, time bounds, registry identity and permissions, all record bindings, and exact predecessor/candidate Git state must agree.
- Issuance, push, and pull-request paths consume one shared immutable successor view and never accept a caller-selected candidate SHA or mutable ref.
- Push and pull-request paths revalidate immediately before consumption; a failed successor check neither consumes authorization nor starts a remote action.
- The record remains separate from repository-owner authorization and review evidence; all existing authorization, remote-target, lineage, and one-time operation checks still apply.
- Fixtures cover cross-repository substitution, record revocation and expiry, registry replay/recovery ambiguity, moved candidate/target state, and the REK-40-shaped predecessor-to-candidate lifecycle.
- Non-adopted predecessor-run tests retain their current behavior.

## Architecture Decision

`docs/decisions/descendant-remediation-adoption.md` is accepted and requires the Phase 3 side-effect boundary to revalidate the signed record. No new ADR is required for this bounded consumption slice.
