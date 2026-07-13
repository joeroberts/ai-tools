# Canonical Spec: codex-governance

## Purpose And Precedence

`codex-governance` is a reusable toolkit for Jira-backed, governed
Codex-assisted engineering work. It prevents scope, ticket, review, and
validation drift without duplicating the same records across Jira, Git, and CI.

Requirements take precedence in this order: explicit user instruction,
canonical spec, approved stage plan, master prompt, stage prompt, generated
template. Lower-precedence instructions must not widen scope.

The master prompt initializes a session and requests approval. Stage 1 creates
`docs/governance-toolkit-spec.md` exactly once after approval. Later changes to
that generated spec require an approved scope change.

## Authority Model

| Concern | Source of truth |
| --- | --- |
| Product goal and acceptance criteria | Jira story |
| Scoped technical work | Jira implementation subtask |
| Code, commits, and changed files | Git and pull request |
| Tests, lint, build, and runtime checks | CI artifacts |
| Durable technical decisions | ADRs under `docs/decisions/` |

The initial workflow reads Jira only. Updating Jira, pushing, merging,
deploying, publishing, installing globally, tagging, releasing, applying
Terraform, mutating cloud resources, accessing secrets, or destructive
filesystem operations requires separate explicit approval.

## Initial Response And Autonomy

Before creating files, summarize the intended structure, propose Stage 1,
explain the authority model and hard stops, state assumptions and risks, and
request approval. After a stage is approved, work autonomously within its
approved scope. Do not repeatedly ask approval for routine edits, safe local
validation, or subagent dispatch that the stage explicitly authorizes.

Local commits require explicit approval in the stage plan. A hard stop is also
required for scope, public-interface, architecture, security, dependency, or
release changes outside the approved stage.

## Jira Work-Item Model

A Jira story owns the business intent. Each independently reviewable technical
slice is a Jira implementation subtask. A subtask must contain:

- scope, non-goals, and technical acceptance criteria;
- validation plan and change class: `trivial`, `standard`, or `high-risk`;
- phase and allowed repository paths or modules;
- review budget for changed files, lines, and components;
- ADR link or `No ADR needed` rationale;
- pull-request link when available;
- concise handoff: status, latest CI run or validated commit, completed work,
  blocker, and exact next action.

Use a stable description template rather than requiring Jira custom fields in
the first release. The generated toolkit must define the template and a
normalized JSON work-item contract in `schemas/jira-work-item.schema.json`.

### Work-Item Authority And Stop Contract

One approved Jira implementation subtask, captured in a versioned normalized
work item, is the executable authority for a run. Jira remains the
human-readable record of intent. The work item binds the ticket
identity/revision/digests, source mode and baseline, allowed paths, acceptance
criteria, required commands, review budget, ADR references, permitted provider
and task class, local-commit permission, and remote-publication permission.

The control plane stops rather than continuing when a binding field differs;
source evidence is missing or expired; the diff is out of scope; a review
budget is exceeded without an approved exception; a required check fails; a
provider is unavailable or unqualified; or required audit evidence is
incomplete. A stopped run is not unblocked in place. Resumption requires a new
signed, versioned rebaseline or exception record that identifies the prior run,
changed fields, rationale, approver, expiry, and revised bounds.

The Jira owner resolves intent. The technical owner resolves architecture or
scope exceptions. The repository owner authorizes local-commit policy and
remote publication. The control plane verifies these records but never infers
an approval.

All signed governance records use one common envelope: canonical JSON payload,
SHA-256 payload digest, `key_id`, `algorithm`, signer role, issuance time,
optional expiry, and signature. The initial algorithm is Ed25519 over the
payload digest. A versioned repository policy maps trusted public keys to the
Jira-owner, technical-owner, repository-owner, and export-issuer roles. At each
gate, the CLI verifies key trust, permitted role, expiry, signature, record
version, and revocation. Private keys never enter the repository or runtime
ledger; tests use ephemeral fixture keys only.

### Ticket Alignment And Drift

