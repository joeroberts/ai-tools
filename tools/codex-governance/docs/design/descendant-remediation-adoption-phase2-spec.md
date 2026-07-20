# Descendant Remediation Adoption Phase 2 Specification

## Planning Record

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
backlog source. Jira Story
[REK-46](https://rekonlabs.atlassian.net/browse/REK-46) and primary Subtask
[REK-47](https://rekonlabs.atlassian.net/browse/REK-47) are the approved Phase
2 execution contract and were read back in `To Do` on 2026-07-19.

## Technical Design

Phase 2 implements the `adoption-validation` slice already declared by the
accepted program specification. It adds an implementation command with preview
and explicitly approved modes. The command receives the immutable predecessor
run, fresh normalized work item and signed Jira export, exact task bundle,
candidate worktree, complete-range review evidence, strict deterministic-check
evidence, owner-local adoption registry, reason, issuance/expiry values, and—
only for approved execution—the technical-owner signer.

Validation runs in this order and stops on the first invalid trust boundary:

1. load current repository policy, work item, signed source, task bundle, and
   immutable predecessor without rewriting any input;
2. verify repository, Jira, work-item, source, bundle, configuration, guidance,
   signer-role, freshness, expiry, and digest bindings;
3. require a clean candidate worktree on the predecessor branch and derive the
   exact candidate commit rather than accepting a mutable alias;
4. prove ancestry from the original base and predecessor commit, reject merges
   and rewritten or out-of-scope intermediate commits, and derive the adopted
   range deterministically;
5. evaluate the complete original-base-to-candidate diff against allowed paths,
   file, line, component, generated-file, and lockfile budgets;
6. verify deterministic-check evidence matches the approved validation plan and
   binds passing exit state plus output digests;
7. verify independent passing reviewer and verifier artifacts bind the complete
   diff and distinct executor identities; and
8. construct and validate the exact version-1 `AdoptionRecord`, then check the
   owner-local registry for duplicate, conflicting, expired, revoked, or replayed
   identity.

Preview prints the exact payload and intended registry identity but does not
read private key bytes, sign, create directories or files, append audit state,
or contact a remote system. Approved execution repeats validation, loads the
owner-local signer, requires an exact match to the configured technical-owner
public trust record, signs the same payload, and persists the signed envelope.

Persistence uses an owner-only registry outside the repository. The final path
is derived from the immutable adoption identity. Creation is same-directory,
atomic, owner-only, and non-overwriting. A failure before the atomic boundary
leaves no trusted record; cleanup failures are explicit blockers. The
predecessor run, worktree, branch, and Git history are never modified.

Phase 2 may record the preceding audit-event identity in the signed payload but
does not change publication consumers. Phase 3 owns audit-chain consumption,
revocation-at-publication checks, authorization issuance, push, and pull-request
integration.

## Allowed Paths

Implementation is limited to `internal/implementation`, `internal/signature`,
`internal/cli`, and `testdata/implementation`.

## Review Budget

This phase is limited to 10 changed files, 800 changed lines, and complete-range adoption validation, atomic non-overwriting successor persistence.

## Declared Implementation Slice

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "adoption-validation",
      "phase": "Phase 2",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": [
        "internal/implementation",
        "internal/signature",
        "internal/cli",
        "testdata/implementation"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 800,
        "components": [
          "complete-range adoption validation",
          "atomic non-overwriting successor persistence"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- All preview and approved modes use one validation and payload-construction
  path; approved mode cannot persist a payload that preview would reject.
- Candidate and range identities are derived from Git and cannot be supplied as
  `HEAD`, a default branch, `refs/*`, or another mutable alias.
- Scope and budget evaluation covers the complete original-base-to-candidate
  diff and every intermediate commit, not only the remediation delta.
- Source, bundle, configuration, guidance, deterministic checks, and review
  evidence are fresh, exact, and internally consistent.
- The technical-owner signer is accessed only during explicitly approved
  execution and must exactly match configured trust after read-back.
- The registry is outside the repository, owner-only, symlink-safe, atomic,
  non-overwriting, and replay-safe.
- Invalid lineage, dirty or moved state, stale evidence, partial-range review,
  same-executor evidence, changed guidance, mismatched signer, duplicate record,
  permission failure, crash boundary, cleanup failure, and retry all fail closed.
- Existing adoption-contract fixtures and unremediated implementation runs keep
  their current behavior.

## Architecture Decision

`docs/decisions/descendant-remediation-adoption.md` is accepted and selects a
separately signed adoption record authorized only by the technical owner. No new
ADR is required for this slice.
