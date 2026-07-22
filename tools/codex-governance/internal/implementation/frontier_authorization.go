package implementation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

// FrontierAssessmentAuthorization is the signed, single-assessment owner
// grant required before a frontier subagent can be dispatched. It deliberately
// carries no fallback semantics or publication authority.
type FrontierAssessmentAuthorization struct {
	FormatVersion   int       `json:"format_version"`
	Provider        string    `json:"provider"`
	Role            string    `json:"role"`
	Model           string    `json:"model"`
	ReasoningEffort string    `json:"reasoning_effort"`
	WorkItem        string    `json:"work_item"`
	DiffDigest      string    `json:"diff_digest"`
	ScopeDigest     string    `json:"scope_digest"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	ConsumptionID   string    `json:"consumption_id"`
}

func ValidateFrontierAssessmentAuthorization(envelope signature.Envelope, registry signature.Registry, policy config.FrontierSubagentPolicy, role, workItem, diffDigest, scopeDigest string, now time.Time) (FrontierAssessmentAuthorization, error) {
	if !policy.Enabled {
		return FrontierAssessmentAuthorization{}, fmt.Errorf("frontier assessment provider is disabled")
	}
	if err := registry.Verify(envelope, []string{"repository-owner", "technical-owner"}, now); err != nil {
		return FrontierAssessmentAuthorization{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Payload))
	decoder.DisallowUnknownFields()
	var authorization FrontierAssessmentAuthorization
	if err := decoder.Decode(&authorization); err != nil {
		return FrontierAssessmentAuthorization{}, fmt.Errorf("parse frontier assessment authorization: %w", err)
	}
	if decoder.More() {
		return FrontierAssessmentAuthorization{}, fmt.Errorf("parse frontier assessment authorization: multiple JSON values")
	}
	if authorization.FormatVersion != 1 || authorization.Provider != "frontier-subagent" || (authorization.Role != "reviewer" && authorization.Role != "verifier") || authorization.Role != role || authorization.WorkItem != workItem || authorization.DiffDigest != diffDigest || authorization.ScopeDigest != scopeDigest || authorization.ConsumptionID == "" || authorization.IssuedAt.IsZero() || authorization.ExpiresAt.IsZero() || !authorization.ExpiresAt.After(now) || authorization.ExpiresAt.Before(authorization.IssuedAt) {
		return FrontierAssessmentAuthorization{}, fmt.Errorf("frontier assessment authorization is incomplete, stale, or mismatched")
	}
	if !policy.AllowsFrontierAssessment(authorization.Model, authorization.ReasoningEffort) {
		return FrontierAssessmentAuthorization{}, fmt.Errorf("frontier assessment authorization requests a model or effort outside repository policy")
	}
	return authorization, nil
}

// ConsumeFrontierAssessmentAuthorization atomically records a one-time
// authorization use. A second invocation with the same consumption ID fails
// closed and never overwrites the first record.
func ConsumeFrontierAssessmentAuthorization(directory string, authorization FrontierAssessmentAuthorization, envelope signature.Envelope) (string, error) {
	if authorization.ConsumptionID == "" || envelope.PayloadDigest == "" {
		return "", fmt.Errorf("frontier assessment authorization is incomplete")
	}
	if err := os.MkdirAll(filepath.Clean(directory), 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Clean(directory), authorization.ConsumptionID+".json")
	record := struct {
		ConsumptionID string    `json:"consumption_id"`
		PayloadDigest string    `json:"payload_digest"`
		ConsumedAt    time.Time `json:"consumed_at"`
	}{authorization.ConsumptionID, envelope.PayloadDigest, time.Now().UTC()}
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return "", fmt.Errorf("frontier assessment authorization has already been consumed")
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	return path, nil
}
