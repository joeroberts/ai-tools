# Canonical Spec: codex-governance

## Current Task

Create an internal distributable toolkit named `codex-governance`.

The toolkit must make our governed packet workflow repeatable across
repositories. Roadmaps are a required governance gate, not just a template.

Before writing files, present a concrete implementation plan and wait for
approval.

After the plan is approved, proceed autonomously within the approved scope. Do
not ask for repeated approvals for routine file edits, validation commands,
reviewer/verifier subagent dispatch, code-editor subagent fixes, evidence
updates, or local commits if those actions are explicitly included in the
approved plan.

## First Response Requirements

The first response must include:

- Summary of the intended toolkit structure.
- Concrete implementation plan.
- Explanation of the roadmap-first governance flow.
- Explanation of how implementation packets trace to roadmap phases.
- Autonomy envelope: allowed actions, hard stops, commit rules, and
  push/publish restrictions.
- Open questions, assumptions, and risks.
- A request for approval before implementation begins.

Do not create or edit files before approval.

## Autonomy Model

Allowed after plan approval:

- Create and edit files inside the target `codex-governance` toolkit directory.
- Create/update templates, scripts, docs, examples, and skill files.
- Run local validation commands.
- Run shell syntax checks.
- Run ShellCheck if available.
- Dispatch reviewer, verifier, and code-editor subagents.
- Close completed subagents.
- Update evidence files.
- Make small focused local commits only if the approved plan explicitly allows
  commits.

Hard stops requiring explicit approval:

- Running `rm -rf` or destructive filesystem operations.
- Publishing or installing the skill globally for the team.
- Pushing to a remote repository.
- Creating or updating remote PRs.
- Posting PR comments.
- Creating tags or releases.
- Deploying.
- Applying Terraform.
- Mutating cloud resources.
- Changing credentials, secrets, or config outside the repo.
- Adding new external dependencies not listed in the approved plan.
- Modifying files outside the target repo/toolkit directory.
- Changing scope, architecture, security posture, public interfaces, release
  behavior, or dependencies beyond the approved plan.
- Accessing secrets or credentials not already available.
- Making product/design decisions required by failing tests.

Continue autonomously until:

- Reviewer and verifier gates are green.
- A blocker cannot be resolved within the approved scope.
- A hard-stop action is required.
- Tests fail in a way that requires a product/design decision.
- Scope materially changes and the roadmap must be updated or replaced.

## Progress Reporting

- Provide concise progress updates at phase boundaries.
- Do not stop to ask "continue?" after each phase.
- Persist session handoffs and evidence as work progresses.

## Commit Model

Local commits are allowed only if explicitly approved in the plan.

If commits are approved, make small focused local commits such as:

1. Scaffold and base templates.
2. Skill and reference docs.
3. Validation scripts and tests.
4. Examples and README/docs.

Separate governance/evidence workflow commits from code/behavior fix commits.

Do not push unless separately approved.

Pushes, tags, releases, publishing, deployment, Terraform apply, cloud
mutation, and remote PR updates always require separate explicit approval.

## Governance Model

### Roadmap-First Gate

Governed work starts with a roadmap under `.codex/assessments/`.

Do not create implementation packets or edit implementation files until the
roadmap exists and is approved, unless the user explicitly bypasses the roadmap
gate.

Roadmap bypass is allowed only when explicitly requested by the user. Any
bypassed packet must record:

```text
Roadmap bypassed: yes
Bypass reason:
Approving instruction:
```

### Roadmap Purpose

The roadmap is the planning artifact. It must capture:

- Goal.
- Source context.
- Design principles.
- Phased work.
- Open decisions.
- Risks.
- Approval status.

Allowed roadmap statuses:

- `proposed`
- `approved`
- `superseded`
- `closed`

If the work is exploratory, create a roadmap with status `proposed`.

If the user approves the roadmap, update status to `approved`.

Once implementation begins, update packet status, not roadmap status, unless
roadmap scope changes.

Material scope changes require updating the approved roadmap or creating a
replacement roadmap before implementation continues. If a roadmap is replaced,
mark the old roadmap as `superseded` and link to the replacement.

