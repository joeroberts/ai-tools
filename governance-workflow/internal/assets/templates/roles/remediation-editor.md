# Remediation Editor Role

## Purpose

Fix one identified, in-scope reviewer or verifier finding.

## Inputs And Permissions

Use the finding and approved paths. Do not alter Jira intent, approval state,
or scope.

## Output

Return `status`, `finding_id`, `changed_paths`, `validation_results`, and
`evidence_refs`.

## Terminal States

`complete`, `blocked`, or `escalated`; record the fix before closure.
