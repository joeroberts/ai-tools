package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReviewEvidence binds independent passing assessments to one exact diff.
type ReviewEvidence struct {
	FormatVersion int              `json:"format_version"`
	DiffDigest    string           `json:"diff_digest"`
	Reviewer      AssessmentRecord `json:"reviewer"`
	Verifier      AssessmentRecord `json:"verifier"`
}

type AssessmentRecord struct {
	ExecutorID       string `json:"executor_id"`
	AssessmentPath   string `json:"assessment_path"`
	AssessmentDigest string `json:"assessment_digest"`
}

// ValidateReviewEvidence ensures both distinct assessments pass and bind the
// supplied diff bytes. Evidence records are owner-created local artifacts.
func ValidateReviewEvidence(path string, diff []byte) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	var evidence ReviewEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return fmt.Errorf("parse review evidence: %w", err)
	}
	if evidence.FormatVersion != 1 || evidence.DiffDigest != digestBytes(diff) || evidence.Reviewer.ExecutorID == "" || evidence.Verifier.ExecutorID == "" || evidence.Reviewer.ExecutorID == evidence.Verifier.ExecutorID {
		return fmt.Errorf("review evidence is incomplete, mismatched, or not independent")
	}
	for _, record := range []AssessmentRecord{evidence.Reviewer, evidence.Verifier} {
		assessmentData, err := os.ReadFile(filepath.Clean(record.AssessmentPath))
		if err != nil || record.AssessmentPath == "" || record.AssessmentDigest != digestBytes(assessmentData) {
			return fmt.Errorf("review evidence assessment artifact is missing or altered")
		}
		assessment, err := LoadAssessment(record.AssessmentPath)
		if err != nil || hasActionableFinding(assessment.Findings) {
			return fmt.Errorf("review evidence assessment does not pass")
		}
	}
	return nil
}

func DiffBytes(worktree string, args ...string) ([]byte, error) {
	output, err := git(worktree, append([]string{"diff"}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("read diff for review evidence: %w: %s", err, output)
	}
	return output, nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
