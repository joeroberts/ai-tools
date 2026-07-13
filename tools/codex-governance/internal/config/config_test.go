package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "governance.yml")
	content := `format_version: 1
profile: generic
jira:
  project: ""
  issue_key_pattern: "^[A-Z]+-[0-9]+$"
  required_sections: [Scope]
review_budget:
  max_changed_files: 1
  max_changed_lines: 1
  max_components: 1
ci:
  provider: github-actions
  mode: warn
upstream: {}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadSigningConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "governance.yml")
	publicKey := base64.StdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))
	content := fmt.Sprintf(`format_version: 1
profile: generic
jira:
  issue_key_pattern: "^[A-Z]+-[0-9]+$"
  required_sections: [Scope]
review_budget:
  max_changed_files: 1
  max_changed_lines: 1
  max_components: 1
ci:
  provider: github-actions
  mode: warn
upstream: {}
signing:
  format_version: 1
  trusted_keys:
    - key_id: repository-owner-1
      role: repository-owner
      algorithm: ed25519
      public_key: %q
`, publicKey)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, err := config.TrustedKeyRegistry(); err != nil {
		t.Fatalf("TrustedKeyRegistry() error = %v", err)
	}
}
