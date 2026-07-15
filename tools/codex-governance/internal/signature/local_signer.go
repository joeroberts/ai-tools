package signature

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LocalSigner is an owner-only machine-local key for read-only Jira exports.
// Its private material must never be committed or copied into a run ledger.
type LocalSigner struct {
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
}

func CreateLocalExportSigner(path string) (TrustedKey, error) {
	if _, err := os.Stat(path); err == nil {
		return TrustedKey{}, fmt.Errorf("refusing to overwrite local signer: %s", path)
	} else if !os.IsNotExist(err) {
		return TrustedKey{}, err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return TrustedKey{}, err
	}
	keyID := Digest(publicKey)[:20]
	signer := LocalSigner{KeyID: keyID, PrivateKey: base64.StdEncoding.EncodeToString(privateKey)}
	data := []byte(fmt.Sprintf("{\n  \"key_id\": %q,\n  \"private_key\": %q\n}\n", signer.KeyID, signer.PrivateKey))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return TrustedKey{}, err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return TrustedKey{}, err
	}
	return TrustedKey{KeyID: keyID, Role: "export-issuer", Algorithm: Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}, nil
}

// LoadLocalExportSigner loads owner-only private material for a local export
// operation. The caller must never serialize the returned private key.
func LoadLocalExportSigner(path string) (string, ed25519.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	if info.Mode().Perm() != 0o600 {
		return "", nil, fmt.Errorf("local signer must have 0600 permissions")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", nil, err
	}
	var signer LocalSigner
	if err := json.Unmarshal(data, &signer); err != nil {
		return "", nil, fmt.Errorf("parse local signer: %w", err)
	}
	if signer.KeyID == "" {
		return "", nil, fmt.Errorf("local signer is missing key ID")
	}
	privateKey, err := base64.StdEncoding.DecodeString(signer.PrivateKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return "", nil, fmt.Errorf("decode local signer private key")
	}
	return signer.KeyID, ed25519.PrivateKey(privateKey), nil
}
