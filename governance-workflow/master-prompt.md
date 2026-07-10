# Master Prompt

We are building `codex-governance`, a reusable team toolkit for governed
Codex-assisted engineering work.

Use a staged approach. Do not try to implement the full toolkit in one pass.

Before writing implementation files, create a canonical spec file named
`docs/governance-toolkit-spec.md` from the requirements I provide. Then propose
Stage 1 only and wait for approval.

After Stage 1 is approved, proceed autonomously within that stage. At the end
of each stage:

- run validation for that stage
- persist a session handoff
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
4. Example repo
5. Review, verification, and polish

Use the canonical spec as the source of truth. If a stage discovers the spec is
wrong or incomplete, propose a spec update before continuing.

First response:

- summarize the intended toolkit structure
- explain the roadmap-first governance flow
- explain the stage sequence
- define the autonomy envelope
- list assumptions and risks
- request approval to create `docs/governance-toolkit-spec.md` and the Stage 1
  plan
