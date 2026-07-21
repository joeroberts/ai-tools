# ADR 0004: Bind Governed Work To Explicit Roadmap Transitions

## Status

Proposed

## Context

Structured roadmaps validate their internal aggregate and phase states, but a
governed work item does not declare whether it changes a roadmap milestone.

## Decision

Add a versioned, repository-neutral roadmap-adoption configuration and bounded
`roadmap_impact` declarations to the work-item and ticket-plan contracts.
Required impacts bind a configured canonical roadmap, phase, and explicit
transition; not-applicable impacts carry a bounded reason.

Validation is deterministic and fail-closed. It verifies repository-relative
configuration, roadmap identity and digest bindings, allowed-path and review
budget coverage, and valid aggregate/phase transitions using the #51 state
table. A preview helper prints an exact transition preview but makes no local
or remote change.

## Consequences

- Lifecycle gates gain an explicit, reviewable milestone-state binding.
- Existing repositories migrate through explicit mappings or bounded exemptions.
- The implementation adds no synchronization engine or external write ability.

## Validation

- Two isolated fixtures prove repository, Jira-project, path, and roadmap-ID
  neutrality.
- Table-driven tests cover declarations, configuration, transitions, stale and
  replayed evidence, and affected lifecycle gates.

## Follow-Up

Create Jira subtasks from the approved planning baseline. Do not begin a slice
before its primary Subtask is read back `In Progress`.
