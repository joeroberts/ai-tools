# Synchronous Implementation Start PRD

## Status

This is the product source for the bounded headless execution correction.
GitHub issue #111 separately tracks a durable execution refactor as a future
feature enhancement.

## Problem

`implementation start` dispatches `codex exec` asynchronously and then exits.
In environments that reap descendants when the launcher exits, the approved
implementation process disappears before producing a result. Its terminal
failure is retained only in the adapter's in-memory task map and is therefore
lost to later reconciliation.

## Goal

Make the current CLI execution path reliable by keeping the launching command
alive through child completion and persisting the terminal run outcome before
returning control to the caller.

## Product Outcomes

- The launching CLI invocation owns the child for its complete lifetime.
- Successful runs are immediately ready for governed verification.
- Failed runs preserve an actionable terminal state and diagnostics.
- The change does not imply detached or restart-safe supervision.

## Acceptance Criteria

- The start command remains active until the approved child exits.
- A successful child produces a persisted `implementation-complete` run and a
  readable result reference.
- A failed child produces a persisted `escalated` run, a nonzero command exit,
  and actionable diagnostic locations.
- Deterministic tests exercise both terminal paths without external services.

## Non-Goals

- Implementing the durable execution architecture described by GitHub issue #111.
- Surviving termination of the launching CLI or its host.
- Adding automatic retry, redispatch, or remote mutation behavior.