At work-item creation, the work item must declare `live-jira` or
`offline-export` as its source mode. Capture the story and subtask key, URL,
capture time, revision or update timestamp, and digests of the description and
acceptance criteria. At implementation, review, and closure gates,
`live-jira` performs a new read; `offline-export` validates its declared
snapshot age and provenance against policy.

If tracked content changes, set the work item to `source-drift-blocked`. An
agent may show changed fields but cannot decide whether the intent still aligns.
A human must rebaseline the work, split it, approve an explicit exception, or
stop it. The validator checks field presence and textual changes; it must not
claim to prove semantic alignment.

An offline export is eligible only when it carries the declared source identity,
capture time, revision or update timestamp, content digests, and provenance
required by policy. For a governed implementation run, it must be a signed
export envelope from a configured trusted issuer key, binding the ticket URL and
key, capture time, Jira revision/update time, content digests, and export
digest. Unsigned exports are permitted only as explicitly marked test fixtures.
The initial maximum offline age is 24 hours and is policy-configurable per
repository. A versioned trusted-key registry controls issuer rotation and
revocation; a revoked issuer immediately invalidates its exports. An export
fails closed when its age or provenance is unacceptable. CI uses `live-jira`
and a fresh read-only query when credentials and network access are configured;
it may use `offline-export` only when that mode is explicitly declared and
policy accepts the supplied snapshot.

## Pull-Request Reviewability

One pull request has one primary Jira implementation subtask by default. A
story may span many pull requests, but a pull request may not silently span
multiple subtasks or phases.

The validator compares the pull-request diff with the subtask's allowed paths,
phase, and review budget. Generated files and lockfiles are classified
separately and cannot hide unrelated changes. Review budgets begin in warning
mode, are calibrated from real repository metrics, then become CI-enforced.

An unsplittable change needs an approved Jira review exception with a reason,
named approver, logical review plan, and rollback or containment plan for
high-risk work. An integration subtask may only wire already-reviewed work or
validate an end-to-end flow; it is not a loophole for unrelated implementation.

## ADR And Handoff Policy

Create an ADR before implementation for a durable architecture, interface,
security, data, operational, dependency, deployment, rollback, or
accepted-risk decision. Routine bug fixes, localized refactors, tests, and
documentation that follow an existing decision do not need one. Every subtask
records the ADR link or the reason it is not needed.

Jira holds concise human handoffs. Stages prepare handoff text but do not post
it; posting to Jira requires separate explicit approval. Detailed command
output, raw logs, test reports, reviewer findings, and CI evidence remain in
PR and CI systems and are linked from Jira. Do not persist local worktree
paths: they are machine-local state. Jira assignee, branch naming, and PR
linkage coordinate work across people.

## Repository Structure

The following are the only per-work-item governance artifacts in a new
repository; the toolkit itself also contains its shared schemas, templates,
profiles, scripts, and references:

```text
governance.yml
docs/
  decisions/
```

The initializer must not create `implementation-packets/`,
`verification/evidence/`, or `.codex/summaries/` for new work. Existing copies
of those directories are legacy records and must not be deleted automatically.

`governance.yml` declares format version, Jira project and issue-key pattern,
required issue sections, selected profile, review-budget policy, CI conventions,
and documentation-root override. It must contain no credentials.

## Templates And References

Create templates for:

- `AGENTS.md`: authority model, approval boundaries, response format, security,
  testing, and agent lifecycle rules;
- `PROJECT_CONTEXT.md`: repository purpose, modules, commands, profile,
  generated artifacts, CI conventions, and non-goals;
- `jira-story.md` and `jira-subtask.md`: required Jira description templates;
- `review-exception.md`: reason, approver, review plan, and containment;
- `adr.md`: status, context, options, decision, consequences, validation, and
  follow-up;
- `future-profiles.md`: deferred profile requirements and promotion criteria.

Create references for the manager, ticket analyst, implementer, reviewer,
verifier, and remediation editor. Each role must state purpose, allowed actions,
inputs, expected structured output, terminal states, escalation conditions, and
closure criteria.

