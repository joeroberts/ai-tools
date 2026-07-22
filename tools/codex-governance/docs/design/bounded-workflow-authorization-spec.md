# Bounded Workflow Authorization Specification

## Authorization Contract

The versioned signed payload contains repository identity, GitHub source issue,
Story and Subtask keys, plan-contract and source digests, base SHA, paths,
budget, acceptance criteria, remediation limit, permitted operations, branch,
remote, PR target, expiry, consumption and revocation state, and derivation
rules for commit SHA, evidence digest, checks, and PR URL.

Each attempted operation appends a privacy-safe audit event containing the
authorization digest, preview digest, result, and read-back digest. Consumption
is atomic and operation-specific; restart reconciliation must not duplicate a
side effect.

## Allowed Paths

- `docs/decisions/0004-bounded-workflow-authorization.md`
- `docs/design/bounded-workflow-authorization-prd.md`
- `docs/design/bounded-workflow-authorization-spec.md`

## Declared Slices

The first declared slice is `authorization-contract` with phase
`planning-document slice`, change class `standard`, no dependencies, and ADR
`docs/decisions/0004-bounded-workflow-authorization.md`. It changes only the
three allowed planning documents. Its acceptance criteria are the versioned
payload, privacy-safe audit event, atomic consumption, and signer,
persistence, replay, expiry, and revocation coverage defined above.

The following manifest is canonical for this slice. The manager may explain
the decomposition, but must preserve every source-derived value verbatim.

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "authorization-contract",
      "summary": "Define the versioned authorization contract planning documents.",
      "phase": "planning-document slice",
      "change_class": "standard",
      "scope": "It changes only the three allowed planning documents.",
      "non_goals": [
        "This does not authorize releases, deployments, tags, force pushes, destructive operations, secret access, infrastructure changes, or unnamed remote targets."
      ],
      "acceptance_criteria": [
        "The versioned payload, privacy-safe audit event, atomic consumption, and signer, persistence, replay, expiry, and revocation coverage defined above."
      ],
      "validation_plan": [
        "focused deterministic tests",
        "make test",
        "make vet",
        "make build",
        "git diff --check",
        "independent exact-diff reviewer and verifier evidence"
      ],
      "allowed_paths": [
        "docs/decisions/0004-bounded-workflow-authorization.md",
        "docs/design/bounded-workflow-authorization-prd.md",
        "docs/design/bounded-workflow-authorization-spec.md"
      ],
      "review_budget": {
        "max_changed_files": 3,
        "max_changed_lines": 350,
        "components": ["authorization-contract"]
      },
      "dependencies": [],
      "adr": "No ADR needed: This is a docs-only task."
    }
  ]
}
```

Later slices integrate lifecycle gates, then commit, publication, merge-check,
and finalization behavior; they are not authorized by this planning slice.

The lifecycle gate integration slice covers Jira preview/read-back, #29 status
evidence, exact-diff evidence, and bounded remediation cycles. The commit,
publication, merge-check, and finalization integration slice covers restart
reconciliation and audit records.

Every slice requires focused deterministic tests, `make test`, `make vet`,
`make build`, `git diff --check`, and independent exact-diff reviewer and
verifier evidence.

## Review Budget

The planning-document slice is limited to 3 changed files, 350 changed lines,
and authorization-contract. Later implementation slices require their own approved
budgets and allowed paths.
