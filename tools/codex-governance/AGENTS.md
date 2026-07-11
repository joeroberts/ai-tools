# Repository Guidelines

## Project Structure & Module Organization

This repository contains the Jira-backed `codex-governance` Go CLI and its
design documentation.

- `cmd/codex-governance/` is the executable entry point; `internal/` contains
  implementation packages and embedded assets.
- `docs/design/north-star.md` is the approved north-star design; the canonical
  spec and staged prompts must remain aligned with it.
- `docs/roadmaps/go-cli-migration.md` controls phased implementation work.

Keep Go production code under `cmd/` and `internal/`; keep design and planning
documents under `docs/`.

## Build, Test, and Development Commands

Use the repository commands before committing:

- `rg -n "TODO|TBD|FIXME" .` finds unresolved placeholders.
- `make test` runs Go tests.
- `make vet` runs Go static checks.
- `make build` builds the CLI.
- `git diff --check` catches trailing whitespace and whitespace errors.
- `git diff --check` catches trailing whitespace and whitespace errors.

## Coding Style & Naming Conventions

Write Markdown with concise headings, short paragraphs, and actionable bullets. Use ATX headings (`#`, `##`) and fenced code blocks. Wrap filenames, commands, repo profiles, and toolkit names in backticks.

Format Go with `gofmt`. Keep command wiring in `cmd/` thin; place behavior in
focused `internal/` packages. When editing governance rules, update the
canonical spec under `docs/design/` first, then align staged prompts and README.

## Testing Guidelines

Add focused Go tests beside behavior and fixtures under `testdata/`. Run `make
test`, `make vet`, `make build`, and `git diff --check` for functional changes.

## Commit & Pull Request Guidelines

Git history currently uses short, plain commits such as `Initial commit` and `first push`. Continue using concise imperative messages, for example `Add Jira drift validation` or `Clarify stage 3 CI guidance`.

Pull requests should include a short summary, files changed, validation commands run, and any changes to governance scope or hard-stop behavior. Link the primary Jira work item and explain any approved review exception. Screenshots are not required for Markdown-only changes.

## Security & Configuration Tips

Preserve the repository's approval boundaries. Do not add instructions that permit pushes, publishes, remote PR updates, tags, releases, Terraform apply, cloud mutations, destructive commands, or secret access without explicit approval.
