# Descendant Remediation Adoption Roadmap

## Status

`active` — all three #69 implementation phases and the GitHub issue #72
prerequisite are complete. The remaining transition is separately authorized
resumption of the blocked REK-40 / #67 publication path.

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
backlog source. Phase 1 is planned in Jira Story
[REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary Subtask
[REK-42](https://rekonlabs.atlassian.net/browse/REK-42). The ADR is accepted
for Option B. The reviewed Phase 1 implementation merged in
[PR #71](https://github.com/joeroberts/ai-tools/pull/71), and REK-41 / REK-42
were finalized. Publication remains unauthorized.

## Milestone Relationship

GitHub issue #69 is the immediate blocker for publishing the final reviewed
REK-40 branch.
It provides the focused successor transition that #22 must later consume for
bounded remediation cycles. It coordinates with #68 for future roadmap-impact
enforcement, but roadmap changes remain explicit reviewed Git diffs until #68
is implemented.

## Phase 1: Select And Define Successor Authority

Status: `complete` — PR #71 merged and REK-41 / REK-42 were finalized.

The accepted ADR selects the separately signed adoption record. Its versioned
contract and deterministic schema fixtures are merged.

## Prerequisite: Bootstrap Technical-Owner Trust

Status: `complete` — GitHub issue
[#72](https://github.com/joeroberts/ai-tools/issues/72) repository delivery
merged in [PR #73](https://github.com/joeroberts/ai-tools/pull/73) as
`6f70e448c0834fb0b416076f19ff21832d2a12a5`. Jira remains the authoritative
execution record; this roadmap does not duplicate its mutable lifecycle
status.

Provide a distinct governed technical-owner signer and reviewed public trust
entry before Phase 2 attempts live registry integration. This prerequisite did
not implement adoption validation or persistence.

[GitHub issue #74](https://github.com/joeroberts/ai-tools/issues/74) records an
uncommitted future platform-contract decision. It does not block #72 closeout
or #69 sequencing and makes no promise of Windows, Plan 9, or general
cross-platform support.

## Phase 2: Validate And Persist Adoption

Status: `complete` — the separately signed record is validated and persisted.

Implement full-range lineage, source, scope, budget, guidance, validation, and
review-evidence checks. Provide exact preview and explicitly approved atomic,
owner-only, non-overwriting successor persistence.

## Phase 3: Consume Successor In Publication

Status: `complete` — [PR #80](https://github.com/joeroberts/ai-tools/pull/80)
merged at `6dba9ebc6cfae13a286bc779245358cd605526de`; REK-48 / REK-49 were
finalized with verified Jira read-back.

Authorization issuance, push, and pull-request creation now verify the complete
successor chain, including cross-repository, recovery, replay, revocation,
moved-state, and REK-40-shaped lifecycle coverage. REK-40 publication remains
separately unauthorized.

## Delivery Order

```text
GitHub issue #69 Phase 1 -> GitHub issue #72
-> GitHub issue #69 Phase 2 -> GitHub issue #69 Phase 3
-> resume REK-40 publication
        |
        +-> #22 consumes the settled transition
```

Each #69 phase is a separate Jira implementation Subtask and contains no more
than two review components. GitHub issue #72 is a separate prerequisite work
item. A phase cannot start until its dependency is merged and the next Subtask
is exactly `In Progress` in a fresh trusted read-back.

## Validation Gates

Every phase requires focused deterministic tests, `make test`, `make vet`,
`make build`, `git diff --check`, and independent reviewer/verifier evidence for
the exact diff. Publication remains separately authorized.

## Completion Rule

The roadmap becomes `complete` only after all three phases merge, their Jira
Subtasks finalize, and the successor transition has successfully unblocked the
REK-40 publication path. GitHub or Jira state never silently changes this file.
