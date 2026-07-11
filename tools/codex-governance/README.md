# codex-governance

`codex-governance` is a Go CLI for Jira-backed governance of Codex-assisted
engineering work. The completed initial implementation validates normalized
offline Jira snapshots, ADR and PR links, explicit Git ranges, scoped diffs,
review budgets, agent closure, release manifests, and governed local-model
summary jobs.

The toolkit is implemented as a Go CLI. A `Makefile` may expose developer
shortcuts but does not contain governance logic.

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

Release manifests are checked locally with `sync --check` or described with
`sync --dry-run`. [releases/1.0.0-draft.json](releases/1.0.0-draft.json) is a
local draft only: it is not published or adopted in `governance.yml`.

The implementation initially supports only the `generic` profile. Terraform,
Node/Kubernetes, and Python/Vue/Kubernetes profiles remain future work.
