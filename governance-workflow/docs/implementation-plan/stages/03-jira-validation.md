# Stage 3 Prompt: Jira Validation

Use this prompt after Stage 2 is complete and approved.

## Objective

Create initialization, read-only Jira validation, and smoke tests.

## Scope

- Implement `codex-governance init` and
  `codex-governance validate-work-item` in Go.
- Implement a read-only Jira adapter with normalized offline-export support.
- Create CI-check and branch-protection guidance. Start review budgets in
  warning mode; do not require a hosted CI provider implementation yet.
- Create fixtures for valid work, missing ticket content, ticket drift, invalid
  parent/subtask links, missing PR link, scope drift, over-budget changes,
  approved review exceptions, missing ADR rationale, offline exports, and
  unsupported profiles.
- Create Go fixture tests and an optional `Makefile` with `build`, `test`, and
  `lint` shortcuts.

## Requirements

The Go CLI returns non-zero on validation failures and never writes Jira or
mutates remote systems. The validator checks the JSON work-item contract, ticket drift, PR linkage,
scope-to-diff, phase, review budgets, exceptions, ADR rationale, and profile
requirements. Support `--warn`, `--strict`, `--work-item`, `--repo-root`, and
`--offline-export`, `--base-sha`, and `--head-sha`.

## Validation

- Run `go test ./...`, `go vet ./...`, optional Go linting, and fixture tests.
- Prove valid fixtures pass and invalid fixtures fail or warn as specified.
- Prepare concise Stage 3 Jira handoff text; do not post it without approval.

## Completion

Summarize work and validation, propose Stage 4, and wait for approval.
