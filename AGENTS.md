# Repository Guidelines

## Portfolio Layout

This repository contains products and AI-agent integration layers. Put
independently testable tools in `tools/<product>/`. Put Codex-specific plugins
and standalone skills in `integrations/codex/plugins/` and
`integrations/codex/skills/`. Reserve `shared/` for assets used by more than
one product.

Read the nearest `AGENTS.md` before changing a product. Product instructions
define its build, test, and release conventions; this file defines repository
boundaries only.

## Changes And Validation

Keep changes within one product or integration package unless a cross-product
contract requires otherwise. Update root documentation and CI paths when a
product is moved or introduced. Run the affected product's documented checks
from its directory, plus `git diff --check`.

## Safety

Do not add credentials, tokens, private prompts, or local caches. Do not push,
publish, deploy, write Jira, or modify cloud infrastructure without explicit
approval.
