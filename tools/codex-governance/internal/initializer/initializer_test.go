package initializer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeCreatesFiles(t *testing.T) {
	root := t.TempDir()
	created, err := Initialize(root)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if len(created) == 0 {
		t.Fatal("Initialize() created no files")
	}
	if _, err := os.Stat(filepath.Join(root, "governance.yml")); err != nil {
		t.Fatalf("governance.yml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "governance", "templates", "roles", "reviewer.md")); err != nil {
		t.Fatalf("role template missing: %v", err)
	}
}

func TestInitializeRefusesOverwrite(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "governance.yml"), []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(root); err == nil {
		t.Fatal("Initialize() succeeded with an existing governance.yml")
	}
}
