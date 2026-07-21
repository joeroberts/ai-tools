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
  repository_id: github.com/acme/repo
  offline_export_max_age: 2h
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
	age, err := config.OfflineExportMaxAge()
	if err != nil || age.String() != "2h0m0s" {
		t.Fatalf("OfflineExportMaxAge() = %v, %v", age, err)
	}
	if repositoryID, err := config.PublicationRepositoryID(); err != nil || repositoryID != "github.com/acme/repo" {
		t.Fatalf("PublicationRepositoryID() = %q, %v", repositoryID, err)
	}
}

func TestPublicationRepositoryIDRequiresConfiguration(t *testing.T) {
	if _, err := (Config{}).PublicationRepositoryID(); err == nil {
		t.Fatal("PublicationRepositoryID accepted an empty identity")
	}
}

func TestRoadmapAdoptionRejectsUnsafeOrIncompleteConfiguration(t *testing.T) {
	for _, adoption := range []RoadmapAdoption{
		{CanonicalPath: "/tmp/roadmap.yaml", ID: "roadmap", FormatVersion: 1, Enforcement: "required"},
		{CanonicalPath: "docs/roadmap.yaml", ID: "roadmap", FormatVersion: 2, Enforcement: "required"},
		{CanonicalPath: "docs/roadmap.yaml", ID: "roadmap", FormatVersion: 1, Enforcement: "unknown"},
		{CanonicalPath: "docs/*.yaml", ID: "roadmap", FormatVersion: 1, Enforcement: "required"},
		{CanonicalPath: "C:/roadmap.yaml", ID: "roadmap", FormatVersion: 1, Enforcement: "required"},
	} {
		if err := adoption.Validate(); err == nil {
			t.Fatalf("RoadmapAdoption.Validate accepted %#v", adoption)
		}
	}
}

func TestRoadmapAdoptionAcceptsPortableRequiredMapping(t *testing.T) {
	adoption := RoadmapAdoption{CanonicalPath: "governance/roadmaps/program.yaml", ID: "program-v1", FormatVersion: 1, Enforcement: "required"}
	if err := adoption.Validate(); err != nil {
		t.Fatalf("RoadmapAdoption.Validate() error = %v", err)
	}
}
