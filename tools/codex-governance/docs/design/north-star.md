# Proposed Governance Redesign

## Decision Summary

Replace the repository-hosted implementation-packet workflow with a
ticket-backed workflow. Jira is the human-facing system of record for work;
Git, pull requests, and CI remain the record of implementation and validation;
ADRs remain versioned with the code. Do not duplicate the same plan, handoff,
or evidence in all three systems.

This document is the approved north-star design. The canonical spec and staged
prompts implement it; later changes must remain aligned with it.

## Sources Of Truth

| Concern | Source of truth |
| --- | --- |
| Product goal and acceptance criteria | Jira story |
| Scoped technical work | Jira implementation subtask |
| Code and exact commits | Git and pull request |
| Test, lint, and build results | CI artifacts |
| Durable technical decisions | `docs/decisions/` ADRs |

Each Jira story may contain multiple implementation subtasks. A subtask is the
normal replacement for an implementation packet. It must include scope,
non-goals, technical acceptance criteria, validation plan, change class, ADR
link or `No ADR needed` rationale, PR link, and concise handoff state.

The initial workflow reads Jira only. Any Jira transition, comment, or field
update requires explicit approval.

## Work-Item Authority And Stop Behavior

An approved Jira implementation subtask is captured in a versioned normalized
work item that is the executable authority for one run. It binds ticket
identity/revision/digests, the source baseline and mode, allowed paths,
acceptance criteria, required commands, review budget, ADR references,
permitted provider/task class, and local-commit and remote-publication
permissions. Jira remains the human-readable record of intent.

The control plane stops on a binding mismatch, missing or expired source
evidence, out-of-scope or unexcepted over-budget diff, required-check failure,
unavailable or unqualified provider, or incomplete audit evidence. It does not
unblock that snapshot in place. A human resumes work only through a new signed,
versioned rebaseline or exception record naming the prior run, changed fields,
rationale, approver, expiry, and revised bounds.

The Jira owner decides intent; the technical owner decides architecture or scope
exceptions; and the repository owner authorizes local-commit policy and remote
publication. The control plane verifies explicit records and never infers an
approval.

Rebaselines, exceptions, remote-publish authorizations, and offline exports use
one common signed envelope: canonical JSON payload, SHA-256 payload digest,
`key_id`, `algorithm`, signer role, issuance time, optional expiry, and
signature. The initial algorithm is Ed25519 over the payload digest. A versioned
repository policy maps trusted public keys to permitted roles and verifies key
trust, role, expiry, signature, record version, and revocation at each gate.
Private keys never enter the repository or runtime ledger; tests use ephemeral
fixture keys only.

## Ticket Alignment And Drift

At work-item creation, the work item declares either `live-jira` or
`offline-export` as its source mode. Record the story/subtask key, URL, capture
time, Jira revision or update timestamp, and digests of the description and
acceptance criteria. At implementation, review, and closure gates,
`live-jira` validates a fresh read, while `offline-export` validates the
snapshot's declared age and provenance against policy.

If tracked ticket content changes, mark the work item `source-drift-blocked`.
An agent may identify changed fields but cannot decide semantic alignment.
A human must rebaseline the work, split it, approve an explicit exception, or
stop it. The validator checks facts and field changes; it does not claim to
prove that prose still has the same intent.

An `offline-export` for a governed implementation run is a signed envelope
from a configured trusted issuer key. It binds the ticket URL/key, capture time,
Jira revision/update time, content digests, and export digest. Unsigned exports
are test fixtures only. The initial maximum age is 24 hours, configurable per
repository. A versioned trusted-key registry supports rotation and revocation;
revoking an issuer immediately invalidates its exports. An unacceptable age or
provenance fails closed.

## Pull Request Reviewability

One PR has one primary Jira implementation subtask by default. A story may
span many PRs, but a PR may not silently span multiple subtasks or phases.

The subtask defines allowed paths, phase, change class, and a review budget for
changed files, lines, and components. CI compares those rules with the PR diff.
Generated files and lockfiles are classified separately and cannot hide
unrelated changes.

Changes that genuinely cannot be split require an approved Jira review
exception with a reason, review plan, named approver, and rollback or
containment plan when high risk. Integration subtasks are allowed only to wire
already-reviewed work together or validate an end-to-end flow.

Begin with warning-mode metrics, then calibrate and enforce review budgets per
repository. Do not impose universal line-count thresholds before observing
real review patterns.

## Repository Artifacts And Handoffs

Do not generate a parallel packet/evidence/handoff directory for every work
item. Remove these as default new-work artifacts:

```text
implementation-packets/
verification/evidence/
.codex/summaries/
```

