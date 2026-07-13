# Specification: Implementation-Agent Orchestration

## Status And Precedence

This specification implements the approved
[implementation-agent PRD](implementation-agent-prd.md). The
[canonical specification](canonical-spec.md) remains authoritative if a
conflict exists.

## Architecture

The Go CLI is the governance control plane. Execution providers implement an
adapter interface; headless Codex is the first provider. A user may choose a
local-LLM provider for code execution or remediation only if the governed local
model policy authorizes its exact provider/model/adapter/config stack, role,
task class, bundle size, and concurrency, and preflight verifies its passing
versioned qualification record. Initial code-edit eligibility is limited to
`scoped-code-edit` and `finding-bound-remediation`; high-risk work is
ineligible. Adapters execute work but cannot approve policy, alter work-item
intent, or perform remote actions outside an authorization verified by the
control plane.

```text
work item + fresh ticket baseline
  -> deterministic preflight
  -> disposable worktree + versioned task bundle
  -> execution adapter
  -> deterministic diff/scope checks
  -> reviewer/verifier/remediation loop
  -> local commit gate
  -> remote-publish authorization gate
  -> push and PR adapter -> closure
```

## Run Contract

An `implementation-run` is a versioned, immutable-at-start record containing:

- run ID, work-item key, source digests, adapter ID and version;
- worktree identity, base SHA, branch name, and approved path/budget snapshot;
- task-bundle digest and result-reference directory;
- lifecycle state, transition times, attempt and review-cycle counters;
- command outcomes, diff SHA, commit SHA, agent task ID, and redacted summary;
- remote-publish authorization, if granted, and resulting PR URL or identifier.

For `offline-export` source mode, the private task bundle retains the signed
export envelope. The run records its envelope digest, issuer-key identity and
registry version, capture time, and applied policy age limit. Governed runs
accept only envelopes signed by a configured, unrevoked issuer key; unsigned
exports are test fixtures only. The default maximum age is 24 hours,
configurable per repository. Preflight and adapter dispatch reject expired,
unverifiable, revoked-issuer, or bundle-mismatched source evidence.

The owner-only runtime ledger is structured and hash-linked. Each event records
run and event IDs, UTC time, actor and role, CLI/schema version, lifecycle
state, relevant policy/evaluation/task-bundle/source digests, gate decision,
and preceding-event hash. It records redacted references plus digests and
normalized outcomes for source checks, commands, diff/commit, adapter results,
findings, rebaseline/exception records, and publication authorization.

Closure writes a final evidence manifest containing the terminal event hash.
Missing, altered, or unverifiable required evidence escalates the run and
blocks closure. The ledger is tamper-evident within the governed local runtime,
not tamper-proof against a party with filesystem control. Retain test evidence
for the full test program; establish production retention before distribution.

The task bundle is versioned and includes only the normalized work item, fresh
ticket baseline, allowed paths, required commands, ADR references, repository
guidance, and a structured-result schema. It must not include credentials or
unbounded local logs.

All signed governance records use a common envelope: canonical JSON payload,
SHA-256 payload digest, `key_id`, `algorithm`, signer role, issuance time,
optional expiry, and signature. The initial algorithm is Ed25519 over the
payload digest. A versioned repository policy maps trusted public keys to the
Jira-owner, technical-owner, repository-owner, and export-issuer roles. The CLI
verifies key trust, role, expiry, signature, record version, and revocation at
each gate. Private keys are never stored in the repository or runtime ledger;
tests use ephemeral fixture keys only.

## State Machine

| State | Entry condition | Exit condition |
| --- | --- | --- |
| `preflight` | approved run requested | deterministic gates pass or escalate |
| `queued` | task bundle and worktree created | adapter accepts dispatch |
| `running` | adapter task ID recorded | adapter reports terminal execution result |
| `implementation-complete` | result captured | post-run scope/diff checks pass |
| `review` | valid diff available | findings clear or remediation/escalation selected |
| `verification` | blocking review findings clear | checks pass or remediation/escalation selected |
| `remediation` | named in-scope findings selected | changed result returns to review |
| `ready-to-commit` | verification passes | local commit gate passes |
| `locally-committed` | exact commit SHA recorded | remote authorization recorded or local handoff closes |
| `ready-for-remote-approval` | user requests publication | authorization validates or run remains local |
| `pushed` | authorized exact SHA is pushed to allowed branch | PR creation succeeds or escalates |
| `PR-created` | authorized PR identifier recorded | closure checks pass |
| `escalated` | a gate cannot safely continue | human-approved next action creates a new transition |
| `closed` | all dispatched agents closed and evidence complete | terminal |

