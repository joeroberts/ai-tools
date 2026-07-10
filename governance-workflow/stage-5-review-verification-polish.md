# Stage 5 Prompt: Review, Verification, And Polish

Use this prompt after Stage 4 is complete and approved.

## Objective

Run the governed manager loop for the completed toolkit and polish the final
deliverable.

## Scope

- Create a review scope for the toolkit work.
- Launch a reviewer subagent.
- Persist reviewer findings.
- If reviewer finds issues, dispatch a code-editor subagent and rerun a fresh
  reviewer.
- Repeat until reviewer has no blocking findings.
- Launch verifier subagent only after reviewer gate is green.
- Persist verifier evidence.
- If verifier fails, fix the issue or environment within approved scope and
  rerun a fresh verifier.
- Repeat until verifier passes or only accepted caveats remain.
- Close all completed subagents.
- Update final handoff and README if needed.

## Validation

- Run all script smoke tests.
- Run shell syntax checks.
- Run ShellCheck if available.
- Run governance validator against fixtures and example.
- Verify required files and headings exist.
- Confirm no forbidden remote/deploy/cloud/destructive actions occurred.

## Completion

At the end of Stage 5:

- summarize reviewer decision
- summarize verifier decision
- list final validation results
- list remaining caveats or non-goals
- propose local commit plan if commits are approved
- do not push, publish, install globally, tag, release, deploy, or update remote
  PRs without explicit approval
