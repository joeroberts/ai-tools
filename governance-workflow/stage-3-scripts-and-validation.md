# Stage 3 Prompt: Scripts And Validation

Use this prompt after Stage 2 is complete and approved.

## Objective

Create governance initialization and validation scripts with smoke tests.

## Scope

- Implement `scripts/init-governance.sh`.
- Implement `scripts/check-governance-packet.sh`.
- Create `tests/fixtures/`:
  - `valid-packet/`
  - `missing-roadmap/`
  - `unapproved-roadmap/`
  - `missing-review-scope/`
  - `missing-adr-reference/`
  - `missing-verifier-artifact/`
  - `known-profile-missing-validation/`
  - `unknown-profile/`
- Create `tests/smoke-test.sh`.
- Ensure scripts are executable.

## Script Requirements

- Use POSIX shell or Bash with clear shebangs.
- If Bash, use `set -euo pipefail`.
- Quote variables.
- Avoid unsafe globbing.
- Print clear errors.
- Return non-zero on validation failure.
- Avoid remote mutations.
- Support profile-aware initialization and validation.
- Support docs-root overrides without requiring repo-specific values to match
  across repos.

## Validation

- Run shell syntax checks.
- Run ShellCheck if available.
- Run smoke tests.
- Prove valid fixture passes.
- Prove invalid fixtures fail as expected.
- Prove known profiles are accepted and unknown profiles fail or warn according
  to mode.
- Record validation results in a Stage 3 handoff.

## Completion

At the end of Stage 3:

- summarize files created
- list validation run and results
- identify gaps
- propose Stage 4 plan
- wait for approval before Stage 4
