# GitHub #45: Repository-configuration baseline

## Purpose

Establish one testable, least-privilege repository-configuration baseline for
`joeroberts/ai-tools`. The baseline distinguishes policy that can be reviewed
in Git from settings that only a repository owner can change at GitHub. It
must make drift visible without granting automation authority to repair it.

## Current Inventory

The inventory below was read from GitHub on 2026-07-21 and is a starting
observation, not a desired configuration:

- `main` has no branch protection or repository ruleset.
- Merge commits, squash merges, and rebase merges are enabled; merged branches
  are not automatically deleted.
- Dependabot security updates are disabled.
- Secret scanning and secret-scanning push protection are enabled.
- `CODEOWNERS`, Dependabot configuration, `SECURITY.md`, pull-request and
  issue templates, `.gitattributes`, and `.editorconfig` are absent.
- The stable read-only checks delivered by #18 and #44 are `go`, `advisory`,
  and `semantic-version`.

## Outcomes

- Checked-in policy files define ownership, vulnerability reporting, dependency
  updates, contribution intake, and portable text/editor behavior.
- A versioned baseline document records every in-scope control's desired state,
  rationale, accountable owner, enforcement surface, verification, and
  rollback.
- A deterministic local command validates checked-in invariants and a
  repository-owned workflow runs that command without tokens, secrets, or
  mutable API calls.
- A separately invoked owner-run audit obtains only a redacted summary of
  GitHub-hosted settings and compares it to the documented desired state.
- Hosted settings have an exact, reviewable preview and rollback plan, but no
  code in this issue mutates GitHub settings, rulesets, secrets, webhooks,
  variables, or access permissions.

## User Experience

Contributors find concise pull-request and issue forms, a private security
reporting route, and ownership expectations in the repository. Dependabot
opens bounded grouped updates for Go modules and GitHub Actions; it receives no
authority to merge, publish, or change repository settings.

Maintainers run `codex-governance repository baseline check --repo-root PATH`
to identify a missing, malformed, or conflicting checked-in control. A
failure identifies the exact file and expected invariant. The CI check is
read-only and produces no Jira writes, GitHub writes, tokens, prompts, or raw
settings payloads.

A repository owner may run the separately documented hosted-settings audit.
It prints a minimal redacted comparison and returns nonzero on drift. It never
changes a setting. Applying a generated hosted-settings preview remains a
future, separately approved owner action after the exact target, required
checks, and rollback command are reviewed.

## Non-Goals

- Changing the behavior or required-check contract of #18 (`go`, `advisory`)
  or #44 (`semantic-version`).
- Applying branch protection, rulesets, Actions permissions, merge settings,
  security settings, secrets, variables, webhooks, or access permissions.
- Auto-merging pull requests, publishing, deploying, releasing, tagging, or
  authorizing Jira writes.
- Reformatting existing content merely because text/editor configuration is
  introduced.

## Compatibility And Safety

- The checked-in configuration is additive and does not impose approval on a
  path without a named owner rule; a root fallback documents the accountable
  repository owner.
- Issue and pull-request templates ask only for information needed to route
  work; they do not replace Jira as the execution contract.
- The proposed `main` ruleset fails closed if any required check is absent or
  cannot be verified, and names only `go`, `advisory`, and `semantic-version`.
- Hosted audit output includes booleans, names, and counts only. It must redact
  authorization headers, tokens, cookie values, raw API bodies, private
  reporting details, and arbitrary command output.
