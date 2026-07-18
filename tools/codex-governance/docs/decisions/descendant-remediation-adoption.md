# ADR: Descendant Remediation Successor Authority

## Status

Proposed. No option is selected and this record does not authorize
implementation.

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

Questions to resolve:

- whether the successor is a new run ID or a versioned continuation identity;
- how current lifecycle transitions and audit events represent adoption;
- how format-version-1 runs migrate without reinterpretation; and
- which role signs or authorizes successor creation.

## Option B: Signed Adoption Record

Keep the predecessor run format unchanged and introduce a separately signed,
versioned adoption record. Publication resolves the predecessor plus adoption
record into an immutable successor publication view.

Questions to resolve:

- whether the extra resolution layer simplifies or complicates every gate;
- how chained adoptions are prohibited or bounded;
- how record expiry and revocation affect the resolved view; and
- which role signs or authorizes adoption.

## Required Decision

Phase 1 must select exactly one option, document rejected alternatives, define
the signer and decision-rights matrix, specify lifecycle and audit transitions,
define expiry/replay/revocation and crash recovery, document migration and
rollback, and change this ADR to `Accepted` before implementation behavior is
committed.

## Non-Options

- Editing the predecessor run in place
- Copying mutable JSON without signed authority
- Inferring adoption from branch `HEAD`, Jira, GitHub, or commit messages
- Weakening publication equality, lineage, source, or exact-review gates
- Treating adoption as publication authorization