### Optional Active Execution State

Some app repositories need a lightweight current-state layer in addition to
roadmaps and packets. The toolkit must support, but not require,
`execution-state/` with:

- `current-state.md`: active objective, selected packet, constraints, and next
  expected action.
- `backlog.md`: future governed work candidates.
- `escalations.md`: unresolved blockers, external dependencies, and required
  approvals.

This layer is optional because small libraries and Terraform modules may not
need it. When present, Codex must treat it as coordination state and reconcile
it with roadmaps, packets, and evidence before proposing next work.

## Repo Profiles And Validation Contracts

The governance model is shared across repositories, but validation is
profile-specific. The toolkit must provide profile templates and examples
without forking the core workflow.

Required profiles:

- `generic`: minimum governance scaffold for unknown or mixed repositories.
- `terraform-module`: Terraform formatting, initialization without backend,
  validation, tests, static/security scans, test-contract checks, and optional
  release tag validation.
- `node-fullstack-k8s`: npm workspace install/build/typecheck/test, optional
  coverage gates, Prisma generation/migration validation, Helm lint/template,
  GitOps checks, container runtime checks, and background worker/cron/migration
  job coverage.
- `python-vue-fullstack-k8s`: Python lint/test/coverage, schema snapshot
  checks, Vue/Vite coverage and build, Helm lint/template, Compose/local smoke
  checks, and GitOps validation.

Profiles must define:

- Discovery questions for stack, package manager, CI system, deployment model,
  docs path, generated artifacts, and secret/config boundaries.
- Required local validation commands.
- Optional heavyweight validation commands that require Docker, network,
  credentials, live services, or explicit approval.
- Coverage expectations or a documented reason coverage is not enforced.
- Generated artifact handling: what is committed, ignored, regenerated, or
  treated as evidence only.
- Deployment and release boundaries that distinguish local validation from
  remote mutation.
- Background workload expectations for APIs, workers, cron jobs, queue
  consumers, migration jobs, and scheduled importers.

Profiles are additive. They must not replace roadmap approval, packet status,
ADR requirements, evidence storage, review-scope checkpoints, or reviewer and
verifier loops.

## App Workflow Gaps To Cover

The toolkit must explicitly account for app repos where implementation changes
span runtime code, databases, CI, containers, and GitOps. Templates and prompts
must cover:

- Configurable docs roots such as `docs/` or `doc/`; ADR placement should
  default to `docs/decisions/` but support a documented override.
- Database migration gates, including Prisma, Alembic, schema snapshots,
  migration deployment checks, and seed/export artifacts.
- External integration checks for Jira, Bitbucket, GitHub, AWS, Argo CD, Kargo,
  and similar systems. These checks must be opt-in, secret-safe, and normally
  read-only unless explicitly approved.
- Generated assets such as Prisma clients, OpenAPI types, SQLite schema
  snapshots, coverage files, built frontend assets, and local database sidecar
  files.
- Local app smoke checks that start API/frontend processes only when scoped and
  safe, and that clean up child processes.
- Container/image runtime checks that verify files needed by entrypoints,
  workers, cron jobs, and migration jobs are present in runtime images.
- Deployment workflow boundaries for image publishing, GitOps updates,
  Argo/Kargo sync, environment promotion, and database migrations.

## Packet Lifecycle

Implementation packets live under `implementation-packets/`.

Each implementation packet must:

- Link to its source roadmap under `.codex/assessments/`.
- Map packet scope back to roadmap phases.
- Map implementation steps back to roadmap phases.
- Define explicit non-goals.
- Define validation requirements.
- Define evidence requirements.
- Track reviewer and verifier status.
- Reference an ADR, or explicitly record `No ADR needed` with a reason.

Allowed packet statuses:

- `draft`
- `ready-for-implementation`
- `in-implementation`
- `pending-review`
- `review-blocked`
- `pending-verification`
- `verification-blocked`
- `accepted`
- `closed`

Status-driven artifact expectations:

- `draft`: packet may be incomplete.
- `ready-for-implementation`: source roadmap must exist and be approved unless
  bypassed; required sections must be present.
- `in-implementation`: source roadmap must exist and be approved unless
  bypassed; evidence directory must exist.
