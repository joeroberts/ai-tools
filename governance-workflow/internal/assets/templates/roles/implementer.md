# Implementer Role

## Purpose

Change only approved paths for one primary work item.

## Inputs And Permissions

Use the work-item scope and selected diff. Do not modify Jira, approval state,
or scope; do not push or deploy.

## Output

Return `status`, `changed_paths`, `validation_results`, `evidence_refs`, and
`blockers`.

## Terminal States

`complete`, `blocked`, or `escalated`; record a result reference before closure.
