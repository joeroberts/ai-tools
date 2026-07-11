# Release And Sync Contract

## Direction And Ownership

`codex-governance` is the upstream design source. A `codex-governance`
implementation repository is downstream. Changes flow upstream to downstream
as versioned releases; implementation discoveries return upstream only as a
proposed design change. Do not use bidirectional file synchronization.

Upstream owns the north-star, canonical requirements, normalized work-item
schema, template contracts, format versions, and migration rules. Downstream
owns scripts, adapters, tests, CI examples, gateway code, and repository-local
configuration. `README.md`, `AGENTS.md`, and `governance.yml` are merged
artifacts: updates add required sections but do not overwrite local content.

## Release Manifest

Each upstream release provides a manifest with release version, source commit,
format version, compatibility range, artifact digests, changelog, and migration
notes. A downstream repository records its adopted release and source commit in
`governance.yml`.

```json
{
  "release": "1.0.0",
  "source_commit": "<git-sha>",
  "format_version": 1,
  "artifacts": { "canonical_spec": "sha256:<digest>" }
}
```

## Migration

`governance-sync --dry-run --release <version>` compares the downstream lock
with the selected manifest and reports required schema, template, validator,
CI, and configuration changes. It writes nothing and makes no Jira, Git, or CI
mutation.

`governance-sync --check` validates that the current downstream implementation
matches its recorded release. A future apply mode must require an approved Jira
migration subtask, create a reviewable branch-level change set, preserve legacy
packet artifacts, and never overwrite locally owned or merged files. Breaking
format changes require explicit migration notes and a compatibility check.

Implementation migration work is split into reviewed Jira subtasks. Updating
the upstream source does not silently alter any downstream repository.