- `pending-review`: `review-scope.md` and evidence directory must exist.
- `review-blocked`: reviewer artifact and findings must exist.
- `pending-verification`: reviewer artifact must exist and show no blocking
  findings; verification commands must be listed.
- `verification-blocked`: verifier artifact and failure details must exist.
- `accepted`: reviewer and verifier artifacts must exist; acceptance criteria
  must be satisfied or caveats explicitly accepted.
- `closed`: session handoff must exist and final evidence must be linked.

## Canonical Directory Structure

```text
codex-governance/
  README.md
  skills/
    governed-packet-workflow/
      SKILL.md
  templates/
    AGENTS.md
    PROJECT_CONTEXT.md
    implementation-packet.md
    validation-profile.md
    review-scope.md
    discovery-evidence.md
    adr.md
    session-handoff.md
    roadmap.md
  profiles/
    generic.md
    terraform-module.md
    node-fullstack-k8s.md
    python-vue-fullstack-k8s.md
  references/
    manager-loop.md
    reviewer-prompt.md
    verifier-prompt.md
    code-editor-prompt.md
  scripts/
    init-governance.sh
    check-governance-packet.sh
  tests/
    fixtures/
      valid-packet/
      missing-roadmap/
      unapproved-roadmap/
      missing-review-scope/
      missing-adr-reference/
      missing-verifier-artifact/
    smoke-test.sh
  examples/
    terraform-module/
    node-fullstack-k8s/
    python-vue-fullstack-k8s/
      README.md
      AGENTS.md
      PROJECT_CONTEXT.md
      .codex/
        assessments/
        summaries/
      implementation-packets/
      verification/
        evidence/
      docs/
        decisions/
```

## Generated Repo Scaffold

`scripts/init-governance.sh` must initialize a target repo with:

```text
AGENTS.md
PROJECT_CONTEXT.md
.codex/
  assessments/
  summaries/
implementation-packets/
verification/
  evidence/
docs/
  decisions/
```

The script must not overwrite existing files unless `--force` is provided.

If `AGENTS.md` already exists and `--force` is not provided, do not overwrite
it. Instead, create `AGENTS.governance.example.md` or print merge guidance.

Do the same for `PROJECT_CONTEXT.md`: do not overwrite unless `--force` is
provided; create `PROJECT_CONTEXT.governance.example.md` or print merge
guidance.

The script must support profile selection:

- `--profile generic`
- `--profile terraform-module`
- `--profile node-fullstack-k8s`
- `--profile python-vue-fullstack-k8s`

The selected profile should influence generated validation guidance,
`PROJECT_CONTEXT.md` placeholders, packet validation examples, and example
evidence. If no profile is supplied, default to `generic` and print the
available profiles.

## Codex Skill

Create `skills/governed-packet-workflow/SKILL.md`.

Skill name: `governed-packet-workflow`

Purpose: Teach Codex how to run a governed implementation workflow.

The skill must cover:

- Roadmap placement under `.codex/assessments/`.
- Roadmap-first planning gate.
- Implementation packets under `implementation-packets/`.
- Evidence under `verification/evidence/<PACKET>/`.
- ADRs under `docs/decisions/`.
- Session handoffs under `.codex/summaries/`.
- Review-scope checkpoints.
- Reviewer/verifier subagent loops.
- Dispatching code-editor subagents for findings.
- Closing completed subagents.
- Separating governance/evidence commits from code-fix commits.
- Final commit/push behavior only after reviewer and verifier gates are green.
- Push/publish/deploy behavior requiring explicit approval.
- Repo profile selection and profile-specific validation contracts.
- Optional `execution-state/` current-state coordination.
- Configurable documentation root and ADR path overrides.

## Template Requirements

All templates must include instructional placeholder text, not blank sections.

Each section must state:

- What belongs there.
- Whether it is required or optional.
- One concise example where useful.

The toolkit must define exact expected contents and required sections for every
generated file. Do not leave `PROJECT_CONTEXT.md`, `AGENTS.md`, packet
templates, evidence templates, roadmap templates, ADR templates, or folder
contents vague.

