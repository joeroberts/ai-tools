package jira

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-governance/internal/signature"
)

type OfflineExport struct {
	CapturedAt string `json:"captured_at"`
	Story      Issue  `json:"story"`
	Subtask    Issue  `json:"subtask"`
}

// OfflineExportEvidence records the policy-verified provenance of a governed
// offline export without retaining its full signed envelope in the run ledger.
type OfflineExportEvidence struct {
	EnvelopeDigest            string `json:"envelope_digest"`
	IssuerKeyID               string `json:"issuer_key_id"`
	TrustedKeyRegistryVersion int    `json:"trusted_key_registry_version"`
	CapturedAt                string `json:"captured_at"`
	AppliedMaxAge             string `json:"applied_max_age"`
}

type SignedOfflineExport struct {
	Export   OfflineExport
	Envelope signature.Envelope
	Evidence OfflineExportEvidence
}

type Issue struct {
	Key                string `json:"key"`
	URL                string `json:"url"`
	Status             string `json:"status"`
	UpdatedAt          string `json:"updated_at"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

func LoadOfflineExport(path string) (OfflineExport, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return OfflineExport{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var export OfflineExport
	if err := decoder.Decode(&export); err != nil {
		return OfflineExport{}, fmt.Errorf("parse offline Jira export: %w", err)
	}
	if err := validateOfflineExport(export); err != nil {
		return OfflineExport{}, err
	}
	return export, nil
}

// LoadSignedOfflineExport loads a governed offline export. Raw exports are
// intentionally not accepted here: governed runs require a configured,
// unrevoked export-issuer key and an export within the policy age limit.
func LoadSignedOfflineExport(path string, registry signature.Registry, maxAge time.Duration, now time.Time) (SignedOfflineExport, error) {
	if maxAge <= 0 {
		return SignedOfflineExport{}, fmt.Errorf("offline export maximum age must be positive")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SignedOfflineExport{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope signature.Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return SignedOfflineExport{}, fmt.Errorf("parse signed offline Jira export: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return SignedOfflineExport{}, fmt.Errorf("parse signed offline Jira export: %w", err)
	}
	return VerifySignedOfflineExport(envelope, registry, maxAge, now)
}

// VerifySignedOfflineExport rechecks an in-memory signed envelope against the
// current key registry and age policy before a governed dispatch.
func VerifySignedOfflineExport(envelope signature.Envelope, registry signature.Registry, maxAge time.Duration, now time.Time) (SignedOfflineExport, error) {
	if maxAge <= 0 {
		return SignedOfflineExport{}, fmt.Errorf("offline export maximum age must be positive")
	}
	if err := registry.Verify(envelope, []string{"export-issuer"}, now); err != nil {
		return SignedOfflineExport{}, fmt.Errorf("verify signed offline Jira export: %w", err)
	}
	var export OfflineExport
	payloadDecoder := json.NewDecoder(bytes.NewReader(envelope.Payload))
	payloadDecoder.DisallowUnknownFields()
	if err := payloadDecoder.Decode(&export); err != nil {
		return SignedOfflineExport{}, fmt.Errorf("parse signed offline Jira export payload: %w", err)
	}
	if err := requireEOF(payloadDecoder); err != nil {
		return SignedOfflineExport{}, fmt.Errorf("parse signed offline Jira export payload: %w", err)
	}
	if err := validateOfflineExport(export); err != nil {
		return SignedOfflineExport{}, err
	}
	capturedAt, _ := time.Parse(time.RFC3339, export.CapturedAt)
	age := now.Sub(capturedAt)
	if age < 0 || age > maxAge {
		return SignedOfflineExport{}, fmt.Errorf("signed offline Jira export is outside the policy age limit")
	}
	envelopeDigest, err := digestEnvelope(envelope)
	if err != nil {
		return SignedOfflineExport{}, err
	}
	return SignedOfflineExport{Export: export, Envelope: envelope, Evidence: OfflineExportEvidence{EnvelopeDigest: envelopeDigest, IssuerKeyID: envelope.KeyID, TrustedKeyRegistryVersion: registry.Version, CapturedAt: export.CapturedAt, AppliedMaxAge: maxAge.String()}}, nil
}

func digestEnvelope(envelope signature.Envelope) (string, error) {
	data, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		return "", err
	}
	return signature.Digest(canonical), nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err == nil {
		return fmt.Errorf("input contains multiple JSON values")
	} else {
		return err
	}
}

func validateOfflineExport(export OfflineExport) error {
	if export.Story.Key == "" || export.Subtask.Key == "" || export.Story.URL == "" || export.Subtask.URL == "" || strings.TrimSpace(export.Story.Status) == "" || strings.TrimSpace(export.Subtask.Status) == "" || export.Story.Description == "" || export.Subtask.Description == "" || export.Story.AcceptanceCriteria == "" || export.Subtask.AcceptanceCriteria == "" {
		return fmt.Errorf("offline Jira export is incomplete")
	}
	for _, value := range []string{export.CapturedAt, export.Story.UpdatedAt, export.Subtask.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("offline Jira export timestamp is invalid")
		}
	}
	return nil
}

func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// CreateSignedOfflineExport captures the exact Story/subtask pair and signs
// the normalized snapshot with the configured export issuer.
func CreateSignedOfflineExport(client ReadClient, storyKey, subtaskKey, keyID string, privateKey ed25519.PrivateKey, now time.Time, maxAge time.Duration) (signature.Envelope, error) {
	if maxAge <= 0 {
		return signature.Envelope{}, fmt.Errorf("offline export maximum age must be positive")
	}
	story, err := client.ReadIssue(storyKey)
	if err != nil {
		return signature.Envelope{}, err
	}
	subtask, err := client.ReadIssue(subtaskKey)
	if err != nil {
		return signature.Envelope{}, err
	}
	export := OfflineExport{CapturedAt: now.UTC().Format(time.RFC3339), Story: story, Subtask: subtask}
	payload, err := json.Marshal(export)
	if err != nil {
		return signature.Envelope{}, err
	}
	expires := now.Add(maxAge)
	return signature.Sign(payload, keyID, "export-issuer", privateKey, now, &expires)
}
