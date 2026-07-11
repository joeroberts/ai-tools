package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() returned %d, want 0", code)
	}
	if got := stdout.String(); got == "" {
		t.Fatal("Run() wrote no help output")
	}
}

func TestRunInitAndConfigCheck(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"init", "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("init returned %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "governance.yml")); err != nil {
		t.Fatalf("governance.yml missing: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"config", "check", "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("config check returned %d: %s", code, stderr.String())
	}
}

func TestRunValidateHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"validate-work-item", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate-work-item --help returned %d", code)
	}
}

func TestRunRoadmapStatus(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	path := filepath.Join("..", "..", "docs", "roadmaps", "go-cli-migration.yaml")

	if code := Run([]string{"roadmap", "status", "--roadmap", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("roadmap status returned %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Adoption And Synchronization")) {
		t.Fatalf("roadmap status output = %q", stdout.String())
	}
}

func TestRunSyncDryRun(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init returned %d", code)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	manifest := filepath.Join("..", "..", "testdata", "releases", "1.0.0.json")

	if code := Run([]string{"sync", "--dry-run", "--manifest", manifest, "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("sync dry-run returned %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Target release: 1.0.0")) {
		t.Fatalf("sync output = %q", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() returned %d, want 2", code)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("Run() wrote no error output")
	}
}
