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
ruleset_id=$(jq -r '[.[] | select(.target == "branch" and .enforcement == "active")][0].id // empty' <<<"$rulesets")
ruleset=""
if [[ -n $ruleset_id ]] && ! ruleset=$(get "repos/$repo/rulesets/$ruleset_id"); then
	echo "audit_status=drift reason=main_ruleset_unavailable"
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

controls_status=$checks_status
if [[ $(jq -r 'if type == "array" then length else 0 end' <<<"$rulesets") -lt 1 ]]; then
	controls_status=drift
fi
ruleset_checks=""
if [[ -z $ruleset ]]; then
	controls_status=drift
else
	ruleset_checks=$(jq -r '[.rules[]? | select(.type == "required_status_checks") | .parameters.required_status_checks[]?.context] | unique | .[]?' <<<"$ruleset")
	if [[ $ruleset_checks != "$expected_checks" ]]; then
		controls_status=drift
	fi
	for rule in pull_request non_fast_forward deletion; do
		if ! jq -e --arg rule "$rule" '.rules[]? | select(.type == $rule)' <<<"$ruleset" >/dev/null; then
			controls_status=drift
		fi
	done
fi
for expectation in \
	".allow_squash_merge:true" \
	".allow_merge_commit:false" \
	".allow_rebase_merge:false" \
	".delete_branch_on_merge:true" \
	".allow_auto_merge:false" \
	".security_and_analysis.secret_scanning.status:enabled" \
	".security_and_analysis.secret_scanning_push_protection.status:enabled"; do
	path=${expectation%%:*}
	expected=${expectation#*:}
	if [[ $(jq -r "$path" <<<"$repository") != "$expected" ]]; then
		controls_status=drift
	fi
done

echo "audit_status=$([[ $controls_status == aligned ]] && echo aligned || echo drift)"
echo "required_checks=$checks_status"
echo "ruleset_count=$(jq -r 'if type == "array" then length else 0 end' <<<"$rulesets")"
echo "ruleset_required_checks=$([[ $ruleset_checks == "$expected_checks" ]] && echo aligned || echo drift)"
echo "squash_merge=$(bool '.allow_squash_merge')"
echo "merge_commits=$(bool '.allow_merge_commit')"
echo "rebase_merge=$(bool '.allow_rebase_merge')"
echo "delete_branch_on_merge=$(bool '.delete_branch_on_merge')"
echo "auto_merge=$(bool '.allow_auto_merge')"
echo "secret_scanning=$(bool '.security_and_analysis.secret_scanning.status == "enabled"')"
echo "push_protection=$(bool '.security_and_analysis.secret_scanning_push_protection.status == "enabled"')"
if [[ $controls_status == drift ]]; then
	exit 1
fi
