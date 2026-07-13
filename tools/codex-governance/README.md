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

## Verification And Advisory CI

Run the governed local smoke check with:

```bash
make smoke-ticket-plan
```

It validates the checked-in plan fixture and exercises the approved-workflow
publication dry run. It does not contact Jira, dispatch a model, read
credentials, or create a publication record. The `Governance Advisory` GitHub
Actions workflow runs this check with tests, vet, build, whitespace, and
roadmap validation. It has read-only repository permissions and is advisory;
it receives no Jira credentials or model prompts.

## Development

```bash
make fmt
make vet
make test
make build
go run ./cmd/codex-governance --help
```

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

View the implementation roadmap locally:

```bash
go run ./cmd/codex-governance roadmap status \
  --roadmap docs/roadmaps/go-cli-migration.yaml
go run ./cmd/codex-governance roadmap check \
  --roadmap docs/roadmaps/go-cli-migration.yaml
```

Phase 3 supports read-only validation from a normalized work item and an
offline Jira export:

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
and its companion roadmap. It is not implemented by the current CLI. It will
use adapter-first orchestration with headless Codex as the first adapter. A
user may select a local LLM only after its policy and code-edit benchmark gates
pass. Push and pull-request creation require separate, run-specific approval.

Release manifests are checked locally with `sync --check` or described with
`sync --dry-run`. [releases/1.0.0-draft.json](releases/1.0.0-draft.json) is a
local draft only: it is not published or adopted in `governance.yml`.

The implementation initially supports only the `generic` profile. Terraform,
Node/Kubernetes, and Python/Vue/Kubernetes profiles remain future work.
