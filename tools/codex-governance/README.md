# codex-governance

`codex-governance` is a Go CLI for Jira-backed governance of Codex-assisted
engineering work. The completed initial implementation validates normalized
offline Jira snapshots, ADR and PR links, explicit Git ranges, scoped diffs,
review budgets, agent closure, release manifests, and governed local-model
summary jobs.

The toolkit is implemented as a Go CLI. A `Makefile` may expose developer
shortcuts but does not contain governance logic.

## Enforcement Boundary

`codex-governance` enforces policy only for the workflow and operations it
controls. It can block a governed run, but it cannot prevent a user or another
tool from editing, committing, or publishing outside that path. Its audit
records provide evidence of observed governed actions; they are not proof that
no bypass occurred. For the proposed implementation-agent extension,
provider/model evaluation records qualify a specific, versioned code-editing
configuration for a defined task scope; they do not guarantee safety for every
repository or task.

## Self-Development Boundary

This Git repository does not use `codex-governance` or ScopeLock to govern its
own development. The CLI, plugin, skill, runtime, Jira workflow, repository
configuration, hooks, and generated review evidence are not authorities or
gates for changes to this repository. Running the product in tests is
diagnostic only.

Self-development uses GitHub for tracking, ordinary repository checks, distinct
independent external reviewer and verifier assessments, and explicit owner
approval.

## Files

- `cmd/codex-governance/`: CLI entry point.
- `internal/`: implementation packages and embedded runtime assets.
- `testdata/`: deterministic Jira, work-item, and release-manifest fixtures.
- `docs/design/`: north-star, canonical specification, and release contract.
- `docs/implementation-plan/`: historical staged implementation prompts.
- `docs/roadmaps/`: approved implementation roadmaps.

## Operating Model

Jira stories own product intent and acceptance criteria. Jira implementation
subtasks own scoped technical work. Pull requests and CI own code and validation
evidence. ADRs remain under `docs/decisions/` with the code.

GitHub issues are backlog and planning records; they do not authorize
implementation. Before implementation begins, create the approved Jira Story
and primary Subtask, link the committed work item, and verify that lifecycle
state in preflight. Jira is the running execution record: approved updates
capture each governed commit or blocker, then PR and merge closure. Every Jira
write is previewed, explicitly approved, and read back.

The initial workflow reads Jira only. A Jira write, push, merge, publish,
release, deployment, Terraform apply, cloud mutation, destructive command, or
secret access always requires explicit approval.

## Ticket Planning

Phase 1 validates a ticket-plan contract against the approved PRD,
specification, and roadmap. It checks source digests, Markdown traceability,
dependencies, bounded paths, ADR references, and workflow-state integrity.
Hosted-manager dispatch, local worker review, stakeholder approval, and Jira
publication are separately governed. Jira publication requires an approved
workflow state and exactly one of `--dry-run` or `--approve`. The write path
reads `JIRA_BASE_URL`, `JIRA_EMAIL`, and `JIRA_API_TOKEN` from the environment;
`governance.yml` never stores credentials. A private publication record is
written before any Jira request and blocks automatic retries, including after a
partial creation.

```bash
codex-governance jira plan validate \
  --plan ticket-plan.json \
  --repo-root .
```

The plan contract records source paths and SHA-256 digests, a Story, independent
subtasks, traceability, phase, change class, review budget, allowed paths, and
ADR rationale. It does not write Jira.

## Jira Work Records

Use `jira work update` after a known commit or as soon as work is blocked. It
prints the exact Jira comment by default. Add `--approve` only after reviewing
that preview; the command then reads credentials from the environment, writes
the comment, and verifies the exact comment by read-back.

```bash
codex-governance jira work update \
  --issue REK-5 --kind commit --commit FULL_SHA \
  --scope "Add approval-gated Jira work records" \
  --check "go test ./internal/jira ./internal/cli" \
  --evidence "/private/review-evidence.json"
```

After a pull request is merged, use `jira work finalize` to verify the merged
pull request and current Jira hierarchy. It previews the merged-PR record and
the Subtask-then-Story transition order; `--approve` records the PR, performs
those transitions, and verifies each ticket is done with a resolution.

## Verification And CI

Repository CI uses ordinary formatting, vet, test, and build checks. It does
not run ScopeLock as a self-development authority.

## Development

Before committing or publishing a change, obtain independent reviewer and
verifier assessments for the exact diff. Both must be external to
`codex-governance`/ScopeLock and distinct from the implementer and from each
other. ScopeLock-generated evidence is not accepted for self-development.

```bash
make fmt
make vet
make test
make build
```

### Adopting-repository frontier assessment policy

In an adopting repository, local policy-approved models remain the default
assessment provider. A frontier subagent is available only when that
repository's `governance.yml` explicitly enables
`assessment.frontier_subagent`, allowlists its model identity, and sets a
maximum reasoning effort. The authorization for an individual assessment must
then bind that configured model, its role, and the exact diff. There is no
fallback from a local assessment to a frontier provider, and no external review
adapter is supported.

## Installation

Add Go's configured binary directory to `PATH`, then install the CLI:

```bash
go_bin_dir="$(go env GOBIN)"
if [ -z "$go_bin_dir" ]; then
  go_bin_dir="$(go env GOPATH | cut -d: -f1)/bin"
fi
export PATH="$go_bin_dir:$PATH"

make install
codex-governance --help
```

Go uses `GOBIN` when configured; otherwise it installs to the `bin` directory
under the first `GOPATH` entry, usually `~/go/bin`. Persist the resolved
`go_bin_dir` in your shell configuration if you need it in future terminals.

Remove the installed binary with:

```bash
make uninstall
```

Phase 3 supports read-only validation from a normalized work item and a signed
offline Jira export. The export must be signed by an unrevoked configured
`export-issuer` key and remain within `signing.offline_export_max_age`:

```bash
codex-governance validate-work-item \
  --work-item work-item.json \
  --offline-export jira-export.json \
  --repo-root .
```

The local runtime records agent closure under `~/.codex-governance-runtime/`.
It can initialize an owner-only Ollama policy, but Phase 5 permits only
benchmark-approved summary tasks; code-edit tasks remain disabled.

The proposed implementation-agent extension is documented in
[`docs/design/implementation-agent-prd.md`](docs/design/implementation-agent-prd.md),
[`docs/design/implementation-agent-spec.md`](docs/design/implementation-agent-spec.md),
and its companion roadmap. It is not complete in the current CLI. It will use
adapter-first orchestration with headless Codex as the first adapter. A
user may select a local LLM only after its policy and code-edit benchmark gates
pass. Its current preflight foundation requires a signed, policy-fresh offline
export; immediately before adapter dispatch it rechecks the current signing
policy and bundle digest. The private task bundle retains the envelope, while
the run records its provenance. Push and pull-request creation require
separate, run-specific approval.

Release manifests are checked locally with `sync --check` or described with
`sync --dry-run`. [releases/1.0.0-draft.json](releases/1.0.0-draft.json) is a
local draft only: it is not published or adopted in `governance.yml`.

The implementation initially supports only the `generic` profile. Terraform,
Node/Kubernetes, and Python/Vue/Kubernetes profiles remain future work.
