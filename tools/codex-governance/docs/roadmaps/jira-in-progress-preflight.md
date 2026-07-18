# Jira In-Progress Preflight Delivery Roadmap

## Phase 1: Capture Signed Status Evidence

Capture and validate Story and Subtask source status in signed Jira exports.
This slice must complete before preflight starts relying on status evidence.

## Phase 2: Enforce the Preflight Gate

Require the verified primary Subtask status to be exactly `In Progress` before
preflight creates artifacts or dispatches an adapter. Align `AGENTS.md` with
the required Jira transition, read-back, and active-ticket scope separation.

## Delivery Order

`in-progress-preflight-gate` depends on `signed-status-evidence`.
