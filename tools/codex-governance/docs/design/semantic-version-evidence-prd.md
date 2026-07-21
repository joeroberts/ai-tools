# GitHub #44: Semantic-version evidence

## Purpose

Provide one deterministic Semantic Versioning 2.0.0 contract for
`codex-governance`. Local tooling and GitHub Actions must calculate version
impact without writing Git state, and must reject release tags whose version,
metadata, or ancestry cannot be proven from repository state.

## Source Of Truth

`releases/1.0.0-draft.json` is the current repository release manifest and
its `release` field is the authoritative current version. Its successor
manifest remains the source of truth when a reviewed change advances that
version. No Go module version, GitHub Release, Git tag, workflow variable, or
`governance.yml` field independently defines the project version.

The manifest's `source_commit` is the commit represented by that version. The
existing `governance.yml` upstream fields remain a downstream-adoption lock;
they consume, rather than define, an upstream release version.

## Outcomes

- A local, deterministic command parses, compares, and increments SemVer
  2.0.0 versions including prerelease and build metadata forms.
- A pull request selects exactly one repository-owned impact input:
  `release:major`, `release:minor`, `release:patch`, or `release:none`.
  Absence, duplication, or an unknown release-impact label fails clearly.
- Pull-request validation reports the manifest version, selected impact, and
  proposed next version without changing repository state.
- Tag validation proves a `v<semver>` tag matches the manifest version,
  targets the manifest source commit, has a valid monotonic transition from
  the prior reachable release tag, and has no drift in versioned metadata.
- GitHub Actions expose concise, stable, read-only summaries that #22 can
  consume as evidence only; they do not grant release authority.
- The CLI exposes the manifest release in its normal version output and keeps
  upstream adoption checks compatible with the release-sync contract.

## User Experience

For a pull request, maintainers apply exactly one of the four release labels.
The `semantic-version` check reports `current`, `impact`, and `next` in its
step summary. A failure states the label or manifest condition to correct and
prints the equivalent local command.

For a release candidate, a maintainer creates no tag through this feature.
The tag-triggered check only validates an already supplied tag and reports the
specific version, ancestry, source-commit, or metadata mismatch. Correcting a
failed check requires a reviewed metadata or tag correction under the existing
approval process; it never causes automation to retag or publish.

## Non-Goals

- Creating, pushing, deleting, or rewriting Git tags.
- Creating GitHub Releases or publishing any artifact.
- Changing hosted branch protection, required-check settings, or label
  administration.
- Treating a successful check as release approval or bypassing Jira and
  explicit remote-mutation approval boundaries.

## Compatibility And Safety

- Workflow permissions remain `contents: read`; PR checks use only the event
  payload and local checkout, so forks need no secrets or write token.
- SemVer failures are deterministic, actionable, and reproducible locally.
- A repository with no prior reachable release tag is a supported first-release
  case. It still requires a valid manifest version and matching tag/source
  commit.
- Pre-release identifiers compare by SemVer precedence; build metadata does
  not affect precedence but remains syntactically validated and must match the
  manifest version exactly for tag validation.
