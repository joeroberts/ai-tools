package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-governance/internal/gitdiff"
	"codex-governance/internal/jira"
	"codex-governance/internal/workitem"
)

type Violation struct {
	Code    string
	Message string
}

func Evaluate(item workitem.Item, export jira.OfflineExport, repoRoot, baseSHA, headSHA string) ([]Violation, error) {
	violations := structuralViolations(item)
	violations = append(violations, driftViolations(item, export)...)
	violations = append(violations, decisionViolations(item, repoRoot)...)

	if baseSHA == "" {
		baseSHA = item.GitRange.BaseSHA
	}
	if headSHA == "" {
		headSHA = item.GitRange.HeadSHA
	}
	if baseSHA != item.GitRange.BaseSHA || headSHA != item.GitRange.HeadSHA {
		violations = append(violations, Violation{Code: "git-range-mismatch", Message: "requested Git range does not match the work item"})
		return violations, nil
	}
	changes, err := gitdiff.Changes(repoRoot, baseSHA, headSHA)
	if err != nil {
		return nil, err
	}
	violations = append(violations, scopeViolations(item, changes)...)
	return violations, nil
}

func structuralViolations(item workitem.Item) []Violation {
	issues := item.Validate()
	violations := make([]Violation, 0, len(issues)+1)
	for _, issue := range issues {
		violations = append(violations, Violation{Code: "invalid-work-item", Message: issue})
	}
	if item.GovernanceStatus != "ready" && (item.Links.PullRequest == nil || *item.Links.PullRequest == "") {
		violations = append(violations, Violation{Code: "missing-pr-link", Message: "non-ready work items require a pull request link"})
	}
	return violations
}

func driftViolations(item workitem.Item, export jira.OfflineExport) []Violation {
	var violations []Violation
	if export.Story.Key != item.Source.StoryKey || export.Subtask.Key != item.Source.SubtaskKey || export.Story.URL != item.Source.StoryURL || export.Subtask.URL != item.Source.SubtaskURL {
		return []Violation{{Code: "source-identity-mismatch", Message: "offline export does not match the recorded story and subtask"}}
	}
	if export.CapturedAt != item.Source.CapturedAt || export.Story.UpdatedAt != item.Source.StoryUpdatedAt || export.Subtask.UpdatedAt != item.Source.SubtaskUpdatedAt {
		violations = append(violations, Violation{Code: "ticket-revision-drift", Message: "offline export capture or issue revision does not match the recorded source"})
	}
	checks := []struct {
		name     string
		actual   string
		expected string
	}{
		{"story description", jira.Digest(export.Story.Description), item.Source.StoryDescriptionDigest},
		{"story acceptance criteria", jira.Digest(export.Story.AcceptanceCriteria), item.Source.StoryAcceptanceCriteriaDigest},
		{"subtask description", jira.Digest(export.Subtask.Description), item.Source.SubtaskDescriptionDigest},
		{"subtask acceptance criteria", jira.Digest(export.Subtask.AcceptanceCriteria), item.Source.SubtaskAcceptanceCriteriaDigest},
	}
	for _, check := range checks {
		if check.actual != check.expected {
			violations = append(violations, Violation{Code: "ticket-drift", Message: check.name + " changed since capture"})
		}
	}
	return violations
}

func decisionViolations(item workitem.Item, repoRoot string) []Violation {
	if strings.HasPrefix(item.Decision.ADR, "No ADR needed: ") {
		return nil
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(item.Decision.ADR))
	if _, err := os.Stat(path); err != nil {
		return []Violation{{Code: "missing-adr", Message: fmt.Sprintf("ADR is unavailable: %s", item.Decision.ADR)}}
	}
	return nil
}

func scopeViolations(item workitem.Item, changes []gitdiff.Change) []Violation {
	var violations []Violation
	components := make(map[string]struct{})
	lines := 0
	for _, change := range changes {
		if !matchesAny(change.Path, item.Scope.AllowedPaths) && !matchesAny(change.Path, item.Scope.FileClassification.GeneratedPaths) && !matchesAny(change.Path, item.Scope.FileClassification.LockfilePaths) {
			violations = append(violations, Violation{Code: "scope-drift", Message: "changed path is outside scope: " + change.Path})
		}
		lines += change.Added + change.Deleted
		component := strings.Split(change.Path, "/")[0]
		components[component] = struct{}{}
	}
	budgetExceeded := len(changes) > item.Scope.ReviewBudget.MaxChangedFiles || lines > item.Scope.ReviewBudget.MaxChangedLines || len(components) > item.Scope.ReviewBudget.MaxComponents
	if len(changes) > item.Scope.ReviewBudget.MaxChangedFiles && item.Links.ReviewException == nil {
		violations = append(violations, Violation{Code: "review-budget", Message: "changed file count exceeds review budget"})
	}
	if lines > item.Scope.ReviewBudget.MaxChangedLines && item.Links.ReviewException == nil {
		violations = append(violations, Violation{Code: "review-budget", Message: "changed line count exceeds review budget"})
	}
	if len(components) > item.Scope.ReviewBudget.MaxComponents && item.Links.ReviewException == nil {
		violations = append(violations, Violation{Code: "review-budget", Message: "changed component count exceeds review budget"})
	}
	if budgetExceeded && item.Links.ReviewException == nil {
		violations = append(violations, Violation{Code: "missing-review-exception", Message: "over-budget work requires an approved review exception"})
	}
	return violations
}

func matchesAny(path string, roots []string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(strings.TrimPrefix(root, "./"), "/")
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}
