# ADR 0001: Persist Ticket-Plan Authority Contract

## Status

Accepted on 2026-07-16 for GitHub issue #38 planning.

## Context

Ticket-plan generation currently combines verified sources, an owner-assigned
constraints file, and manager-authored plan content. Generation applies the
constraints in memory and marks some plan traces with a temporary assignment
authority flag. Standalone validation later receives only the plan and source
documents, so it cannot reconstruct or verify the assignment authority that
generation used.

The same missing durable boundary prevents deterministic enforcement of
source-declared implementation slices. Cardinality, ordering, dependency,
budget, and path decisions can be lost or reinterpreted between decomposition,
generation, approval, and standalone validation.

## Decision

Introduce one strict, versioned ticket-plan authority contract. The contract is
created from verified source descriptors plus explicit owner assignments before
manager generation. It records:

- source identities and digests;
- canonical source-derived Story and Subtask values;
- assignment-owned fields and declared implementation slices;
- permitted manager-narrative fields and their validation rules; and
- stable slice IDs, order, dependencies, review budgets, and canonical allowed
  paths.

The generated plan identifies the contract digest. Workflow approval binds the
plan digest, contract digest, and verified source digests. Generation and
standalone validation call the same contract-aware validator. Equality with the
contract, rather than manager-authored trace text, establishes authority for
assigned and canonical fields.

Contract artifacts are private runtime records, written owner-only and without
overwrite. Supported plan, contract, and workflow versions are explicit.
Unsupported combinations fail closed with an actionable migration or rejection
message. They are never silently reinterpreted.

Source-declared slice coverage is part of this contract. No parallel slice
contract or manager-selected cardinality mechanism is introduced.

## Consequences

- Ticket-plan validation requires the matching persisted contract in addition
  to the plan and verified sources.
- Contract substitution, plan drift, source drift, omitted or extra slices,
  reordered slices, and aggregated path values fail deterministically.
- Deterministic contract failures do not consume manager-remediation cycles.
- Only fields explicitly classified as manager narrative may enter the bounded
  semantic remediation loop.
- Legacy artifacts require an explicit supported migration path or are rejected.
- Future bounded workflow authorization may bind to this contract digest without
  duplicating ticket-plan authority.

## Alternatives Considered

### Keep the temporary assignment trace flag

Rejected because the flag is stored in the plan without the durable assignment
and source-derived contract that gives it meaning.

### Reconstruct constraints from source text during standalone validation

Rejected because owner assignments are not necessarily derivable from source
text and reconstruction could silently reinterpret previously approved values.

### Add a separate declared-slice manifest

Rejected because two authority artifacts could drift. Declared slices belong in
the same persisted contract as the rest of the approved ticket-plan boundary.
