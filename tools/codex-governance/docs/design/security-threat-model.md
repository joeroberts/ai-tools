# Security Threat Model

## Overview

`codex-governance` is a local Go CLI that governs Jira-backed engineering work.
Its primary surfaces are normalized work-item validation, offline Jira-export
parsing, ticket-plan orchestration, local runtime and cache state, release
manifest comparison, and the governed local Ollama gateway. The proposed
implementation-agent extension adds a disposable-worktree execution boundary,
execution-provider adapters, and separately authorized remote publication.

The CLI is developer or operator tooling, not a network service. It does not
host tenant data or authenticate end users. Its security impact is nevertheless
high when it operates in repositories containing source code, controls a Git
worktree, reads ticket exports, or invokes a local model runtime.

## Threat Model, Trust Boundaries, And Assumptions

### Assets And Security Objectives

- Repository integrity: governed work must remain within the approved subtask,
  paths, review budget, and exact Git range.
- Remote integrity: no Jira, Git remote, pull-request, release, deployment, or
  cloud mutation occurs without the required explicit authorization.
- Secret safety: credentials must not enter configuration, task bundles,
  prompts, logs, cache entries, result references, or CI artifacts.
- Evidence integrity: runtime lifecycle records, source digests, validation
  results, and commit/diff SHAs must accurately describe the run.
- Local-system containment: an adapter or model must not escape its worktree,
  access arbitrary files, or gain shell/network authority beyond its policy.

### Trust Boundaries

| Boundary | Trusted side | Untrusted or constrained side |
| --- | --- | --- |
| CLI input | Operator-approved command and local config | CLI flags, paths, work-item JSON, ticket exports, manifests, and fixture content |
| Ticket baseline | Fresh read or validated offline export | Ticket text can change or be malformed and cannot determine its own semantic alignment |
| Repository | Approved base SHA, allowed paths, and disposable worktree | Existing working tree, diff, hooks, and repository guidance can be hostile or stale |
| Execution adapter | Governance control plane and policy | Headless Codex or a local model may produce unsafe edits, misleading output, or fail mid-run |
| Local-model gateway | Owner-only policy and pinned inventory | Model prompts and responses are untrusted; model selection is not an approval grant |
| Remote publication | Run-specific authorization bound to exact SHA and branch | Git remote and PR provider requests must not broaden scope or reuse authorization |
| Runtime store | Owner-only ledger, cache, and result references | Local tampering, unsafe permissions, and stale recovery state |

### Assumptions

- The local operator and the approved `governance.yml` are trusted within their
  stated scope; credentials are supplied out of band when a separately approved
  remote action requires them.
- Jira descriptions, offline exports, repository files, model output, and Git
  metadata are treated as untrusted input at their respective boundaries.
- Deterministic checks can establish structural compliance, but only a human
  can adjudicate semantic ticket drift or accepted risk.
- Hosted Codex and local-model providers are not trusted with remote credentials
  or unconstrained filesystem and shell access.

## Attack Surface, Mitigations, And Attacker Stories

### Primary Surfaces And Controls

- CLI parsers and configuration loaders validate structured inputs and use
  bounded profiles; malformed work items, exports, policy files, and manifests
  must fail closed.
- `internal/validate`, `internal/workitem`, and `internal/gitdiff` enforce
  ticket fields, source drift, ADR rationale, explicit Git ranges, allowed
  paths, and review budgets.
- `internal/runtime` uses an owner-only local ledger/cache. Result summaries
  are redacted; raw credentials and unrestricted logs are excluded.
- `internal/ollama` restricts endpoints to local HTTP, verifies pinned model
  IDs, validates owner-only policy permissions, and currently disables
  `code-edit` tasks.
- The implementation-agent extension must create a dedicated disposable
  worktree, use a versioned minimal task bundle, reconcile agent task IDs after
  restart, and block closure when agents remain open.
- Remote publication is permitted only by a run-specific authorization bound to
  work item, remote fingerprint, branch, exact commit SHA, allowed operation,
  approver, and expiry. Force pushes, protected/default-branch writes, merges,
  releases, tags, and Jira writes remain outside that authorization.

### Attacker Stories

- A malicious or compromised ticket export attempts to broaden scope, suppress
  acceptance criteria, or cause an unsafe path to be changed. Digest, fresh
  baseline, and structural validation should stop the run; semantic ambiguity
  escalates to a human.
- Repository guidance, source code comments, or model output attempt prompt
  injection to exfiltrate secrets, run arbitrary commands, or alter remote
  state. Task bundles must be minimal, adapters constrained, and remote
  credentials withheld.
- A local model returns a patch outside approved paths, a deceptive summary, or
  a patch that passes superficial checks. The control plane stages and checks
  exact diffs, reviewer/verifier evidence, and review budgets independently.
- A stale, forged, or reused publication approval attempts to push a different
  commit or branch. Exact-SHA, remote, branch, operation, expiry, and single-use
  checks must fail closed.
- A host crash creates an unknown in-flight run, allowing duplicate edits or
  orphaned agents. Recovery must query the stored adapter task ID and escalate
  unknown state rather than re-dispatching.

Out of scope are attacks requiring compromise of the operating system account,
the Git hosting provider, or a trusted human approver. Such compromise may
raise downstream impact but is not mitigated solely by this CLI.

## Severity Calibration

### Critical

Critical issues enable unauthorized remote mutation or broad secret compromise:
a bypass of exact-SHA remote-publish authorization that permits arbitrary Git
pushes, a path from model input to credential exfiltration, or arbitrary command
execution outside the governed worktree with developer credentials.

### High

High issues let an agent materially violate repository governance or corrupt
evidence: escaping the approved worktree, bypassing source-drift enforcement,
forging closed-agent state, or applying an unreviewed patch outside allowed
paths. These normally require local execution or an operator-controlled input,
but can compromise source integrity.

### Medium

Medium issues cause incorrect bounded behavior or denial of governed work:
failure to reject a stale ticket export, incomplete redaction of a runtime
summary, retry behavior that leaves a recoverable run blocked, or a policy
parser accepting an unpinned local model. Impact is limited when no remote
authority or secrets are exposed.

### Low

Low issues affect diagnostics, non-sensitive local metadata, or advisory-only
output without changing enforcement: misleading progress text, incomplete
fixture documentation, or a roadmap-rendering inconsistency. They do not alter
Git, Jira, model policy, or evidence integrity.

Repository: sha256:ae851f6c63e9594f2ecc8e585bfd8d23ea30d44c6f0afffb776c90e1d7913107
Version: 28667484a4ec5b9a1ce7a90d98fd9a3246b45ed8
