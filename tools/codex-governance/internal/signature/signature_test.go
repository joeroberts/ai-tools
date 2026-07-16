package signature

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateLocalExportSignerWritesOwnerOnlyKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime", "export-signer.json")
	key, err := CreateLocalExportSigner(path)
	if err != nil {
		t.Fatal(err)
	}
	if key.KeyID == "" || key.Role != "export-issuer" || key.Algorithm != Algorithm || key.PublicKey == "" {
		t.Fatalf("trusted key = %#v", key)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("signer permissions = %v, %v", info.Mode().Perm(), err)
	}
	if _, err := CreateLocalExportSigner(path); err == nil {
		t.Fatal("CreateLocalExportSigner() overwrote an existing signer")
	}
	keyID, privateKey, err := LoadLocalExportSigner(path)
	if err != nil || keyID != key.KeyID || len(privateKey) != ed25519.PrivateKeySize {
		t.Fatalf("LoadLocalExportSigner() = %q, %d, %v", keyID, len(privateKey), err)
	}
}

func TestRepositoryOwnerSignerRefusesOverwriteAndUnsafePermissions(t *testing.T) {
	permissiveDirectory := t.TempDir()
	if err := os.Chmod(permissiveDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateLocalRepositoryOwnerSigner(filepath.Join(permissiveDirectory, "unsafe-owner.json")); err == nil {
		t.Fatal("CreateLocalRepositoryOwnerSigner() accepted a permissive directory")
	}
	safeDirectory := t.TempDir()
	if err := os.Chmod(safeDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(safeDirectory, "repository-owner.json")
	key, err := CreateLocalRepositoryOwnerSigner(path)
	if err != nil {
		t.Fatal(err)
	}
	if key.Role != "repository-owner" {
		t.Fatalf("signer role = %q", key.Role)
	}
	loadedKey, _, err := LoadLocalRepositoryOwnerSigner(path)
	if err != nil || loadedKey != key {
		t.Fatalf("loaded repository-owner key = %#v, %v", loadedKey, err)
	}
	if _, err := CreateLocalRepositoryOwnerSigner(path); err == nil {
		t.Fatal("CreateLocalRepositoryOwnerSigner() overwrote an existing signer")
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLocalRepositoryOwnerSigner(path); err == nil {
		t.Fatal("LoadLocalRepositoryOwnerSigner() accepted unsafe permissions")
	}
	tamperedDirectory := t.TempDir()
	if err := os.Chmod(tamperedDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	tamperedPath := filepath.Join(tamperedDirectory, "repository-owner.json")
	if _, err := CreateLocalRepositoryOwnerSigner(tamperedPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(tamperedPath)
	if err != nil {
		t.Fatal(err)
	}
	var signer LocalSigner
	if err := json.Unmarshal(data, &signer); err != nil {
		t.Fatal(err)
	}
	signer.KeyID = "sha256:wrong"
	data, err = json.Marshal(signer)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tamperedPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLocalRepositoryOwnerSigner(tamperedPath); err == nil {
		t.Fatal("LoadLocalRepositoryOwnerSigner() accepted mismatched key material")
	}
}

func TestVerifyAcceptsCanonicalPayload(t *testing.T) {
	publicKey, privateKey := fixtureKey(t)
	registry, err := NewRegistry(FormatVersion, []TrustedKey{{KeyID: "repo-owner-1", Role: "repository-owner", Algorithm: Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	envelope, err := Sign([]byte(`{"b":2,"a":1}`), "repo-owner-1", "repository-owner", privateKey, now, &expires)
	if err != nil {
		t.Fatal(err)
	}
	if string(envelope.Payload) != `{"a":1,"b":2}` {
		t.Fatalf("canonical payload = %s", envelope.Payload)
	}
	if err := registry.Verify(envelope, []string{"repository-owner"}, now); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyRejectsInvalidRoleExpiryRevocationAndPayload(t *testing.T) {
	publicKey, privateKey := fixtureKey(t)
	registry, err := NewRegistry(FormatVersion, []TrustedKey{{KeyID: "tech-owner-1", Role: "technical-owner", Algorithm: Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	envelope, err := Sign([]byte(`{"scope":"docs"}`), "tech-owner-1", "technical-owner", privateKey, now, &expires)
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Verify(envelope, []string{"repository-owner"}, now); err == nil {
		t.Fatal("Verify() accepted disallowed role")
	}
	if err := registry.Verify(envelope, []string{"technical-owner"}, expires); err == nil {
		t.Fatal("Verify() accepted expired record")
	}
	revoked, err := NewRegistry(FormatVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := revoked.Verify(envelope, []string{"technical-owner"}, now); err == nil {
		t.Fatal("Verify() accepted revoked key")
	}
	envelope.Payload = []byte(`{"scope":"code"}`)
	if err := registry.Verify(envelope, []string{"technical-owner"}, now); err == nil {
		t.Fatal("Verify() accepted altered payload")
	}
}

func TestCanonicalizeRejectsTrailingJSON(t *testing.T) {
	if _, err := Canonicalize([]byte(`{} {}`)); err == nil {
		t.Fatal("Canonicalize() accepted multiple JSON values")
	}
}

func TestSignRejectsMalformedPayload(t *testing.T) {
	_, privateKey := fixtureKey(t)
	if _, err := Sign([]byte(`{"unterminated":`), "repository-owner-1", "repository-owner", privateKey, time.Now().UTC(), nil); err == nil {
		t.Fatal("Sign() accepted malformed JSON")
	}
}

func TestNewRegistryRejectsUnknownRole(t *testing.T) {
	publicKey, _ := fixtureKey(t)
	if _, err := NewRegistry(FormatVersion, []TrustedKey{{KeyID: "unknown", Role: "operator", Algorithm: Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}}); err == nil {
		t.Fatal("NewRegistry() accepted an unknown role")
	}
}

func fixtureKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return publicKey, privateKey
}
