# Bounded Workflow Authorization Specification

## Authorization Contract

The versioned signed payload contains repository identity, GitHub source issue,
Story and Subtask keys, plan-contract and source digests, base SHA, paths,
budget, acceptance criteria, remediation limit, permitted operations, branch,
remote, PR target, expiry, and derivation rules for commit SHA, evidence
digest, checks, and PR URL.

Each attempted operation appends a privacy-safe audit event containing the
authorization digest, preview digest, result, and read-back digest. Durable
mutable state, keyed by the authorization digest, records operation-specific
consumption and revocation. Consumption is atomic; restart reconciliation must
not duplicate a side effect or alter the signed payload.

## Allowed Paths

- `README.md`
- `docs/design`
- `docs/design/bounded-workflow-authorization-spec.md`
- `internal/cli`
- `internal/implementation`
- `internal/jira`
- `testdata`

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
      "adr": "docs/decisions/0004-bounded-workflow-authorization.md"
    }
  ]
}
```

Later slices integrate lifecycle gates, then commit, publication, merge-check,
and finalization behavior; they are not authorized by this planning slice.

The `authorization-contract` documentation slice was delivered in PR #116 and
is historical. The following manifest is the only pending implementation
manifest. It replaces the historical slice as the source for the next ticket
plan; no pending slice may recreate the documentation-only work.

```json ticket-plan-slices
{
  "format_version": 1,
  "slices": [
    {
      "id": "workflow-authorization-core",
      "summary": "Add the general signed workflow authorization and durable state core.",
      "phase": "Phase 1",
      "change_class": "high-risk",
      "scope": "It adds the immutable authorization payload, signature validation, and owner-only durable consumption, revocation, and privacy-safe audit state.",
      "non_goals": [
        "This does not authorize a Jira write, implementation dispatch, commit, push, pull request, merge, or finalization."
      ],
      "acceptance_criteria": [
        "Authorization binds repository, GitHub issue, Jira keys, approved contract and source digests, base revision, paths, budget, review limit, operations, branch, remote, pull-request target, expiry, and derivation rules.",
        "The signed claims are immutable; durable state keyed by the authorization digest atomically records per-operation consumption, revocation, and privacy-safe audit events without duplicate side effects.",
        "Focused tests prove signature, expiry, binding, replay, revocation, atomic-consumption, audit-redaction, and restart-reconciliation failures fail closed."
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
        "docs/design/bounded-workflow-authorization-spec.md",
        "internal/implementation"
      ],
      "review_budget": {
        "max_changed_files": 8,
        "max_changed_lines": 700,
        "components": ["signed workflow authorization contract", "durable authorization state and deterministic fixtures"]
      },
      "dependencies": [],
      "adr": "docs/decisions/0004-bounded-workflow-authorization.md"
    },
    {
      "id": "authorized-local-lifecycle",
      "summary": "Gate deterministic Jira and local lifecycle actions with workflow authorization.",
      "phase": "Phase 2",
      "change_class": "high-risk",
      "scope": "It consumes the core authorization before deterministic Jira planning, In Progress transition, implementation entry, review-cycle progression, local commit, and factual work-record actions.",
      "non_goals": [
        "This does not replace #29 Jira status evidence, #18 required-check evidence, existing exact-diff review, ticket-drift detection, or credential isolation."
      ],
      "acceptance_criteria": [
        "Each covered local or Jira action renders its exact preview, verifies the live authorization and existing prerequisite evidence before its side effect, records the read-back, and fails closed on drift, expiry, consumption, revocation, or a third remediation cycle.",
        "Tests cover eligible lifecycle progression and every denied hard stop without creating duplicate Jira or local side effects."
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
        "internal/cli",
        "internal/implementation",
        "internal/jira",
        "testdata"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 850,
        "components": ["authorized deterministic Jira and local actions", "preview read-back and hard-stop fixtures"]
      },
      "dependencies": ["workflow-authorization-core"],
      "adr": "docs/decisions/0004-bounded-workflow-authorization.md"
    },
    {
      "id": "authorized-publication-and-merge",
      "summary": "Bind publication and merge eligibility to workflow authorization.",
      "phase": "Phase 3",
      "change_class": "high-risk",
      "scope": "It consumes the authorization for the exact reviewed commit, branch, remote, pull request, and merge target after verified #18 required-check and branch-protection evidence.",
      "non_goals": [
        "This does not authorize releases, deployments, tags, force pushes, unnamed remotes, infrastructure changes, or a bypass of authoritative required checks."
      ],
      "acceptance_criteria": [
        "Push, pull-request creation, and merge require matching exact-diff evidence, deterministic target derivation, unconsumed live authority, and verified #18 evidence; absent, pending, failed, stale, or unverifiable checks fail before the side effect.",
        "Tests prove replay and target drift cannot publish or merge another commit, branch, repository, or pull request."
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
        "internal/cli",
        "internal/implementation",
        "testdata"
      ],
      "review_budget": {
        "max_changed_files": 9,
        "max_changed_lines": 800,
        "components": ["authorized publication target validation", "authoritative merge-evidence consumption"]
      },
      "dependencies": ["authorized-local-lifecycle"],
      "adr": "docs/decisions/0004-bounded-workflow-authorization.md"
    },
    {
      "id": "authorized-finalization-and-operator-guidance",
      "summary": "Finalize authorized work with reconciliation and operator guidance.",
      "phase": "Phase 4",
      "change_class": "high-risk",
      "scope": "It binds merged-state Jira finalization and restart reconciliation to authorization audit state and documents informational previews, machine gates, and new-approval boundaries.",
      "non_goals": [
        "This does not weaken child-before-parent completion, exact read-back, audit retention, or any hard-stop boundary."
      ],
      "acceptance_criteria": [
        "Finalization verifies merged state and Jira hierarchy before each write, records exactly one reconciled outcome after partial failure, and leaves failed or ambiguous side effects visible and blocking.",
        "Operator documentation distinguishes informational previews, enforced gates, and the actions that require new owner approval."
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
        "README.md",
        "docs/design",
        "internal/cli",
        "internal/implementation",
        "internal/jira",
        "testdata"
      ],
      "review_budget": {
        "max_changed_files": 10,
        "max_changed_lines": 850,
        "components": ["authorized finalization and reconciliation", "operator authorization-boundary guidance"]
      },
      "dependencies": ["authorized-publication-and-merge"],
      "adr": "docs/decisions/0004-bounded-workflow-authorization.md"
    }
  ]
}
```

Every slice requires focused deterministic tests, `make test`, `make vet`,
`make build`, `git diff --check`, and independent exact-diff reviewer and
verifier evidence.

## Review Budget

The completed planning-document slice was limited to 3 changed files, 350
changed lines, and authorization-contract. The full pending implementation plan
is limited to 37 changed files, 3200 changed lines, and signed workflow authorization contract, durable authorization state and deterministic fixtures, authorized deterministic Jira and local actions, preview read-back and hard-stop fixtures, authorized publication target validation, authoritative merge-evidence consumption, authorized finalization and reconciliation, operator authorization-boundary guidance. Each pending slice has its own stricter declared budget and allowed paths.
