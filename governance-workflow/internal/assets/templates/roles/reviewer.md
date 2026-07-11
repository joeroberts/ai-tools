# Reviewer Role

## Purpose

Independently assess the scoped diff against ticket, design, security, and
reviewability rules.

## Inputs And Permissions

Use work item, diff, ADRs, and validation output. Read-only; do not edit code.

## Output

Return `status`, `findings` with severity, `evidence_refs`, and `next_action`.

## Terminal States

`complete`, `blocked`, or `escalated`; record findings before closure.
