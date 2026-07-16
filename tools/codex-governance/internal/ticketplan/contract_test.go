package ticketplan

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAuthorityContractRoundTripAndNoOverwrite(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	contract, err := LoadAuthorityContract(filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "contract.json"), repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "private", "contract.json")
	digest, err := SaveAuthorityContract(output, repoRoot, contract)
	if err != nil || !digestPattern.MatchString(digest) {
		t.Fatalf("SaveAuthorityContract() digest = %q, error = %v", digest, err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("contract permissions = %v, error = %v", info, err)
	}
	loaded, err := LoadAuthorityContract(output, repoRoot)
	if err != nil || !reflect.DeepEqual(loaded, contract) {
		t.Fatalf("round-trip contract: %v", err)
	}
	if savedDigest, err := FileDigest(output); err != nil || savedDigest != digest {
		t.Fatalf("persisted digest = %q, error = %v", savedDigest, err)
	}
	if _, err := SaveAuthorityContract(output, repoRoot, contract); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("overwrite error = %v", err)
	}
}

func TestAuthorityContractRejectsSourceAliasesAndDrift(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "contract.json")
	contract, err := LoadAuthorityContract(path, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	contract.Sources[1].Digest = contract.Sources[0].Digest
	if err := contract.Validate(); err == nil {
		t.Fatal("Validate accepted byte-identical source aliases")
	}
	contract, _ = LoadAuthorityContract(path, repoRoot)
	contract.Sources[0].Path = contract.Sources[1].Path
	if err := contract.Validate(); err == nil {
		t.Fatal("Validate accepted repeated source paths")
	}
	contract, _ = LoadAuthorityContract(path, repoRoot)
	contract.Sources[0].Digest = "sha256:" + strings.Repeat("0", 64)
	if err := contract.ValidateAgainst(repoRoot); err == nil {
		t.Fatal("ValidateAgainst accepted source drift")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	for index, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		if err := os.WriteFile(filepath.Join(root, "docs", name), []byte(strings.Repeat(name, index+1)), 0o600); err != nil {
			t.Fatal(err)
		}
		contract.Sources[index].Path = "docs/" + name
		contract.Sources[index].Digest, _ = FileDigest(filepath.Join(root, "docs", name))
	}
	if err := os.Symlink("prd.md", filepath.Join(root, "docs", "prd-link.md")); err != nil {
		t.Fatal(err)
	}
	contract.Sources[0].Path = "docs/prd-link.md"
	if err := contract.ValidateAgainst(root); err == nil {
		t.Fatal("ValidateAgainst accepted a symlink source alias")
	}
}

func TestAuthorityContractRejectsMalformedDeclaredSlices(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "contract.json")
	tests := []struct {
		name   string
		mutate func(*AuthorityContract)
	}{
		{"duplicate ID", func(c *AuthorityContract) { c.Slices = append(c.Slices, c.Slices[0]) }},
		{"missing dependency", func(c *AuthorityContract) { c.Slices[0].Assignment.Dependencies = []string{"missing"} }},
		{"invalid budget", func(c *AuthorityContract) { c.Slices[0].Assignment.ReviewBudget.MaxChangedLines = 0 }},
		{"malformed path", func(c *AuthorityContract) { c.Slices[0].Assignment.AllowedPaths = []string{"../outside"} }},
		{"missing dependencies", func(c *AuthorityContract) { c.Slices[0].Assignment.Dependencies = nil }},
		{"missing narrative rules", func(c *AuthorityContract) { c.NarrativeRules = nil }},
		{"invalid evidence field", func(c *AuthorityContract) { c.Evidence[0].Field = "manager.selected" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract, err := LoadAuthorityContract(path, repoRoot)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&contract)
			if err := contract.Validate(); err == nil {
				t.Fatal("Validate accepted malformed contract")
			}
		})
	}
}

func TestAuthorityContractRejectsUnknownFieldsAndUnsupportedVersion(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	fixture, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "contract.json")
	bad := strings.Replace(string(fixture), `"format_version": 1`, `"format_version": 2, "unknown": true`, 1)
	if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAuthorityContract(path, repoRoot); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
	contract, _ := LoadAuthorityContract(filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "contract.json"), repoRoot)
	contract.FormatVersion = 2
	if err := contract.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported-version error = %v", err)
	}
}
