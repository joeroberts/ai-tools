# Publication Enablement and Rebaseline Work Item

## Status

This work item was discovered while publishing the completed `REK-28`
implementation. No trusted `repository-owner` signer or signed publication
authorization exists, and the current publication payload requires the remote
target SHA to equal the implementation run base.

Jira Story `REK-29` and primary implementation Subtask `REK-30` are the
approved execution contract. `REK-30` is `In Progress` before the planning
baseline is committed and before implementation begins.

## Scope

Provide an explicit owner-side path to bootstrap an owner-only
`repository-owner` signer, trust only its public key in repository policy, and
issue a run-specific signed publication authorization without performing a
remote operation. Separate the implementation base SHA from the expected
remote target SHA while proving that both are ancestors of the authorized
commit.

## Non-Goals

- Changing the reviewed `REK-28` implementation diff.
- Automatically pushing or creating a pull request after signing.
- Storing private keys in the repository, task bundle, ledger, or Jira.
- Force pushes, default-branch writes, merges, tags, releases, or deployment.
- General-purpose certificate, revocation, or organizational key management.

## Acceptance Criteria

- An explicit approved bootstrap creates non-overwriting `0600` owner-only
  signer material outside the repository and adds only its public key as a
  trusted `repository-owner`.
- An explicit approved issuance command creates a non-overwriting `0600`
  signed authorization bound to one run, repository, remote fingerprint,
  branch, commit, target branch, implementation base, expected target SHA,
  operations, issuance time, and expiry without invoking GitHub.
- Versioned validation distinguishes the implementation base from the current
  remote target and requires both to be ancestors of the authorized commit.
- Legacy version-1 authorization validation remains fail-closed and unchanged.
- Focused tests cover permission, overwrite, signature, mismatch, ancestry,
  expiry, and no-remote-side-effect boundaries.

## Review Budget

Maximum 10 changed files, 800 changed lines, and repository-owner authorization
issuance, remote target and lineage binding. The architectural decision is
recorded in `docs/decisions/repository-owner-authorization-issuance.md`.
