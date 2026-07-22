package implementation

import (
	"fmt"
	"time"
)

// RequiredMergeChecks are the stable, read-only checks established by #18 and
// #44. A merge is eligible only when all are required on the target branch and
// have passed for the exact pull-request head commit.
var RequiredMergeChecks = []string{"go", "advisory", "semantic-version"}

type MergeCheckEvidence struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type MergeEvidence struct {
	PullRequestURL string               `json:"pull_request_url"`
	HeadSHA        string               `json:"head_sha"`
	HeadBranch     string               `json:"head_branch"`
	TargetBranch   string               `json:"target_branch"`
	MergeState     string               `json:"merge_state"`
	RequiredChecks []string             `json:"required_checks"`
	Checks         []MergeCheckEvidence `json:"checks"`
	ObservedAt     time.Time            `json:"observed_at"`
}

func ValidateMergeEvidence(evidence MergeEvidence, run Run, maxAge time.Duration, now time.Time) error {
	if maxAge <= 0 || evidence.PullRequestURL == "" || evidence.HeadSHA == "" || evidence.HeadBranch == "" || evidence.TargetBranch == "" || evidence.ObservedAt.IsZero() || now.Sub(evidence.ObservedAt) > maxAge || evidence.ObservedAt.After(now.Add(time.Minute)) {
		return fmt.Errorf("merge evidence is incomplete or stale")
	}
	if run.CommitSHA == "" || run.Branch == "" || evidence.HeadSHA != run.CommitSHA || evidence.HeadBranch != run.Branch || evidence.MergeState != "CLEAN" {
		return fmt.Errorf("merge evidence does not match the authorized pull request target")
	}
	for _, required := range RequiredMergeChecks {
		if !contains(evidence.RequiredChecks, required) {
			return fmt.Errorf("required branch check %q is absent", required)
		}
		passed := false
		for _, check := range evidence.Checks {
			if check.Name == required && check.Status == "COMPLETED" && check.Conclusion == "SUCCESS" {
				passed = true
			}
		}
		if !passed {
			return fmt.Errorf("required check %q has not passed", required)
		}
	}
	return nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