`escalated` and `closed` are terminal for automatic execution. A locally
committed run may close without remote publication. The CLI must never infer a
remote authorization from a local-commit approval.

## Adapter Contract

The execution adapter must support `Start`, `Status`, `Cancel`, and `Result`.
`Start` accepts a task-bundle path and worktree path and returns an opaque task
ID. `Status` reports only lifecycle facts. `Result` returns structured output
that is stored as an immutable local result reference. The adapter must not
receive remote credentials and must run with the control plane's declared
worktree and path constraints.

The initial headless Codex adapter must execute in the disposable worktree,
use a structured output schema, and expose a recoverable task identifier. A
local-LLM adapter must submit jobs only through the governed gateway; it returns
a patch or structured edit result for the control plane to stage in the
worktree and check deterministically. A fake adapter is required for
deterministic lifecycle and recovery tests.

The headless Codex adapter runs a non-ephemeral `codex exec --json` process and
records the `thread_id` emitted by its `thread.started` event together with its
PID, command fingerprint, worktree path reference, and result path. Status
checks use the process while it exists; after a restart the adapter may resume
only the recorded session in the recorded worktree. If either identity cannot
be verified, the run becomes `escalated` and is never silently re-dispatched.

## Enforcement And Recovery

Preflight validates work-item structure, fresh ticket baseline, source drift,
agent closure, policy, repository state, worktree availability, paths, and
review budget. Post-run validation compares the exact base and resulting SHAs
and records command results. The control plane checks scope before review,
before commit, and before any remote action.

On restart, a run in `running` is reconciled through `Status` using its stored
task ID. Unknown or unavailable adapter state becomes `escalated`; the CLI
must not automatically start a replacement task. Retry limits are stored in the
run record and include failed dispatches and remediation cycles.

## Commit And Remote Publication

The local commit command is available only from `ready-to-commit`, only in the
run worktree, and only when the approved work item enables local commits.

A remote-publish authorization is a signed or locally auditable record with a
work-item key and run ID, repository identity, remote name and URL fingerprint,
target ref, exact commit SHA and expected base SHA, approver identity, issuance
time, expiry, record version and digest, and allowed operations (`push`,
`create-pr`). One record may authorize both operations, but each is explicit,
independently verified, and consumed separately; `create-pr` also binds its PR
target branch. The CLI must reject expired, mismatched, broadened, or reused
authorizations, including changed remotes, refs, and SHAs. It must reject
force-pushes and protected/default-branch targets.

## Security And Data Handling

The runtime ledger and result store are owner-only. Store redacted summaries,
digests, and references by default; do not persist credentials, raw prompts,
or unrestricted logs. Worktree cleanup occurs only after terminal closure and
must retain the evidence needed to reproduce the exact diff and commit.

The implementer and remediation editor reuse the same owner-only local-model
policy, pinned model ID, benchmark evidence, input-size limit, and no-fallback
behavior as reviewer and verifier. Their `code-edit` authorization is separate
from `ticket-plan-review`: it must demonstrate scoped patch generation,
forbidden-path rejection, deterministic validation, and redaction before it is
enabled. The existing review benchmark alone does not authorize code edits.

## Evaluation Qualification

Each qualification record binds provider, exact model ID/revision, adapter
version, tool-permission/config version, prompt/task-bundle schema and version,
benchmark-corpus and harness versions/hashes, role, task class, metrics,
thresholds, evaluator identity, and approval. Qualification is not inherited
across roles or task classes. The corpus includes normal tasks and adversarial
cases for forbidden paths, source drift, invalid output, validation failure,
authorization attempts, and sensitive-data redaction. All safety-control cases
must pass. A versioned baseline corpus determines the recorded non-safety
task-success threshold before any provider is enabled. Any bound-input change,
relevant incident, or scheduled review requires requalification.

## Required Tests

- State-transition, invalid-transition, and terminal-state tests.
- Task-bundle schema, digest, and secret-exclusion tests.
- Worktree isolation, allowed-path, review-budget, and exact-SHA tests.
- Fake-adapter start/status/result/recovery tests, including host crash and
  unknown-task escalation.
- Local-model policy, benchmark, pinned-ID, and unavailable-provider tests;
  verify no fallback or model escalation occurs.
- Review-loop and finding-bound remediation tests.
- Local-commit gate and remote authorization mismatch, expiry, reuse,
  force-push, protected-branch, and PR-operation tests.
