# Signed Publication and Review Gate Handover

## Status

- Branch: `codex/signed-publication-authorization`
- Base: `9ce4349` (`main` / merged PR #8)
- Working tree: uncommitted and intentionally not published.
- Do not commit, push, create a PR, or create Jira issues until the gates below
  are satisfied.

## Completed Local Work

- Added signed repository-owner publication authorization verification.
- Bound governed push and PR dispatch to repository identity, remote URL
  fingerprint, target ref, base SHA, branch, and commit SHA.
- Added separate, atomic consumption for `push` and `create-pr` operations.
- Added a current-remote target-ref check and explicit GitHub repository/base
  selection for PR creation.
- Added review-evidence validation: two distinct executors, unaltered passing
  assessment artifacts, and an exact diff digest match.
- Added tracked pre-commit and pre-push hooks and configured this checkout's
  `core.hooksPath` to `.githooks`.
- Added manager-only ticket-plan decomposition to break the previous circular
  dependency between decomposition and per-subtask constraints.

## Current Governance State

The completed implementation work has **not** received independently executed
reviewer and verifier assessments. Passing unit tests, roadmap prose, or
historical PR merges do not substitute for those records.

The repository now requires passing reviewer and verifier evidence before a
commit, push, or pull-request creation. The current branch has no such evidence
and must remain uncommitted.

## Jira Planning State

- Jira authentication was verified using environment credentials; do not record
  or print credential values.
- Available Jira project: `REK` (RekonLabs); `governance.yml` now sets
  `jira.project: REK`.
- Approved narrow planning source:
  [`docs/design/signed-publication-review-work-item.md`](../design/signed-publication-review-work-item.md).
- Approved review budget: 26 changed files, 1500 changed lines, 9 components
  (scope adjustment approved 2026-07-13).
- Manager decomposition artifact (owner-only runtime path):
  `~/.codex-governance-runtime/signed-publication-ticket-plan/decomposition.json`.
- Drafted subtask: `ST-1`, signed remote-publication authorization and
  review-evidence hardening.

## Resolved Planning Prerequisite

`jira constraints assign` now takes the manager decomposition and an approved
assignment, validates source identity, allowed paths, budget, dependencies, and
traceability, and emits the owner-only assigned constraints file required by
reviewer/verifier plan generation. Do not manually edit that output file.

## Safe Next Steps

1. Use `jira constraints assign` to assign `ST-1` the approved paths,
   `24 / 1400 / 8` budget,
   dependencies, and traceability.
2. Run `jira plan generate` with the assigned constraints and the owner-only
   local policy. It must dispatch independent reviewer and verifier roles.
3. Obtain stakeholder approval of the resulting plan/workflow.
4. Create the Jira Story/subtask through `jira plan create --approve` using
   the approved workflow and private environment credentials.
5. Obtain a fresh signed offline export for the created subtask.
6. Create the implementation run and task bundle, then run independent
   reviewer and verifier assessments against the exact current diff.
7. Run `make review-gate EVIDENCE=/absolute/path/review-evidence.json`.
8. Only after all gates pass may the branch be committed, pushed, or proposed
   as a PR.

## Validation Already Run

Before the later ticket-plan/decomposition changes:

- `make test`
- `make vet`
- `make build`
- `git diff --check`
- `go run ./cmd/codex-governance config check --repo-root .`

After adding the manager-only decomposition path:

- `go test ./internal/agentplan ./internal/cli`
- `git diff --check`

Rerun the full validation suite after the assignment command is implemented.

## Deferred Follow-Up: Governed Model Unload

Add a policy-checked `codex-governance ollama unload --model NAME` command for
explicit, auditable local-model transitions. It must only unload an allowlisted
model through the governed gateway, record no prompts or credentials, and
verify the model is no longer resident before a high-memory verifier is run.
Do not bypass this with raw Ollama API calls or an unrestricted shell command.
