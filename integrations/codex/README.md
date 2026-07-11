# Codex Integrations

This directory contains Codex-specific adoption layers for products under
`tools/`.

- `plugins/` contains plugin packages.
- `skills/` contains standalone skills.
- `.agents/plugins/marketplace.json` is the repository marketplace registry.

Each plugin owns a `scripts/register-local.sh` command when it supports local
development registration. That script must create its symlink under
`~/.codex/plugins`, add this marketplace through `codex plugin marketplace
add`, and install the plugin through `codex plugin add` without overwriting an
unrelated local path.
