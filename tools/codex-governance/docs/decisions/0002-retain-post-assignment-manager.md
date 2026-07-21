# ADR-0002: Retain the Post-Assignment Manager

## Status

Accepted on 2026-07-21 for GitHub issue #59 and Jira REK-58/REK-59.

## Context

Ticket planning currently uses a hosted manager after owner assignment. The
manager turns the approved constraints into a Jira-ready plan, while local
validation, reviewer, verifier, and Jira gates remain authoritative. The
authority contract had not explicitly selected whether to retain that manager,
bind and reuse an earlier decomposition, or materialize the final plan locally.

The decision must preserve an independently reviewable ownership boundary,
prevent constraint drift and replay, and leave a clear process-ownership model
for #55 persistent supervision.

## Decision

Retain the post-assignment hosted manager. Its output is a bounded,
contract-aware proposal, never an authority source.

The owner-approved assignment remains the source of truth for Story and
Subtask identity, allowed paths, dependencies, change class, review budget,
and acceptance criteria. The manager may emit only explicitly permitted
narrative and ordering fields. A versioned schema derived from the assignment
rejects omitted, added, aggregated, out-of-pool, oversized, malformed, or
ambiguous values before a plan can be created or written to Jira.

## Ownership And Validation

- The manager owns no Jira mutation, approval, constraint, or review decision.
- The local planner binds manager input to the source-export digest, approved
  assignment digest, schema digest, manager model identity, and invocation ID.
- The manager result records those digests, terminal status, owner-only
  diagnostic references, and a result digest. A result with any mismatch,
  missing binding, invalid signature where required, or unexpected terminal
  status fails closed.
- Reviewer and verifier assess the exact generated plan/diff independently;
  neither may reuse the manager as evidence of correctness.
- Jira creation uses only the locally validated canonical plan and retains the
  manager-result binding as private execution evidence.

## Replay, Drift, And Recovery

A result is single-use for one assignment digest and invocation ID. Reusing it
for a different export, assignment, repository revision, or plan attempt is a
replay violation. Changed source evidence, assignment, schema, or allowed
model invalidates an in-flight result and requires a new run; it must not be
silently patched or reinterpreted.

Transient manager failure is terminal for that invocation. An owner may start a
new invocation only after the prior terminal record and diagnostics are
preserved. The retained-manager choice does not itself provide detached process
survival: #55 owns durable process identity, recovery, duplicate prevention,
and lifecycle diagnostics.

## Migration And Rollback

Existing artifacts without the new binding remain readable only as historical
records; they cannot authorize a new Jira write. New planning runs use the
bounded schema and binding immediately. Rollback disables new manager dispatch,
marks affected runs failed with diagnostics, and returns planning to the last
valid owner-approved assignment. It never reopens or overwrites an existing
Jira plan.

## Required Verification

The follow-on implementation must deterministically verify a valid bounded
result; malformed output; source, assignment, and schema drift; replayed
invocation IDs; forbidden fields; and rollback after dispatch failure. It must
also verify that reviewer/verifier evidence remains distinct from manager
output and that no failed manager result can reach Jira creation.

## Consequences

This preserves the current orchestration topology and limits change risk while
making authority and failure boundaries explicit. It retains hosted-manager
latency and availability as operational dependencies; those risks are bounded
by the existing lifecycle controls and the persistent-supervisor work in #55.
