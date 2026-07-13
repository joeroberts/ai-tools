// Package signature verifies signed governance records without accessing
// private keys or external services.
package signature

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	FormatVersion = 1
	Algorithm     = "ed25519"
)

// TrustedKey identifies one public key and the governance role it may sign
// for. The public key is standard base64-encoded Ed25519 key material.
type TrustedKey struct {
	KeyID     string
	Role      string
	Algorithm string
	PublicKey string
}

// Registry is the configured, versioned set of public keys trusted by a
// repository. Its version is carried by signed-record consumers.
type Registry struct {
	Version int
	keys    map[string]TrustedKey
}

// Envelope is the common signed governance record. Payload is canonicalized
// before its digest and signature are verified.
type Envelope struct {
	FormatVersion int             `json:"format_version"`
	Payload       json.RawMessage `json:"payload"`
	PayloadDigest string          `json:"payload_digest"`
	KeyID         string          `json:"key_id"`
	Algorithm     string          `json:"algorithm"`
	SignerRole    string          `json:"signer_role"`
	IssuedAt      time.Time       `json:"issued_at"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
	Signature     string          `json:"signature"`
}

func NewRegistry(version int, keys []TrustedKey) (Registry, error) {
	if version != FormatVersion {
		return Registry{}, fmt.Errorf("unsupported trusted-key registry version %d", version)
	}
	registry := Registry{Version: version, keys: make(map[string]TrustedKey, len(keys))}
	for _, key := range keys {
		if err := validateTrustedKey(key); err != nil {
			return Registry{}, err
		}
		if _, exists := registry.keys[key.KeyID]; exists {
			return Registry{}, fmt.Errorf("duplicate trusted key %q", key.KeyID)
		}
		registry.keys[key.KeyID] = key
	}
	return registry, nil
}

// Verify confirms that an envelope is structurally valid, is unexpired, and
// was signed by a configured key for one of the allowed roles.
func (r Registry) Verify(envelope Envelope, allowedRoles []string, now time.Time) error {
	if envelope.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported signed-record version %d", envelope.FormatVersion)
	}
	if envelope.Algorithm != Algorithm {
		return fmt.Errorf("unsupported signature algorithm %q", envelope.Algorithm)
	}
	if envelope.KeyID == "" || envelope.SignerRole == "" || envelope.IssuedAt.IsZero() || envelope.PayloadDigest == "" || envelope.Signature == "" {
		return fmt.Errorf("signed record is missing required fields")
	}
	if envelope.ExpiresAt != nil && !envelope.ExpiresAt.After(now) {
		return fmt.Errorf("signed record is expired")
	}
	if !allowsRole(envelope.SignerRole, allowedRoles) {
		return fmt.Errorf("signer role %q is not allowed", envelope.SignerRole)
	}
	key, ok := r.keys[envelope.KeyID]
	if !ok {
		return fmt.Errorf("trusted key %q is not configured or has been revoked", envelope.KeyID)
	}
	if key.Role != envelope.SignerRole {
		return fmt.Errorf("trusted key %q is not authorized for role %q", envelope.KeyID, envelope.SignerRole)
	}
	canonical, err := Canonicalize(envelope.Payload)
	if err != nil {
		return fmt.Errorf("canonicalize signed payload: %w", err)
	}
	digest := Digest(canonical)
	if envelope.PayloadDigest != digest {
		return fmt.Errorf("signed payload digest does not match payload")
	}
	publicKey, err := decodePublicKey(key.PublicKey)
	if err != nil {
		return fmt.Errorf("decode trusted key %q: %w", key.KeyID, err)
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	digestBytes, _ := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	if !ed25519.Verify(publicKey, digestBytes, signature) {
		return fmt.Errorf("signed record signature is invalid")
	}
	return nil
}

// Sign creates an envelope for tests and approved signing integrations. Callers
// must keep the private key outside repositories and runtime ledgers.
func Sign(payload []byte, keyID, role string, privateKey ed25519.PrivateKey, issuedAt time.Time, expiresAt *time.Time) (Envelope, error) {
	if keyID == "" || role == "" || issuedAt.IsZero() || len(privateKey) != ed25519.PrivateKeySize {
		return Envelope{}, fmt.Errorf("signing inputs are invalid")
	}
	canonical, err := Canonicalize(payload)
	if err != nil {
		return Envelope{}, err
	}
	digest := Digest(canonical)
	digestBytes, _ := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	return Envelope{
		FormatVersion: FormatVersion,
		Payload:       canonical,
		PayloadDigest: digest,
		KeyID:         keyID,
		Algorithm:     Algorithm,
		SignerRole:    role,
		IssuedAt:      issuedAt.UTC(),
		ExpiresAt:     expiresAt,
		Signature:     base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, digestBytes)),
	}, nil
}

// Canonicalize returns a deterministic JSON encoding with object keys sorted.
// It rejects trailing data and non-JSON payloads.
func Canonicalize(payload []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("payload contains multiple JSON values")
		}
		return nil, err
	}
	var out bytes.Buffer
	if err := writeCanonicalJSON(&out, value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func Digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateTrustedKey(key TrustedKey) error {
	if key.KeyID == "" || !isGovernanceRole(key.Role) || key.Algorithm != Algorithm {
		return fmt.Errorf("trusted key is invalid")
	}
	_, err := decodePublicKey(key.PublicKey)
	return err
}

func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public-key size")
	}
	return ed25519.PublicKey(decoded), nil
}

func allowsRole(role string, allowed []string) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}

func isGovernanceRole(role string) bool {
	switch role {
	case "jira-owner", "technical-owner", "repository-owner", "export-issuer":
		return true
	default:
		return false
	}
}

func writeCanonicalJSON(out *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		out.WriteString(strconv.FormatBool(typed))
	case string:
		encoded, _ := json.Marshal(typed)
		out.Write(encoded)
	case json.Number:
		if _, err := strconv.ParseFloat(typed.String(), 64); err != nil {
			return fmt.Errorf("invalid JSON number %q", typed)
		}
		out.WriteString(typed.String())
	case []any:
		out.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				out.WriteByte(',')
			}
			if err := writeCanonicalJSON(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				out.WriteByte(',')
			}
			encoded, _ := json.Marshal(key)
			out.Write(encoded)
			out.WriteByte(':')
			if err := writeCanonicalJSON(out, typed[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("unsupported JSON value %T", value)
	}
	return nil
}
