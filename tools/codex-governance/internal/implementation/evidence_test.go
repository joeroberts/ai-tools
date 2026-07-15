package implementation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateReviewEvidenceRequiresIndependentPassingAssessments(t *testing.T) {
	root := t.TempDir()
	writeAssessment := func(name string) string {
		path := filepath.Join(root, name)
		if err := SaveAssessment(path, Assessment{}); err != nil {
			t.Fatal(err)
		}
		return path
	}
	reviewer, verifier := writeAssessment("reviewer.json"), writeAssessment("verifier.json")
	record := func(path string, executor string) AssessmentRecord {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return AssessmentRecord{ExecutorID: executor, AssessmentPath: path, AssessmentDigest: digestBytes(data)}
	}
	diff := []byte("diff")
	evidence := ReviewEvidence{FormatVersion: 1, DiffDigest: digestBytes(diff), Reviewer: record(reviewer, "reviewer-1"), Verifier: record(verifier, "verifier-1")}
	data, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "evidence.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewEvidence(path, diff); err != nil {
		t.Fatal(err)
	}
	evidence.Verifier.ExecutorID = evidence.Reviewer.ExecutorID
	data, _ = json.Marshal(evidence)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewEvidence(path, diff); err == nil {
		t.Fatal("accepted non-independent assessments")
	}
}
