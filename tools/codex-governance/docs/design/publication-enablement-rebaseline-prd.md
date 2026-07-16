# Publication Enablement and Rebaseline PRD

## Status

This is the product source for the separately authorized blocker fix required
to publish the completed `REK-28` implementation.

## Problem

The publication verifier requires a signed `repository-owner` authorization,
but the repository trusts only an `export-issuer` and provides no approved
owner-side issuance path. Its payload also treats the implementation run base
as the current remote target SHA. Once a planning baseline is committed locally
before implementation, those SHAs legitimately differ and publication becomes
impossible even when the final commit descends from both.

## Goal

Make signed publication operational without weakening separation, exact
binding, ancestry checks, explicit approval, or one-time consumption.

## Product Outcomes

- Repository owners can provision and use local owner-only signing material
  without exposing private keys.
- Authorization issuance and remote publication remain separate approved
  commands.
- A run may include local planning commits while its pull request still targets
  the exact current remote base.
- Existing version-1 authorization semantics do not silently broaden.

## Acceptance Criteria

- Bootstrap and issuance fail closed on missing approval, unsafe permissions,
  overwrite attempts, untrusted keys, invalid operations, or mismatched runs.
- Version-2 authorization binds distinct implementation and target SHAs.
- The authorized commit must descend from both bound SHAs.
- Signing performs no network mutation and does not consume the authorization.
- Focused regression tests prove valid and rejection paths.

## Non-Goals

- Weakening signed authorization or review-evidence enforcement.
- Automatic publication, merging, releasing, or deployment.
- Broad enterprise key lifecycle management.
