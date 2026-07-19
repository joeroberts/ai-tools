package signature

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
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

// CreateLocalTechnicalOwnerSigner creates the fixed-role signer used only for
// technical-owner adoption decisions.
func CreateLocalTechnicalOwnerSigner(path string) (TrustedKey, error) {
	if err := ValidateLocalSignerPath(path); err != nil {
		return TrustedKey{}, err
	}
	return createLocalSigner(path, "technical-owner")
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
	return loadLocalTrustedSigner(path, "repository-owner")
}

// LoadLocalTechnicalOwnerSigner reloads a fixed-role technical-owner signer.
func LoadLocalTechnicalOwnerSigner(path string) (TrustedKey, ed25519.PrivateKey, error) {
	return loadLocalTrustedSigner(path, "technical-owner")
}

func loadLocalTrustedSigner(path, role string) (TrustedKey, ed25519.PrivateKey, error) {
	keyID, privateKey, err := loadLocalSigner(path)
	if err != nil {
		return TrustedKey{}, nil, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return TrustedKey{
		KeyID:     keyID,
		Role:      role,
		Algorithm: Algorithm,
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
	}, privateKey, nil
}

// ValidateLocalSignerPath checks a proposed destination without creating
// directories or files. It is intended for no-write command previews.
func ValidateLocalSignerPath(path string) error {
	if _, err := os.Lstat(filepath.Clean(path)); err == nil {
		return fmt.Errorf("refusing to overwrite local signer: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return validateExistingOwnerOnlyAncestor(filepath.Dir(path))
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

func validateExistingOwnerOnlyAncestor(path string) error {
	for ancestor := filepath.Clean(path); ; ancestor = filepath.Dir(ancestor) {
		info, err := os.Lstat(ancestor)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 || ownerID(info) != uint32(os.Getuid()) {
				return fmt.Errorf("local signer directory must be owner-only and not a symlink")
			}
			return validateSignerAncestorChain(ancestor)
		}
		if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return fmt.Errorf("local signer directory has no existing ancestor")
		}
	}
}

// validateSignerAncestorChain rejects directories that another user could
// replace. Root-owned sticky directories such as /tmp are the sole writable
// exception because their sticky bit prevents cross-user replacement.
func validateSignerAncestorChain(path string) error {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	for ancestor := canonical; ; ancestor = filepath.Dir(ancestor) {
		info, err := os.Lstat(ancestor)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("local signer ancestor must be a directory and not a symlink")
		}
		owner := ownerID(info)
		perms := info.Mode().Perm()
		writableByOthers := perms&0o022 != 0
		if writableByOthers {
			if owner != 0 || info.Mode()&os.ModeSticky == 0 {
				return fmt.Errorf("local signer ancestor is replaceable")
			}
		} else if owner != 0 && owner != uint32(os.Getuid()) {
			return fmt.Errorf("local signer ancestor is not owned by the current user or root")
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return nil
		}
	}
}

func ownerID(info os.FileInfo) uint32 {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ^uint32(0)
	}
	return stat.Uid
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
