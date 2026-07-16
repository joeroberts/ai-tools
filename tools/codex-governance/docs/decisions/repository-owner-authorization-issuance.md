# ADR: Guided Repository-Owner Authorization Issuance

## Status

Proposed for stakeholder approval with the publication-enablement work item.

## Context

Remote publication correctly requires externally authorized repository-owner
evidence, but the repository has no trusted repository-owner key and no safe,
repeatable way for its owner to issue that evidence. Manual envelope assembly
would be error-prone and could omit run, remote, target, lineage, expiry, or
operation bindings.

The existing CLI already separates verification and remote dispatch. Jira
export signer bootstrap also demonstrates owner-only local key material whose
public key alone enters repository policy.

## Decision

Add explicitly invoked owner-side bootstrap and authorization-issuance commands
to the governance CLI. Treat them as repository-owner operations, not control
plane inference. They require explicit approval, accept operator-selected
owner-only paths, refuse overwrite, and never run automatically from preflight,
implementation, push, or PR commands.

Issuance may read local run state and read-only Git remote metadata to construct
the exact signed payload. It must not push, contact GitHub issue or PR APIs,
write Jira, consume an authorization, or copy private key material into the
repository, task bundle, logs, or ledger.

## Consequences

- Owners gain a deterministic way to create evidence that existing publication
  gates can verify.
- The same binary contains owner-side issuance and verifier code, so separation
  depends on explicit commands, private-file permissions, non-automatic wiring,
  exact signed fields, and one-time publication consumption.
- General key rotation, revocation distribution, shared organizational signing,
  and hardware-backed custody remain future decisions.
