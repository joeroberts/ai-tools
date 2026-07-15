# Signed Publication Integration Work Item

## Status

Approved for local integration planning on 2026-07-15.

## Scope

Complete the integration boundary for the signed-publication workflow: expose
the governed planner and review-evidence controls through the CLI and Makefile;
provide a guided, local-only Jira export signer and read-only export path; and
enforce signed, one-time publication authorization before a push or pull request
can have a remote side effect.

## Allowed Paths

- `AGENTS.md`
- `Makefile`
- `README.md`
- `.githooks/`
- `docs/design/`
- `docs/handovers/`
- `governance.yml`
- `internal/cli/`
- `internal/implementation/`
- `internal/jira/`
- `internal/ollama/`
- `internal/signature/`

## Non-Goals

- Merge, release, tag, deployment, or force-push automation.
- Jira writes except the separately approved Story/subtask publication flow.
- Bypassing review evidence, local-model policy, signed authorization, or
  one-time-consumption checks.
- Broad model-provider evaluation or remote infrastructure changes.

## Architecture Decision

No ADR needed: this work integrates the approved local governance primitives
without changing the existing publication architecture.

## Acceptance Criteria

- The CLI and Makefile expose review-evidence validation and read-only local
  model inventory without authorizing an unapproved model to execute.
- A guided local signer bootstrap creates owner-only material and a read-only
  Jira export can be signed without placing credentials or keys in the repo.
- Push and pull-request paths verify signed authorization, exact review
  evidence, repository identity, worktree integrity, and one-time consumption
  before the remote side effect.
- Focused regression tests cover valid paths and rejection paths for each
  external-boundary control.

## Validation Plan

- `make test`
- `make vet`
- `make build`
- `git diff --check`
- Independent reviewer and verifier assessments bound to the exact diff.

## Review Budget

This work is limited to 24 changed files, 2000 changed lines, and 8 components.

Components: documentation, hooks, configuration, CLI, Jira export, local
signing, publication authorization, and tests.
