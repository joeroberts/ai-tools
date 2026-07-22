# ADR-0004: Bounded Workflow Authorization

## Decision

Use one signed, owner-approved authorization for the normal lifecycle of one
approved ticket-plan subtask. The authorization binds the repository identity,
contract digest, source digests, Jira keys, base revision, allowed paths,
budget, branch, remote, review limit, expiry, and deterministic derivation
rules. It is immutable, single-use per operation, revocable, and audited.

Previews remain mandatory before Jira and GitHub writes. A matching live
authorization makes that preview informational; it never permits an
out-of-contract action.

## Hard Stops

Scope or source drift, invalid status evidence, a third remediation cycle,
expired, consumed, or revoked authorization, missing exact-diff evidence,
failed or stale required checks, and any release, deployment, tag, force push,
secret, or unnamed remote endpoint require a new approval.

## Consequences

The implementation consumes existing #29 status evidence, #18 check evidence,
and the persisted ticket-plan authority contract. It does not replace them or
widen publication, repository, or cloud authority.
