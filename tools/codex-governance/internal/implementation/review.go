package implementation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Finding struct {
	ID        string `json:"id"`
	Severity  string `json:"severity"`
	Location  string `json:"location,omitempty"`
	Condition string `json:"condition,omitempty"`
	Summary   string `json:"summary"`
}

type Assessment struct {
	Findings []Finding `json:"findings"`
}

func SaveAssessment(path string, assessment Assessment) error {
	data, err := json.MarshalIndent(assessment, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return fmt.Errorf("refusing to overwrite assessment")
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

// SaveRawAssessment preserves a malformed model response for local diagnosis.
// It is owner-only and never overwrites an earlier failure artifact.
func SaveRawAssessment(path string, response []byte) (string, error) {
	rawPath := filepath.Clean(path) + ".raw"
	if err := os.MkdirAll(filepath.Dir(rawPath), 0o700); err != nil {
		return "", err
	}
	if _, err := os.Stat(rawPath); err == nil {
		return "", fmt.Errorf("refusing to overwrite raw assessment response")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.WriteFile(rawPath, response, 0o600); err != nil {
		return "", err
	}
	return rawPath, nil
}

func LoadAssessment(path string) (Assessment, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Assessment{}, err
	}
	var assessment Assessment
	if err := json.Unmarshal(data, &assessment); err != nil {
		return Assessment{}, fmt.Errorf("parse assessment: %w", err)
	}
	for _, finding := range assessment.Findings {
		if finding.ID == "" || finding.Summary == "" || !validSeverity(finding.Severity) {
			return Assessment{}, fmt.Errorf("assessment contains an invalid finding")
		}
	}
	return assessment, nil
}

// ApplyReview advances an independently produced review. Blocking findings
// require remediation; important findings require remediation in this initial
// implementation because accepted-risk records are not yet implemented.
func ApplyReview(run *Run, assessment Assessment) error {
	if run.State != StateReview {
		return fmt.Errorf("run is not ready for review")
	}
	if hasActionableFinding(assessment.Findings) {
		if run.ReviewCycles >= 2 {
			return run.Transition(StateEscalated)
		}
		run.ReviewCycles++
		return run.Transition(StateRemediation)
	}
	return run.Transition(StateVerification)
}

func ApplyVerification(run *Run, assessment Assessment) error {
	if run.State != StateVerification {
		return fmt.Errorf("run is not ready for verification")
	}
	if hasActionableFinding(assessment.Findings) {
		if run.ReviewCycles >= 2 {
			return run.Transition(StateEscalated)
		}
		run.ReviewCycles++
		return run.Transition(StateRemediation)
	}
	return run.Transition(StateReadyToCommit)
}

// ApplyRemediation requires the caller to name every actionable finding being
// addressed. It never accepts a blanket "fix review" transition.
func ApplyRemediation(run *Run, assessment Assessment, findingIDs []string) error {
	if run.State != StateRemediation || len(findingIDs) == 0 {
		return fmt.Errorf("remediation requires a run in remediation and finding IDs")
	}
	actionable := map[string]bool{}
	for _, finding := range assessment.Findings {
		if finding.Severity == "blocking" || finding.Severity == "important" {
			actionable[finding.ID] = true
		}
	}
	for _, id := range findingIDs {
		if !actionable[id] {
			return fmt.Errorf("remediation finding %q is not actionable", id)
		}
	}
	return run.Transition(StateReview)
}

func hasActionableFinding(findings []Finding) bool {
	for _, finding := range findings {
		if finding.ID == "" || finding.Summary == "" || !validSeverity(finding.Severity) {
			return true
		}
		if finding.Severity == "blocking" || finding.Severity == "important" {
			return true
		}
	}
	return false
}

func validSeverity(value string) bool {
	return value == "blocking" || value == "important" || value == "minor" || value == "informational"
}
