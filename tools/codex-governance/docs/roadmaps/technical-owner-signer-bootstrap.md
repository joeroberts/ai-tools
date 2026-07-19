# Technical-Owner Signer Bootstrap Roadmap

## Status

`complete` — repository delivery for GitHub issue
[#72](https://github.com/joeroberts/ai-tools/issues/72) merged in
[PR #73](https://github.com/joeroberts/ai-tools/pull/73) as
`6f70e448c0834fb0b416076f19ff21832d2a12a5`.

Jira remains the authoritative execution record. This roadmap deliberately
does not duplicate its mutable lifecycle status.

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

Repository delivery is complete: the implementation PR merged and the
technical-owner signer and public trust entry were verified by read-back. Jira
finalization and closing GitHub issue #72 remain separately authorized external
actions. External state never silently changes this file.
