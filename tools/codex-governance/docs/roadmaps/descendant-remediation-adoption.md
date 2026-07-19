# Descendant Remediation Adoption Roadmap

## Status

`active` — Phase 1 is complete; GitHub issue #72 is the prerequisite before
Phase 2 planning.

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

Status: `Jira-planning` — GitHub issue
[#72](https://github.com/joeroberts/ai-tools/issues/72), Jira Story
[REK-43](https://rekonlabs.atlassian.net/browse/REK-43), and primary Subtask
[REK-44](https://rekonlabs.atlassian.net/browse/REK-44). REK-44 is `To Do`.

Provide a distinct governed technical-owner signer and reviewed public trust
entry before Phase 2 attempts live registry integration. This prerequisite does
not implement adoption validation or persistence.

## Phase 2: Validate And Persist Adoption

Status: `pending` — blocked by GitHub issue #72.

Implement full-range lineage, source, scope, budget, guidance, validation, and
review-evidence checks. Provide exact preview and explicitly approved atomic,
owner-only, non-overwriting successor persistence.

## Phase 3: Consume Successor In Publication

Status: `pending`.

Make authorization issuance, push, and pull-request creation verify the complete
successor chain. Add cross-repository, recovery, replay, revocation, moved-state,
and REK-40-shaped lifecycle fixtures. Update canonical and operator documents
to the approved behavior.

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
