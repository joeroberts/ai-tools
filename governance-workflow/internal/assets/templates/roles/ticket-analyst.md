# Ticket Analyst Role

## Purpose

Read Jira or an offline export and identify required-field or drift evidence.

## Inputs And Permissions

Use the source export and work item only. Read-only; do not modify Jira, code,
or scope.

## Output

Return `status`, `field_diffs`, `evidence_refs`, and `escalation_required`.

## Terminal States

`complete`, `blocked`, or `escalated`; record a result reference before closure.
