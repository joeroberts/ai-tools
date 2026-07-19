# Technical-Owner Signer Bootstrap Work Item

## Status

`implementation-ready` — REK-43 is `To Do`; REK-44 is `In Progress`.

GitHub issue [#72](https://github.com/joeroberts/ai-tools/issues/72) is the
approved backlog source. Jira Story
[REK-43](https://rekonlabs.atlassian.net/browse/REK-43) and primary Subtask
[REK-44](https://rekonlabs.atlassian.net/browse/REK-44) were created from the
approved plan. The next transition is independent exact-diff review and
verification evidence before an authorized commit.

## Scope

Add a fixed-role, approval-gated local technical-owner signer bootstrap and
repository trust-onboarding path. Preserve all existing signer roles and keep
private material outside the repository.

## Non-Goals

- Creating or consuming descendant-adoption records.
- Implementing #69 Phase 2 validation or persistence.
- Arbitrary roles, key import, rotation, or general key management.
- Jira or GitHub writes, publication, merge, release, or deployment.
- Weakening signed-source, review, publication, or one-use authorization gates.

## Technical Acceptance Criteria

- No-write preview identifies the fixed role and target paths without creating
  state.
- Explicitly approved execution creates one distinct owner-only signer outside
  the repository and one matching public trusted-key entry.
- Unsafe paths, permissions, overwrites, duplicate trust, role mismatch,
  policy failure, cleanup failure, and read-back mismatch fail closed.
- Existing export-issuer and repository-owner paths remain unchanged.
- Private key material never appears in repository or diagnostic artifacts.
- Deterministic tests and all repository validation gates pass.

## Planning Sources

- PRD: `docs/design/technical-owner-signer-bootstrap-prd.md`
- Specification: `docs/design/technical-owner-signer-bootstrap-spec.md`
- Roadmap: `docs/roadmaps/technical-owner-signer-bootstrap.md`
- Existing ADR: `docs/decisions/descendant-remediation-adoption.md`

## Review Budget

Maximum 11 changed files, 850 changed lines, approved paths, and exactly two
components. Scope expansion requires a separately approved amendment or work
item before implementation edits.

## Dependency

This work must merge and the technical-owner trust entry must be reviewed
before #69 Phase 2 planning resumes. It does not authorize Phase 2 itself.
