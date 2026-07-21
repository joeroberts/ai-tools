# GitHub #18: Authoritative CI and governance evidence

## Purpose

Make the repository's active GitHub Actions workflows give contributors
actionable, deterministic feedback for `tools/codex-governance` while
preserving local commands as the authority and retaining explicit approval
boundaries.

## Current Baseline

- Root `.github/workflows/ci.yml` runs formatting, vet, test, and build for
  pull requests and pushes to `main` with read-only contents permission.
- Root `.github/workflows/governance.yml` runs on pull requests, produces a
  compact summary artifact, performs no Jira access, and treats work-item
  validation as an opt-in local-fixture check.
- `tools/codex-governance/.github/workflows/governance-advisory.yml` is not an
  active GitHub workflow because GitHub reads workflow files only from the
  repository root.

## Outcomes

- Deterministic checks map directly to documented local commands and produce
  privacy-safe actionable summaries.
- CI and governance triggers, path behavior, Go setup/cache behavior, and
  fork safety are explicit and tested where practical.
- The design identifies candidate authoritative inputs for #22 without making
  them required branch protection checks.
- Review-evidence status is represented without uploading prompts,
  credentials, private runtime artifacts, or raw local evidence.

## Non-Goals

- Changing hosted branch protection or required-check settings.
- Jira writes, remote publication, model execution, secret-dependent actions,
  or automatic merge behavior.
- Treating CI as a replacement for the required local exact-diff review gate.
