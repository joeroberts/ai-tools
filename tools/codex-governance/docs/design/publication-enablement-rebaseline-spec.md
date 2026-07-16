# Publication Enablement and Rebaseline Specification

## Technical Design

Add explicit owner-side bootstrap and authorization-issuance commands. Bootstrap
creates a new Ed25519 signer only at an operator-selected path outside the
repository, requires explicit approval, refuses overwrite, enforces owner-only
directories and `0600` signer permissions, and appends only the public key to
`governance.yml` with role `repository-owner`.

Issuance loads a `0600` repository-owner signer, the locally committed run, the
configured repository identity, and the selected Git remote and target branch.
It reads the remote URL and current target SHA, builds a version-2 payload,
signs it with a bounded expiry, and writes a new `0600` envelope. It requires
explicit approval, refuses overwrite, and never calls push, GitHub PR, Jira, or
authorization-consumption code.

Version 2 binds `implementation_base_sha` to the run base and
`expected_target_sha` to the selected remote target ref. Publication validates
the current remote ref against `expected_target_sha` and requires both bound
SHAs to be ancestors of the exact authorized commit. Version-1 records retain
their existing equality and validation semantics; no version is silently
upgraded or broadened.

## Allowed Paths

Implementation is limited to `docs/decisions`, `docs/design`, `docs/roadmaps`,
`governance.yml`, `internal/signature`, `internal/implementation`, and
`internal/cli`.

## Review Budget

The total review budget is 10 changed files, 800 changed lines, and repository-owner authorization issuance, remote target and lineage binding.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "publication-enablement-rebaseline",
      "phase": "Phase 1",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": [
        "docs/decisions",
        "docs/design",
        "docs/roadmaps",
        "governance.yml",
        "internal/signature",
        "internal/implementation",
        "internal/cli"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 800,
        "components": [
          "repository-owner authorization issuance",
          "remote target and lineage binding"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Bootstrap repository-owner signer material and policy trust only after an
  explicit approved command; refuse unsafe permissions and overwrite.
- Issue a version-2 signed authorization after explicit approval, bind all
  run/repository/remote/target/commit/operation/time fields, write it owner-only,
  and perform no remote side effect or consumption.
- Preserve version-1 validation while validating version-2 implementation-base
  and expected-target fields independently.
- Require the authorized commit to descend from both bound SHAs and reject a
  moved target ref before push or PR creation.
- Add deterministic tests, then run `make test`, `make vet`, `make build`, and
  `git diff --check`.

## Architecture Decision

Required ADR: `docs/decisions/repository-owner-authorization-issuance.md`.
