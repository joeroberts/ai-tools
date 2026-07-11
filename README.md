# ai-tools

`ai-tools` is a portfolio repository for tools and integration layers that
support AI-assisted development workflows.

## Layout

- `tools/` contains independently testable products, such as CLIs, services,
  libraries, their documentation, and fixtures. Each tool owns its runtime and
  release lifecycle.
- `integrations/codex/plugins/` contains Codex plugin packages. A plugin is an
  adoption layer around one or more tools; it must not duplicate tool logic.
- `integrations/codex/skills/` contains standalone Codex skills that do not
  require plugin packaging or a backing executable.
- `shared/` is reserved for schemas, fixtures, and documents that are truly
  consumed by multiple products. Do not move tool-specific assets here merely
  for centralization.

## Current Products

- [`tools/codex-governance/`](tools/codex-governance/) provides the
  `codex-governance` Go CLI for deterministic governance checks, roadmap
  reporting, and local-model policy enforcement.
- [`integrations/codex/plugins/codex-governance/`](integrations/codex/plugins/codex-governance/)
  provides the optional Codex adoption layer for the CLI.

Codex plugins are registered through the repository marketplace at
`integrations/codex/.agents/plugins/marketplace.json`.

## Conventions

Keep a product's implementation, tests, fixtures, and durable documentation
together under `tools/<product>/`. Keep Codex-specific packaging under
`integrations/codex/` and have it invoke the installed product CLI rather than
embedding a copy. Add a root-level entry here when introducing a new product or
integration family.
