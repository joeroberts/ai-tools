package implementation

import (
	"strings"
	"testing"
	"time"
)

func TestValidateMergeEvidenceRequiresExactTargetAndPassingChecks(t *testing.T) {
	now := time.Now().UTC()
	run := Run{CommitSHA: strings.Repeat("a", 40), Branch: "codex/REK-97"}
	evidence := MergeEvidence{PullRequestURL: "https://github.test/acme/repo/pull/7", HeadSHA: run.CommitSHA, HeadBranch: run.Branch, TargetBranch: "main", MergeState: "CLEAN", RequiredChecks: append([]string(nil), RequiredMergeChecks...), Checks: []MergeCheckEvidence{{Name: "go", Status: "COMPLETED", Conclusion: "SUCCESS"}, {Name: "advisory", Status: "COMPLETED", Conclusion: "SUCCESS"}, {Name: "semantic-version", Status: "COMPLETED", Conclusion: "SUCCESS"}}, ObservedAt: now}
	if err := ValidateMergeEvidence(evidence, run, time.Minute, now); err != nil {
		t.Fatal(err)
	}
	evidence.Checks[1].Conclusion = "FAILURE"
	if err := ValidateMergeEvidence(evidence, run, time.Minute, now); err == nil {
		t.Fatal("failed required check was accepted")
	}
	evidence.Checks[1].Conclusion = "SUCCESS"
	evidence.HeadSHA = strings.Repeat("b", 40)
	if err := ValidateMergeEvidence(evidence, run, time.Minute, now); err == nil {
		t.Fatal("different pull-request commit was accepted")
	}
	evidence.HeadSHA = run.CommitSHA
	evidence.RequiredChecks = []string{"go", "advisory"}
	if err := ValidateMergeEvidence(evidence, run, time.Minute, now); err == nil {
		t.Fatal("missing branch-protection check was accepted")
	}
}
