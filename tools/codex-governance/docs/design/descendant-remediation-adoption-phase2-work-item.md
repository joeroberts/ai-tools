# Descendant Remediation Adoption Phase 2 Work Item

## Status

`Jira-planning`

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
approved backlog source. Phase 1, Jira REK-41 / REK-42, and the technical-owner
signer prerequisite in Jira REK-43 / REK-44 are complete. The user authorized
Phase 2 planning on 2026-07-19. Jira Story
[REK-46](https://rekonlabs.atlassian.net/browse/REK-46) and primary Subtask
[REK-47](https://rekonlabs.atlassian.net/browse/REK-47) were created from the
approved ticket plan and read back in `In Progress`. The approved 2026-07-20
scope amendment authorizes a narrow Phase 2 remediation planning update.

## Scope

Validate one approved reviewed linear descendant against the immutable
predecessor, fresh source, complete-range scope and budget rules, current
guidance, deterministic checks, and independent exact-diff evidence. Provide an
exact no-write preview and explicitly approved atomic persistence of one
technical-owner-signed adoption record.

## Non-Goals

- Publication successor consumption, push, or pull-request integration.
- Resuming REK-40 publication before Phase 3.
- Predecessor mutation, chained adoption, or general run-history repair.
- Jira or GitHub writes, commit, push, pull-request creation, merge, release, or
  deployment from the adoption command.
- Bounded workflow authorization from #22.

## Technical Acceptance Criteria

- The implementation matches the Phase 2 PRD and specification without
  aggregating Phase 3.
- The complete-range validation and atomic-persistence components remain within
  the declared allowed paths and 10-file, 850-line budget.
- Focused regressions prove that complete-range `--numstat` parsing is
  independent of Git rename detection and that a crash-left temporary registry
  file cannot block a later safe retry.
- Preview performs no trusted persistence, signing, audit append, remote write,
  or private-signer access.
- Approved persistence repeats validation, verifies technical-owner trust, and
  creates exactly one immutable signed record without altering predecessor or
  repository state.
- All required deterministic and independent review gates pass.

## Planning Sources

- PRD: `docs/design/descendant-remediation-adoption-phase2-prd.md`
- Specification: `docs/design/descendant-remediation-adoption-phase2-spec.md`
- Roadmap: `docs/roadmaps/descendant-remediation-adoption-phase2.md`
- Accepted ADR: `docs/decisions/descendant-remediation-adoption.md`
- Program contract: `docs/design/descendant-remediation-adoption-spec.md`

## Review Budget

The future implementation Subtask is limited to 10 files, 850 changed lines,
the four declared implementation paths, and exactly two components. Scope
expansion requires a separately approved amendment before implementation edits.

## Next Transition

Obtain independent reviewer and verifier evidence for this exact planning diff,
commit the amended planning baseline after explicit approval, record that
commit in REK-47 after exact preview and approval, then run a fresh
implementation preflight against the current `In Progress` Subtask.
