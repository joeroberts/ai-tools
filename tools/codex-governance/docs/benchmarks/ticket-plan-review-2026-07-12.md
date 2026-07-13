# Ticket Plan Review Benchmark - 2026-07-12

## Scope

This synthetic benchmark exercises the bounded `ticket-plan-review` worker
task. The input deliberately contains missing source traceability, an
unbounded path, non-measurable validation, inadequate ADR rationale, and an
undefined dependency. A passing worker must block approval and identify those
defects. It contains no Jira credentials or customer source material.

## Results

| Role | Model | Pinned digest | Result |
| --- | --- | --- | --- |
| Reviewer | `gemma4:12b-mlx` | `117d0d84cf2ab865feb59afc2cd30ff5d55f0035e05eb8d1b814f9688e3f3671` | Pass: returned all five expected blocking findings. |
| Verifier | `qwen3-coder:30b` | `06c1097efce0431c2045fe7b2e5108366e43bee1b4603a7aded8f21689e90bca` | Pass: returned all five expected blocking findings. |

The commands used the owner-only policy fixture at
`testdata/ollama/ticket-plan-review-policy.yaml`, the synthetic prompt at
`testdata/ollama/ticket-plan-review-prompt.txt`, and the governed
`codex-governance ollama run` command. Both results were cached only under a
temporary local runtime directory.

## Promotion Boundary

The fixture is benchmark evidence, not an installed owner policy. Enabling
live plan generation requires an owner-only runtime policy containing these
same pinned entries, an explicit Phase 2 approval, and a representative
non-sensitive smoke run. Any digest change requires a new benchmark.
