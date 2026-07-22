package implementation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateReviewEvidenceRequiresIndependentPassingAssessments(t *testing.T) {
	root := t.TempDir()
	writeAssessment := func(role, modelID string) string {
		path := filepath.Join(root, role+".json")
		if err := SaveAssessment(path, Assessment{}); err != nil {
			t.Fatal(err)
		}
		rawPath := path + ".raw.valid"
		if err := writeAssessmentArtifact(rawPath, []byte("NONE")); err != nil {
			t.Fatal(err)
		}
		findings, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(rawPath)
		if err != nil {
			t.Fatal(err)
		}
		envelopePath := path + ".envelope.json"
		now := time.Now().UTC()
		envelope := AssessmentEnvelope{FormatVersion: 1, Provider: "local", Role: role, ModelName: role + "-model", ModelID: modelID, PolicyDigest: "sha256:policy", DiffDigest: digestBytes([]byte("diff")), PromptDigest: "sha256:prompt", RawOutputPath: rawPath, RawOutputDigest: digestBytes(raw), FindingsPath: path, FindingsDigest: digestBytes(findings), StartedAt: now, CompletedAt: now}
		if err := SaveAssessmentEnvelope(envelopePath, envelope); err != nil {
			t.Fatal(err)
		}
		return envelopePath
	}
	reviewer, verifier := writeAssessment("reviewer", "model-reviewer"), writeAssessment("verifier", "model-verifier")
	record := func(path string) AssessmentRecord {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return AssessmentRecord{EnvelopePath: path, EnvelopeDigest: digestBytes(data)}
	}
	diff := []byte("diff")
	evidence := ReviewEvidence{FormatVersion: 1, DiffDigest: digestBytes(diff), Reviewer: record(reviewer), Verifier: record(verifier)}
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
	if err := os.WriteFile(filepath.Join(root, "reviewer.json.raw.valid"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewEvidence(path, diff); err == nil {
		t.Fatal("accepted altered raw assessment output")
	}
	if err := os.WriteFile(filepath.Join(root, "reviewer.json.raw.valid"), []byte("NONE"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence.Verifier = evidence.Reviewer
	data, _ = json.Marshal(evidence)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewEvidence(path, diff); err == nil {
		t.Fatal("accepted non-independent assessments")
	}
}
