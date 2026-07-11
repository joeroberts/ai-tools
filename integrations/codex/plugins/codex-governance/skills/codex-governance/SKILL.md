---
name: codex-governance
description: Route governed engineering work through the local codex-governance CLI. Use for roadmap status, configuration checks, offline work-item validation, and governed local-model policy checks.
---

# Codex Governance

Use this plugin as an adoption layer for the installed `codex-governance` CLI.
The CLI is authoritative for validation and state; do not reproduce its rules
from memory or paste its long design documents into context.

## Preflight

1. Run `codex-governance --help`. If unavailable, state that the CLI must be
   installed; do not substitute ad hoc validation.
2. Locate the target repository and check for `governance.yml`.
3. Run `codex-governance config check --repo-root <repo>` before governed work.
   Do not run `init` or modify configuration without approval.

## Routing

- For roadmap state, run `roadmap status` or `roadmap check` with the
  repository's structured roadmap path.
- For a governed change, run `validate-work-item` only with the normalized
  work-item JSON, offline ticket export, and explicit Git range supplied by the
  repository or user.
- For local-model work, use only `ollama policy init` or `ollama run` commands
  allowed by the installed policy. Do not call Ollama directly.
- Treat a nonzero CLI result as a blocking signal unless the user explicitly
  approves a documented exception.

## Context And Escalation

Read local design documents only when a CLI error or a durable design decision
requires it. Prefer CLI summaries and JSON output when available. Escalate
ticket-intent disputes, source drift, missing approvals, open agents, or policy
violations instead of working around them.

## Boundaries

Never use this plugin to write Jira, push, merge, publish, deploy, access
secrets, download models, or bypass a policy check without explicit approval.
