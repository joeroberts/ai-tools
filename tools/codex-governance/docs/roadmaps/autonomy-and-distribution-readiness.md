# Governance Toolkit Autonomy And Distribution Readiness

## Status

Planning estimate rebaselined on 2026-07-18. This is not an implementation
authorization, delivery commitment, or release-readiness decision. Re-estimate
each item when its implementation-ready work item is approved.

## Current State

The CLI retains explicit approval and fail-closed gates. ADR-0003 supplies
restart-safe supervision for approved headless runs; it does not widen
authorization. The CLI does not provide bounded autonomous authorization,
operational local-model code-edit qualification, protected-branch
administration, or distribution. Policy eligibility is not operational model
qualification.

## Rebaselined Sequence And Effort

The numeric ranges preserve the 2026-07-17 SOL Medium estimate. They are active
agent effort, not human person-hours or uninterrupted elapsed time. The known
range subtotal is 152-269 agent-hours and excludes all unestimated work.

| Order | Issue or work item | Active agent effort | Confidence |
| --- | --- | ---: | --- |
| Complete | [#29](https://github.com/joeroberts/ai-tools/issues/29): verified Jira `In Progress` preflight | Historical 4-8; remaining 0 | Delivered |
| 1 | [#51](https://github.com/joeroberts/ai-tools/issues/51): consistent aggregate and phase roadmap states | Not yet estimated | Pending decomposition |
| 2 | [#68](https://github.com/joeroberts/ai-tools/issues/68): repository-neutral roadmap configuration and machine-enforced lifecycle/impact contract | Not yet estimated | ADR and decomposition required |
| Complete | [#99](https://github.com/joeroberts/ai-tools/issues/99): restart/recovery design correction | Delivered in ADR-0003; remaining 0 | Delivered |
| 4 | [#18](https://github.com/joeroberts/ai-tools/issues/18): authoritative CI and governance evidence | 6-12 | Medium-high |
| 5 | [#44](https://github.com/joeroberts/ai-tools/issues/44): semantic-version evidence | Not yet estimated | Pending decomposition |
| 6 | [#45](https://github.com/joeroberts/ai-tools/issues/45): repository and protected-branch baseline | Not yet estimated | Pending decomposition |
| 7 | [#22](https://github.com/joeroberts/ai-tools/issues/22): bounded workflow authorization | 30-50 | Medium |
| Complete | [#59](https://github.com/joeroberts/ai-tools/issues/59): manager ownership ADR | Historical 3-6; remaining 0 | Delivered |
| Complete | [#55](https://github.com/joeroberts/ai-tools/issues/55): persistent supervision | Delivered; remaining 0 | Delivered |
| 10 | [#19](https://github.com/joeroberts/ai-tools/issues/19): root entry guidance propagation | 4-8 | Medium-high |
| 11a | [#21](https://github.com/joeroberts/ai-tools/issues/21): operational model qualification registry | 10-18 | Medium |
| 11b | [#50](https://github.com/joeroberts/ai-tools/issues/50): finding adjudication | 30-55 | Medium-low |
| 11c | [#13](https://github.com/joeroberts/ai-tools/issues/13): qualified local coding agent | 45-80 | Medium-low |

```text
#67 -> #51 -> #68 -> #22
#51 + #99
#18 -> #44 -> #45 -> #22
#59 -> #55
#68 -> #19
#21 -> #50 -> #13
```

#68 must settle the roadmap-impact contract before #22 consumes it. It owns
repository-neutral configuration and machine enforcement; #19 only propagates
the settled root-entry guidance. #55 follows #59's selected ownership model:
retain the post-assignment manager as a bounded proposal producer, with local
validation and Jira gates remaining authoritative.

## Required Design Amendments

Open dependencies remain future work, not implemented behavior. Before a
listed issue changes the governed product contract, the canonical PRD/spec must
be amended as follows:

| Issue | Required canonical change |
| --- | --- |
| #18 | Authoritative checks, evidence ownership, privacy-safe review status, fork safety, and failure handling. |
| #19 | Missing-root `AGENTS.md` creation, merge-required addenda, and adopted-guidance preflight enforcement. |
| #22 | An ADR and scope, identity, expiry, consumption, replay, revocation, derivation, and hard-stop semantics. |
| #44 | Version source of truth, deterministic version-impact input, tag validation, and stable evidence. |
| #45 | Required-check ownership, hosted-setting approval/read-back, drift, rollback, and protected-branch boundaries. |
| #50 | Versioned disputes, immutable evidence, supersession, adjudication, and authorized escalation. |
| #51 | Aggregate/phase state validation and failure semantics. |
| #55 | Delivered with ADR-0003: restart-safe supervisor ownership, durable identity, recovery, duplicate prevention, terminal state, and diagnostics. |
| #99 | Delivered in ADR-0003: restart/recovery ownership, reconciliation, duplicate prevention, terminal diagnostics, and fail-closed recovery conditions. |
| #59 | Delivered in ADR-0002: retained bounded manager ownership, decomposition binding, migration, review, and rollback. |
| #68 | ADR plus roadmap configuration ownership, impact declarations, transitions, enforcement, compatibility, and failure semantics. |

The broader implementation documents must separately distinguish implemented
terminal-artifact reconciliation from missing process survival and persistent
supervision. The canonical design already covers role/task-specific model
qualification (#21), qualified local-model coding (#13), and the fact that
migration completion does not authorize distribution (#52).

## Distribution Boundary

Distribution remains blocked. #52 must consume stable outputs of #18, #21,
#22, and #44-#49 with the required security, retention, provenance,
compatibility, privacy, and support evidence. Completion of a roadmap is not
distribution approval.
