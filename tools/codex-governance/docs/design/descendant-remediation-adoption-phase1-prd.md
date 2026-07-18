# Descendant Remediation Adoption Phase 1 PRD

## Status

Approved planning source for the first sequential Jira plan under GitHub issue
[#69](https://github.com/joeroberts/ai-tools/issues/69). Jira Story
[REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary Subtask
[REK-42](https://rekonlabs.atlassian.net/browse/REK-42) were created and read
back in `To Do`. Implementation is not yet authorized.

## Problem

The CLI has no approved immutable authority representation for adopting a
reviewed descendant remediation commit. Without that contract, REK-40 cannot
bind its final reviewed branch to publication, and later validation or
publication work would have no canonical successor identity to consume.

## Goal

Select and persist the successor-authority architecture in
`docs/decisions/descendant-remediation-adoption.md`, then implement only its
versioned contract and deterministic schema/validation fixtures. Do not yet
implement adoption persistence or publication-gate integration.

## Product Outcomes

- One ADR selects either a versioned successor run or a separately signed
  adoption record.
- The ADR defines decision rights, signer role, expiry, replay, revocation,
  migration, recovery, audit linkage, rollback, and publication consumption.
- A strict versioned contract binds predecessor, candidate, source, task bundle,
  guidance, validation, full-range review, authorization, and audit identity.
- Malformed, incomplete, ambiguous, aliased, or version-mismatched contract
  fixtures fail deterministically.
- No predecessor mutation, successor persistence command, or publication
  behavior is introduced in this phase.

## Acceptance Criteria

- `docs/decisions/descendant-remediation-adoption.md` becomes `Accepted` with
  exactly one selected representation and explicit rejected alternatives.
- Canonical design sources are updated only to the extent required to define the
  selected successor authority and its decision boundaries.
- The versioned successor/adoption contract has strict field, identity, digest,
  role, timestamp, expiry, and format validation.
- Deterministic fixtures cover valid round-trip plus missing predecessor,
  mismatched repository/work item, malformed digests, unsupported version,
  invalid role, expiry/revocation input, and ambiguous candidate identity.
- Existing format-version-1 implementation runs remain unchanged and continue
  to validate under their current behavior.
- Focused tests, `make test`, `make vet`, `make build`, `git diff --check`, and
  independent exact-diff reviewer/verifier evidence pass.

## Non-Goals

- Full lineage, scope, budget, guidance, or complete-diff adoption validation.
- Exact preview or persistence of a successor instance.
- Publication authorization, push, or pull-request integration.
- Resuming REK-40 publication in this phase.
- Implementing bounded lifecycle authorization from #22.
- Writing Jira or GitHub, publishing, merging, releasing, or deploying from
  implementation commands.
