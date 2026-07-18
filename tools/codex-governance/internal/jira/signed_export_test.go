package jira

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codex-governance/internal/signature"
)

func TestLoadSignedOfflineExport(t *testing.T) {
	export := fixtureExport(t)
	now := time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC)
	path, registry := signedExport(t, export, "export-issuer", now.Add(-time.Hour), now.Add(time.Hour))

	loaded, err := LoadSignedOfflineExport(path, registry, 24*time.Hour, now)
	if err != nil {
		t.Fatalf("LoadSignedOfflineExport() error = %v", err)
	}
	if loaded.Export != export || loaded.Evidence.IssuerKeyID != "fixture-issuer" || loaded.Evidence.TrustedKeyRegistryVersion != signature.FormatVersion {
		t.Fatalf("LoadSignedOfflineExport() = %#v", loaded)
	}
}

func TestLoadSignedOfflineExportRejectsInvalidInputs(t *testing.T) {
	export := fixtureExport(t)
	now := time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC)
	path, registry := signedExport(t, export, "export-issuer", now.Add(-time.Hour), now.Add(time.Hour))
	if _, err := LoadSignedOfflineExport(path, registry, 30*time.Minute, now); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted an expired export")
	}
	if _, err := LoadSignedOfflineExport(path, signature.Registry{}, 24*time.Hour, now); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted a revoked issuer")
	}
	expiredEnvelopePath, expiredEnvelopeRegistry := signedExport(t, export, "export-issuer", now.Add(-2*time.Hour), now.Add(-time.Minute))
	if _, err := LoadSignedOfflineExport(expiredEnvelopePath, expiredEnvelopeRegistry, 24*time.Hour, now); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted an expired envelope")
	}
	wrongRolePath, wrongRoleRegistry := signedExport(t, export, "technical-owner", now.Add(-time.Hour), now.Add(time.Hour))
	if _, err := LoadSignedOfflineExport(wrongRolePath, wrongRoleRegistry, 24*time.Hour, now); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted a non-export issuer role")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope signature.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Payload = []byte(`{"captured_at":"2026-07-11T10:00:00Z","story":{},"subtask":{}}`)
	tampered, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignedOfflineExport(path, registry, 24*time.Hour, now); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted an altered payload")
	}
}

func TestLoadSignedOfflineExportRejectsUnsignedFixture(t *testing.T) {
	registry, err := signature.NewRegistry(signature.FormatVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignedOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"), registry, 24*time.Hour, time.Now().UTC()); err == nil {
		t.Fatal("LoadSignedOfflineExport() accepted an unsigned export")
	}
}

func TestLoadOfflineExportRejectsMissingOrMalformedStatusEvidence(t *testing.T) {
	valid, err := os.ReadFile(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]byte) []byte{
		"missing": func(data []byte) []byte {
			return bytes.Replace(data, []byte("    \"status\": \"In Progress\",\n"), nil, 1)
		},
		"malformed": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"status": "In Progress"`), []byte(`"status": {"name":"In Progress"}`), 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := mutate(valid)
			path := filepath.Join(t.TempDir(), "export.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadOfflineExport(path); err == nil {
				t.Fatal("LoadOfflineExport() accepted invalid status evidence")
			}
		})
	}
}

func TestLoadSignedOfflineExportRejectsInvalidStatusEvidence(t *testing.T) {
	export := fixtureExport(t)
	now := time.Date(2026, 7, 11, 11, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(export)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func([]byte) []byte{
		"missing": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"status":"In Progress",`), nil, 1)
		},
		"blank": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"status":"In Progress"`), []byte(`"status":" \t "`), 1)
		},
		"malformed": func(data []byte) []byte {
			return bytes.Replace(data, []byte(`"status":"In Progress"`), []byte(`"status":{"name":"In Progress"}`), 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			path, registry := signedExportPayload(t, mutate(payload), "export-issuer", now.Add(-time.Hour), now.Add(time.Hour))
			if _, err := LoadSignedOfflineExport(path, registry, 24*time.Hour, now); err == nil {
				t.Fatal("LoadSignedOfflineExport() accepted invalid status evidence")
			}
		})
	}
}

func fixtureExport(t *testing.T) OfflineExport {
	t.Helper()
	export, err := LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	return export
}

func signedExport(t *testing.T, export OfflineExport, role string, issuedAt, expiresAt time.Time) (string, signature.Registry) {
	t.Helper()
	payload, err := json.Marshal(export)
	if err != nil {
		t.Fatal(err)
	}
	return signedExportPayload(t, payload, role, issuedAt, expiresAt)
}

func signedExportPayload(t *testing.T, payload []byte, role string, issuedAt, expiresAt time.Time) (string, signature.Registry) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := signature.Sign(payload, "fixture-issuer", role, privateKey, issuedAt, &expiresAt)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signed-export.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := signature.NewRegistry(signature.FormatVersion, []signature.TrustedKey{{KeyID: "fixture-issuer", Role: role, Algorithm: signature.Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}})
	if err != nil {
		t.Fatal(err)
	}
	return path, registry
}
