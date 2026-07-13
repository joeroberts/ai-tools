# Implementation-Agent Orchestration Roadmap

## Status

`proposed`

Structured phase state:
[implementation-agent-orchestration.yaml](implementation-agent-orchestration.yaml).

## Goal

Add adapter-first, governed orchestration for a bounded implementation agent.
Headless Codex is the initial adapter. Users may later select a policy-approved
local LLM for code execution or remediation through the governed gateway. The
implementation must preserve the existing Jira, policy, review, evidence, and
explicit-approval boundaries.

## Approval Record

- Approval status: approved for planning
- Approving instruction: user approval on 2026-07-13
- Approved scope: PRD, specification, and phased implementation roadmap
- Not authorized: implementation, local commits, remote pushes, PR creation,
  Jira writes, merges, releases, deployments, secrets, or cloud actions

## Phased Work

1. **Discovery And Contract Approval**
   - Finalize the PRD, state machine, adapter interface, and threat model.
   - Define fixtures for valid execution, drift, scope violation, crash
     recovery, authorization failure, and escalation.

2. **Foundations**
   - Add the `implementation-run` schema and runtime-ledger upgrades.
   - Build task-bundle generation, policy/config additions, and dry-run
     preflight with deterministic tests.

3. **Read-Only Orchestration**
   - Implement fake-adapter launch, status, result, and reconciliation.
   - Test lifecycle integrity, retry bounds, host-crash recovery, and no
     duplicate dispatch or edit behavior.

4. **Controlled Codex Execution**
   - Add disposable worktree isolation and the scoped headless Codex adapter.
   - Capture evidence and run deterministic post-run diff, path, budget, and
   command checks.

   - Add the local-LLM adapter contract and policy/benchmark fixtures, but keep
     code-edit availability disabled until its representative benchmark passes.

5. **Review And Remediation**
   - Orchestrate independent reviewer and verifier roles.
   - Enforce finding-bound remediation, two-cycle limits, and escalation.

6. **Adoption And Hardening**
   - Add operator UX, metrics, redacted audit export, and representative smoke
     runs.
   - Implement local commit and separately authorized push/PR adapters only
     after their contract and security tests pass.
   - Enable a local-LLM code-edit provider only after its benchmark and policy
     gate pass; add other provider-specific adapters after headless Codex is
     stable.

## Progress

- Phase 1 is complete: the approved PRD, state machine, adapter contract,
  repository threat model, and synthetic fixture catalog are documented.
- Phase 2 is complete: versioned implementation runs, private task bundles,
  adapter policy configuration, lifecycle transitions, and dry-run preflight
  are implemented with deterministic tests.
- Phase 3 is complete: the non-editing fake adapter supports launch, status,
  result, cancellation, bounded retries, and crash-recovery reconciliation.
- Phase 4 is complete: explicit approval and adapter policy gate controlled
  execution; detached worktrees, persisted Codex session/PID evidence,
  conservative reconciliation, and deterministic scope verification are
  implemented.
- Phase 5 is complete: policy-gated independent reviewer and verifier
  assessments, owner-only result records, bounded review/verification cycles,
  and finding-bound remediation are implemented.
- Phase 6 is in progress: operator UX, audit export, smoke coverage, local
  commit gates, and separately authorized remote publication are next.

## Cross-Phase Gates

- Do not dispatch an agent without fresh source-drift and policy checks.
- Do not permit a local commit without a passing verification gate and explicit
  work-item support.
- Do not push or create a PR without a valid, run-specific remote-publish
  authorization bound to the exact SHA and branch.
- Do not promote a phase without its fixtures, deterministic tests, and
  documented evidence.
