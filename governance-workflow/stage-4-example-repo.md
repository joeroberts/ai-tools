# Stage 4 Prompt: Example Repo

Use this prompt after Stage 3 is complete and approved.

## Objective

Create profile examples showing how Terraform modules and app repos use the
same governance scaffold with different validation contracts.

## Scope

Create `examples/terraform-module/` with:

- `examples/terraform-module/README.md`
- `examples/terraform-module/AGENTS.md`
- `examples/terraform-module/PROJECT_CONTEXT.md`
- `examples/terraform-module/.codex/assessments/`
- `examples/terraform-module/.codex/summaries/`
- `examples/terraform-module/implementation-packets/`
- `examples/terraform-module/verification/evidence/`
- `examples/terraform-module/docs/decisions/`
- Example roadmap.
- Example implementation packet.
- Example review scope.
- Example discovery evidence.
- Example ADR if demonstrating behavior/workflow change.

Create `examples/node-fullstack-k8s/` with the same governance artifacts plus:

- npm workspace validation examples.
- TypeScript typecheck, Vitest test, optional coverage, and build commands.
- Prisma generate and migration validation examples.
- Helm lint/template and GitOps value validation examples.
- API, frontend, worker, cron, and migration job scope examples.
- Runtime image validation examples.
- External integration checks documented as opt-in and secret-safe.

Create `examples/python-vue-fullstack-k8s/` with the same governance artifacts
plus:

- Makefile-style command aggregation.
- Python lint, schema check, test, and coverage examples.
- Vue/Vite coverage and build examples.
- Optional `execution-state/current-state.md`, `backlog.md`, and
  `escalations.md`.
- Helm/GitOps validation and local smoke check examples.
- Database migration/schema snapshot handling.

## Example Data Rules

- Use fake IDs such as `EXAMPLE-0001`.
- Use fake names such as `example-app`, `example-service`, and
  `example-owner`.
- Do not include private URLs, real issue IDs, real repo names, credentials,
  cloud state, backend secrets, provider tokens, or production identifiers.
- Do not include Terraform apply behavior as an automated action.
- Do not include live app deploy, Argo/Kargo sync, image publish, or database
  migration execution as automated actions.

## Validation

- Run governance validator against each example.
- Run smoke tests if the example is covered.
- Record validation results in a Stage 4 handoff.

## Completion

At the end of Stage 4:

- summarize files created
- list validation run and results
- identify gaps
- propose Stage 5 plan
- wait for approval before Stage 5