Keep `docs/decisions/` for ADRs and add a small `governance.yml` for the Jira
integration, issue-key pattern, required issue sections, CI conventions, and
format version. Existing repository packets remain legacy records; do not
delete them automatically.

Use Jira for concise handoffs: current status, PR link, latest CI run or
validated commit, completed work, blocker, and exact next action. Keep raw
logs, command output, detailed findings, and test reports in PR/CI systems.
Do not persist local worktree paths; they are machine-local state. Jira
assignee, branch naming, and PR linkage coordinate work across people.

## ADR Policy

Create an ADR before implementation when work changes a durable architectural,
interface, security, data, operational, dependency, deployment, rollback, or
accepted-risk decision. A packet/subtask must otherwise state why no ADR is
needed. Routine bug fixes, localized refactors, tests, and documentation do not
need ADRs when they follow an existing decision.

## Agent Roles And Disagreements

Ticket planning is orchestrated by the Go application: a manager prepares a
structured plan from approved product sources, while fresh reviewer and verifier
agents independently assess the plan. The application validates source digests,
persists role evidence, and only marks the plan approved after both roles pass.
Jira publication remains a separate explicitly approved action.

Define manager, ticket analyst, implementer, reviewer, verifier, and
remediation editor roles. Each directive specifies inputs, permitted actions,
scope, expected structured output, terminal state, and closure criteria.

Implementation orchestration is adapter-first, with headless Codex as the
initial adapter. Users may select a local LLM for code execution or remediation
only after its policy entry and reproducible evaluation record approve that
role and task type. Code-edit qualification binds the exact provider, model and
pinned ID, adapter, tool-permission/config version, prompt/task-bundle schema
and version, benchmark corpus and harness versions/hashes, role, task class,
metrics, thresholds, evaluator identity, and approval. Qualification is
separate for each role and task class. Initial code-edit eligibility is limited
to `scoped-code-edit` and `finding-bound-remediation`; high-risk work remains
ineligible pending a separate policy decision.

The versioned corpus includes normal tasks and adversarial cases for forbidden
paths, source drift, invalid output, validation failure, authorization attempts,
and sensitive-data redaction. All safety-control cases must pass. Before any
provider is enabled, a non-safety task-success threshold is derived from a
versioned baseline corpus and recorded in policy. Requalify when any bound input
changes, after a relevant incident, and on the defined recurring review date. A
passing evaluation is scoped evidence rather than a general safety guarantee.
One approved implementation subtask runs in one dedicated disposable Git
worktree. A versioned task bundle supplies the normalized work item, fresh
ticket baseline, allowed paths, commands, ADRs, and repository guidance. The
governance application owns preflight, lifecycle evidence, reconciliation, and
deterministic post-run checks.

Local commits are permitted only when enabled by the approved work item, the
exact staged diff has passing assessments from independent reviewer and
verifier executors, and pre-commit gates pass. The same diff-bound evidence is
required before pushing the exact resulting commit to a non-protected
branch and creating its PR require a separate run-specific human authorization.
It binds the work item and run ID, repository identity, remote name and URL
fingerprint, target ref, exact commit and expected base SHAs, approver identity,
issuance time, expiry, and record version/digest. A single authorization may
list both `push` and `create-pr`; each remains an explicit independently
checked and consumed operation, and `create-pr` binds its target branch. It
never permits force-pushes, protected/default-branch writes, merges, releases,
tags, Jira writes, or unrelated remote actions. Changed remotes, refs, or SHAs,
and expired, reused, or broadened records fail closed.

The governance application enforces this workflow only where it controls the
operation. It cannot prevent edits, commits, or publication by a user or a
separate tool outside the governed path. Its evidence records demonstrate the
governed actions it observed; they do not prove that bypasses never occurred.

For testing, each run has an owner-only, structured, hash-linked event ledger
outside the repository. Events record run/event IDs, UTC time, actor and role,
CLI/schema version, state, relevant policy/evaluation/task-bundle/source
digests, gate decision, and preceding-event hash. The ledger records redacted
references plus digests and normalized outcomes for source checks, commands,
diff/commit, adapter results, findings, exceptions, and authorization.

Closure writes a final evidence manifest containing the terminal event hash.
Missing, altered, or unverifiable required evidence escalates the run and
prevents closure. This is tamper-evident within the governed local runtime, not
tamper-proof against a party with filesystem control. Retain test evidence for
the full test program; define production retention separately before
distribution.

The manager coordinates but cannot override policy checks, ticket drift,
required CI failures, or human decision rights. It must persist/link each
agent result, verify completion criteria, close the agent immediately, and
block finalization when an agent remains open without an approved exception.

