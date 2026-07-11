package config

import (
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