### templates/AGENTS.md

Required sections:

- Execution Guardrails
- Output / Feedback Format
- Coding Standards
- Testing Expectations
- Security & Secrets Hygiene
- Commit / PR Format
- Documentation & Governance
- Reviewer / Verifier Workflow
- Subagent Lifecycle Rules

Must include planning/approval rules, secret redaction, required response
format, no destructive commands without approval, repo-standard tooling,
behavior-change tests, ADRs, roadmap location, session handoffs, review
artifacts, subagent closure, and roadmap-first governance unless explicitly
bypassed.

### templates/PROJECT_CONTEXT.md

Required sections:

- Project Purpose
- Architecture / Module Overview
- Tooling & Commands
- Governance Model
- Active Work Streams
- Validation Gates
- Release / Deployment Notes
- Security & Configuration Notes
- Known Constraints / Non-Goals

Must include repo purpose, directories, validation commands, governance artifact
locations, roadmap approval gates, review-scope checkpoints, release/deploy
behavior, and what must never be committed.

Must also include profile-specific placeholders for:

- Repo profile name.
- Docs root, for example `docs/` or `doc/`.
- Package/build tools.
- Test and coverage commands.
- Database migration/schema commands.
- Container, Helm, GitOps, or deployment validation commands.
- Generated artifacts that are intentionally committed or intentionally
  ignored.

### templates/implementation-packet.md

Required sections:

- Objective
- Discovery Snapshot
- Scope
- Non-Goals
- Required Decisions
- Implementation Steps
- Roadmap Phase Mapping
- Required Validation
- Generated Artifacts
- Deployment / Runtime Boundaries
- Evidence Requirements
- Acceptance Criteria
- Reviewer / Verifier Status

Must include packet ID, status, source roadmap link, ADR link or `No ADR
needed`, non-goals, checklist steps, roadmap phase mapping, validation,
evidence path, acceptance criteria, reviewer/verifier placeholders, and roadmap
bypass fields.

For app repos, validation must identify which processes or workloads are in
scope: API, frontend, worker, cron job, queue consumer, migration job, batch
importer, Helm chart, GitOps values, or CI pipeline.

### templates/validation-profile.md

Required sections:

- Profile Name
- Applies To
- Stack Discovery
- Required Validation Commands
- Optional Heavyweight Commands
- Coverage Expectations
- Database / Schema Gates
- Runtime Workloads
- Generated Artifacts
- Deployment Boundaries
- Security / Secret Notes
- Example Packet Validation Block

Must include instructional placeholders and examples for each required profile.
Validation profile content must be reusable in `PROJECT_CONTEXT.md`,
implementation packets, review scope, and verifier prompts.

### templates/review-scope.md

Required sections:

- Scope Snapshot
- Files In Scope
- Expected Review Artifacts
- Required Verification Commands
- Forbidden Commands
- Stale-Scope Mitigation
- Subagent Lifecycle
- Manager Loop Instructions

Must include packet ID, branch, base branch, commit under review, optional PR
URL, files in scope, reviewer/verifier output paths, required commands,
forbidden mutation commands, intended-scope warning, current git comparison,
later-commit inspection, subagent closure, reviewer/verifier loops, and
historical evidence naming:

- latest reviewer artifact: `reviewer.md`
- historical reviewer artifacts: `reviewer-001.md`, `reviewer-002.md`
- latest verifier artifact: `verifier.md`
- historical verifier artifacts: `verifier-001.md`, `verifier-002.md`

### templates/discovery-evidence.md

Required sections:

- Purpose
- Known Inputs
- Discovery Findings
- Validation Evidence
- Security Notes
- Environment Caveats
- Evidence Mismatches
- Open Questions

Must include redaction reminder, command/result format, environment caveats,
and forbidden mutation reminder.

### templates/session-handoff.md

Required sections:

- Summary
- Work Completed
- Decisions
- Tests Run
- Open Questions
- TODO

Must include date, packet/workstream placeholder, and links to roadmap, packet,
evidence, ADRs, and PR if available.

### templates/roadmap.md

Required sections:

- Goal
- Status
- Source Context
- Design Principles
- Phased Work
- Open Decisions
- Risks
- Approval Record
- Scope Change Policy

