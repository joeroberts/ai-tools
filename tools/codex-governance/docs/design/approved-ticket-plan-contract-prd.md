# Approved Ticket-Plan Contract PRD

## Status

GitHub issue #38 is the source request and includes the requirements consolidated
from closed GitHub issue #20. This PRD is a work-item draft. Jira planning and
owner approval are required before implementation.

## Problem

Ticket-plan generation uses verified source documents and owner-assigned
constraints, but it does not persist the authority that joins them. The plan can
therefore validate differently during generation and standalone validation.

The planner can also present one document under multiple source roles, duplicate
that content in the manager prompt, and omit or reshape source-declared
implementation slices.

## Goal

Make every approved ticket plan reproducible from one durable contract that
binds distinct product requirements, technical rules, delivery sequencing, and
owner assignments.

## Product Outcomes

- A plan accepted during generation receives the same deterministic result from
  standalone validation.
- PRD, specification, and roadmap evidence retain distinct identities and role
  bindings throughout planning and traceability.
- Manager prompts contain no repeated source body.
- Every declared implementation slice appears exactly once and in approved
  dependency order.
- Unsupported legacy artifacts fail explicitly instead of being silently
  reinterpreted.

## Acceptance Criteria

- The workflow persists an integrity-bound, versioned ticket-plan contract.
- Source substitution, role aliasing, source drift, contract drift, and plan
  drift fail before any downstream side effect.
- Assignment-owned and canonical source-derived fields do not depend on
  manager-authored excerpts for authority.
- Deterministic failures do not consume semantic manager-remediation cycles.
- Regression coverage includes the omitted finalization slice and aggregated
  path output observed during GitHub issue #11 planning.

## Non-Goals

- Changing Jira write, transition, approval, publication, merge, or finalization
  policy.
- Implementing bounded workflow authorization from GitHub issue #22.
- Weakening verified reads, traceability, independent review, or fail-closed
  behavior.
