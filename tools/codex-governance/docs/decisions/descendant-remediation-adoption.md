# ADR: Descendant Remediation Successor Authority

## Status

Accepted on 2026-07-19 for REK-41 / REK-42 Phase 1.

## Context

A governed run is bound to its committed SHA. An approved, reviewed descendant
remediation can make the final branch differ from that SHA, correctly blocking
publication. The predecessor cannot be mutated, and re-executing identical work
is wasteful. A new immutable authority representation is required to bind the
predecessor, candidate, fresh source, full-range validation, and publication
handoff.

## Decision Drivers

- Predecessor immutability and tamper-evident lineage
- Common signed-envelope and trusted-role compatibility
- Exact full-range scope, budget, validation, and review binding
- Atomic, non-overwriting, replay-safe persistence
- Clear decision rights, expiry, revocation, migration, recovery, and rollback
- Minimal special casing in publication issuance, push, and PR creation
- Repository-neutral adoption and deterministic testing

## Option A: Versioned Successor Run

Introduce a new implementation-run format whose successor fields bind the
predecessor run digest, prior commit, adopted range, candidate commit, refreshed
authority, validation, and review evidence.

Rejected. Adding successor fields to `implementation.Run` would make a
format-version-1 run ambiguous: it would be both immutable predecessor
evidence and a mutable lifecycle container for a later remediation. It would
also require every current run reader and publication gate to understand a new
meaning before Phase 3.

## Option B: Signed Adoption Record

Keep the predecessor run format unchanged and introduce a separately signed,
versioned adoption record. Publication resolves the predecessor plus adoption
record into an immutable successor publication view.

Selected. The adoption record is a separate, versioned, signed payload. It
binds the immutable predecessor and exact candidate without changing either
the predecessor run or format-version-1 parsing. A later publication gate may
resolve one verified record with its predecessor into a successor view.

## Authority And Contract

The Phase 1 payload is `implementation.AdoptionRecord`, version 1. It carries
immutable repository, work-item, predecessor-run, original-base, predecessor
commit, candidate-commit, adopted-range, source, task-bundle, configuration,
guidance, complete-diff, review, deterministic-check, reason, expiry, and
preceding-audit-event bindings. Candidate commit and full range are authority;
the candidate branch is descriptive and rejects `HEAD`, default branches, and
`refs/*` aliases.

The payload is strict: unknown or trailing JSON, unsupported versions, missing
bindings, malformed digests or timestamps, mutable aliases, duplicate or
unsorted checks, non-passing checks, and invalid identities fail closed. Its
encoder refuses invalid values and produces deterministic JSON; parsing never
repairs input into compliance.

| Decision | Authority | Phase 1 behavior |
| --- | --- | --- |
| Ticket intent and scope | Jira owner | Bound by the work-item/source digest; no inference. |
| Adoption architecture and record approval | Technical owner | The only permitted `authorized_role`; Phase 1 verifies the payload's shared signed envelope. |
| Local commit and remote publication | Repository owner | Remain separately authorized; this record grants neither. |
| Record creation and storage | Future approved command | No command creates, signs, or persists a record in Phase 1. |

Expiry is mandatory. Phase 1 fixtures reject records that are not yet valid,
expired, signed for an unpermitted role, or signed by a key absent from the
trusted registry. Phase 2 adds live registry integration, persistence, and
replay checks. Chained adoptions are prohibited until a future ADR amendment
explicitly defines a bounded chain. The record's preceding audit-event identity
links the later write to the existing tamper-evident ledger.

## Migration, Recovery, And Rollback

Format-version-1 runs retain their current parsing and publication behavior;
they are predecessors, never silently upgraded successor runs. Phase 2 may
construct a separately signed record only after full validation and must use
atomic owner-only non-overwriting storage. A crash before completion leaves no
trusted record; recovery restarts validation rather than repairing a partial
write. Rollback revokes or expires a record and records that fact in the audit
ledger; it never rewrites the predecessor, candidate, or prior audit event.

Phase 3 publication consumption must revalidate record signature, role,
revocation, expiry, immutable bindings, exact candidate, and audit linkage at
the side-effect boundary. Adoption remains distinct from publication authority.

## Non-Options

- Editing the predecessor run in place
- Copying mutable JSON without signed authority
- Inferring adoption from branch `HEAD`, Jira, GitHub, or commit messages
- Weakening publication equality, lineage, source, or exact-review gates
- Treating adoption as publication authorization
