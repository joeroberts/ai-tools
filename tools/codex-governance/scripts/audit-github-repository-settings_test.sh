#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
script="$root/tools/codex-governance/scripts/audit-github-repository-settings.sh"
temp=$(mktemp -d)
trap 'rm -rf "$temp"' EXIT

mkdir -p "$temp/bin"
cat >"$temp/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$GH_LOG"
endpoint=${!#}
if [[ ${GH_MODE:-} == missing && $endpoint == repos/example/repo/branches/main/protection ]]; then
  exit 1
fi
case "$endpoint" in
  repos/example/repo)
    printf '%s\n' '{"allow_squash_merge":true,"allow_merge_commit":false,"allow_rebase_merge":false,"delete_branch_on_merge":true,"allow_auto_merge":false,"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"enabled"}},"private_note":"secret-value"}'
    ;;
  repos/example/repo/branches/main/protection)
    printf '%s\n' '{"required_status_checks":{"contexts":["go","advisory","unexpected-check"]}}'
    ;;
  repos/example/repo/rulesets) printf '%s\n' '[]' ;;
  *) exit 1 ;;
esac
EOF
chmod +x "$temp/bin/gh"

PATH="$temp/bin:$PATH" GH_LOG="$temp/gh.log" "$script" --repo example/repo >"$temp/output" || status=$?
if [[ ${status:-0} -ne 1 ]]; then
  echo "expected drift exit, got ${status:-0}" >&2
  exit 1
fi
grep -qx 'audit_status=drift' "$temp/output"
grep -qx 'required_checks=drift' "$temp/output"
grep -qx 'secret_scanning=enabled' "$temp/output"
if grep -Eq 'secret-value|unexpected-check|private_note' "$temp/output"; then
  echo "audit output exposed a raw API value" >&2
  exit 1
fi
if ! grep -Eq '^api --method GET repos/example/repo($|/)' "$temp/gh.log"; then
  echo "audit did not use gh api GET" >&2
  exit 1
fi
if grep -Ev '^api --method GET ' "$temp/gh.log" >/dev/null; then
  echo "audit issued a non-GET request" >&2
  exit 1
fi

set +e
PATH="$temp/bin:$PATH" GH_LOG="$temp/gh-missing.log" GH_MODE=missing "$script" --repo example/repo >"$temp/missing-output"
missing_status=$?
set -e
if [[ $missing_status -ne 1 ]]; then
  echo "expected unavailable-endpoint exit, got $missing_status" >&2
  exit 1
fi
grep -qx 'audit_status=drift reason=main_protection_unavailable' "$temp/missing-output"
