package validate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codex-governance/internal/gitdiff"
	"codex-governance/internal/jira"
	"codex-governance/internal/workitem"
)

func TestEvaluateValidFixture(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA = base
	item.GitRange.HeadSHA = head
	writeADR(t, root)
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("Evaluate() violations = %#v", violations)
	}
}

func TestEvaluateDetectsTicketDrift(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA = base
	item.GitRange.HeadSHA = head
	writeADR(t, root)
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "ticket-drift.json"))
	if err != nil {
		t.Fatal(err)
	}

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "ticket-drift") {
		t.Fatalf("ticket drift violation missing: %#v", violations)
	}
}

func TestEvaluateDetectsMissingADR(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA = base
	item.GitRange.HeadSHA = head
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "missing-adr") {
		t.Fatalf("missing ADR violation missing: %#v", violations)
	}
}

func TestEvaluateAllowsDeclaredPendingADRButWorkingTreeRequiresIt(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	item.Scope.AllowedPaths = append(item.Scope.AllowedPaths, "docs/decisions")
	item.Decision.ADRPreflightPending = true
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if containsCode(violations, "missing-adr") {
		t.Fatalf("preflight validation rejected declared pending ADR: %#v", violations)
	}
	violations, err = EvaluateWorking(item, root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "missing-adr") {
		t.Fatalf("working-tree validation accepted missing pending ADR: %#v", violations)
	}
}

func TestWorkItemRejectsPendingADROutsideAllowedPaths(t *testing.T) {
	item := loadFixture(t)
	item.Decision.ADRPreflightPending = true
	if len(item.Validate()) == 0 {
		t.Fatal("work item accepted pending ADR outside allowed paths")
	}
}

func TestEvaluateDetectsSourceIdentityMismatch(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA = base
	item.GitRange.HeadSHA = head
	writeADR(t, root)
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	export.Story.Key = "CG-99"

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "source-identity-mismatch") {
		t.Fatalf("source identity violation missing: %#v", violations)
	}
}

func TestEvaluateRequiresPullRequestForReview(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA = base
	item.GitRange.HeadSHA = head
	item.Links.PullRequest = nil
	writeADR(t, root)
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}

	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "missing-pr-link") {
		t.Fatalf("missing PR violation missing: %#v", violations)
	}
}

func TestScopeViolationsRequireException(t *testing.T) {
	item := loadFixture(t)
	item.Scope.ReviewBudget.MaxChangedFiles = 1
	violations := scopeViolations(item, []gitdiff.Change{{Path: "internal/a.go"}, {Path: "internal/b.go"}})
	if !containsCode(violations, "missing-review-exception") {
		t.Fatalf("missing review exception violation: %#v", violations)
	}
}

func TestWorkItemRejectsHighRiskExceptionWithoutContainment(t *testing.T) {
	item := loadFixture(t)
	item.Scope.ChangeClass = "high-risk"
	item.Links.ReviewException = &workitem.ReviewException{
		JiraReference: "https://jira.example.test/browse/CG-2?focusedCommentId=1",
		Reason:        "Must land together",
		Approver:      "owner",
		ReviewPlan:    []string{"Review migration"},
		ApprovedAt:    "2026-07-11T10:00:00Z",
	}
	issues := item.Validate()
	if len(issues) == 0 {
		t.Fatal("high-risk exception was accepted without containment")
	}
}

func TestEvaluateDetectsTicketRevisionDrift(t *testing.T) {
	root, base, head := testRepository(t)
	item := loadFixture(t)
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	writeADR(t, root)
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	export.Story.UpdatedAt = "2026-07-11T11:00:00Z"
	violations, err := Evaluate(item, export, root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCode(violations, "ticket-revision-drift") {
		t.Fatalf("revision drift missing: %#v", violations)
	}
}

func TestWorkItemRejectsUnlinkedReviewException(t *testing.T) {
	item := loadFixture(t)
	item.Links.ReviewException = &workitem.ReviewException{Reason: "needed", Approver: "owner", ReviewPlan: []string{"review"}, ApprovedAt: "2026-07-11T10:00:00Z"}
	if len(item.Validate()) == 0 {
		t.Fatal("unlinked review exception was accepted")
	}
}

func loadFixture(t *testing.T) workitem.Item {
	t.Helper()
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func testRepository(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "user.email", "test@example.test")
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "value.go"), []byte("package internal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(root, "internal", "value.go"), []byte("package internal\n\nconst Value = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "head")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	return root, base, head
}

func writeADR(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "docs", "decisions", "ADR-0001.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# ADR\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}

func containsCode(violations []Violation, code string) bool {
	for _, violation := range violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