## Agent Governance

### Ticket Plan Orchestration

The Go application owns ticket-plan orchestration. Given approved product
sources, it dispatches a hosted Codex manager to create a structured draft
plan, then dispatches independent policy-governed local Ollama reviewer and
verifier roles against that plan. The application validates source digests and
plan structure after each agent result and records every lifecycle transition
in the runtime ledger. Agent approval promotes a plan only to
`ready-for-approval`; a stakeholder must explicitly approve it before Jira
creation. Agents do not write Jira.

### Implementation-Agent Orchestration

The Go application is the governance control plane for implementation of one
approved Jira implementation subtask. It uses an adapter-first execution model;
headless Codex is the initial adapter, and a policy-approved local LLM may be
selected for code execution or remediation after its benchmark gate passes. The
application performs deterministic preflight, builds a versioned task bundle,
dispatches and reconciles the agent, and records lifecycle evidence. It does
not delegate policy decisions to the implementation agent.

Each run uses a dedicated disposable Git worktree and may change only the
primary subtask's approved paths and review budget. Its task bundle contains
the normalized work item, fresh ticket baseline, allowed paths, required
commands, relevant ADRs, and repository guidance. The agent cannot change
ticket intent, work-item approval state, or acceptance criteria.

The lifecycle is `preflight` -> `queued` -> `running` ->
`implementation-complete` -> `review` -> `verification` -> `remediation` or
`escalated` -> `ready-to-commit` -> `locally-committed` ->
`ready-for-remote-approval` -> `pushed` -> `PR-created` -> `closed`.

`escalated` is terminal until a human supplies a new approved action. The
review and verification loop is limited to two normal cycles. Remediation must
name the finding IDs it addresses and remain within approved paths; a third
cycle requires explicit human approval and unused policy budget.

The application records immutable local result references, commit and diff
SHAs, command outcomes, adapter task IDs, and redacted summaries. After a host
restart it reconciles an in-flight task and never silently re-dispatches it or
duplicates edits. Open agents block closure unless an approved exception is
recorded.

Provider selection is a user preference constrained by policy, not an authority
grant. Code-edit eligibility additionally requires a reproducible passing
evaluation record for the exact provider, model and pinned ID, adapter,
tool-permission/config version, prompt/task-bundle schema and version,
benchmark-corpus and harness versions/hashes, role, task class, metrics,
thresholds, evaluator identity, and approval. Qualification is separate for
each role and task class. Initial code-edit eligibility is limited to
`scoped-code-edit` and `finding-bound-remediation`; high-risk task classes are
ineligible pending a separate policy decision.

The corpus must include normal tasks and adversarial cases for forbidden paths,
source drift, invalid output, validation failure, authorization attempts, and
sensitive-data redaction. Every safety-control case must pass. Before any
provider is enabled, the policy must record a non-safety task-success threshold
derived from a versioned baseline corpus; it must not rely on an unexamined
arbitrary percentage. The policy and evaluation registry must be versioned;
changing any qualified input, a relevant incident, or the recurring review date
requires requalification. A passing evaluation is scoped evidence, not proof
of safety for every repository or task class. The policy must allow the
selected provider, model name and pinned ID, role, task type, task-bundle size,
and concurrency. A local LLM uses the governed gateway and receives no direct
Jira, Git push, cloud, secret, or arbitrary shell access. If the selected
provider is unavailable, revoked, or not approved for code edits, preflight
fails without falling back or escalating models.

Local commits are allowed only when the approved work item enables them and
all required pre-commit gates pass. Push and pull-request creation require a
separate, run-specific remote-publish authorization binding the work-item key
and run ID, repository identity, remote name and URL fingerprint, target ref,
exact commit and expected base SHAs, approver identity, issuance time, expiry,
and authorization-record version and digest. A single authorization may list
both `push` and `create-pr`, but they are separate explicit operations and each
is independently checked and consumed. `create-pr` also binds its target
branch. The authorization cannot permit force-pushes, protected or default
branch writes, merges, tags, releases, Jira writes, or unrelated repository
actions. The CLI rejects expired, reused, mismatched, or broadened records.

