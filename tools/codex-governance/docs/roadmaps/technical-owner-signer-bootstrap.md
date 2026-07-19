# Technical-Owner Signer Bootstrap Roadmap

## Status

`Jira-planning` — GitHub issue
[#72](https://github.com/joeroberts/ai-tools/issues/72) is the backlog source.
Jira Story [REK-43](https://rekonlabs.atlassian.net/browse/REK-43) and primary
Subtask [REK-44](https://rekonlabs.atlassian.net/browse/REK-44) were read back
as `To Do`. Implementation remains unauthorized until the planning baseline is
committed and REK-44 is explicitly transitioned to `In Progress` and read back.

## Phase 1: Fixed-Role Signer Bootstrap

Add a no-write preview and explicitly approved command that creates one
non-overwriting owner-only technical-owner signer outside the repository.
Validate path containment, symlink resolution, directory permissions, fixed
role, destination absence, and existing trust before generation.

## Phase 2: Trust Onboarding And Verification

Add only the generated public key record to repository policy, reload the
signer and configuration, and verify key ID, role, algorithm, and public key.
Remove newly created signer material when policy persistence fails and surface
cleanup or read-back failures as blockers.

## Delivery Order

```text
fixed-role signer bootstrap
  -> trust onboarding and verification
  -> #69 Phase 2 planning resumes
```

Both phases form one Jira implementation Subtask with exactly two review
components. This prerequisite does not implement adoption validation,
persistence, or publication consumption.

## Validation Gates

Run focused deterministic tests, `make test`, `make vet`, `make build`, and
`git diff --check`. Require independent reviewer and verifier evidence for the
exact diff before commit or publication.

## Completion Rule

This roadmap becomes `complete` only after the implementation PR merges, the
technical-owner signer and public trust entry are verified by read-back, and
the Jira Story/Subtask finalize. External state never silently changes this
file.
