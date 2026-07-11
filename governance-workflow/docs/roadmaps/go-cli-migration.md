# Go CLI Migration Roadmap

## Status

`in-progress`

Structured phase state: [go-cli-migration.yaml](go-cli-migration.yaml).

## Goal

Turn this documentation-first repository into the `codex-governance` Go CLI:
a reusable, testable governance utility that validates Jira-backed work items,
PR scope, reviewability, and governed local-model jobs.

## Source Context

The approved [north-star design](../design/north-star.md) and
[canonical specification](../design/canonical-spec.md) define the target model.
This roadmap is the implementation plan for this repository. A Jira story key
is not yet assigned; add it before ticket-backed validation is enabled.

## Design Principles

- Go owns governance behavior; the `Makefile` exposes developer shortcuts only.
- Jira is read-only unless an explicit approval authorizes a write.
- One PR should implement one reviewable Jira subtask by default.
- Preserve legacy prompt assets and packet-era artifacts; do not delete them
  automatically.
- Build deterministic validation before optional local-model automation.

## Phased Work

1. **Repository Foundation** - complete 2026-07-11
   - Move design and staged-prompt documents under `docs/`.
   - Add `go.mod`, `cmd/`, `internal/`, `testdata/`, `Makefile`, and Go CI.
   - Deliver `codex-governance --help`.
   - Add `.github/workflows/ci.yml` for pull requests and pushes to the default
     branch. Use `contents: read` permission only and run formatting, `go vet`,
     `go test`, and CLI build checks.

2. **Configuration And Assets** - complete 2026-07-11
   - Add embedded templates and schemas.
   - Implement `governance.yml` loading and release manifest support.
   - Deliver `codex-governance init` without overwriting local files.

3. **Work-Item Validation** - complete 2026-07-11
   - Implement normalized offline Jira export parsing.
   - Implement `validate-work-item` for ticket drift, ADR rationale, PR link,
     explicit Git range, scope-to-diff, review budgets, and exceptions.
   - Add fixture-driven tests and deterministic exit codes.

4. **Adoption And Synchronization** - complete 2026-07-11
   - Implement `sync --check` and `sync --dry-run`.
   - Add structured roadmap state under `docs/roadmaps/*.yaml`; keep narrative
     roadmap context in companion Markdown files.
   - Implement `roadmap status` with table, Markdown, and JSON output so status
     matrices are rendered locally without an agent parsing prose.
   - Implement `roadmap check` to validate phase states, approval records,
     active-phase constraints, and required validation evidence.
   - Add `.github/workflows/governance.yml` for pull-request governance
     validation. Use pull-request base/head SHAs, repository read permission,
     and no Jira write capability.
   - Upload compact validation summaries as workflow artifacts; do not upload
     credentials, raw prompts, unrestricted logs, or cache contents.
   - Start scope and review-budget validation in warning mode, publish its
     result as a non-required check, then calibrate thresholds from observed
     PRs before proposing branch protection and required-check rollout.
   - Make read-only Jira validation opt-in only after a scoped CI secret and
     explicit approval are available. Offline fixtures remain the default.
   - Document release manifest and migration compatibility behavior.

5. **Agent And Local-Model Runtime** - complete 2026-07-11
   - Add role directives, local execution ledger, and closure checks.
   - Implement governed Ollama jobs, model policy allowlists, cache safeguards,
     and benchmark gates.

6. **Review And Hardening** - complete 2026-07-11
   - Run independent review and verification.
   - Resolve blocking findings, document accepted caveats, and prepare a local
     release manifest without publishing it.

## Scope Boundaries

This roadmap does not authorize Jira writes, pushes, merges, releases,
deployments, Terraform or cloud mutations, secret access, arbitrary model
downloads, or destructive cleanup.

## GitHub Actions Rollout

`ci.yml` is required in Phase 1. It runs on `pull_request` and default-branch
`push` events with least-privilege `contents: read` permissions. It checks Go
formatting, `go vet ./...`, `go test ./...`, and `go build ./cmd/codex-governance`.

`governance.yml` is introduced in Phase 4. It runs on pull requests, compares
the explicit base and head range, and reports ticket/PR scope and review-budget
results. It starts advisory-only. Required status checks and branch protection
are proposed only after warning-mode results have been reviewed and thresholds
approved. Jira reads use an explicitly approved, read-only CI secret; no
workflow may update Jira, push, merge, deploy, or access unrelated secrets.

Workflow artifacts contain concise, redacted validation summaries and are
retained only for the repository's agreed CI retention period. Raw ticket
content, credentials, model prompts, unrestricted logs, and local cache data
must never be uploaded.

Phase 4 also adds local roadmap reporting. `codex-governance roadmap status`
renders the structured roadmap state as a table by default and supports Markdown
and JSON output. `codex-governance roadmap check` validates the structured
state. Neither command reads Jira or modifies repository files.

## Approval Record

- Approval status: approved
- Approving instruction: user approved Phases 1 through 6 on 2026-07-11
- Approved scope: Full Go CLI Migration Roadmap

## Progress

- Phase 1 is complete: documentation is under `docs/`, the Go module and CLI
  foundation exist, and basic GitHub Actions CI is configured.
- Phase 2 is complete: reusable assets, `governance.yml` loading, release
  manifest parsing, and no-overwrite initialization are implemented.
- Phase 3 is complete: offline Jira export parsing and deterministic work-item
  validation cover ticket drift, ADRs, PR links, Git ranges, scope, budgets,
  and review exceptions.
- Phase 4 is complete: roadmap status/check rendering, release comparison,
  advisory governance CI, and artifact safeguards are implemented.
- Phase 5 is complete: role directives, execution ledger, owner-only Ollama
  policy, benchmark gates, summary-only jobs, and redacted cache are implemented.
- Phase 6 is complete: independent review and post-remediation verification
  were run, blocking implementation findings were resolved, and a local draft
  release manifest was prepared. Publishing and tagging remain prohibited.

## Risks And Open Decisions

- Jira API authentication and provider details remain intentionally deferred;
  offline exports are the first validation path.
- Review-budget defaults require observed repository metrics before enforcement.
- Local Ollama models must pass benchmark gates before code-edit authority.
- The release/versioning scheme must remain compatible with repositories that
  adopt the CLI later.
