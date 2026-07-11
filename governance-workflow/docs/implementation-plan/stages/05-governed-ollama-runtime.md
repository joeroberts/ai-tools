# Stage 5 Prompt: Governed Ollama Runtime And Cache

Use this prompt after Stage 4 is complete and approved.

## Objective

Create the local execution boundary for approved Ollama model jobs and private
cache behavior.

## Scope

- Implement `codex-governance ollama run` in Go for atomic governed Ollama
  jobs.
- Load a policy-controlled allowlist that pins model name, version, and Ollama
  ID from `~/.codex-governance-runtime/policy.yaml`; only the platform owner
  may change that owner-only policy.
- Validate role, task type, scope, input bundle, output schema, attempts,
  timeout, tool-call limit, and change size before invoking a model.
- Return code edits as patches; stage them in a disposable worktree and run
  deterministic scope and test checks.
- Keep execution ledger, audit, and content-addressed cache data under
  `~/.codex-governance-runtime/` with owner-only permissions.
- Cache only ticket baselines, immutable diff summaries, exact-commit command
  results, and bounded summaries; never cache approval or review conclusions.

## Boundaries

Models must not call Jira directly, push, access secrets, deploy, mutate cloud
resources, invoke arbitrary shell commands, select arbitrary models, download
models, or silently escalate to a more expensive model. Run local jobs
sequentially by default. Token and dollar accounting are telemetry only unless
the execution runtime can enforce them from provider usage data.

Cache only digests and redacted summaries by default; never store credentials,
raw prompts, or unrestricted logs. Expire cache entries after 30 days by
default and provide a local clear command.

## Validation

- Verify rejected models, task types, paths, output schemas, and limits fail.
- Verify code patches are isolated and scope validation runs before acceptance.
- Verify cache keys include all relevant immutable inputs and a fresh Jira read
  is still required at each gate.
- Benchmark each enabled local model on representative extraction, summary,
  bounded edit, and seeded-review tasks before granting its task permissions.

## Completion

Summarize policy, benchmark, validation, and caveats; propose Stage 6 and wait
for approval.
