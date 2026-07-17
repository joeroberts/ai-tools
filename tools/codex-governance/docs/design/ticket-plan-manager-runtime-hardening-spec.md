# Ticket-Plan Manager Runtime Hardening Specification

## Runtime Contract

The CLI propagates one signal-aware context through ticket-plan orchestration
and every hosted manager call. `jira plan decompose` and `jira plan generate`
require positive Go-duration values for `--manager-timeout` and
`--manager-wait-delay`. Invalid, missing, zero, or negative values fail before
agent lifecycle state is opened.

Each manager call derives a deadline from the configured timeout. The Codex
command uses that context and sets `exec.Cmd.WaitDelay` to the configured
wait-delay so cancellation cannot wait indefinitely for the process or for
inherited output pipes. Context cancellation returns through `runRole`, which
persists the existing private error artifact and terminal `failed` and `closed`
ledger events.

The CLI uses `signal.NotifyContext` for `os.Interrupt` and `SIGTERM` and stops
the signal subscription before returning. This bounded path does not claim
recovery after an uncatchable signal or parent-process loss.

## Constraint-Aware Output Schema

Post-assignment generation builds its schema from the loaded approved
constraints. Every field that has an approved finite value set uses that set
as a JSON Schema `enum`. Arrays declare `minItems` and `maxItems` derived from
their approved finite pools or exact expected counts. The schema must not ask a
manager to satisfy the obsolete rule that every path contains `/`.

At minimum, allowed-path items use the exact approved path pool, component
items use the approved component pool, Subtask IDs use the assigned ID set,
and dependency items use the assigned ID set. Array maxima are no larger than
their corresponding approved pools. The existing local application of
constraints and contract-aware validation remain authoritative.

Decomposition remains manager-authored before assignment. Its schema uses a
bounded repository-relative path shape that accepts root-level entries, rejects
commas, traversal, newlines, wildcards, and aggregated values, and bounds every
array. It does not claim approved-value enums before constraints exist.

## Diagnostics

Hosted Codex runs add `--json`. Standard output is written to an owner-only
JSONL diagnostic, standard error to a separate owner-only log, the exact schema
to an owner-only artifact, and the final structured message to an owner-only
result file. Successful and failed lifecycle records retain references to the
diagnostic set without embedding prompts, source bodies, credentials, or raw
diagnostic content in console output.

Diagnostic directories and files use mode `0700` and `0600`. Existing files
are never overwritten. User-facing errors identify diagnostic paths and redact
secret-like values before persistence or display.

## Allowed Paths

Implementation is limited to `docs/design`, `docs/roadmaps`,
`internal/agentplan`, `internal/cli`, and `testdata/ticket-plans`.

## Review Budget

The total review budget is 12 changed files, 900 changed lines, constraint-aware manager schema and fixtures, supervised manager lifecycle diagnostics.
The change has exactly two components and must remain within this budget.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "constraint-aware-manager-schema",
      "phase": "Phase 1",
      "change_class": "standard",
      "dependencies": [],
      "allowed_paths": [
        "internal/agentplan",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 6,
        "max_changed_lines": 450,
        "components": [
          "constraint-aware manager schema and fixtures"
        ]
      }
    },
    {
      "id": "supervised-manager-lifecycle",
      "phase": "Phase 2",
      "change_class": "standard",
      "dependencies": [
        "constraint-aware-manager-schema"
      ],
      "allowed_paths": [
        "internal/agentplan",
        "internal/cli",
        "testdata/ticket-plans"
      ],
      "review_budget": {
        "max_changed_files": 6,
        "max_changed_lines": 450,
        "components": [
          "supervised manager lifecycle diagnostics"
        ]
      }
    }
  ]
}
```

## Technical Acceptance Criteria

- Generate schema fixtures that accept `AGENTS.md`, `testdata`, and approved
  nested paths.
- Reject aggregated path strings, values outside the approved pool, arrays over
  their approved bounds, and protocol-like path strings larger than the
  accepted finite values.
- Persist JSONL, stderr, schema, and result diagnostics with owner-only modes.
- Preserve terminal Codex usage events when Codex emits them.
- Cancel a deterministic fake manager on deadline and on propagated context
  cancellation without dispatching a replacement.
- Bound a deterministic inherited-pipe fixture with `WaitDelay`.
- Record `started`, `failed`, and `closed` events for controlled timeout and
  cancellation failures.
- Run `make test`, `make vet`, `make build`, and `git diff --check`.

## Architecture Decision

No ADR needed: this preserves the accepted hosted-manager ownership contract
while hardening its schema and synchronous lifecycle. Whether to eliminate the
post-assignment manager or bind and reuse the earlier decomposition is a
separate architectural decision tracked by GitHub issue #59 and requiring its
own product sources, contract-migration analysis, and ADR.

## Non-Goals

- Deciding or implementing manager elimination or decomposition reuse.
- Adding automatic retries or changing the two-cycle semantic review policy.
- Changing local reviewer/verifier execution or model residency.
- Changing remote-write authorization.