Must include `.codex/assessments/` location rule, approval status, allowed
statuses, phased checklist, risks, unresolved decisions, packet linkage, and
scope-change policy.

### templates/adr.md

Required sections:

- Status
- Date
- Deciders
- Contributors
- Context
- Decision
- Options Considered
- Consequences
- Validation & Rollout Plan
- Open Questions
- Follow-up Tasks

Must include statuses `Proposed`, `Accepted`, `Rejected`, `Deprecated`, at
least three options when practical, positive consequences, and risk/tradeoff
consequences.

## Reference Prompt Templates

Create:

- `references/manager-loop.md`
- `references/reviewer-prompt.md`
- `references/verifier-prompt.md`
- `references/code-editor-prompt.md`

These references must encode no hard-coded PR dependency, review-scope as
intended scope, git state comparison, later commit inspection, persisted
findings under `verification/evidence/<PACKET>/`, and closure of completed
subagents unless a documented reason exists.

## Reviewer Loop

1. Launch reviewer.
2. Wait for reviewer.
3. Persist reviewer findings.
4. If issues exist, dispatch a code-editor subagent.
5. Persist code-editor changes and evidence.
6. Repeat with a fresh reviewer until no blocking findings remain.
7. Close completed reviewer/code-editor subagents unless a documented reason
   exists.

## Verifier Loop

1. Launch verifier only after reviewer gate is green.
2. Wait for verifier.
3. Persist verifier results.
4. If verification fails, fix the issue or environment within approved scope.
5. Rerun a fresh verifier.
6. Repeat until verifier passes or only accepted caveats remain.
7. Close completed verifier/code-editor subagents unless a documented reason
   exists.

## Scripts

Create:

- `scripts/init-governance.sh`
- `scripts/check-governance-packet.sh`

Scripts must be POSIX shell or Bash with clear shebangs. If using Bash, use
`set -euo pipefail`. Scripts must quote variables, avoid unsafe globbing, print
clear errors, return non-zero on validation failure, pass ShellCheck when
available, and avoid external dependencies beyond standard shell tooling unless
documented.

### scripts/init-governance.sh

Must initialize a repo with standard governance directories/files, copy scaffold
templates, create missing directories, avoid overwriting unless `--force` is
provided, create governance example files or merge guidance for existing
`AGENTS.md`/`PROJECT_CONTEXT.md`, support a target repo path, print a summary,
and fail safely on invalid target paths.

Recommended options:

- `--force`
- `--dry-run`
- `--help`

### scripts/check-governance-packet.sh

Must validate required governance structure.

Required capabilities:

- Verify required files exist.
- Verify required headings exist in each template/generated artifact.
- Verify `AGENTS.md` or `AGENTS.governance.example.md` has required governance
  headings.
- Verify `PROJECT_CONTEXT.md` or `PROJECT_CONTEXT.governance.example.md` has
  required governance headings.
- Verify selected profile exists when a packet or context declares one.
- Verify profile-specific validation sections exist when a known profile is
  declared.
- Verify packet roadmap link or explicit roadmap bypass with reason.
- Verify linked roadmap exists and has required headings.
- Warn or fail if roadmap status is not approved for packets in implementation
  or review status.
- Fail if `review-scope.md` is missing for a packet pending review.
- Fail if reviewer/verifier artifacts are missing when packet status requires
  them.
- Recognize latest and historical reviewer/verifier evidence names.
- Fail if behavior/workflow changes do not reference an ADR or explicitly say
  `No ADR needed`.
- Require structure and sections to be consistent without requiring identical
  repo-specific values.

Recommended options:

- `--strict`
- `--warn`
- `--packet <packet-file>`
- `--repo-root <path>`
- `--profile <profile-name>`
- `--help`

Validation modes:

- `--strict`: fail on all governance violations.
- `--warn`: print warnings for non-critical issues and fail only on critical
  structure errors.

## Test Fixtures And Smoke Tests

Create test fixtures for:

- Valid packet.
- Missing roadmap.
- Unapproved roadmap.
- Missing review scope.
- Missing ADR reference.
- Missing verifier artifact.
- Known profile with missing validation block.
- Unknown profile.

