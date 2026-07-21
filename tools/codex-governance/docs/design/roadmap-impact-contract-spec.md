# Roadmap-Impact Contract Specification

## Contract

`governance.yml` gains a versioned roadmap-adoption block. Required mode names
exactly one repository-relative structured roadmap and stable identity. Paths
must be clean, contained by the repository, and free of machine-local, Jira,
issue-number, and repository-specific code assumptions.

Each normalized work item and ticket-plan subtask includes either:

```json
{"mode":"required","roadmap_id":"...","canonical_path":"...","phase":"...","transition":"start"}
```

or:

```json
{"mode":"not-applicable","reason":"bounded explanation"}
```

## Allowed Paths

- `governance.yml`
- `cmd/codex-governance`
- `internal/assets`
- `internal/config`
- `internal/initializer`
- `internal/jira`
- `internal/implementation`
- `internal/agentplan`
- `internal/roadmap`
- `internal/ticketplan`
- `internal/validate`
- `internal/workitem`
- `testdata`
- `docs/decisions`
- `docs/design`
- `docs/roadmaps`

Do not change GitHub workflow files, hosted repository settings, root guidance,
or unrelated product paths.

## Review Budget

The total review budget is 42 changed files, 3600 changed lines, and roadmap configuration and contracts, validation fixtures, non-destructive adoption assets, migration fixtures, planning and preflight enforcement, entry-gate fixtures, transition preview and digest binding, state-evidence fixtures, commit and publication gates, lifecycle fixtures, finalization enforcement, non-interactive check fixtures. Each declared slice must stay within its own budget.

## Declared Implementation Slices

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {"id":"roadmap-impact-contract","phase":"Phase 1","change_class":"high-risk","dependencies":[],"allowed_paths":["governance.yml","internal/assets","internal/config","internal/ticketplan","internal/workitem","internal/validate","testdata","docs/decisions","docs/design","docs/roadmaps"],"review_budget":{"max_changed_files":10,"max_changed_lines":800,"components":["roadmap configuration and contracts","validation fixtures"]}},
    {"id":"roadmap-adoption-assets","phase":"Phase 2","change_class":"standard","dependencies":["roadmap-impact-contract"],"allowed_paths":["internal/assets","internal/initializer","internal/config","testdata","docs/design"],"review_budget":{"max_changed_files":7,"max_changed_lines":600,"components":["non-destructive adoption assets","migration fixtures"]}},
    {"id":"roadmap-entry-enforcement","phase":"Phase 3","change_class":"high-risk","dependencies":["roadmap-adoption-assets"],"allowed_paths":["internal/agentplan","internal/ticketplan","internal/validate","internal/implementation","internal/roadmap","testdata"],"review_budget":{"max_changed_files":8,"max_changed_lines":700,"components":["planning and preflight enforcement","entry-gate fixtures"]}},
    {"id":"roadmap-transition-evidence","phase":"Phase 4","change_class":"high-risk","dependencies":["roadmap-entry-enforcement"],"allowed_paths":["cmd/codex-governance","internal/roadmap","internal/validate","internal/workitem","testdata"],"review_budget":{"max_changed_files":7,"max_changed_lines":650,"components":["transition preview and digest binding","state-evidence fixtures"]}},
    {"id":"roadmap-publication-gates","phase":"Phase 5","change_class":"high-risk","dependencies":["roadmap-transition-evidence"],"allowed_paths":["internal/implementation","internal/validate","internal/roadmap","testdata"],"review_budget":{"max_changed_files":5,"max_changed_lines":450,"components":["commit and publication gates","lifecycle fixtures"]}},
    {"id":"roadmap-finalization-check","phase":"Phase 6","change_class":"high-risk","dependencies":["roadmap-publication-gates"],"allowed_paths":["cmd/codex-governance","internal/jira","internal/implementation","internal/validate","internal/roadmap","testdata"],"review_budget":{"max_changed_files":5,"max_changed_lines":400,"components":["finalization enforcement","non-interactive check fixtures"]}}
  ]
}
```

## Rules

Reject missing declarations, empty reasons, invalid configuration, absolute or
escaping paths, absent or duplicate identities, stale digests, skipped states,
replayed or cross-repository records, and contradictory aggregate state.

Phase 4 introduces the required `prior_digest` and resulting-digest evidence
binding; Phase 1 declarations intentionally do not infer or validate a digest.

A required transition must be in the approved paths, review budget, and
planning baseline. A completion transition must appear in the reviewed diff
before commit, publication, or finalization; non-completing work does not churn
roadmap state.

The preview helper validates bindings then prints the resulting digest. It
never edits files or reads or writes Jira, GitHub, or hosted CI.

Planning validates structure; preflight validates state and baseline; commit,
publication, and Jira-finalization revalidate bindings. A stable
non-interactive check supports later read-only CI adoption.

## Architecture Decision

Use `docs/decisions/0004-roadmap-impact-contract.md` for the durable
repository-neutral roadmap-impact authority boundary, transition binding, and
explicit migration decision.

Every declared implementation slice, including `roadmap-adoption-assets`, uses
`docs/decisions/0004-roadmap-impact-contract.md`; none may substitute a
`No ADR needed` rationale.
