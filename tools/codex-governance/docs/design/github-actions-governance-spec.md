# GitHub #18: CI and governance workflow specification

## Boundaries

- Workflows run with `contents: read` only and must be safe for fork pull
  requests.
- Go setup uses the pinned project version and dependency caching when it does
  not broaden access or expose private material.
- CI runs deterministic local-equivalent checks. Governance validation emits a
  concise redacted summary and never uploads a prompt, token, offline export,
  runtime ledger, or review-evidence file.
- Missing optional local governance fixtures is a clearly reported advisory
  condition, not a hidden success or external lookup.
- Branch-protection and required-check changes are excluded. This work may
  document a proposed #22 input only.

## Allowed Paths

- `tools/codex-governance/docs/design/github-actions-governance-prd.md`
- `tools/codex-governance/docs/design/github-actions-governance-spec.md`
- `.github/workflows/ci.yml`
- `.github/workflows/governance.yml`
- `tools/codex-governance/.github/workflows/governance-advisory.yml`

## Validation

- Validate workflow syntax and path behavior with focused deterministic tests
  or fixtures where available.
- Run `make test`, `make vet`, `make build`, and `git diff --check`.
- Require independent exact-diff reviewer and verifier evidence before each
  commit and publication action.

## Review Budget

This plan is limited to 5 changed files, 400 changed lines, and design contract, deterministic CI, privacy-safe governance evidence.

## Declared Implementation Slices

```json
[
  {"id":"github-18-design-contract","phase":"Phase 1","change_class":"standard","dependencies":[],"allowed_paths":["tools/codex-governance/docs/design/github-actions-governance-prd.md","tools/codex-governance/docs/design/github-actions-governance-spec.md"],"review_budget":{"max_changed_files":2,"max_changed_lines":160,"components":["design contract"]}},
  {"id":"github-18-deterministic-ci","phase":"Phase 2","change_class":"standard","dependencies":["github-18-design-contract"],"allowed_paths":[".github/workflows/ci.yml"],"review_budget":{"max_changed_files":1,"max_changed_lines":120,"components":["deterministic CI"]}},
  {"id":"github-18-governance-evidence","phase":"Phase 3","change_class":"standard","dependencies":["github-18-design-contract"],"allowed_paths":[".github/workflows/governance.yml","tools/codex-governance/.github/workflows/governance-advisory.yml"],"review_budget":{"max_changed_files":2,"max_changed_lines":120,"components":["privacy-safe governance evidence"]}}
]
```

## Slice Evidence

- `github-18-design-contract` is a standard Phase 1 slice with no dependencies,
  two changed files, 160 changed lines, and design contract. It changes only
  `tools/codex-governance/docs/design/github-actions-governance-prd.md` and
  `tools/codex-governance/docs/design/github-actions-governance-spec.md`.
- `github-18-deterministic-ci` is a standard Phase 2 slice depending on
  `github-18-design-contract`, with one changed file, 120 changed lines, and
  deterministic CI. It changes only `.github/workflows/ci.yml`.
- `github-18-governance-evidence` is a standard Phase 3 slice depending on
  `github-18-design-contract`, with two changed files, 120 changed lines, and
  privacy-safe governance evidence. It changes only
  `.github/workflows/governance.yml` and
  `tools/codex-governance/.github/workflows/governance-advisory.yml`.
