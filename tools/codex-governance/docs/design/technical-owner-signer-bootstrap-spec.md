# Technical-Owner Signer Bootstrap Specification

## Technical Design

Add a dedicated command:

```text
codex-governance implementation bootstrap-technical-owner \
  --signer PATH [--repo-root PATH] [--approve]
```

The role is fixed internally to `technical-owner`; the command must not accept
a role flag. Without `--approve`, it validates the proposed locations and
prints a no-write preview. The preview states that the generated key ID cannot
exist until approved creation.

Approved execution requires the signer path to resolve outside the repository,
its existing ancestor to be owner-only, the destination not to exist, and the
current trusted-key registry not to contain a technical-owner entry. Symlink
resolution must not permit an apparent outside path to resolve into the
repository.

Extend the local-signer package with technical-owner-specific create and load
functions backed by the shared fixed-role primitive. The local record contains
only key ID and private key, is created with `0600` permissions, and remains
outside the repository. Loading reconstructs the public key and returns a
trusted-key value fixed to the technical-owner role.

After signer creation, append only the returned public key record to the
repository configuration and use the existing validated configuration save
path. Reload the signer and configuration, then require exact agreement on key
ID, role, algorithm, and public key. If policy persistence fails, remove only
the signer created by that invocation. A cleanup failure is a blocking error
reported with both failure contexts.

Existing export-issuer and repository-owner commands and key semantics remain
unchanged. Bootstrap grants no adoption or publication authority by itself;
later consumers must still validate a signed, unexpired record against the
current trusted registry.

## Allowed Paths

- `docs/design`
- `docs/roadmaps`
- `governance.yml`
- `internal/cli`
- `internal/signature`

## Review Budget

8 changed files, 600 changed lines, and fixed-role technical-owner signer bootstrap and path safety, repository trust onboarding plus read-back and failure recovery.

Exactly two components are permitted:

1. fixed-role technical-owner signer bootstrap and path safety; and
2. repository trust onboarding plus read-back and failure recovery.

## Declared Implementation Slice

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "technical-owner-trust-bootstrap",
      "phase": "Prerequisite",
      "change_class": "high-risk",
      "dependencies": [],
      "allowed_paths": [
        "docs/design",
        "docs/roadmaps",
        "governance.yml",
        "internal/cli",
        "internal/signature"
      ],
      "review_budget": {
        "max_changed_files": 8,
        "max_changed_lines": 600,
        "components": [
          "fixed-role technical-owner signer bootstrap and path safety",
          "repository trust onboarding plus read-back and failure recovery"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Preview validates inputs and reports fixed role and target locations without
  creating a directory, key, configuration change, or other side effect.
- Approved execution creates a distinct Ed25519 technical-owner signer and
  refuses overwrite, unsafe permissions, in-repository resolution, symlink
  escape, or pre-existing technical-owner trust.
- The saved local signer and public trust record match exactly after read-back.
- Policy-save failure removes newly generated untrusted signer material;
  cleanup failure remains visible and blocking.
- Private key material is never serialized into repository configuration,
  output, diagnostics, tests, prompts, or ledgers.
- Tests cover preview, approval, permissions, overwrite, unsafe ancestry,
  symlink resolution, duplicate trust, role mismatch, policy failure, cleanup
  failure, read-back mismatch, and unchanged existing bootstrap behavior.
- `make test`, `make vet`, `make build`, `git diff --check`, and independent
  exact-diff reviewer/verifier evidence pass.

## Architecture Decision

No ADR needed: this work implements the fixed technical-owner authority already
selected by the accepted descendant-remediation adoption ADR without changing
that authority model or the signed adoption-record contract.
