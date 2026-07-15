# Governed Ollama Model Lifecycle Work Item

## Status

Jira planning is complete: GitHub issue #12, Story `REK-7`, and primary
Subtask `REK-8`. This work item is a draft until reviewed and committed.

## Scope

Add the policy-checked lifecycle operations required by the Makefile:

- `make unload-reviewer-model`
- `make unload-verifier-model`
- `make load-reviewer-model`
- `make load-verifier-model`

The current owner-only runtime policy supplies the fixed operational roles:
`gemma4:12b-mlx` reviewer and `devstral:24b` verifier. Unload uses the
governed no-prompt `keep_alive: 0` request and verifies `loaded=false`; load
prewarms through the governed gateway and verifies `loaded=true`.

## Allowed Paths

- `AGENTS.md`
- `Makefile`
- `README.md`
- `docs/design/`
- `internal/cli/`
- `internal/ollama/`

## Non-Goals

- Raw user-facing Ollama API or CLI calls.
- Unrestricted shell execution, prompts, credentials, or automatic stopping.
- Guessing models or asking users to unload models manually.
- Changing the owner-only policy file format or model selection.

## Acceptance Criteria

- The four Make targets work with no required model or policy arguments.
- The governed CLI rejects a model not allowlisted for its lifecycle role.
- Stop and load issue no prompt content and verify the resulting residency.
- `AGENTS.md` directs agents to use the Make targets and status checks before
  switching between high-memory reviewer and verifier models.
- Focused tests cover successful load/stop, rejection, and failed verification.

## Review Budget

Maximum 8 files, 500 changed lines, 4 components: Makefile, CLI, Ollama
gateway, and documentation/tests. No ADR needed: bounded operational workflow.
