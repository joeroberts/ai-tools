# GitHub #44: Semantic-version evidence specification

## Contract

The product release version is read from a supplied release manifest. The
manifest parser must require a valid SemVer 2.0.0 `release` value in addition
to its existing required fields. `source_commit` must be a full immutable Git
object identifier accepted by the tag-validation command; manifests used by
the existing downstream sync command retain their current compatibility rules
until a later migration expressly tightens them.

The CLI provides read-only version commands:

```text
codex-governance version impact --manifest PATH --impact major|minor|patch|none
codex-governance version validate-tag --manifest PATH --tag v<semver> --repo-root PATH
codex-governance version current --manifest PATH
```

`current` prints a stable machine-readable and human-readable representation
of the manifest version. `impact` prints `current`, `impact`, and `next` in a
stable line-oriented form. `none` returns the unmodified current version.
None of these commands writes files, refs, tags, releases, or network state.

## SemVer Rules

- Parse the complete SemVer 2.0.0 grammar: numeric `major.minor.patch`, an
  optional dot-separated prerelease, and optional dot-separated build
  metadata.
- Numeric core and numeric prerelease identifiers reject leading zeroes except
  `0`. Identifiers contain only ASCII alphanumerics and hyphen and are nonempty.
- Comparison follows SemVer 2.0.0: core numbers compare numerically, a stable
  version has higher precedence than the corresponding prerelease, prerelease
  identifiers compare in order, and build metadata does not affect precedence.
- `major`, `minor`, and `patch` produce the next stable core version and clear
  prerelease/build metadata. `major` increments major and zeros minor/patch;
  `minor` increments minor and zeros patch; `patch` increments patch. Overflow
  fails rather than wrapping.
- `none` preserves the complete manifest version. An impact command must reject
  any other spelling.

## Pull-Request Input And Workflow

The root `semantic-version.yml` workflow runs for pull requests affecting the
release manifest, CLI version implementation, or workflow itself. It has one
stable check name, `semantic-version`, and `contents: read` permission.

The workflow derives release-impact labels only from
`github.event.pull_request.labels`. It passes the selected label to the local
CLI and writes the CLI result to `GITHUB_STEP_SUMMARY`. It does not check out
the PR with credentials, call a mutable API, or use secrets. Exactly one label
from the four contract labels is required. Missing or multiple contract labels
must fail with the names of the labels found and the four valid choices.

The workflow validates the current manifest before calculating impact. Its
summary always includes the stable field names `current`, `impact`, and `next`
on success. Test fixtures exercise major, minor, patch, none, missing,
conflicting, and malformed inputs.

## Tag Validation

The same workflow runs for `push` tags matching `v*`; this trigger validates
but never creates or changes the tag. It invokes `version validate-tag` against
the checked-out tag target and manifest.

Validation requires all of the following:

1. The tag is exactly `v` plus a valid SemVer version and equals the manifest
   `release` exactly, including prerelease/build metadata.
2. The tag target resolves to the manifest `source_commit`; annotated tags
   resolve to their peeled commit.
3. The manifest version and its required artifacts parse and verify without
   versioned-metadata drift.
4. If a reachable prior valid `v<semver>` tag exists, the candidate version has
   strictly greater SemVer precedence and the prior tag commit is an ancestor
   of the candidate target. Invalid or non-ancestor candidate history fails.
5. If no prior valid reachable release tag exists, validation reports the
   supported `first-release` condition and still applies the other rules.

The command reports each failed predicate with a corrective action. It must
not treat build-only differences as a monotonic release transition, because
they have equal SemVer precedence.

## Manifest And Adoption Exposure

`sync --check` and `sync --dry-run` continue to read the same manifest release
and compare it with `governance.yml` upstream adoption values. The version
commands share the parser used by manifest loading so local release evidence
and downstream adoption cannot disagree about validity. The existing draft
manifest remains valid as SemVer prerelease `1.0.0-draft`; this change neither
adopts nor publishes it.

## Allowed Paths

- `tools/codex-governance/docs/design/semantic-version-evidence-prd.md`
- `tools/codex-governance/docs/design/semantic-version-evidence-spec.md`
- `tools/codex-governance/internal/syncer/manifest.go`
- `tools/codex-governance/internal/syncer/manifest_test.go`
- `tools/codex-governance/internal/version`
- `tools/codex-governance/internal/cli/cli.go`
- `tools/codex-governance/internal/cli/cli_test.go`
- `.github/workflows/semantic-version.yml`
- `tools/codex-governance/testdata/releases`

## Validation

