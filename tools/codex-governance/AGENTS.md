# Repository Guidelines

## Project Structure

This product is the Jira-integrated Go CLI currently named
`codex-governance` and scheduled for the atomic ScopeLock rename.

- `cmd/codex-governance/` is the executable entry point.
- `internal/` contains production packages and embedded assets.
- `testdata/` contains deterministic fixtures.
- `docs/` contains product design, roadmaps, decisions, and operating guidance.

Keep Go production code under `cmd/` and `internal/`. Keep product-specific
design and planning documents under `docs/`.

## Self-Development Exclusion

ScopeLock and its legacy `codex-governance` identity are outside the authority
chain for developing this product and the Git repository that contains it.
This exclusion applies to every branch, worktree, fork, and successor attempt.

Do not invoke or rely on the product CLI, plugin, skill, runtime,
`governance.yml`, Jira workflow, hooks, review-evidence checker, generated
records, or local-model policy as an approval, preflight, planning, review,
verification, commit, publication, or completion gate for this repository.
There is no self-development exception, bootstrap mode, compatibility path, or
manual override.

The product may execute only as the subject of ordinary tests. Test output is
diagnostic and cannot authorize the tested change. GitHub is the sole work
tracker for self-development; Jira is an integration under test, not an
execution authority.

Use the following external development process:

- The primary agent coordinates work and does not use ScopeLock as authority.
- Implementation uses `gpt-5.6-sol` with medium reasoning.
- Independent planning, architecture, TPM, reviewer, and verifier roles use
  `gpt-5.6-sol` with high reasoning when those roles are required by scope.
- Reviewer and verifier are distinct from each other and from the implementer.
- No role uses `xhigh`, `max`, or `ultra`.
- Ordinary tests and independent assessments are retained as evidence outside
  ScopeLock's runtime.
- Explicit owner approval remains the final authority for publication.

## Build And Validation

Run the relevant commands from this directory:

- `make test`
- `make vet`
- `make build`
- `git diff --check`

Run `gofmt` on changed Go files. Behavior changes require focused tests at the
lowest practical layer. Candidate CLI smoke tests are allowed as diagnostics;
they do not govern the change.

## Coding Style

Keep command wiring in `cmd/` thin and place behavior in focused `internal/`
packages. Follow existing Go and Markdown style. Avoid unrelated refactors,
drive-by formatting, and new dependencies without a clear need.

## Commit And Pull Request Requirements

Before commit or publication, obtain passing assessments for the exact
candidate diff from a distinct external reviewer and verifier. These
assessments must not be produced, validated, or approved by ScopeLock or the
legacy application.

Pull requests should summarize the change, affected files, ordinary validation
commands, and any change to authority or fail-closed behavior. Do not require a
Jira link for this repository's self-development.

After a merged pull request, synchronize the goal-owned worktree with
`origin/main` only when it is clean. Never stash, discard, reset, or overwrite
unrelated user work to force synchronization.

## Safety

Do not add credentials, tokens, private prompts, local runtime files, caches,
or generated assessment artifacts. Do not push, publish, deploy, tag, mutate
GitHub or Jira, change cloud infrastructure, or delete historical evidence
without the applicable explicit owner authorization.
