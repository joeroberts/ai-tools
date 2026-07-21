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
	if _, err := os.Stat(filepath.Join(root, "docs", "governance", "templates", "roadmap-adoption.yaml")); err != nil {
		t.Fatalf("roadmap adoption template missing: %v", err)
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

func TestPreviewIsIdempotentAndReportsMergeRequired(t *testing.T) {
	root := t.TempDir()
	if _, err := Initialize(root); err != nil {
		t.Fatal(err)
	}
	outcomes, err := Preview(root)
	if err != nil {
		t.Fatal(err)
	}
	foundGovernance := false
	for _, outcome := range outcomes {
		if outcome.Path == "governance.yml" {
			foundGovernance = true
			if outcome.Action != "merge-required" {
				t.Fatalf("governance preview = %#v", outcome)
			}
		}
	}
	if !foundGovernance {
		t.Fatal("Preview() did not report governance.yml")
	}
	if _, err := Initialize(root); err == nil {
		t.Fatal("Initialize overwrote existing repository-owned files")
	}
}

func TestPreviewDoesNotWriteUserOwnedFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "governance.yml")
	if err := os.WriteFile(path, []byte("user-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	outcomes, err := Preview(root)
	if err != nil {
		t.Fatal(err)
	}
	foundGovernance := false
	for _, outcome := range outcomes {
		if outcome.Path == "governance.yml" {
			foundGovernance = true
			if outcome.Action != "merge-required" {
				t.Fatalf("governance preview = %#v", outcome)
			}
		}
	}
	if !foundGovernance {
		t.Fatal("Preview() did not report governance.yml")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "user-owned" {
		t.Fatalf("Preview changed user file: %q, %v", data, err)
	}
}

func TestPreviewReportsDecisionFileConflict(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "decisions"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	outcomes, err := Preview(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range outcomes {
		if outcome.Path == "docs/decisions/" && outcome.Action == "conflict" {
			return
		}
	}
	t.Fatal("Preview() did not report docs/decisions file conflict")
}

func TestPreviewRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "docs")); err != nil {
		t.Fatal(err)
	}
	if _, err := Preview(root); err == nil {
		t.Fatal("Preview() accepted a symlinked initialization path")
	}
}
