# Technical-Owner Signer Bootstrap PRD

## Status

`Jira-planning` — GitHub issue
[#72](https://github.com/joeroberts/ai-tools/issues/72) is the approved backlog
source. Jira Story [REK-43](https://rekonlabs.atlassian.net/browse/REK-43) and
primary Subtask [REK-44](https://rekonlabs.atlassian.net/browse/REK-44) were
created from the approved plan and read back as `To Do`. Implementation remains
unauthorized.

## Problem

The signed-envelope contract supports the `technical-owner` role, and the
accepted descendant-remediation ADR makes that role the only authority allowed
to approve an adoption record. This repository has no trusted technical-owner
key, no approved local technical-owner signer, and no governed command that can
create and onboard one.

The existing bootstrap paths are fixed to the export issuer and repository
owner. Reusing either key under a different role would break authority
separation and fail live adoption-record verification. Adding this work to
GitHub issue #69 Phase 2 would exceed that slice's approved component boundary.

## Goal

Provide one narrow, explicit bootstrap path that creates a distinct owner-only
technical-owner signer outside the repository and adds only its public trust
record to `governance.yml`. Preview must not mutate state; approved execution
must verify the signer and trust mapping by read-back.

## Product Outcomes

- The repository can verify a live adoption record signed by its configured
  technical owner.
- Export-issuer, repository-owner, and technical-owner keys remain distinct.
- Private key material remains outside the repository and owner-readable only.
- Bootstrap refuses overwrite, unsafe paths, duplicate role trust, and partial
  trust without silently recovering or weakening policy.
- GitHub issue #69 Phase 2 resumes with its validation and persistence scope
  unchanged.

## Required Workflow

1. Validate the repository configuration and proposed signer destination.
2. Show a no-write preview naming the fixed role, signer destination, and
   repository policy path.
3. Require explicit approval before generating a key or modifying policy.
4. Create one non-overwriting owner-only technical-owner signer outside the
   repository.
5. Add only the matching public key record to repository policy.
6. Reload both records and verify key ID, role, algorithm, and public key.
7. On a policy-write failure, remove the newly created untrusted signer and
   surface any cleanup failure.

## Success Criteria

- Preview produces no filesystem or repository mutation.
- Approved bootstrap creates one `0600` signer under an owner-only directory
  and one matching technical-owner trusted-key entry.
- Unsafe paths, symlink escape, existing destinations, existing role trust,
  permission failures, role mismatch, policy failure, and read-back mismatch
  fail closed with actionable diagnostics.
- Private key bytes never enter repository files, output, diagnostics, test
  snapshots, prompts, or runtime ledgers.
- Existing signer bootstrap and verification behavior remains unchanged.

## Non-Goals

- Creating, signing, persisting, revoking, or consuming an adoption record.
- Implementing #69 Phase 2 validation or persistence.
- Arbitrary caller-selected roles, general key management, rotation, or
  certificate infrastructure.
- Importing an unverifiable existing private key.
- Jira or GitHub writes, publication, push, pull-request creation, merge,
  release, deployment, or unrelated secret access.
