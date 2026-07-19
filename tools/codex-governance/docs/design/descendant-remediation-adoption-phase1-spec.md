# Descendant Remediation Adoption Phase 1 Specification

## Technical Design

Phase 1 first changes the proposed ADR at
`docs/decisions/descendant-remediation-adoption.md` to `Accepted` with one
selected representation. Contract implementation may begin only after that
decision is explicit in the reviewed diff.

Implement the minimum versioned successor authority contract selected by the
ADR. It must represent repository and work-item identity, predecessor run and
digest, original base, prior and candidate commits, branch, adopted range,
source/work-item/task-bundle/configuration/guidance identities, full-range
review evidence identities, normalized validation outcomes, reason, authorized
role, signer, issuance/expiry, and audit-chain linkage.

The contract layer validates structure, binding syntax, and the shared signed
envelope in this phase. Phase 2 will connect the contract to Git history,
current worktrees, live trusted-key policy, source freshness, complete-diff
scope/budget evaluation, review artifacts, replay checks, and atomic
persistence. Phase 3 will connect verified successors to publication.

Preserve format-version-1 run behavior. Do not silently reinterpret, mutate, or
upgrade an existing run.

## Allowed Paths

Implementation is limited to `docs/decisions`, `docs/design`, `docs/roadmaps`,
and `internal/implementation`.

## Review Budget

This phase is limited to 9 changed files, 800 changed lines, and ADR and successor authority contract, versioned schema and validation fixtures.

## Declared Implementation Slice

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
        "max_changed_files": 9,
        "max_changed_lines": 800,
        "components": [
          "ADR and successor authority contract",
          "versioned schema and validation fixtures"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- ADR status, selected option, rejected option, authority matrix, migration,
  recovery, rollback, and audit consequences are explicit and internally
  consistent.
- The contract is strict and versioned; unknown fields, incomplete bindings,
  mutable aliases, invalid identities, malformed digests/timestamps, unsupported
  versions, and unpermitted roles fail closed.
- Contract encoding is deterministic and round-trippable without normalizing
  invalid input into compliance.
- Format-version-1 runs retain current parsing and publication behavior.
- No command creates or persists a successor instance or performs a remote
  operation in this phase.
- Focused deterministic tests and repository validation commands pass.

## Architecture Decision

`docs/decisions/descendant-remediation-adoption.md`

Option B is selected: a separately signed, versioned adoption-record payload.
Phase 1 implements strict payload validation, deterministic encoding, and
shared-envelope verification fixtures. Live trusted-key policy integration,
persistence, replay enforcement, and publication resolution remain later-phase
work.
