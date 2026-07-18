# Descendant Remediation Adoption Specification

## Status

Proposed technical contract for GitHub issue #69. The ADR decision described
below is required before implementation and remains deliberately unresolved.

## Architecture Decision Required

The ADR must select one canonical representation:

1. a versioned immutable successor implementation run that references and
   digest-binds its predecessor; or
2. a separately signed descendant-adoption record that resolves to an immutable
   successor publication view without altering the predecessor run.

The ADR must also decide authorized signer role, expiry, replay and revocation
semantics, crash recovery, migration for format-version-1 runs, audit-ledger
linkage, rollback, and how publication consumes the selected representation.

Both options must use the common signed-envelope and trusted-role model. A
plain mutable JSON copy, inferred transition, or in-place `commit_sha` update is
not an eligible option.

## Common Validation Contract

Adoption receives:

- predecessor run and its digest;
- fresh normalized work item, signed Jira export, task bundle, and repository
  configuration/guidance identity;
- clean candidate worktree, branch, and `HEAD`;
- deterministic validation outcome summary; and
- independent reviewer and verifier evidence for
  `original_implementation_base..candidate_head`.

Before persistence, validation must prove:

- repository, work-item, branch, original base, and predecessor identities
  match;
- source evidence is trusted, fresh, unrevoked, and agrees with the work item
  and task bundle;
- candidate `HEAD` descends from the original base and predecessor commit;
- the adopted predecessor-to-candidate history is linear, bounded, contains no
  merge or unrelated commits, and has not been rewritten;
- the complete original-base-to-candidate diff satisfies approved paths, file,
  line, and component budgets and file classifications;
- repository guidance and required deterministic checks match the approved
  snapshot;
- review evidence identifies distinct executors, contains no actionable
  findings, and binds the complete diff rather than only the last commit; and
- the successor identity has not already been issued or consumed.

## Persisted Successor Contract

The selected record must bind at minimum:

- format and repository identity;
- work-item key and approved-source identity;
- predecessor run ID and canonical digest;
- original implementation base, predecessor commit, candidate branch, and
  candidate commit;
- adopted range identity and complete-diff digest;
- work-item, source envelope, task-bundle, configuration, and guidance digests;
- reviewer/verifier executor identities, assessment digests, and combined
  evidence digest;
- normalized deterministic-check outcomes;
- reason, authorized role, signer identity, issuance time, and expiry when
  applicable; and
- preceding audit event or equivalent tamper-evident chain identity.

Persistence must be atomic, owner-only, non-overwriting, and replay-safe. The
predecessor remains byte-for-byte unchanged. Failure before atomic completion
must leave no trusted partial successor.

## CLI Contract

Provide an adoption command with two modes:

- preview: validates every input and renders the exact successor payload without
  persistence or remote effects; and
- approve: repeats validation, verifies explicit authority, and atomically
  persists exactly the previewed successor.

Provide stable human-readable diagnostics and versioned machine-readable
output. Diagnostics must identify the failed binding and safe remediation but
must not expose credentials, signing keys, private prompts, raw model output,
or unnecessary machine-local paths.

Publication authorization issuance, push, and pull-request creation must accept
only the ADR-selected verified successor representation and must revalidate the
chain, branch `HEAD`, source evidence, full-range review evidence, remote target,
expiry, revocation, and operation consumption immediately before side effects.

## Allowed Paths

Implementation slices may select only the applicable paths from this approved
pool: `docs/decisions`, `docs/design`, `docs/roadmaps`, `internal/implementation`,
`internal/signature`, `internal/cli`, and `testdata/implementation`.

## Review Budget

Each slice is limited to 10 changed files, 800 changed lines, and ADR and successor authority contract, versioned schema and validation fixtures, complete-range adoption validation, atomic non-overwriting successor persistence, publication successor-chain integration, cross-repository lifecycle and recovery fixtures.
Each slice may consume only its exact two assigned components from this pool;
the stricter per-slice file limits below remain binding.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "successor-contract",
      "phase": "Phase 1",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": [
        "docs/decisions",
        "docs/design",
        "docs/roadmaps",
        "internal/implementation"
      ],
      "review_budget": {
        "max_changed_files": 8,
        "max_changed_lines": 800,
        "components": [
          "ADR and successor authority contract",
          "versioned schema and validation fixtures"
        ]
      }
    },
    {
      "id": "adoption-validation",
      "phase": "Phase 2",
      "change_class": "high-risk",
      "dependencies": [
        "successor-contract"
      ],
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
    },
    {
      "id": "successor-publication",
      "phase": "Phase 3",
      "change_class": "high-risk",
      "dependencies": [
        "adoption-validation"
      ],
      "allowed_paths": [
        "docs/design",
        "docs/roadmaps",
        "internal/implementation",
        "internal/cli",
        "testdata/implementation"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 800,
        "components": [
          "publication successor-chain integration",
          "cross-repository lifecycle and recovery fixtures"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Cover valid successor adoption plus every failure named in GitHub #69 with
  deterministic table-driven tests.
- Prove predecessor immutability and atomic, non-overwriting persistence under
  validation, permission, crash, and retry failures.
- Prove two distinct repository/Jira/path configurations cannot replay or
  cross-adopt successor evidence.
- Preserve current behavior for runs without descendant remediation.
- Update the north star, canonical specification, implementation PRD/spec,
  roadmap, CLI help, and operator documentation only after the ADR selects the
  representation and authority model.
- Run focused tests, `make test`, `make vet`, `make build`, and
  `git diff --check` for every slice.

## Security Boundaries

- Adoption is local authority construction, not publication authority.
- No command may infer approval from current state or mutate the predecessor.
- Private keys and Jira credentials never enter repository files, bundles,
  diagnostics, audit summaries, or model prompts.
- A candidate that cannot be proven safe fails closed and requires a new
  governed run or separately approved exception; it is never normalized into
  compliance.
