# GitHub #45: Repository-configuration baseline specification

## Baseline Record

`tools/codex-governance/docs/design/repository-configuration-baseline.md` is
the authoritative human-readable baseline. It has one row for every in-scope
control with these fields: control, desired state, rationale, accountable
owner, enforcement surface, verification command or read-back, and rollback.
It separates checked-in controls from GitHub-hosted controls and states which
controls are proposals rather than live configuration.

The checked-in baseline must contain these exact artifacts:

- `.github/CODEOWNERS` assigns the repository root, products,
  `integrations/codex/`, `.github/`, `governance.yml`, and security-sensitive
  controls to the named repository owner. Its comments explain the intentionally
  small owner set and that ownership is an approval signal, not a public access
  grant.
- `.github/dependabot.yml` schedules monthly grouped updates for `gomod` under
  `tools/codex-governance` and `github-actions` at `/`; it sets no auto-merge,
  workflow-write, release, publication, or repository-administration action.
- `SECURITY.md` directs confidential vulnerability details to GitHub private
  vulnerability reporting, identifies the supported `main` line, sets response
  and disclosure expectations, and says not to file sensitive details in a
  public issue.
- `.github/pull_request_template.md` asks for scope, validation, linked issue
  or Jira reference when applicable, governance/release impact, and security
  consideration; it says `N/A` is acceptable where a field does not apply.
- `.github/ISSUE_TEMPLATE/bug_report.yml` and
  `.github/ISSUE_TEMPLATE/config.yml` collect actionable bug reports and link
  security reports away from public issues.
- `.gitattributes` normalizes text through `text=auto`, marks common binary
  image/PDF formats as binary, and does not rewrite existing content.
- `.editorconfig` applies UTF-8, LF, final newlines, and trimmed trailing
  whitespace for ordinary text while preserving Makefile tabs and exempting
  generated/binary paths from formatting claims.

## Checked-In Validation

The CLI exposes this read-only command:

```text
codex-governance repository baseline check --repo-root PATH
```

It reads only the named repository. On success it emits a stable line-oriented
summary. On failure it names the missing, malformed, conflicting, or
permission-expanding invariant and exits nonzero. It must not write a file,
network state, Git state, Jira record, or GitHub setting.

Validation verifies the exact required artifacts above, rejects a Dependabot
configuration with an `open-pull-requests-limit` greater than five or a
schedule more frequent than weekly, and rejects any configuration that grants
auto-merge, `pull-requests: write`, `contents: write`, or a workflow/repository
administration action. It validates CODEOWNERS coverage for the listed
governance and security paths, `SECURITY.md` private-reporting wording, forms'
required routing fields, and text/editor baseline markers. It does not require
or attempt online access.

`.github/workflows/repository-baseline.yml` runs the command for pull requests
that touch a baseline artifact, its implementation, or the workflow. Its sole
stable job/check name is `repository-baseline`; it uses `contents: read`, no
secrets, no mutable API calls, and a non-credentialed checkout. It is advisory
until a repository owner separately approves including it in a hosted ruleset.

## Hosted Settings Proposal And Audit

The baseline document proposes, but this issue does not apply, a `main`
ruleset requiring pull requests, one approving review, stale-approval
dismissal, resolved conversations, blocked force pushes and deletion, and
administrators subject to the rule. It names exactly `go`, `advisory`, and
`semantic-version` as required checks. The proposal fails closed when one of
those check names is absent from an owner read-back; it must not silently omit
or substitute a check.

The proposal selects squash-only merging, automatic deletion of merged
branches, no automatic branch update, no auto-merge, no required signed
commits, no required web sign-off, vulnerability alerts and Dependabot security
updates enabled, secret scanning and push protection enabled, and private
vulnerability reporting enabled. It preserves least-privilege Actions defaults
and fork-safe `contents: read` workflows. Each proposed value has an exact
GitHub API preview, owner, verification read-back, and rollback command in the
baseline document.

`tools/codex-governance/scripts/audit-github-repository-settings.sh` is an
owner-run, read-only helper. It requires `gh` authentication only at execution,
uses `gh api` GET endpoints only, and accepts `--repo OWNER/REPO`. It never
prints bearer tokens, headers, private-report details, raw settings JSON, or
arbitrary API errors. It returns a deterministic, redacted control summary and
reports hosted drift without attempting remediation.

## Allowed Paths

- `tools/codex-governance/docs/design/repository-configuration-baseline-prd.md`
- `tools/codex-governance/docs/design/repository-configuration-baseline-spec.md`
- `tools/codex-governance/docs/design/repository-configuration-baseline.md`
- `.github/CODEOWNERS`
- `.github/dependabot.yml`
- `SECURITY.md`
- `.github/pull_request_template.md`
- `.github/ISSUE_TEMPLATE/bug_report.yml`
- `.github/ISSUE_TEMPLATE/config.yml`
- `.gitattributes`
- `.editorconfig`
- `tools/codex-governance/internal/baseline/baseline.go`
- `tools/codex-governance/internal/baseline/baseline_test.go`
- `tools/codex-governance/internal/cli/cli.go`
- `tools/codex-governance/internal/cli/cli_test.go`
- `tools/codex-governance/scripts/audit-github-repository-settings.sh`
- `tools/codex-governance/scripts/audit-github-repository-settings_test.sh`
- `.github/workflows/repository-baseline.yml`

## Validation

- Unit-test successful validation plus missing artifacts, malformed YAML,
  conflicting configuration, unexpected required checks, permission expansion,
  and redaction.
- Exercise the shell audit helper with a fake `gh` executable to prove it uses
  GET only, produces a redacted summary, and reports absent or conflicting
  hosted controls without network access.