Smoke tests must cover:

- Shell syntax checks.
- `init-governance.sh --dry-run`.
- `init-governance.sh` against a temporary repo.
- `check-governance-packet.sh` against valid and invalid fixtures.
- profile validation for Terraform, Node fullstack, and Python/Vue fullstack
  examples.
- ShellCheck if available.

If ShellCheck is unavailable, record that it was skipped with a reason.

## Examples

Create `examples/terraform-module/`.

The example must show how a Terraform module repo would use the scaffold and
include example `AGENTS.md`, `PROJECT_CONTEXT.md`, roadmap, implementation
packet, evidence directory, ADR if behavior/workflow change is demonstrated,
and README explaining the governed workflow.

Use fake IDs/placeholders such as `EXAMPLE-0001`, `example-app`,
`example-service`, and `example-owner`.

Do not include real credentials, cloud state, backend secrets, provider tokens,
private URLs, real Jira tickets, real repo names, or production identifiers. Do
not include Terraform apply behavior as an allowed automated action.

Create `examples/node-fullstack-k8s/`.

The example must show a Node/NestJS/Vue/Prisma app with:

- npm workspace validation examples.
- TypeScript typecheck, Vitest tests, optional coverage, and build commands.
- Prisma generate and migration validation.
- Helm lint/template and GitOps value checks.
- API, frontend, worker, cron, and migration job scope examples.
- Runtime image validation examples.
- External integration checks documented as opt-in and secret-safe.

Create `examples/python-vue-fullstack-k8s/`.

The example must show a Python FastAPI + Vue/Vite app with:

- Python lint, test, and coverage gates.
- Frontend coverage and build gates.
- Schema snapshot or migration validation.
- Makefile-style command aggregation.
- Optional `execution-state/` current-state files.
- Helm/GitOps validation and local smoke checks.
- Secret-safe handling for Jira, database, cloud, and deployment config.

## Top-Level README

Create `README.md` with:

- Purpose
- Toolkit Contents
- Directory Structure
- Installation / Copy Model
- Roadmap-First Workflow
- Packet Lifecycle
- Reviewer / Verifier Workflow
- Validation Usage
- Repo Profiles
- Example Usage
- Commit / Push Policy
- Security Notes
- Limitations / Non-Goals

Explain how to initialize governance, create a roadmap, use roadmap approval,
map packets to roadmap phases, store evidence, run validation scripts, and
understand actions forbidden without explicit approval.

The README must explain that governance is universal but validation is
profile-specific. It must document the supported profiles, when to use each,
and how to override docs roots or validation commands safely.

## Security And Portability Requirements

The toolkit must be portable and repo-agnostic.

Do not embed:

- Company secrets.
- Real credentials.
- Tokens.
- Private URLs.
- Real issue tracker IDs.
- Real repository names.
- Cloud state.
- Local dumps.
- Production identifiers.

Templates must remind users not to commit secrets, state files, credentials,
local dumps, generated artifacts unless intended, or environment-specific
configuration.

Scripts must avoid mutating remote systems.

## Definition Of Done

The work is complete only when:

- All required templates exist.
- The Codex skill exists.
- Reference prompts exist.
- Scripts exist and are executable.
- Top-level README exists.
- Terraform module, Node fullstack Kubernetes, and Python/Vue fullstack
  Kubernetes examples exist.
- Required profile files exist and define validation contracts.
- Required headings are present in all templates.
- Smoke tests exist.
- Shell syntax checks pass.
- ShellCheck passes if available, or skip is documented.
- `check-governance-packet.sh` validates the valid fixture.
- `check-governance-packet.sh` detects expected invalid fixtures.
- `AGENTS.md` / `AGENTS.governance.example.md` structure is validated.
- `PROJECT_CONTEXT.md` / `PROJECT_CONTEXT.governance.example.md` structure is
  validated.
- Evidence files document commands run and results.
- Reviewer gate is green.
- Verifier gate is green.
- No push, publish, global install, release, deploy, Terraform apply, cloud
  mutation, or PR update has occurred without explicit approval.
