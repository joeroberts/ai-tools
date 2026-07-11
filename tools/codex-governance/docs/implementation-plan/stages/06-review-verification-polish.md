# Stage 6 Prompt: Review, Verification, And Polish

Use this prompt after Stage 5 is complete and approved.

## Objective

Review and verify the completed Jira-backed toolkit.

## Scope

- Launch an independent reviewer; persist or link findings.
- Remediate in-scope findings, then use a fresh reviewer. Bound normal review
  and verification cycles to two; require explicit approval for a third.
- Launch an independent verifier only after blocking review findings are clear.
- Escalate unresolved disagreement, ticket drift, or validation failure rather
  than retrying indefinitely.
- Close every completed agent and fail finalization if one remains open without
  an approved exception.
- Update README and concise Jira handoff if needed.
- Prepare a versioned release manifest and migration notes; do not publish or
  tag them without explicit approval.

## Validation

Run smoke, shell, JSON, and diff checks; validate generic and offline-export
fixtures; verify deferred profiles remain documented; verify no default packet,
evidence, or session-handoff directories are generated; confirm governed
Ollama policy, cache, and execution-ledger closure checks pass; confirm no
forbidden remote or destructive action occurred; verify dry-run and check-mode
release synchronization behavior.

## Completion

Summarize review and verification decisions, validation results, caveats, and
an optional local commit plan. Do not push, publish, release, deploy, or update
remote Jira or PR state without explicit approval.