- Run `make test`, `make vet`, `make build`, and `git diff --check`.
- Require independent exact-diff reviewer and verifier evidence before every
  commit, push, or pull request.

## Acceptance Language

The implementation acceptance criteria are:

- The documented baseline identifies each in-scope checked-in and hosted
  control, its desired state, rationale, owner, enforcement surface,
  verification, and rollback.
- Checked-in validation is deterministic and read-only; it rejects missing,
  malformed, conflicting, or permission-expanding configuration.
- CODEOWNERS, bounded Dependabot configuration, private vulnerability
  reporting, focused contribution forms, and portable text/editor controls are
  present and validated.
- The hosted-settings audit is owner-run, read-only, deterministic, and
  redacts credentials, raw API payloads, and private reporting details.
- Hosted configuration remains a separately approved proposal. This issue
  never changes GitHub settings, rulesets, secrets, variables, webhooks, or
  access permissions.
- The proposed `main` ruleset names exactly `go`, `advisory`, and
  `semantic-version`, and fails closed on an absent or unverifiable check.

The baseline-contract slice acceptance criteria are:

- The PRD and specification distinguish checked-in controls from hosted
  settings and state that hosted settings remain read-only proposals.
- The documents record the observed inventory and use only `go`, `advisory`,
  and `semantic-version` for the proposed ruleset.
- The documents define ownership, verification, rollback, drift reporting,
  redaction, and the non-goals for every control class.

The checked-in-policy slice acceptance criteria are:

- CODEOWNERS covers product, integration, workflow, governance, and
  security-sensitive paths without granting access or remote authority.
- Dependabot covers Go modules and GitHub Actions with bounded grouping and no
  auto-merge or publication authority.
- SECURITY.md, pull-request and issue templates, `.gitattributes`, and
  `.editorconfig` provide the specified focused controls without reformatting
  unrelated source files.

The drift-evidence slice acceptance criteria are:

- `repository baseline check` validates exact checked-in invariants and fails
  for missing files, malformed configuration, conflicts, permission expansion,
  and unexpected proposed required checks.
- The `repository-baseline` workflow is fork-safe, `contents: read`, and
  advisory.
- The owner-run hosted audit issues GET requests only, reports redacted drift,
  and cannot mutate GitHub settings.

The slice non-goals are:

- Do not change GitHub-hosted settings, rulesets, branch protection, secrets,
  variables, webhooks, Actions permissions, or access permissions.
- Do not change #18 or #44 workflow behavior or their required-check names.
- Do not authorize automatic merge, publication, deployment, release, tagging,
  or Jira writes.
- Do not reformat unrelated repository content.

## Review Budget

This plan is limited to 18 changed files, 1700 changed lines, and repository configuration baseline, checked-in policy, deterministic local baseline validation, read-only hosted drift evidence.

## Declared Implementation Slices

The governed decomposition must create exactly the following four independent
subtasks, in this order. It must not merge, omit, or rename a declared slice.

```json
[
  {"id":"github-45-baseline-contract","phase":"Phase 1","change_class":"standard","dependencies":[],"allowed_paths":["tools/codex-governance/docs/design/repository-configuration-baseline-prd.md","tools/codex-governance/docs/design/repository-configuration-baseline-spec.md","tools/codex-governance/docs/design/repository-configuration-baseline.md"],"review_budget":{"max_changed_files":3,"max_changed_lines":500,"components":["repository configuration baseline"]}},
  {"id":"github-45-checked-in-policy","phase":"Phase 2","change_class":"standard","dependencies":["github-45-baseline-contract"],"allowed_paths":[".github/CODEOWNERS",".github/dependabot.yml","SECURITY.md",".github/pull_request_template.md",".github/ISSUE_TEMPLATE/bug_report.yml",".github/ISSUE_TEMPLATE/config.yml",".gitattributes",".editorconfig"],"review_budget":{"max_changed_files":8,"max_changed_lines":300,"components":["checked-in policy"]}},
  {"id":"github-45-local-baseline-validation","phase":"Phase 3","change_class":"standard","dependencies":["github-45-checked-in-policy"],"allowed_paths":["tools/codex-governance/internal/baseline/baseline.go","tools/codex-governance/internal/baseline/baseline_test.go","tools/codex-governance/internal/cli/cli.go","tools/codex-governance/internal/cli/cli_test.go"],"review_budget":{"max_changed_files":4,"max_changed_lines":650,"components":["deterministic local baseline validation"]}},
  {"id":"github-45-hosted-drift-evidence","phase":"Phase 4","change_class":"standard","dependencies":["github-45-local-baseline-validation"],"allowed_paths":["tools/codex-governance/scripts/audit-github-repository-settings.sh","tools/codex-governance/scripts/audit-github-repository-settings_test.sh",".github/workflows/repository-baseline.yml"],"review_budget":{"max_changed_files":3,"max_changed_lines":250,"components":["read-only hosted drift evidence"]}}
]
```

## Slice Evidence

- `github-45-baseline-contract` is a standard Phase 1 slice with no
  dependencies, three changed files, 500 changed lines, and repository
  configuration baseline.
- `github-45-checked-in-policy` is a standard Phase 2 slice depending on
  `github-45-baseline-contract`, with eight changed files, 300 changed lines,
  and checked-in policy.
- `github-45-local-baseline-validation` is a standard Phase 3 slice depending
  on `github-45-checked-in-policy`, with four changed files, 650 changed
  lines, and deterministic local baseline validation.
- `github-45-hosted-drift-evidence` is a standard Phase 4 slice depending on
  `github-45-local-baseline-validation`, with three changed files, 250 changed
  lines, and read-only hosted drift evidence.