Resolve disagreements by evidence rather than agent voting:

1. Rerun deterministic checks for factual disputes.
2. Use one fresh independent reviewer for review disputes.
3. Escalate acceptance-criteria disputes to the Jira owner.
4. Require an ADR and technical-owner decision for durable design disputes.
5. Stop the work item if unresolved after adjudication.

Reviewer and verifier loops are bounded. Findings use `blocking`, `important`,
`minor`, or `informational` severity. A maximum number of cycles, escalation
rules, and accepted-caveat requirements prevent repeated agent churn.

## Model Policy And Local Ollama Use

The toolkit implementation is a Go CLI. A `Makefile` may provide development
shortcuts but does not implement policy or validation logic.

Routine governance should be deterministic tooling, not model calls. Model
selection is policy-controlled: agents request a capability tier, and the
execution layer resolves it to an allowlisted model.

- `local-small`: bounded extraction and summaries using approved Gemma models.
- `local-standard-coder`: scoped implementation or remediation using
  `qwen3-coder:30b` or `devstral:24b` only after evaluation.
- `high-reasoning`: hosted model or human for ticket drift, architecture,
  security, and disputes.

Do not use local models as the sole authority for scope, security, or release
readiness. Run local jobs sequentially by default to avoid resource pressure.
Pin allowed models by name, version, and Ollama model ID; do not allow `latest`,
automatic downloads, arbitrary endpoints, or silent model escalation.

## Governed Ollama Gateway

Introduce a local execution boundary between agents and Ollama. Agents submit
atomic jobs; they do not call Ollama directly. The gateway validates role,
model tier, task type, scope, timeout, call limit, input bundle, and output
schema before invoking Ollama.

For code edits, the gateway accepts a patch, validates its paths, applies it in
a disposable worktree, and runs deterministic checks. Local models receive no
direct Jira, Git push, cloud, secret, deployment, or arbitrary shell access.

Start with a CLI wrapper and local audit store; add a queueing service only if
multiple agents need concurrent local-model jobs. Token and dollar limits are
external-runtime features, not prompt guarantees. The first implementation
enforces observable limits: attempts, timeouts, tool calls, concurrent jobs,
and change size.

## Cache Policy

Use a private, content-addressed cache outside repositories for ticket
baselines, exact Git diff summaries, exact-commit validation results, and
bounded local-model summaries. Keys include all relevant inputs: ticket
revision/digests, commit SHAs, command, policy version, toolchain identity, and
model/prompt version.

At each governance gate, use the work item's declared source mode for drift
detection. A `live-jira` mode requires a fresh Jira read. An `offline-export`
mode fails closed unless policy accepts its declared age and provenance. Cached
data may reduce duplicate work within a job but cannot replace either required
validation. Do not reuse cached reviews, approvals, ADR decisions, or release
conclusions.

## Phased Implementation

1. Approve the ticket-backed authority model and migrate the canonical spec,
   README, prompts, and templates. Preserve legacy packet artifacts.
2. Add `governance.yml`, a normalized Jira work-item contract, and fixture
   exports for offline tests.
3. Implement a read-only Jira adapter and validator for required fields,
   parent/subtask links, ticket drift, PR linkage, scope-to-diff, and review
   exceptions.
4. Add CI checks and branch protection guidance. Start review budgets in
   warning mode and calibrate them before blocking merges.
5. Add explicit agent directives, bounded closure, disagreement handling, and
   ADR decision rules.
6. Implement the governed local Ollama CLI, benchmark approved local models,
   then add caching and optional queueing.

No stage may write to Jira, push, merge, deploy, or access secrets without
separate explicit approval.

## Release And Synchronization

The design repository releases versioned manifests to implementation
repositories. Synchronization is one-way and reviewable: upstream requirements
and schema contracts are versioned; downstream scripts and adapters are
implementation-owned; merged files are never overwritten. See
`release-sync-contract.md` for the required manifest, dry-run, compatibility,
and migration behavior.

## Distribution Readiness

Do not distribute the implementation-agent control plane until all of the
following are complete or an accountable owner records a time-bounded,
published exception:

- Audit evidence has defined integrity, approver-identity binding, retention,
  tamper detection, and tamper-response behavior.
- Tests cover bypass and failure paths, including changed remotes or refs,
  rebases, stale tickets, altered task bundles, and expired or reused remote
  authorizations.
- Versioned policy and evaluation registries support provider revocation and
  incident response; code-edit eligibility is checked at preflight.
- Product and release language describes the enforcement boundary and avoids
  claims that the CLI cannot substantiate.
- Supported environments, compatibility and upgrade expectations, privacy and
  local-data handling, and evidence-retention commitments are documented.
