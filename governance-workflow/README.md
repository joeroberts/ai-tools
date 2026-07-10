# Governance Workflow Prompt Assets

These assets help create `codex-governance`, a reusable toolkit for governed
Codex-assisted engineering work.

## Files

- `master-prompt.md`: starts the work and requires Codex to create a canonical
  spec before implementation.
- `canonical-spec.md`: full source-of-truth requirements for the toolkit.
- `stage-1-spec-and-scaffold.md`: creates base structure and templates.
- `stage-2-skill-and-references.md`: creates the Codex skill and prompt
  references.
- `stage-3-scripts-and-validation.md`: creates bootstrap/validation scripts and
  smoke tests.
- `stage-4-example-repo.md`: creates Terraform, Node fullstack, and
  Python/Vue fullstack examples.
- `stage-5-review-verification-polish.md`: runs the governed review and
  verification loop.

## Recommended Use

1. Start a new Codex session in the repo where `codex-governance` should be
   created.
2. Paste `master-prompt.md`.
3. When Codex asks for the canonical requirements, provide `canonical-spec.md`.
4. Approve Stage 1 only.
5. Continue one stage at a time using the staged prompt files.

## Operating Model

Use the full spec as the source of truth, but keep implementation prompts small.
Each stage should have its own approval gate, validation, session handoff, and
clear next-step proposal.

Do not allow push, publish, global install, remote PR updates, tags, releases,
deploys, Terraform apply, cloud mutations, destructive commands, or secret
access without explicit approval.

## Repo Profiles

The workflow is repo-agnostic, but validation is profile-specific. The toolkit
must support at least:

- `generic`: governance-only defaults for unknown repos.
- `terraform-module`: Terraform formatting, validation, tests, security scans,
  and optional release tag checks.
- `node-fullstack-k8s`: npm workspace checks, TypeScript, Vitest, Prisma,
  Helm/GitOps, image/runtime checks, and background worker coverage.
- `python-vue-fullstack-k8s`: Python lint/test/coverage, schema checks, Vue
  coverage/build, Helm/GitOps, and local app smoke checks.

Bridge is the lightweight Node app adoption target. Execution Lens is the
stronger mature app-governance reference. Profiles should add validation and
artifact requirements without changing the shared roadmap, packet, ADR,
evidence, and reviewer/verifier workflow.
