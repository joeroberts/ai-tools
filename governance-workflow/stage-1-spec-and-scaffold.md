# Stage 1 Prompt: Spec And Scaffold

Use this prompt after the master prompt has been approved.

## Objective

Create the canonical spec and base scaffold for `codex-governance`.

## Scope

- Create `docs/governance-toolkit-spec.md` from the supplied canonical spec.
- Create the canonical directory structure.
- Create top-level `README.md` with required sections.
- Create all files under `templates/` with required headings and instructional
  placeholder text.
- Create `templates/validation-profile.md`.
- Create initial files under `profiles/`:
  - `generic.md`
  - `terraform-module.md`
  - `node-fullstack-k8s.md`
  - `python-vue-fullstack-k8s.md`
- Create minimal `.codex/summaries/` session handoff for Stage 1.

## Do Not Create Yet

- Do not create the Codex skill implementation beyond empty directories if
  needed.
- Do not create scripts beyond placeholders.
- Do not create fixtures or examples beyond placeholders.

## Validation

- Verify all Stage 1 files exist.
- Verify each template has its required headings.
- Verify each required profile file exists and names its validation contract.
- Run markdown/diff checks available in the repo.
- Record validation results in the Stage 1 handoff.

## Completion

At the end of Stage 1:

- summarize files created
- list validation run and results
- identify gaps
- propose Stage 2 plan
- wait for approval before Stage 2
