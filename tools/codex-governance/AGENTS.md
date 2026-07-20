# Repository Guidelines

## Project Structure & Module Organization

This repository contains the Jira-backed `codex-governance` Go CLI and its
design documentation.

- `cmd/codex-governance/` is the executable entry point; `internal/` contains
  implementation packages and embedded assets.
- `docs/design/north-star.md` is the approved north-star design; the canonical
  spec and staged prompts must remain aligned with it.
- `docs/roadmaps/go-cli-migration.md` controls phased implementation work.

Keep Go production code under `cmd/` and `internal/`; keep design and planning
documents under `docs/`.

## Build, Test, and Development Commands

Use the repository commands before committing:

- `rg -n "TODO|TBD|FIXME" .` finds unresolved placeholders.
- `make test` runs Go tests.
- `make vet` runs Go static checks.
- `make build` builds the CLI.
- `git diff --check` catches trailing whitespace and whitespace errors.
- `make review-gate EVIDENCE=/absolute/path/review-evidence.json` verifies
  independent reviewer and verifier evidence for the exact staged diff.
- `git diff --check` catches trailing whitespace and whitespace errors.

## Coding Style & Naming Conventions

Write Markdown with concise headings, short paragraphs, and actionable bullets. Use ATX headings (`#`, `##`) and fenced code blocks. Wrap filenames, commands, repo profiles, and toolkit names in backticks.

Format Go with `gofmt`. Keep command wiring in `cmd/` thin; place behavior in
focused `internal/` packages. When editing governance rules, update the
canonical spec under `docs/design/` first, then align staged prompts and README.

## Testing Guidelines

Add focused Go tests beside behavior and fixtures under `testdata/`. Run `make
test`, `make vet`, `make build`, and `git diff --check` for functional changes.

## Active-Task Continuity

Once the user authorizes a bounded phase, that authorization covers factual Jira
records, fresh exports, preflight, implementation, validation, reviews, and
local commits. Continue without per-action prompts. Stop only for a scope
change, meaningful blocker, failed gate, or external protection that requires
user input.

For an approved bounded phase, automatically push, create the pull request, and
merge when local tests, vet, build, scope checks, and independent review pass;
CI passes; the manager evaluates CodeRabbit comments against the repository
requirements; and no valid blocking issue remains. CodeRabbit is advisory, not
authoritative.

For every subagent or external review process, directly monitor its process and
result artifact until terminal. Keep the active turn open while waiting, show
the current gate and immediate action in commentary, and immediately execute
the next deterministic stage when it completes. Do not announce a future action
as though it has already begun.

## Approval Continuity

An explicit user approval of a governed planning contract, work item, implementation contract, or stated end-to-end workflow authorizes every action that contract expressly requires or permits.

That authorization includes:
- local workflow state updates and any required `--approve` flags;
- planning generation, review, verification, plan approval, and dry-run previews;
- in-scope Jira reads, creates, updates, transitions, finalization, and mandatory read-backs;
- implementation preflight, implementation, validation, reviewer/verifier evidence, remediation within the approved scope, commit, push, and pull-request creation;
- separately authorized publication steps when publication is expressly included in the approved contract.

Do not ask the user to repeat, restate, or separately approve an intermediate step solely because:
- a CLI command uses an `--approve` flag;
- a workflow has a local approval state;
- a Jira write must be previewed and read back;
- a tool reports that a plan is “ready for approval”; or
- a normal governed gate occurs between the user’s approval and the approved outcome.

Instead, perform the required preview, execute the in-scope operation, retain or report the required evidence, perform the required read-back, and continue the workflow.

A new user approval is required only when one of these conditions is true:
- the proposed action would exceed the approved contract’s scope, allowed paths, review budget, acceptance criteria, dependencies, authority model, or stated non-goals;
- the action is a materially different external action not expressly covered by the approved contract;
- a required gate fails and there is more than one materially different safe remediation path;
- proceeding would require a new secret, credential, external owner decision, or authority not already available and authorized; or
- the user explicitly pauses, cancels, or narrows the workflow.

When continuing under prior approval, give concise factual progress updates. Do not phrase a status update as a request for confirmation.

## Commit & Pull Request Guidelines

Reviewer and verifier evidence is a hard gate. Before any commit, push, or
pull-request creation, run independent reviewer and verifier assessments for
the exact diff and run `make review-gate` with the resulting evidence record.
Both records must pass and identify distinct executors. If either record is
missing, stale, mismatched, or reports a blocking or important finding, stop:
do not stage a commit, push, or create a pull request. Unit tests, CI, roadmap
status, or a prior review are not substitutes for this evidence.

Git history currently uses short, plain commits such as `Initial commit` and `first push`. Continue using concise imperative messages, for example `Add Jira drift validation` or `Clarify stage 3 CI guidance`.

Pull requests should include a short summary, files changed, validation commands run, and any changes to governance scope or hard-stop behavior. Link the primary Jira work item and explain any approved review exception. Screenshots are not required for Markdown-only changes.

## Work Tracking And Implementation Entry

Use GitHub issues for backlog, deferred work, and broad planning. Use Jira as
the execution contract for approved implementation work. Do not begin an
implementation edit or call work implementation-ready until a committed work
item links its Jira Story and primary Subtask.

At each work-state change, report whether the item is `backlog`,
`work-item-draft`, `Jira-planning`, `implementation-ready`, or `closed` and
name the next required transition. Before the first implementation edit, verify
the Jira linkage and state the next Jira update trigger.

For every governed commit, blocker, PR, and merge, prepare a factual Jira
update. Jira remains the authoritative execution record. For work covered by an
explicitly approved governed contract, the contract approval is the explicit
authorization for every in-scope Jira write. Before each Jira write, prepare or
run the required factual preview; after it, perform and report the required
read-back. Do not request a duplicate user confirmation between preview and
execution. Require new user approval only for an out-of-contract Jira mutation
or one of the Approval Continuity exceptions. Never make a Jira write from a
hook, background process, or unstated inference.

Before the first implementation edit, the primary Subtask must be transitioned
to exactly `In Progress`, and the Jira read-back must confirm that status. Do
not infer, silently change, or perform that transition from implementation
preflight.

Keep newly discovered defects, improvements, and follow-up work outside the
approved active-ticket scope unless the owner explicitly approves a scope
change. Track that work separately instead of expanding the active Subtask.

After confirming that a pull request is merged, return the local worktree to
`main`, fast-forward it from `origin/main`, and verify the worktree is clean.
If local changes or branch state prevent that sequence, report the blocker; do
not stash, discard, or overwrite work to force synchronization.

## Local Model Residency

Use the guarded Makefile targets to manage the fixed local review roles:
`make unload-reviewer-model`, `make unload-verifier-model`,
`make load-reviewer-model`, and `make load-verifier-model`. Before switching to
a high-memory reviewer or verifier, unload the no-longer-needed role and verify
the governed status reports the expected residency. Do not use raw Ollama CLI
or API commands and do not ask the user to unload models manually.

## Security & Configuration Tips

Preserve the repository's approval boundaries. Do not add instructions that
permit unbounded pushes, publishes, remote PR updates, tags, releases,
Terraform apply, cloud mutations, destructive commands, or secret access.
Bounded-phase authorization may permit push, PR creation, and merge only under
the active-task continuity gates above.
