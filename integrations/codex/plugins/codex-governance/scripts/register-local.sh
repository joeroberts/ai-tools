#!/usr/bin/env bash
set -euo pipefail

plugin_name="codex-governance"
marketplace_name="ai-tools"
plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
marketplace_root="$(cd "${plugin_root}/../.." && pwd -P)"
link_path="${HOME}/.codex/plugins/${plugin_name}"

mkdir -p "${HOME}/.codex/plugins"

if [[ -L "${link_path}" ]]; then
  existing_target="$(cd "${link_path}" && pwd -P)"
  if [[ "${existing_target}" != "${plugin_root}" ]]; then
    printf 'refusing to replace existing symlink: %s -> %s\n' "${link_path}" "${existing_target}" >&2
    exit 1
  fi
elif [[ -e "${link_path}" ]]; then
  printf 'refusing to replace existing path: %s\n' "${link_path}" >&2
  exit 1
else
  ln -s "${plugin_root}" "${link_path}"
fi

if ! codex plugin marketplace list | awk 'NR > 1 {print $1}' | grep -Fxq "${marketplace_name}"; then
  codex plugin marketplace add "${marketplace_root}"
fi

if ! codex plugin list | awk 'NR > 1 {print $1}' | grep -Fxq "${plugin_name}@${marketplace_name}"; then
  codex plugin add "${plugin_name}@${marketplace_name}"
fi

printf 'registered %s from %s\n' "${plugin_name}" "${plugin_root}"