The manager coordinates work but cannot override policy checks, source drift,
required CI failures, or human decision rights. A local runtime execution ledger
outside the repository is owner-only and hash-linked. Each structured event
records run and event IDs, UTC time, actor and role, CLI/schema version,
lifecycle state, relevant policy/evaluation/task-bundle/source digests, gate
decision, and the preceding event hash. It records digests and normalized
outcomes for source checks, commands, diff and commit, provider/adapter results,
review findings, rebaseline/exception records, and publication authorization.

Store redacted references rather than credentials, raw prompts, or unrestricted
logs. Closure writes a final evidence manifest containing the terminal event
hash. Missing, altered, or unverifiable required evidence escalates the run and
blocks closure. This ledger is tamper-evident within the governed local runtime;
it is not tamper-proof against a party with filesystem control. Test evidence
is retained for the full test program. Production retention is a separate
explicit distribution decision. Before finalization, the manager must verify
the ledger, persist or link every agent result, and close every completed agent.
Open agents block finalization unless a documented approved exception exists.

Roles:

- **Ticket analyst** reads Jira or an offline export, validates required fields,
  and reports drift. It is read-only.
- **Implementer** changes code only for the primary subtask and approved paths.
- **Reviewer** independently assesses the diff against ticket scope, design,
  security, and reviewability. It is read-only.
- **Verifier** independently runs or evaluates required validation and CI
  evidence. It is read-only except safe local checks.
- **Remediation editor** fixes specific in-scope findings and cannot change
  ticket intent or acceptance criteria.

Findings use `blocking`, `important`, `minor`, or `informational` severity.
Only `blocking` findings stop verification. An `important` finding requires a
fix or an approved accepted-risk record. Use at most two normal review or
verification cycles; a third requires unused policy budget and explicit human
approval. Exhausted or unresolved work is escalated, not retried indefinitely.

Resolve disagreements by evidence: rerun deterministic checks for factual
disputes; obtain one fresh independent reviewer for review disputes; escalate
acceptance-criteria disputes to the Jira owner; require an ADR and technical
owner decision for durable design disputes. Stop unresolved work.

### Enforcement And Evidence Boundaries

The CLI enforces only the governed workflow it controls. It cannot prevent a
user or another tool from editing, committing, or publishing outside that
workflow. Its audit record therefore documents observed, policy-governed
actions rather than proving that no bypass occurred. Distribution claims must
state this boundary plainly.

Before distribution, define and test audit-record integrity, approver identity
binding, evidence retention, tamper detection and response, authorization
revocation, incident handling, and policy/evaluation-registry versioning. Test
failure paths for changed remotes or refs, rebases, stale tickets, altered task
bundles, expired and reused authorizations, and unavailable or revoked
providers. Define supported environments, compatibility and upgrade policy,
privacy and local-data handling, and evidence-retention expectations before
making availability or safety claims.

## Validation And CI

Create:

- a Go module with `cmd/codex-governance/` and `internal/` packages;
- `codex-governance init` and `codex-governance validate-work-item` commands;
- a read-only Jira adapter with offline-export support;
- fixture-based smoke tests.

Use Go for the CLI, validator, Jira client, synchronization logic, cache, and
Ollama gateway. A `Makefile` may provide `make build`, `make test`, and
`make lint` shortcuts, but it must not implement governance behavior.

The validator must check the normalized work-item contract, required story and
subtask fields, parent/subtask relationship, ticket drift, PR linkage,
scope-to-diff using explicit base and head SHAs, phase, review budget, approved
exceptions, ADR rationale, and required CI evidence. It must emit deterministic
exit codes and support `--warn`, `--strict`, `--work-item`, `--repo-root`,
`--offline-export`, `--base-sha`, and `--head-sha`.

Stage 3 must create CI-check and branch-protection guidance. CI integration is
optional per adopting repository, but when configured it performs the fresh
Jira read, validates the current PR against its primary subtask, and can become
a required branch-protection check after review budgets are calibrated. The
toolkit must never write Jira from validation.

