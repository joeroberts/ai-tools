#!/usr/bin/env bash
# Read-only owner audit for the proposed repository-settings baseline.
set -euo pipefail

usage() {
  echo "usage: audit-github-repository-settings.sh --repo OWNER/REPO" >&2
}

repo=""
if [[ $# -eq 2 && $1 == "--repo" ]]; then
  repo=$2
else
  usage
  exit 2
fi
if [[ ! $repo =~ ^[^/[:space:]]+/[^/[:space:]]+$ ]]; then
  usage
  exit 2
fi
if ! command -v gh >/dev/null || ! command -v jq >/dev/null; then
  echo "audit_status=drift reason=required_tool_unavailable"
  exit 1
fi

get() {
  gh api --method GET "$1" 2>/dev/null
}

if ! repository=$(get "repos/$repo"); then
  echo "audit_status=drift reason=repository_unavailable"
  exit 1
fi
if ! protection=$(get "repos/$repo/branches/main/protection"); then
  echo "audit_status=drift reason=main_protection_unavailable"
  exit 1
fi
if ! rulesets=$(get "repos/$repo/rulesets"); then
  echo "audit_status=drift reason=rulesets_unavailable"
  exit 1
fi

bool() {
  jq -r "$1 | if . == true then \"enabled\" elif . == false then \"disabled\" else \"unavailable\" end" <<<"$repository"
}

actual_checks=$(jq -r '.required_status_checks.contexts[]? // empty' <<<"$protection" | LC_ALL=C sort -u)
expected_checks=$'advisory\ngo\nsemantic-version'
checks_status=aligned
if [[ $actual_checks != "$expected_checks" ]]; then
	checks_status=drift
fi

echo "audit_status=$([[ $checks_status == aligned ]] && echo aligned || echo drift)"
echo "required_checks=$checks_status"
echo "ruleset_count=$(jq -r 'if type == "array" then length else 0 end' <<<"$rulesets")"
echo "squash_merge=$(bool '.allow_squash_merge')"
echo "merge_commits=$(bool '.allow_merge_commit')"
echo "rebase_merge=$(bool '.allow_rebase_merge')"
echo "delete_branch_on_merge=$(bool '.delete_branch_on_merge')"
echo "auto_merge=$(bool '.allow_auto_merge')"
echo "secret_scanning=$(bool '.security_and_analysis.secret_scanning.status == "enabled"')"
echo "push_protection=$(bool '.security_and_analysis.secret_scanning_push_protection.status == "enabled"')"
if [[ $checks_status == drift ]]; then
	exit 1
fi
