# Descendant Remediation Adoption Phase 2 PRD

## Status

`Jira-planning`

The user authorized Phase 2 planning on 2026-07-19. GitHub issue
[#69](https://github.com/joeroberts/ai-tools/issues/69) is the backlog source.
Jira Story [REK-46](https://rekonlabs.atlassian.net/browse/REK-46) and primary
Subtask [REK-47](https://rekonlabs.atlassian.net/browse/REK-47) were created
from the approved ticket plan and read back in `To Do`. Phase 1 and the
technical-owner signer prerequisite are complete. Implementation remains
unauthorized until the linked planning baseline is committed and REK-47 is
explicitly transitioned to exactly `In Progress`.

## Problem

Phase 1 defined the separately signed `AdoptionRecord`, but the CLI cannot yet
prove that a concrete remediation descendant satisfies that contract or persist
an approved signed record. REK-40 therefore remains bound to its immutable
predecessor commit even though its final descendant received complete-range
review and verification.

## Goal

Add one local, explicit adoption transition that validates a single approved
linear descendant against fresh source, exact scope, budgets, guidance,
deterministic checks, and complete-range review evidence, then atomically
persists a technical-owner-signed adoption record without changing the
predecessor run or performing publication.

## Product Outcomes

- Preview proves every required binding and renders the exact unsigned payload
  without trusted persistence or remote effects.
- Approved execution repeats validation, verifies the fixed technical-owner
  signer, signs the previewed payload, and stores one immutable record.
- The predecessor run and candidate repository remain byte-for-byte unchanged.
- Duplicate, replayed, stale, rewritten, merged, unrelated, out-of-scope, or
  incompletely reviewed candidates fail before trusted persistence.
- Phase 3 can later consume the record without inventing another successor
  identity or weakening existing publication authorization.

## Acceptance Criteria

- Validation binds the exact repository, work item, predecessor run, original
  base, predecessor commit, candidate branch and commit, and adopted range.
- The candidate is a clean linear descendant of both the original base and
  predecessor commit, with no merge commit or out-of-scope intermediate commit.
- Fresh signed Jira evidence, normalized work item, task bundle, current
  configuration, repository guidance, and machine-readable deterministic-check
  evidence agree exactly.
- The complete original-base-to-candidate diff satisfies approved paths, file,
  line, component, generated-file, and lockfile limits.
- Independent reviewer and verifier evidence is passing, distinct, unaltered,
  and bound to that complete diff.
- Preview creates no signed record, registry entry, audit event, Jira or GitHub
  write, commit, push, pull request, release, or deployment.
- Approved execution uses the trusted `technical-owner` identity, persists an
  owner-only signed envelope atomically and non-overwriting, and rejects replay
  or conflicting successor identity.
- Failure and retry cannot alter the predecessor or leave a trusted partial
  record. Diagnostics identify safe remediation without exposing private keys,
  credentials, prompts, raw model output, or unnecessary local paths.
- Two repository/Jira/path fixtures prove repository-neutral validation and
  cross-repository replay rejection.
- Focused tests, `make test`, `make vet`, `make build`, `git diff --check`, and
  independent exact-diff reviewer/verifier evidence pass.

## Non-Goals

- Changing publication authorization, push, or pull-request creation; Phase 3
  owns successor consumption.
- Publishing or otherwise resuming REK-40 in this phase.
- Chained adoption, arbitrary history repair, rebases, merges, cherry-picks, or
  rewritten and unrelated commit ranges.
- Mutating or upgrading the predecessor implementation run.
- Automatically writing Jira or GitHub, committing, pushing, creating a pull
  request, merging, releasing, or deploying.
- Implementing bounded lifecycle authorization from #22.