- Unit-test parsing, ordering, increments, invalid forms, and overflow.
- Unit-test manifest integration, CLI output/error conditions, and tag cases
  with temporary Git repositories: first release, ancestor, non-ancestor,
  malformed tag, mismatch, non-incrementing version, metadata drift, and
  annotated tag.
- Add fixture tests for all PR impact inputs and workflow summary contract.
- Run `make test`, `make vet`, `make build`, and `git diff --check`.
- Require independent exact-diff reviewer and verifier evidence before each
  commit, push, or pull request.

## Acceptance Language

The implementation acceptance criteria are:
- The manifest release is the authoritative current version, and no command
  writes files, refs, tags, releases, or network state.
- Pull-request validation requires exactly one supported release-impact label
  and reports `current`, `impact`, and `next`.
- Tag validation proves the exact manifest version, target/source-commit
  relationship, metadata integrity, and monotonic reachable release history,
  including the supported first-release case.
- The implementation remains evidence-only and does not create, push, delete,
  or rewrite tags or publish releases.

The version-contract slice acceptance criteria are:
- The PRD and specification state that the release manifest `release` field is
  authoritative.
- The specified commands are read-only and support `major`, `minor`, `patch`,
  and `none` impact inputs.
- The documents define exact tag/version, source-commit, metadata-drift,
  monotonic-history, and first-release validation requirements.

The local-version-engine slice acceptance criteria are:
- Manifest loading requires a valid SemVer 2.0.0 `release`; tag validation
  accepts a full immutable `source_commit`.
- The CLI emits stable output for current and impact calculations; `none`
  preserves the complete version and unsupported impacts fail.
- Tag validation handles annotated tags, first release, ancestry, malformed
  tags, exact version mismatches, non-incrementing versions, and metadata
  drift.
- Existing `sync --check` and `sync --dry-run` retain their release-manifest
  adoption behavior.

The fork-safe-workflow slice acceptance criteria are:
- The workflow has check name `semantic-version`, `contents: read` permission,
  and derives labels only from `github.event.pull_request.labels`.
- Successful PR summaries include `current`, `impact`, and `next`; missing or
  conflicting contract labels fail with found labels and valid choices.
- Tag pushes matching `v*` invoke validation only and never create or change a
  tag.

The slice non-goals are:
- Do not create, push, delete, or rewrite Git tags.
- Do not create GitHub Releases or publish any artifact.
- Do not write files, refs, tags, releases, or network state.
- Do not adopt or publish the existing draft manifest.
- Do not use secrets, mutable APIs, or a credentialed PR checkout.
- Do not create or change tags.

## Review Budget

This plan is limited to 9 changed files, 760 changed lines, and semantic-version contract, local version engine, fork-safe workflow evidence.

## Declared Implementation Slices
The governed decomposition must create exactly the following three independent
subtasks, in this order. It must not merge, omit, or rename a declared slice.
```json
[
  {"id":"github-44-version-contract","phase":"Phase 1","change_class":"standard","dependencies":[],"allowed_paths":["tools/codex-governance/docs/design/semantic-version-evidence-prd.md","tools/codex-governance/docs/design/semantic-version-evidence-spec.md"],"review_budget":{"max_changed_files":2,"max_changed_lines":260,"components":["semantic-version contract"]}},
  {"id":"github-44-local-version-engine","phase":"Phase 2","change_class":"standard","dependencies":["github-44-version-contract"],"allowed_paths":["tools/codex-governance/internal/syncer/manifest.go","tools/codex-governance/internal/syncer/manifest_test.go","tools/codex-governance/internal/version","tools/codex-governance/internal/cli/cli.go","tools/codex-governance/internal/cli/cli_test.go","tools/codex-governance/testdata/releases"],"review_budget":{"max_changed_files":6,"max_changed_lines":400,"components":["local version engine"]}},
  {"id":"github-44-fork-safe-workflow","phase":"Phase 3","change_class":"standard","dependencies":["github-44-local-version-engine"],"allowed_paths":[".github/workflows/semantic-version.yml"],"review_budget":{"max_changed_files":1,"max_changed_lines":100,"components":["fork-safe workflow evidence"]}}
]
```
## Slice Evidence

- `github-44-version-contract` is a standard Phase 1 slice with no
  dependencies, two changed files, 260 changed lines, and semantic-version
  contract.
- `github-44-local-version-engine` is a standard Phase 2 slice depending on
  `github-44-version-contract`, with six changed files, 400 changed lines,
  and local version engine.
- `github-44-fork-safe-workflow` is a standard Phase 3 slice depending on
  `github-44-local-version-engine`, with one changed file, 100 changed lines,
  and fork-safe workflow evidence.
