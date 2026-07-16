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
	return createLocalSigner(path, "export-issuer")
}

// CreateLocalRepositoryOwnerSigner creates the owner-only private key used to
// authorize a repository's remote publication. The caller must explicitly add
// the returned public key to repository policy.
func CreateLocalRepositoryOwnerSigner(path string) (TrustedKey, error) {
	return createLocalSigner(path, "repository-owner")
}

func createLocalSigner(path, role string) (TrustedKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return TrustedKey{}, err
	}
	keyID := Digest(publicKey)[:20]
	signer := LocalSigner{KeyID: keyID, PrivateKey: base64.StdEncoding.EncodeToString(privateKey)}
	data := []byte(fmt.Sprintf("{\n  \"key_id\": %q,\n  \"private_key\": %q\n}\n", signer.KeyID, signer.PrivateKey))
	if err := ensureOwnerOnlyDirectory(filepath.Dir(path)); err != nil {
		return TrustedKey{}, err
	}
	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return TrustedKey{}, fmt.Errorf("refusing to overwrite local signer: %s", path)
	}
	if err != nil {
		return TrustedKey{}, err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(filepath.Clean(path))
		if writeErr != nil {
			return TrustedKey{}, writeErr
		}
		return TrustedKey{}, closeErr
	}
	return TrustedKey{KeyID: keyID, Role: role, Algorithm: Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}, nil
}

// LoadLocalExportSigner loads owner-only private material for a local export
// operation. The caller must never serialize the returned private key.
func LoadLocalExportSigner(path string) (string, ed25519.PrivateKey, error) {
	return loadLocalSigner(path)
}

// LoadLocalRepositoryOwnerSigner loads owner-only private material used solely
// for explicit publication authorization issuance.
func LoadLocalRepositoryOwnerSigner(path string) (TrustedKey, ed25519.PrivateKey, error) {
	keyID, privateKey, err := loadLocalSigner(path)
	if err != nil {
		return TrustedKey{}, nil, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return TrustedKey{
		KeyID:     keyID,
		Role:      "repository-owner",
		Algorithm: Algorithm,
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
	}, privateKey, nil
}

func loadLocalSigner(path string) (string, ed25519.PrivateKey, error) {
	if err := validateOwnerOnlyDirectory(filepath.Dir(path)); err != nil {
		return "", nil, err
	}
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
	key := ed25519.PrivateKey(privateKey)
	if signer.KeyID != Digest(key.Public().(ed25519.PublicKey))[:20] {
		return "", nil, fmt.Errorf("local signer key ID does not match private key")
	}
	return signer.KeyID, key, nil
}

func ensureOwnerOnlyDirectory(path string) error {
	if err := os.MkdirAll(filepath.Clean(path), 0o700); err != nil {
		return err
	}
	return validateOwnerOnlyDirectory(path)
}

func validateOwnerOnlyDirectory(path string) error {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("local signer directory must be owner-only")
	}
	return nil
}
