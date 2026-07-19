# Technical-Owner Signer Bootstrap Work Item

## Status

`closed` — repository delivery is complete in
[PR #73](https://github.com/joeroberts/ai-tools/pull/73), merged as
`6f70e448c0834fb0b416076f19ff21832d2a12a5`.

Jira remains the authoritative execution record; this document records only
the durable repository closeout and does not restate mutable Jira lifecycle
status. The next transition is the separately authorized Jira and GitHub
closeout, then stop without starting GitHub issue #69 Phase 2.

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

The implementation delivery stayed within its approved scope. This closeout is
limited to three existing Markdown files, 180 changed lines, and one docs
component. Scope expansion requires a separately approved amendment or work
item before implementation edits.

## Dependency

The merged technical-owner bootstrap delivery and reviewed public trust entry
are the prerequisite for #69 sequencing. They do not authorize implementation,
planning, or resumption of GitHub issue #69 Phase 2.

## Closeout Handoff

PR #73 delivered the fixed technical-owner authority model: an approval-gated,
fixed-role local signer bootstrap with public-trust onboarding and exact
read-back verification. The existing safety boundaries remain in force:
private signer material stays outside the repository; no adoption record was
created, signed, persisted, revoked, or consumed; and review, publication, and
one-use authorization gates were not weakened.

The manager treated CodeRabbit comments as advisory evidence. Each comment was
validated against user direction, the canonical specification, the Jira
contract, and repository rules. Three in-scope findings were remediated; a
later platform-contract suggestion was out of scope, and an optional
portability question was separated into uncommitted
[GitHub issue #74](https://github.com/joeroberts/ai-tools/issues/74).
Invalid or out-of-scope automated comments cannot widen the active work item.

GitHub issue #74 is a future platform-contract decision, not a promise of
Windows, Plan 9, or general cross-platform support. It does not block #72
closeout or #69 sequencing.
