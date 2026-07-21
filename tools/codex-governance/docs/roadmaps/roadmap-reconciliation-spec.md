# Roadmap Reconciliation Specification

## Technical Design

Reconcile existing roadmaps against current execution evidence. Structured YAML
is the machine-readable phase state; each companion Markdown status must agree.
Document deferred capability and dependency language without changing product
behavior.

## Allowed Paths

Implementation is limited to `docs/roadmaps`.

## Review Budget

The single slice is limited to 16 files, 800 lines, and two components:
roadmap status/completion records and autonomy dependency/effort documentation.

## Technical Acceptance Criteria

- Make Markdown and YAML aggregate states coherent without relying on #51.
- Use concise completion records with GitHub, Jira, and Git/PR/CI evidence in
  their authoritative records.
- Do not infer roadmap state from external systems.
- Preserve explicit-approval and fail-closed boundaries.
- Identify canonical PRD/spec changes required before #18, #19, #22, #44,
  #45, #50, #51, #55, #59, and #68 change the contract.

## Architecture Decision

No ADR needed: this correction removes duplicated records and documents current
responsibilities without implementing a new authority or lifecycle mechanism.
