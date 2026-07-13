# Product Requirements Document: Implementation-Agent Orchestration

## Status

Approved for specification and roadmap on 2026-07-13. This PRD authorizes
design and implementation planning only; it does not authorize remote actions.

## Problem

`codex-governance` can validate Jira-backed work and orchestrate ticket-plan
creation, but it cannot govern an implementation agent through execution,
review, recovery, and closure. Teams therefore lack a consistent, auditable
way to constrain implementation work to one approved subtask.

## Product Goal

Orchestrate a bounded implementation agent for one approved Jira subtask:
select work, run deterministic preflight, start and observe the agent, then
produce a governed result or an actionable escalation.

## User Journey

1. A user selects one approved normalized work item.
2. The toolkit reads a fresh ticket baseline and runs policy, source-drift,
   path, review-budget, and repository preflight checks.
3. The toolkit creates a disposable Git worktree and a versioned task bundle,
   then dispatches the implementation agent through an execution adapter.
4. The user observes lifecycle state and redacted evidence without needing to
   inspect unrestricted agent logs.
5. The toolkit runs bounded review, verification, and finding-specific
   remediation, or escalates a non-converging or unsafe run.
6. A passing run can create a local commit. A separately approved,
   run-specific authorization may push that commit and create its pull request.

## Functional Requirements

- Use an adapter-first execution boundary; headless Codex is the first adapter.
- Let a user choose an approved local LLM for code execution or remediation.
  The selection is available only when its exact provider/model/adapter/config
  stack and versioned evaluation record are allowed by local policy for that
  role and task class. Initial eligibility is limited to `scoped-code-edit` and
  `finding-bound-remediation`; high-risk work is ineligible.
- Run one primary subtask in one dedicated disposable Git worktree.
- Supply a versioned task bundle containing the normalized work item, fresh
  ticket baseline, allowed paths, required commands, ADRs, and repository
  guidance.
- Enforce approved paths, review budgets, source-drift checks, and deterministic
  post-run validation.
- Persist lifecycle records, immutable local result references, commit/diff
  SHAs, command outcomes, adapter task IDs, and redacted summaries.
- Reconcile a task after a host restart without silently re-dispatching it or
  duplicating edits.
- Limit normal review and verification to two cycles. Remediation must be tied
  to explicit finding IDs and approved paths. A third cycle needs human approval
  and unused policy budget.
- Allow a local commit only when the work item enables it and pre-commit gates
  pass.
- Require a separate remote-publish authorization for push and PR creation.
  One authorization may explicitly permit both operations; it must bind the
  work item and run, remote identity, target ref, exact commit and base SHAs,
  approver, expiry, and PR target branch, and each operation is checked
  independently.

## Hard Boundaries

- One primary subtask per run; no ticket-intent or acceptance-criteria changes.
- No Jira write, merge, deployment, release, tag, cloud action, secret access,
  force-push, or protected/default-branch write.
- No unrestricted shell or credential access for the implementation agent.
- A provider choice cannot bypass the governed gateway, model allowlist,
  benchmark gate, task-bundle limits, or concurrency limits.
- The orchestration layer cannot override drift, policy, required-check, or
  human-decision gates.

## Human Gates

- Explicit approval before execution.
- Stop on source drift until a human rebaselines, splits, or stops the work.
- Explicit approval for a third review/verification cycle.
- Separate, run-specific approval before remote publishing.

## Success Criteria

- Lifecycle records and evidence are reproducible for every completed run.
- No agent remains orphaned after successful closure or recovery.
- Scope and review-budget validation are deterministic.
- Retries are bounded and recoverable host failures do not duplicate edits.
- A user receives either a governed result or a precise escalation.

## Non-Goals

- Autonomous backlog selection.
- Semantic adjudication of ticket drift.
- Merge or release automation.
- Unrestricted shell or credential access.

## Dependencies And Risks

- A stable headless Codex adapter contract is required for the first execution
  provider.
- Worktree cleanup and recovery must preserve evidence without retaining raw
  prompts, credentials, or machine-local paths in Jira.
- Push and PR adapters must enforce the bounded remote-publish authorization
  independently of prompt text.
