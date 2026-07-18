# Jira In-Progress Preflight Work Item

## Status

GitHub issue #29 is the source request. Jira Story
[REK-36](https://rekonlabs.atlassian.net/browse/REK-36) is the execution
contract, with two sequential implementation Subtasks:

1. [REK-37](https://rekonlabs.atlassian.net/browse/REK-37) captures and
   validates signed status evidence.
2. [REK-38](https://rekonlabs.atlassian.net/browse/REK-38) enforces the
   `In Progress` preflight gate after `REK-37` completes.

Implementation remains blocked until this work item is committed and the
active primary Subtask is transitioned to `In Progress` and read back.

## Scope

Make the signed offline Jira export carry the Story and Subtask status needed
for an implementation-entry decision. Implementation preflight must reject a
primary Subtask whose verified status is not exactly `In Progress`, including
`To Do`, blocked, stale, missing, or unverifiable status evidence. The failure
must occur before a task bundle, run, worktree, or adapter dispatch is created.

Align `AGENTS.md` with the enforced workflow. Require the primary Subtask to be
transitioned to `In Progress` with read-back before the first implementation
edit, and keep newly discovered defects or improvements separate from the
approved active-ticket scope unless the owner explicitly changes scope.

This is a standard hardening of the existing signed-export and implementation
preflight boundaries.

## Non-Goals

- Inferring or silently changing Jira status.
- Performing Jira transitions from implementation preflight.
- Treating a work-item handoff string as authoritative ticket status.
- Expanding GitHub issue #26 or changing Jira evidence rendering.
- Implementing unrelated defects discovered while #29 is active.

## Acceptance Criteria

- Signed offline Jira exports include source status for both the Story and
  Subtask and reject missing or malformed status evidence.
- Implementation preflight passes only when the signed, policy-valid Subtask
  snapshot reports exactly `In Progress`.
- `To Do`, blocked, stale, missing, and unverifiable status cases fail before
  task-bundle, run, worktree, or adapter creation with an actionable error.
- Focused tests cover export capture, signature verification, accepted status,
  rejected statuses, and absence of implementation side effects on failure.
- `AGENTS.md` contains the recovered active-ticket separation rule and requires
  the Jira `In Progress` transition plus read-back.
- `make test`, `make vet`, `make build`, and `git diff --check` pass.

## Allowed Paths

- `AGENTS.md`
- `docs/design/`
- `internal/implementation/`
- `internal/jira/`
- `testdata/`

## Review Budget

Deliver the work as two independently reviewed slices:

- `REK-37`: maximum 6 changed files, 400 changed lines, and two components:
  signed Jira export status evidence and export validation fixtures.
- `REK-38`: maximum 6 changed files, 400 changed lines, and two components:
  the `In Progress` preflight gate and workflow/preflight fixtures.

Architecture Decision: No ADR needed: this strengthens existing signed-export
and preflight contracts without adding a new architectural component.
