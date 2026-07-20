# Descendant Remediation Adoption Phase 3 Work Item

## Status

`Jira-planning`

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the approved backlog source. Phase 1, the technical-owner prerequisite, and Phase 2 are complete. Jira Story [REK-48](https://rekonlabs.atlassian.net/browse/REK-48) and primary Subtask [REK-49](https://rekonlabs.atlassian.net/browse/REK-49) were created from the approved plan and read back `In Progress`. No Phase 3 implementation may start until this planning baseline is committed and a fresh preflight succeeds.

## Scope

Resolve one already persisted technical-owner-signed adoption record into an immutable successor publication view and make repository-owner authorization issuance, governed push, and governed pull-request creation consume that view. Revalidate it at every remote side-effect boundary without changing the predecessor run or broadening publication authority.

## Non-Goals

- Publishing, pushing, or creating a pull request for REK-40 / #67 in this implementation task.
- Creating, changing, or repairing adoption records or their Phase 2 registry.
- Chained adoption, legacy-run migration, or generalized history repair.
- Jira/GitHub writes, merge, release, deployment, or force-push automation.
- Weakening repository-owner authorization, exact review evidence, lineage, remote-target, revocation, or one-time consumption gates.

## Technical Acceptance Criteria

- The shared resolver accepts only one complete, valid, currently trusted adoption record that exactly binds the immutable predecessor and checked-out candidate.
- `issue-publish`, `push`, and `create-pr` use that resolver when a successor record is requested; failures occur before signing, authorization consumption, or remote dispatch as applicable.
- Adopted publication retains all existing version-2 authorization bindings and non-adopted predecessor publication behavior.
- Focused fixtures prove cross-repository rejection, revocation, expiry, replay/recovery ambiguity, moved state, and REK-40-shaped lifecycle failure and success paths.
- The implementation stays within the declared five paths, 10 files, 850 lines, and two components; all deterministic and independent review gates pass.

## Planning Sources

- PRD: `docs/design/descendant-remediation-adoption-phase3-prd.md`
- Specification: `docs/design/descendant-remediation-adoption-phase3-spec.md`
- Roadmap: `docs/roadmaps/descendant-remediation-adoption-phase3.md`
- Accepted ADR: `docs/decisions/descendant-remediation-adoption.md`
- Program contract: `docs/design/descendant-remediation-adoption-spec.md`

## Review Budget

The future implementation Subtask is limited to 10 files, 850 changed lines, the declared five paths, and exactly two components. Any expansion requires a separate approved planning amendment before implementation edits.

## Next Transition

Commit this approved planning baseline, record that commit in REK-49, and run a fresh implementation preflight against the signed REK-48 / REK-49 export. Only then may Phase 3 code work begin.
