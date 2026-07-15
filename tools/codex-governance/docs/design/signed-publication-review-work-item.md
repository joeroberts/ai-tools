# Signed Publication and Review-Evidence Hardening Work Item

## Status

Approved for Jira planning on 2026-07-13.

## Scope

Harden this high-risk governed remote-publication workflow so that a signed,
repository-owner authorization is verified against the exact repository,
remote, target ref, base SHA, branch, and commit SHA before push or pull
request creation. Require independently executed reviewer and verifier evidence
for the exact diff before commit, push, or pull-request creation.

## Allowed Paths

- `AGENTS.md`
- `Makefile`
- `README.md`
- `.githooks/`
- `docs/design/`
- `governance.yml`
- `internal/assets/templates/governance.yml`
- `internal/agentplan/`
- `internal/cli/`
- `internal/config/`
- `internal/implementation/`
- `internal/ticketplan/`

## Non-Goals

- Jira writes other than creating this approved Story and subtask.
- Merge, release, tag, deployment, or force-push automation.
- Provider qualification, evaluation-registry, or audit-retention redesign.
- Bypassing required reviewer, verifier, source, authorization, or policy gates.

## Architecture Decision

No ADR needed: signed-authorization hardening remains within the existing
governed publication architecture and does not change the stated non-goals.

## Acceptance Criteria

- Push and pull-request creation reject altered, expired, reused, or mismatched
  signed authorization evidence before the remote side effect.
- Each remote operation is independently consumed before dispatch.
- Reviewer and verifier evidence identifies distinct executors, matches the
  exact diff, and contains no blocking or important findings.
- Commit, push, and pull-request paths fail closed when that evidence is absent
  or altered.
- Focused regression tests cover valid and rejection paths.

## Validation Plan

- `make test`
- `make vet`
- `make build`
- `git diff --check`
- Independent reviewer and verifier assessments bound to the exact diff.

## Review Budget

This work is limited to 26 changed files, 2500 changed lines, and 9 components.
This scope adjustment, including `internal/ticketplan/` for allowed-path
validation and the increased line budget, was approved on 2026-07-14.

Components: documentation, hooks, configuration, agent planning, CLI,
publication authorization, review evidence, tests, and ticket-plan validation.
