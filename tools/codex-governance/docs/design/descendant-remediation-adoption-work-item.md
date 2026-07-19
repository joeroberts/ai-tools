# Descendant Remediation Adoption Work Item

## Status

`implementation-ready` — REK-42 is `In Progress`.

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
approved backlog source. Jira Story
[REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary Subtask
[REK-42](https://rekonlabs.atlassian.net/browse/REK-42) were created from the
approved Phase 1 plan and read back. REK-42 is `In Progress`; the next
transition is completion only after the remediation diff is independently
reviewed, merged, and finalized in Jira.

## Scope

Plan an ADR-first, repository-neutral successor transition for approved,
reviewed linear descendant remediation commits. Preserve predecessor
immutability, require fresh source and complete-range evidence, and integrate
the verified successor with existing separately authorized publication gates.

## Non-Goals

- Implementing any successor representation before the ADR is approved.
- Mutating prior runs or accepting arbitrary, merged, rewritten, unrelated, or
  unverifiable history.
- Implementing the complete bounded authorization owned by #22.
- Automatic Jira, GitHub, push, pull-request, merge, release, or deployment
  actions.
- Weakening current source, scope, budget, review, signer, remote, or
  publication-consumption checks.

## Technical Acceptance Criteria

- Produce three sequential Jira plans across the program, each containing one
  Story and one primary implementation Subtask. The current plan contains only
  Phase 1 and must not aggregate Phases 2 or 3.
- Make the first Subtask select and persist the ADR before behavior changes.
- Bind successor adoption to fresh source authority, predecessor and candidate
  lineage, complete-range scope/budget validation, repository guidance,
  deterministic outcomes, and exact full-range independent review evidence.
- Preserve predecessor bytes and current unremediated-run behavior.
- Fail before persistence or remote side effects on every mismatch, stale input,
  replay, history anomaly, permission failure, or partial write.
- Prove repository neutrality with at least two distinct repository/Jira/path
  fixtures.
- Update the persisted roadmap at planned, active, blocked, and complete
  milestone transitions through reviewed Git diffs.

## Current Phase 1 Planning Sources

- PRD: `docs/design/descendant-remediation-adoption-phase1-prd.md`
- Specification: `docs/design/descendant-remediation-adoption-phase1-spec.md`
- Roadmap: `docs/roadmaps/descendant-remediation-adoption-phase1.md`
- Accepted ADR: `docs/decisions/descendant-remediation-adoption.md`

The umbrella PRD, specification, and roadmap remain the program contract for
all three phases; they are not aggregated into the current ticket plan.

## Review Budget

The current Phase 1 Subtask is limited to 10 files, 800 lines, approved paths,
and exactly two components. The complete planning-base-to-candidate
implementation diff and the remediation delta must remain within those limits.
Later phases require new planning-source digests, constraints, approval, Jira
creation, and committed work items. No ticket may aggregate the three phases
into one implementation diff.

## Architecture Decision

The accepted ADR selects the separately signed adoption record and defines its
decision-rights, signing, lifecycle, migration, recovery, and rollback contract.
