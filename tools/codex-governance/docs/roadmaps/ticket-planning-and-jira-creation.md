# Ticket Planning And Governed Jira Creation

Structured phase state: [ticket-planning-and-jira-creation.yaml](ticket-planning-and-jira-creation.yaml).

## Goal

Generate a governed ticket plan from approved PRD, specification, and roadmap
documents; require stakeholder approval; then create the Jira Story and
subtasks through a separately approved write gate.

## Architecture

The hosted manager performs semantic synthesis. Local Ollama workers perform
independent bounded review and verification through the existing policy
gateway. Go owns source digests, schema and scope checks, lifecycle state,
artifacts, retry limits, approval, escalation, and Jira publication.

## Phases

1. **Plan Contract And Workflow State** - complete 2026-07-13
   - Add field-level source traceability, dependency checks, ADR/path checks,
     source-digest verification, and persisted workflow state.
   - Preserve current mixed work only where it satisfies this contract.

2. **Hosted Manager And Local Workers** - blocked pending deterministic manager remediation
   - Use hosted Codex only for the manager.
   - Route reviewer and verifier through allowlisted local Ollama models.
   - Require benchmark evidence and bounded task types before local execution.

3. **Stakeholder Approval And Escalation** - in progress
   - Add explicit plan approval, two-cycle remediation, redacted findings, and
     stakeholder escalation artifacts.

4. **Governed Jira Publication** - pending approval
   - Require approved workflow state, dry-run support, idempotent result
     records, explicit write approval, and environment/secret-manager inputs.

5. **End-To-End Verification And CI** - pending approval
   - Add fixture-based tests, local smoke tests, documentation, and advisory CI
     coverage without exposing prompts or credentials.

## Boundaries

No Jira issues are created until Phase 4 is approved and a stakeholder-approved
plan passes all deterministic gates. Local models never receive Jira
credentials or direct write authority.
