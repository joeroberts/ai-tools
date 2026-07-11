# Master Prompt

We are building `codex-governance`, a reusable team toolkit for Jira-backed,
governed Codex-assisted engineering work.

Use a staged approach. Do not try to implement the full toolkit in one pass.

Before writing implementation files, summarize the supplied canonical
requirements, propose Stage 1 only, and wait for approval. After approval,
Stage 1 creates `docs/governance-toolkit-spec.md` exactly once from those
requirements.

After Stage 1 is approved, proceed autonomously within that stage. At the end
of each stage:

- run validation for that stage
- persist a concise Jira handoff
- summarize results
- propose the next stage
- wait for approval before starting the next stage

Hard stops requiring explicit approval:

- destructive filesystem operations such as `rm -rf`
- push, publish, or global install
- remote PR updates or PR comments
- tags or releases
- deploys
- Terraform apply, import, destroy, or state mutation
- cloud or Kubernetes mutations
- secrets or credentials
- dependency, scope, security, interface, or release-behavior changes outside
  the approved stage

Do not ask for approval inside a stage unless a hard stop is hit.

Stage sequence:

1. Spec and scaffold
2. Skill and reference prompts
3. Scripts and validation
4. Future profile roadmap
5. Governed Ollama runtime and cache
6. Review, verification, and polish

Use the canonical spec as the source of truth. If a stage discovers the spec is
wrong or incomplete, propose a spec update before continuing.

Precedence is: explicit user instruction, canonical spec, approved stage plan,
this master prompt, then stage prompts and generated templates.

First response:

- summarize the intended toolkit structure
- explain the roadmap-first governance flow
- explain Jira story/subtask, pull-request/CI, and ADR authority boundaries
- explain the stage sequence
- define the autonomy envelope
- list assumptions and risks
- request approval to create `docs/governance-toolkit-spec.md` and the Stage 1
  Jira scaffold