Fixtures must cover valid work, missing required ticket content, changed
acceptance criteria, invalid parent/subtask links, missing PR link,
scope-to-diff violation, over-budget change, approved review exception,
missing ADR rationale, unavailable Jira with offline export, and unsupported
profile.

## Profiles

`generic` is the only supported profile initially. It provides ticket-backed
governance and repository-neutral validation. `terraform-module`,
`node-fullstack-k8s`, and `python-vue-fullstack-k8s` remain documented future
profiles. Each becomes supported only after its discovery rules, commands,
fixtures, examples, smoke checks, and promotion criteria are implemented.

Supported profiles declare commands as data: command ID, command, required or
optional class, prerequisites, timeout, network/credential/Docker requirements,
and permitted skip reasons. Profiles add validation; they never replace the
Jira, PR, ADR, or review model.

## Release And Synchronization

The source design repository releases versioned manifests to downstream toolkit
implementations. Synchronization is one-way: upstream owns requirements,
schemas, template contracts, format versions, and migration rules; downstream
owns scripts, adapters, tests, CI examples, and local configuration. Merged
files must not be overwritten.

The generated toolkit must support `governance-sync --dry-run --release <id>`
and `governance-sync --check`. Dry-run reports required changes without writing
files or mutating Jira, Git, or CI. A future apply mode requires an approved
Jira migration subtask, preserves legacy packet artifacts, and produces a
reviewable change set. See `release-sync-contract.md` for the complete contract.

## Model Policy, Gateway, And Cache

Routine governance is deterministic tooling, not an LLM call. Agents request
a capability tier; a policy-controlled execution layer resolves it to an
allowlisted model. The initial policy supports local-small extraction and
summaries with approved Gemma models, and local-standard scoped code work with
approved `qwen3-coder:30b` or `devstral:24b` only after evaluation.

Local model jobs run through a governed Ollama CLI gateway. Agents submit an
atomic job; they do not call Ollama directly. The gateway validates role, task
type, model ID, scope, input bundle, output schema, attempts, timeout, and
change size. Models receive no direct Jira, Git push, cloud, secret, deployment,
or arbitrary shell access. Code edits are returned as patches, staged in a
disposable worktree, then checked deterministically.

The platform owner maintains the trusted gateway policy at
`~/.codex-governance-runtime/policy.yaml`, with owner-only permissions. It maps
roles and task types to allowlisted model names, versions, and Ollama IDs. The
gateway compares the installed Ollama inventory to that policy before each job.
Disallow `latest`, automatic downloads, arbitrary endpoints, silent model
escalation, and default parallel local jobs. Hosted high-reasoning models or
humans handle architecture, security, semantic ticket drift, and disputes.

Prompt assets cannot enforce provider token or dollar limits. The initial
runtime enforces observable limits: attempts, timeouts, tool calls, concurrent
jobs, and change size. Token and dollar accounting are optional gateway
telemetry when the provider exposes trustworthy usage data.

Use a private content-addressed cache under `~/.codex-governance-runtime/cache`
with owner-only permissions for ticket baselines, exact Git diff summaries,
exact-commit validation, and bounded local-model summaries. Keys include ticket
revisions/digests, commit SHAs, commands, policy and toolchain versions, and
model/prompt versions. Store digests and redacted summaries by default, never
credentials, raw prompts, or unrestricted logs. Expire entries after 30 days by
default and provide a local clear command. A cache never replaces a fresh Jira
read at a gate and never reuses reviews, approvals, ADRs, or release conclusions.

## Definition Of Done

The toolkit is complete when it creates the Jira-backed configuration and
templates, validates normalized Jira work items and offline fixtures, detects
ticket drift and PR scope violations, documents CI enforcement, preserves legacy
packet directories, implements all role directives and closure checks, keeps
deferred profiles documented, and passes Go, JSON, smoke, and diff checks.
No remote mutation, secret access, push, merge, release, deployment, or cloud
operation may occur without explicit approval.
