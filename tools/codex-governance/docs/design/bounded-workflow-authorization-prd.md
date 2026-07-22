# Bounded Workflow Authorization PRD

## Problem

Approved bounded work currently requires repeated command-level approvals even
when every target and gate is already fixed by the approved ticket-plan
contract.

## Goal

Allow one integrity-protected owner approval to authorize the normal lifecycle
of one bounded Jira subtask through merged-state finalization, while retaining
previews, exact read-backs, independent review, and all hard stops.

## Non-Goals

This does not authorize releases, deployments, tags, force pushes, destructive
operations, secret access, infrastructure changes, or unnamed remote targets.

## Success Criteria

Authorization is non-replayable, revocable, expiring, contract-bound, and
audited. Every external write fails closed unless its deterministic preview,
target, source state, evidence, and lifecycle prerequisite match the live
authorization.
