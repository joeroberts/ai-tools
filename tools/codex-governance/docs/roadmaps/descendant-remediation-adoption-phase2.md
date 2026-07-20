# Descendant Remediation Adoption Phase 2 Roadmap

## Status

`in-progress`

The user authorized planning on 2026-07-19. This is the executable planning
roadmap for the second sequential Jira plan under GitHub issue #69. Jira Story
[REK-46](https://rekonlabs.atlassian.net/browse/REK-46) and primary Subtask
[REK-47](https://rekonlabs.atlassian.net/browse/REK-47) were created from the
approved ticket plan and read back in `In Progress`. A 2026-07-20 scope
amendment raises the implementation line budget from 800 to 850 and adds two
focused Phase 2 remediation regressions. Implementation remains paused until
the amended planning baseline is reviewed and committed, then restarted from a
fresh preflight.

## Phase 2: Validate And Persist Adoption

Implement exactly two review components:

1. complete-range adoption validation; and
2. atomic non-overwriting successor persistence.

Exit only when the command proves every repository, work-item, source,
predecessor, candidate, range, scope, budget, guidance, deterministic-check,
review, signer, expiry, registry, and replay binding; approved execution stores
one owner-only signed record; the predecessor remains unchanged; and all
deterministic and independent review gates pass.

The remediation must prove that complete-range `--numstat` parsing is
independent of Git rename detection and that a crash-left temporary registry
file cannot block a later safe retry.

## Delivery Boundary

This plan creates one Jira Story and one primary implementation Subtask. It does
not change publication authorization, push, or pull-request creation and cannot
resume REK-40. Those outcomes require a separately planned Phase 3.

## Dependency Record

```text
#69 Phase 1 complete
  -> #72 technical-owner signer prerequisite complete
  -> Phase 2 validation and persistence
  -> future Phase 3 publication consumption
  -> resume REK-40 / #67 publication
```

## Validation Gates

Run focused deterministic tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Require independent reviewer and verifier evidence for the
exact diff before commit or publication.

## Completion Rule

Phase 2 becomes `complete` only after its implementation PR merges and its Jira
Story/Subtask finalize. The program roadmap must then be updated through a
reviewed Git change. External state never silently changes either roadmap.
