package syncer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.json")
	if err := os.WriteFile(path, []byte(`{"release":"1.0.0","source_commit":"abc1234","format_version":1,"compatibility_range":">=1 <2","artifacts":{"README.md":"sha256:0000000000000000000000000000000000000000000000000000000000000000"},"changelog":"initial","migration_notes":"none"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(path); err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
}

func TestCompare(t *testing.T) {
	changes := Compare(Adoption{}, Manifest{Release: "1.0.0", SourceCommit: "abc1234", FormatVersion: 1})
	if len(changes) != 3 {
		t.Fatalf("Compare() changes = %v", changes)
	}
}

func TestLoadManifestRejectsInvalidSemVer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.json")
	data := `{"release":"01.0.0","source_commit":"abc1234","format_version":1,"compatibility_range":">=1 <2","artifacts":{"README.md":"sha256:0000000000000000000000000000000000000000000000000000000000000000"},"changelog":"initial","migration_notes":"none"}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("LoadManifest() accepted invalid release")
	}
}
