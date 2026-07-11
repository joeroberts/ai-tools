# Codex Governance Plugin

This plugin is the Codex adoption layer for the installed
`codex-governance` CLI. Its bundled skill routes agents to deterministic CLI
checks; it does not duplicate governance policy or validation logic.

## Local Registration

From the repository root, run:

```bash
./integrations/codex/plugins/codex-governance/scripts/register-local.sh
```

The script creates a development symlink at
`~/.codex/plugins/codex-governance`, registers the repository marketplace at
`integrations/codex/.agents/plugins/marketplace.json`, and installs
`codex-governance@ai-tools`. It refuses to replace an unrelated existing
symlink or path.

The CLI itself must be installed separately, for example with:

```bash
(cd tools/codex-governance && make install)
```
