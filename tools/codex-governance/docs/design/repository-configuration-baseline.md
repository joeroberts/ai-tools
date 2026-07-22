# Repository-configuration baseline

## Status and Authority

This document is the authoritative baseline for repository configuration in
`joeroberts/ai-tools`. It records controls that are reviewable in Git and
controls that can be changed only by a GitHub repository owner. The accountable
owner for every control is `@joeroberts`; ownership assigns review
responsibility and does not grant access or automation authority.

The current hosted inventory was read on 2026-07-21. It is not a desired-state
record: `main` has no protection or ruleset; merge, squash, and rebase are
enabled; merged branches are retained; Dependabot security updates are off;
and secret scanning plus push protection are on. The checked-in policy files
listed below were absent at that time.

`codex-governance repository baseline check --repo-root PATH` verifies only
checked-in controls. `tools/codex-governance/scripts/audit-github-repository-settings.sh
--repo OWNER/REPO` is an owner-run, read-only hosted audit. Neither command
applies settings, writes GitHub state, or carries release, merge, publication,
deployment, secret, access, or Jira authority.

## Checked-In Controls

| Control | Desired state and rationale | Enforcement and verification | Rollback |
| --- | --- | --- | --- |
| `.github/CODEOWNERS` | Root, products, Codex integrations, workflows, `governance.yml`, and security policy have a named owner so sensitive changes are routed consistently. | `repository baseline check`; inspect review request routing in a test pull request. | Revert the reviewed ownership change; do not use it to alter membership or access. |
| `.github/dependabot.yml` | Monthly grouped `gomod` and `github-actions` updates, with at most five open PRs; dependency maintenance is visible and bounded. No auto-merge, workflow write, publication, or administration action is permitted. | `repository baseline check`; GitHub Dependabot configuration page read-back. | Revert the file; owner separately disables hosted Dependabot settings only with a reviewed preview. |
| `SECURITY.md` | Confidential reports use GitHub private vulnerability reporting; `main` is the supported line; public issues must not contain vulnerability details. | `repository baseline check`; verify the rendered security policy and private-report entry point. | Revert the file only after a replacement private path is reviewed. |
| Pull-request template | Collects scope, validation, issue/Jira linkage when applicable, governance/release impact, and security consideration; each field may be `N/A`. | `repository baseline check`; open a draft pull request. | Revert the template change. |
| Bug report and issue configuration | Bug reports provide reproducible routing data and issue configuration directs security reports to the private route. | `repository baseline check`; open the new-issue chooser. | Revert the relevant form/configuration change. |
| `.gitattributes` | `text=auto` and binary markers avoid platform-specific churn without rewriting existing content. | `repository baseline check`; inspect attributes with `git check-attr`. | Revert the attribute rule; do not mass-normalize files. |
| `.editorconfig` | UTF-8, LF, final newline, and trimmed text whitespace are portable defaults; Makefile tabs and generated/binary content remain protected. | `repository baseline check`; use an EditorConfig-aware editor on a new file. | Revert the rule without reformatting unrelated content. |

## Proposed GitHub-Hosted Controls

All rows in this section are proposals. They are not live configuration and
must not be applied by this issue. Before any mutation, the repository owner
must review an exact API preview, explicitly approve that preview, apply it
outside automation, and read back every listed setting. A failed read-back is
drift, not permission to retry with broader settings.

| Control | Proposed desired state and rationale | Read-only verification | Exact rollback |
| --- | --- | --- | --- |
| `main` ruleset | Require pull requests, one approving review, stale-approval dismissal, resolved conversations, administrator inclusion, and blocked force-push/deletion. These are the minimum protections for reviewed changes. | Owner runs the hosted audit and confirms all rule predicates. Required checks are valid only when all three names below are present. | Owner previews and restores the prior ruleset JSON captured in the approved change record; delete only the newly created ruleset if there was no prior rule. |
| Required checks | Require exactly `go`, `advisory`, and `semantic-version`. These are the stable, read-only checks established by #18 and #44. The proposal fails closed if one is absent or unverifiable. | Hosted audit reads the ruleset and recent check names without substituting or omitting a check. | Restore the exact prior required-check list from the approved rollback preview. |
| Merge and cleanup | Allow squash merge only and automatically delete merged branches. Disable merge commits and rebase merge to keep linear reviewed history. | Hosted audit compares merge-method and branch-cleanup booleans. | Restore the exact previous merge and cleanup booleans from the approved preview. |
| Pull-request automation | Disable automatic branch updates and auto-merge; branch freshness and merge remain human decisions. | Hosted audit compares `allow_update_branch` and `allow_auto_merge`. | Restore the prior booleans from the approved preview. |
| Commit attestations | Do not require signed commits or web commit sign-off until an owner separately defines support and exception handling. | Hosted audit reports the corresponding booleans. | Restore any previous requirement only through its approved preview. |
| Dependency security | Enable vulnerability alerts and Dependabot security updates; checked-in Dependabot remains bounded and non-merging. | Hosted audit reports enabled/disabled state only. | Restore the prior security-update state from the approved preview. |
| Secret protection | Keep secret scanning and push protection enabled, including supported non-provider patterns only after their false-positive policy is reviewed. | Hosted audit reports enabled/disabled state only and never prints scanning payloads. | Restore prior feature booleans from the approved preview. |
| Private vulnerability reporting | Enable GitHub private vulnerability reporting so `SECURITY.md` has an actionable confidential route. | Owner confirms the repository security feature state without reporting private details. | Disable only through a separately approved rollback after a replacement confidential route exists. |
| Actions and forks | Preserve default least privilege and fork-safe read-only workflows; no workflow receives a repository-admin, secrets, or write-token expansion. | Hosted audit reports repository Actions permission summary; workflow review verifies `contents: read`. | Restore the exact prior Actions settings from the approved preview. |

## Hosted Change Procedure

1. The repository owner runs the read-only audit and records only its redacted
   result plus the prior-state digest.
2. The owner prepares an exact GitHub API request preview for the selected
   rows, including ruleset target, checks, merge settings, and security
   booleans. The preview identifies the previous value and rollback request for
   each field.
3. A separate explicit owner approval authorizes that one preview. It does not
   authorize an unreviewed replacement, access change, secret change, webhook,
   variable, merge, release, or deployment.
4. The owner applies the preview, performs the table's read-back, and records
   a redacted comparison. Any mismatch is remediated by the approved rollback
   or a newly approved preview.

## Drift Handling

Checked-in drift is a deterministic local/CI failure and is fixed in a reviewed
pull request. Hosted drift is reported by the owner-run audit as control names,
booleans, and counts only. It must never expose authorization headers, tokens,
cookies, raw API bodies, private-report details, or arbitrary command errors.

This baseline does not change #18 workflow behavior, #44 version semantics,
or #22 publication authority. It does not make the advisory
`repository-baseline` check required; adding it to a live ruleset needs the
same separate owner-preview and approval procedure.
