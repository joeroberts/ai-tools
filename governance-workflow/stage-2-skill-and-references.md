# Stage 2 Prompt: Skill And Reference Prompts

Use this prompt after Stage 1 is complete and approved.

## Objective

Create the `governed-packet-workflow` Codex skill and reusable prompt
references.

## Scope

- Create `skills/governed-packet-workflow/SKILL.md`.
- Create:
  - `references/manager-loop.md`
  - `references/reviewer-prompt.md`
  - `references/verifier-prompt.md`
  - `references/code-editor-prompt.md`
- Ensure the skill links directly to relevant templates and references.
- Encode roadmap-first planning, packet lifecycle, review-scope checkpoints,
  reviewer/verifier/code-editor loops, subagent closure, and commit separation.
- Encode repo profile selection and profile-specific validation contracts.
- Encode optional `execution-state/` reconciliation before proposing next work.
- Encode docs-root overrides, generated artifact rules, deployment boundaries,
  and external integration guardrails.

## Validation

- Verify required reference files exist.
- Verify `SKILL.md` has clear trigger conditions and workflow instructions.
- Verify `SKILL.md` tells Codex to discover repo type before selecting
  validation commands.
- Run skill validation if available.
- Run markdown/diff checks available in the repo.
- Record validation results in a Stage 2 handoff.

## Completion

At the end of Stage 2:

- summarize files created
- list validation run and results
- identify gaps
- propose Stage 3 plan
- wait for approval before Stage 3
