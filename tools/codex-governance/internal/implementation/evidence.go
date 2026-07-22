package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReviewEvidence binds independent passing assessments to one exact diff.
type ReviewEvidence struct {
	FormatVersion int              `json:"format_version"`
	DiffDigest    string           `json:"diff_digest"`
	Reviewer      AssessmentRecord `json:"reviewer"`
	Verifier      AssessmentRecord `json:"verifier"`
}

type AssessmentRecord struct {
	EnvelopePath   string `json:"envelope_path"`
	EnvelopeDigest string `json:"envelope_digest"`
}

// AssessmentEnvelope is the provenance-bearing record emitted by the governed
// assessment path. It is intentionally separate from the normalized findings
// so evidence cannot be fabricated by supplying an arbitrary executor ID.
type AssessmentEnvelope struct {
	FormatVersion   int       `json:"format_version"`
	Provider        string    `json:"provider"`
	Role            string    `json:"role"`
	ModelName       string    `json:"model_name"`
	ModelID         string    `json:"model_id"`
	PolicyDigest    string    `json:"policy_digest"`
	DiffDigest      string    `json:"diff_digest"`
	PromptDigest    string    `json:"prompt_digest"`
	RawOutputPath   string    `json:"raw_output_path"`
	RawOutputDigest string    `json:"raw_output_digest"`
	FindingsPath    string    `json:"findings_path"`
	FindingsDigest  string    `json:"findings_digest"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
}

func SaveAssessmentEnvelope(path string, envelope AssessmentEnvelope) error {
	if err := envelope.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return fmt.Errorf("refusing to overwrite assessment envelope")
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func LoadAssessmentEnvelope(path string) (AssessmentEnvelope, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return AssessmentEnvelope{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var envelope AssessmentEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return AssessmentEnvelope{}, fmt.Errorf("parse assessment envelope: %w", err)
	}
	if decoder.More() {
		return AssessmentEnvelope{}, fmt.Errorf("parse assessment envelope: multiple JSON values")
	}
	if err := envelope.Validate(); err != nil {
		return AssessmentEnvelope{}, err
	}
	return envelope, nil
}

func (e AssessmentEnvelope) Validate() error {
	if e.FormatVersion != 1 || e.Provider != "local" || (e.Role != "reviewer" && e.Role != "verifier") || e.ModelName == "" || e.ModelID == "" || e.PolicyDigest == "" || e.DiffDigest == "" || e.PromptDigest == "" || e.RawOutputPath == "" || e.RawOutputDigest == "" || e.FindingsPath == "" || e.FindingsDigest == "" || e.StartedAt.IsZero() || e.CompletedAt.IsZero() || e.CompletedAt.Before(e.StartedAt) {
		return fmt.Errorf("assessment envelope is incomplete or invalid")
	}
	return nil
}

// ValidateReviewEvidence ensures both distinct assessments pass and bind the
// supplied diff bytes and require provenance-bearing local assessment records.
func ValidateReviewEvidence(path string, diff []byte) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	var evidence ReviewEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return fmt.Errorf("parse review evidence: %w", err)
	}
	if evidence.FormatVersion != 1 || evidence.DiffDigest != digestBytes(diff) {
		return fmt.Errorf("review evidence is incomplete, mismatched, or not independent")
	}
	roles := []string{"reviewer", "verifier"}
	identities := map[string]bool{}
	for index, record := range []AssessmentRecord{evidence.Reviewer, evidence.Verifier} {
		data, err := os.ReadFile(filepath.Clean(record.EnvelopePath))
		if err != nil || record.EnvelopePath == "" || record.EnvelopeDigest != digestBytes(data) {
			return fmt.Errorf("review evidence envelope is missing or altered")
		}
		envelope, err := LoadAssessmentEnvelope(record.EnvelopePath)
		if err != nil || envelope.Role != roles[index] || envelope.DiffDigest != evidence.DiffDigest {
			return fmt.Errorf("review evidence envelope does not bind the required role and diff")
		}
		raw, rawErr := os.ReadFile(filepath.Clean(envelope.RawOutputPath))
		findings, findingsErr := os.ReadFile(filepath.Clean(envelope.FindingsPath))
		if rawErr != nil || findingsErr != nil || digestBytes(raw) != envelope.RawOutputDigest || digestBytes(findings) != envelope.FindingsDigest {
			return fmt.Errorf("review evidence envelope provenance artifact is missing or altered")
		}
		assessment, err := LoadAssessment(envelope.FindingsPath)
		if err != nil || hasActionableFinding(assessment.Findings) {
			return fmt.Errorf("review evidence assessment does not pass")
		}
		identity := envelope.Provider + ":" + envelope.ModelID
		if identities[identity] {
			return fmt.Errorf("review evidence reviewer and verifier are not independent")
		}
		identities[identity] = true
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
