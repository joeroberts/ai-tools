# Jira Work-Record and Finalization Work Item

## Status

Jira-planning is complete. Source: GitHub issue #11; Story `REK-4`.

- `REK-5` — work-record updates
- `REK-6` — merge finalization (depends on `REK-5`)

The approved owner-authored plan was required because the manager decomposition
did not cover the declared slices; the planner defect is tracked in GitHub
issue #20. The work item and governance-entry policy changes are committed;
`REK-5` is implementation-ready. `REK-6` remains blocked on `REK-5`.

## Scope

Add an approval-gated Jira work-record lifecycle to the governed CLI. The
workflow must prepare factual commit and blocker updates, preview every Jira
write, add the approved comment only after its triggering event is known, and
finalize a merged subtask and eligible parent Story with read-back evidence.

## Allowed Paths

- `AGENTS.md`
- `Makefile`
- `README.md`
- `docs/design/`
- `docs/handovers/`
- `governance.yml`
- `internal/cli/`
- `internal/implementation/`
- `internal/jira/`

## Non-Goals

- Automatic, background, or unapproved Jira writes.
- Replacing Jira ticket planning, signed exports, review evidence, or
  signed-publication authorization.
- Creating, merging, releasing, tagging, deploying, or force-pushing code.
- Changing Jira authentication or storing credentials in the repository.
- Inferring that a Story is complete when any child subtask remains open.

## Architecture Decision

No ADR needed. Add a bounded `jira work` command family that treats Jira as a
fact record while preserving explicit human authority for every write:

- `jira work update` accepts a structured `commit` or `blocker` event and
  renders the exact comment in dry-run mode.
- `jira work finalize` verifies the supplied pull request is merged, reads the
  current ticket hierarchy and transitions, previews the subtask-then-Story
  transition sequence, and writes only with `--approve`.
- Each successful write records the Jira response reference and rereads the
  affected issue. A failed or ambiguous read/write fails closed and reports a
  recoverable next action.

The implementation must not claim that a raw Git commit was recorded unless a
matching approved Jira update and read-back succeeded. It may guide the user
after a commit, but it must not make a Jira write from a hook or background
process.

## Planned Slices

### Work-Record Updates

The `work-record-updates` slice implements `jira work update` for factual
`commit` and `blocker` records. Its change class is `standard`; its ADR is `No
ADR needed: bounded Jira lifecycle behavior`. This slice is limited to 10
changed files, 800 changed lines, and the CLI, Jira client, local
result-record, and test components. Its allowed paths are `internal/cli/`,
`internal/jira/`, `internal/implementation/`, `Makefile`, `README.md`,
`docs/design/`, and `docs/handovers/`.

### Merge Finalization

The `merge-finalization` slice implements `jira work finalize` after the
`work-record-updates` slice. Its change class is `standard`; its ADR is `No ADR
needed: bounded Jira lifecycle behavior`. It verifies merged PR state and
ticket hierarchy, previews and approves the Subtask-then-Story transition
sequence, and verifies read-back status and resolution. This slice is limited
to 10 changed files, 1000 changed lines, and the CLI, Jira client,
lifecycle-validation, and test components. Its allowed paths are
`internal/cli/`, `internal/jira/`, `internal/implementation/`, `Makefile`,
`README.md`, `docs/design/`, and `docs/handovers/`.

## Acceptance Criteria

- A commit update validates the issue key, commit SHA, completed scope, checks,
  and evidence reference; it previews the resulting comment by default and
  creates it only with `--approve`.
- A blocker update validates and records the blocker, impact, owner decision
  needed, and next action using the same preview/approval/read-back boundary.
- Finalization rejects an unmerged or ambiguous pull request, a ticket with
  missing review evidence, an incomplete subtask, or a parent Story with an
  incomplete child.
- Finalization transitions the Subtask before the Story, and transitions the
  Story only after its completion rule is verified. It rereads both tickets and
  verifies their status and resolution.
- All Jira writes require credentials only from the environment, explicit
  `--approve`, short HTTP timeouts, and structured failure messages. No prompts,
  credentials, or raw Jira dumps enter the repository.
- Focused tests cover preview-only execution, approval enforcement, valid
  commit/blocker comments, merged and unmerged pull requests, incomplete
  hierarchy rejection, transition ordering, and failed read-back handling.

## Validation Plan

- `go test ./internal/jira ./internal/implementation ./internal/cli`
- `make test`
- `make vet`
- `make build`
- `git diff --check`
- Independent reviewer and verifier assessments bound to the exact diff.

## Review Budget

This work is limited to 20 changed files, 1800 changed lines, and 7 components.

Components: documentation, CLI, Jira client, lifecycle validation, local
result records, tests, and Makefile guidance.
