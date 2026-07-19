# Descendant Remediation Adoption Phase 1 Roadmap

## Status

`active` — REK-42 is `In Progress`.

This is the executable planning roadmap for the first sequential Jira plan
under GitHub issue #69. The program roadmap remains
`docs/roadmaps/descendant-remediation-adoption.md`.

Jira Story [REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary
Subtask [REK-42](https://rekonlabs.atlassian.net/browse/REK-42) were read back;
the Subtask is `In Progress`.

## Phase 1: Select And Define Successor Authority

The accepted ADR selects the separately signed adoption record. Implement only
its versioned contract and deterministic schema/validation fixtures.

Exit only when:

- the ADR is accepted with one representation and complete decision semantics;
- the contract binds every required authority and evidence identity;
- invalid contract fixtures fail closed;
- format-version-1 behavior remains unchanged; and
- all deterministic and independent review gates pass.

## Delivery Boundary

This plan creates one Jira Story and one primary implementation Subtask. It does
not aggregate Phase 2 adoption validation/persistence or Phase 3 publication
integration. After this phase merges and Jira finalizes, fresh approved sources
and a separate ticket plan are required for Phase 2.

## Dependency Record

```text
#69 Phase 1 -> future Phase 2 plan -> future Phase 3 plan -> resume REK-40 publication
```

## Validation Gates

Run focused deterministic tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Require independent reviewer and verifier evidence for the
exact diff before commit or publication.

## Completion Rule

Phase 1 becomes complete only when its implementation PR merges and its Jira
Story/Subtask finalize. That transition updates both this phase roadmap and the
program roadmap through reviewed Git changes; external state never silently
changes either roadmap.
