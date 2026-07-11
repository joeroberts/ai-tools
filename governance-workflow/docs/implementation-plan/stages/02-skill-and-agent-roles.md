# Stage 2 Prompt: Skill And Agent Roles

Use this prompt after Stage 1 is complete and approved.

## Objective

Create the `governed-jira-workflow` skill and role directives.

## Scope

- Create `skills/governed-jira-workflow/SKILL.md`.
- Create references for manager, ticket analyst, implementer, reviewer,
  verifier, and remediation editor.
- Encode Jira authority, read-only ticket drift checks, one primary subtask per
  PR, review exceptions, ADR policy, CI evidence links, and disagreement rules.
- Give every role purpose, inputs, permissions, structured output, terminal
  states, escalation conditions, and closure criteria.
- Require the manager to record agent lifecycle and closure in the local runtime
  ledger, and block finalization for open agents without an approved exception.
- Encode the governed local-model gateway boundary and advisory versus enforced
  controls.

## Validation

- Verify all role references exist and have the required contract sections.
- Verify the skill links to Jira templates, configuration, and references.
- Run available Markdown and diff checks.
- Prepare concise Stage 2 Jira handoff text; do not post it without approval.

## Completion

Summarize work and validation, propose Stage 3, and wait for approval.
