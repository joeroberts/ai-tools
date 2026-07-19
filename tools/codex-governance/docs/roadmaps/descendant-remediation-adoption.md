# Descendant Remediation Adoption Roadmap

## Status

`active` — REK-42 is the current Phase 1 implementation milestone.

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
backlog source. Phase 1 is planned in Jira Story
[REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary Subtask
[REK-42](https://rekonlabs.atlassian.net/browse/REK-42). The ADR is accepted
for Option B; implementation is limited to its contract and fixtures.
Publication remains unauthorized.

## Milestone Relationship

#69 is the immediate blocker for publishing the final reviewed REK-40 branch.
It provides the focused successor transition that #22 must later consume for
bounded remediation cycles. It coordinates with #68 for future roadmap-impact
enforcement, but roadmap changes remain explicit reviewed Git diffs until #68
is implemented.

## Phase 1: Select And Define Successor Authority

Status: `active` — REK-42 is `In Progress`.

The accepted ADR selects the separately signed adoption record. Implement its
versioned contract and deterministic schema fixtures only; Phase 2 owns
live trusted-key integration, replay enforcement, and persistence.

## Phase 2: Validate And Persist Adoption

Status: `pending`.

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
#69 Phase 1 -> #69 Phase 2 -> #69 Phase 3 -> resume REK-40 publication
                                      |
                                      +-> #22 consumes the settled transition
```

Each phase is a separate Jira implementation Subtask and contains no more than
two review components. A phase cannot start until its dependency is merged and
the next Subtask is exactly `In Progress` in a fresh trusted read-back.

## Validation Gates

Every phase requires focused deterministic tests, `make test`, `make vet`,
`make build`, `git diff --check`, and independent reviewer/verifier evidence for
the exact diff. Publication remains separately authorized.

## Completion Rule

The roadmap becomes `complete` only after all three phases merge, their Jira
Subtasks finalize, and the successor transition has successfully unblocked the
REK-40 publication path. GitHub or Jira state never silently changes this file.
