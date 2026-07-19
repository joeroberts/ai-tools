# Descendant Remediation Adoption PRD

## Status

GitHub issue [#69](https://github.com/joeroberts/ai-tools/issues/69) is the
approved backlog source. Phase 1 is planned in Jira Story
[REK-41](https://rekonlabs.atlassian.net/browse/REK-41) and primary Subtask
[REK-42](https://rekonlabs.atlassian.net/browse/REK-42). The technical owner
accepted the separately signed adoption-record architecture for Phase 1;
Phases 2 and 3 still require separate Jira plans.

## Problem

The implementation run records the commit created by the governed commit gate.
When an explicitly approved remediation adds a reviewed descendant commit, the
final branch no longer equals that recorded commit. Publication correctly
rejects the mismatch, but the CLI cannot create a new immutable authority record
for the final descendant without re-executing identical work or mutating prior
run history.

REK-40 demonstrates the failure: its original run records `f427b90`, while the
approved remediation ends at `13e73cd`. Both remediation-only and combined
base-to-candidate reviews passed, but the final branch cannot enter the signed
publication workflow.

## Goal

Allow a repository owner to adopt one approved, reviewed, linear descendant
remediation range through an explicit fail-closed transition that preserves the
predecessor, binds fresh source and full-range evidence, and produces immutable
successor authority consumable by existing publication gates.

## Product Outcomes

- A valid descendant remediation does not require duplicate implementation.
- The predecessor run remains immutable and independently verifiable.
- The successor binds the original base, predecessor, candidate, source,
  bundle, guidance, validation, and full-range review evidence.
- Publication continues to require a separate signed authorization and
  separately consumed push and pull-request operations.
- Invalid, stale, replayed, ambiguous, rewritten, or out-of-scope adoption
  attempts stop before trusted local persistence or remote side effects.
- The capability is repository-neutral and works across different repository
  identities, Jira projects, branch names, and repository-relative paths.

## Users And Decision Rights

- The Jira owner approves ticket-intent or scope amendments before adoption.
- The technical owner approves the ADR and any architecture or scope exception.
- The repository owner controls local commit and remote-publication authority.
- The ADR must decide which authorized role signs or approves successor
  adoption. The control plane must verify that role and never infer it from
  repository or Jira state.

## Required Workflow

1. Stop publication when branch `HEAD` differs from the recorded run commit.
2. Obtain a fresh signed Jira export and validated work item/task bundle for any
   approved source amendment.
3. Validate the candidate worktree, linear descendant range, complete
   base-to-candidate diff, scope, budgets, repository guidance, deterministic
   checks, and independent reviewer/verifier evidence.
4. Show an exact no-write adoption preview.
5. After explicit authorized approval, atomically persist a new immutable
   successor record without altering the predecessor.
6. Require publication issuance and execution to revalidate the complete
   successor chain and the current remote target.

## Success Criteria

- The REK-40 scenario can resume publication through the new transition without
  rerunning implementation or modifying its predecessor record.
- Every trusted field is deterministic, versioned, digest-bound, and covered by
  fail-closed validation.
- No adoption command writes Jira or GitHub, pushes, creates a pull request,
  merges, releases, deploys, or consumes publication authority.
- Existing unremediated runs retain their current behavior.
- The roadmap records #69 as the immediate REK-40 publication blocker and as a
  focused input to #22.

## Non-Goals

- General run-history repair.
- Adopting arbitrary, merged, rewritten, unrelated, or unverifiable history.
- Semantic adjudication of Jira drift.
- Automatic Jira, GitHub, publication, merge, release, or deployment actions.
- Replacing the bounded lifecycle authorization planned in #22.
- Weakening source, scope, budget, exact-diff review, signer, remote-target, or
  one-use publication checks.
