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

func TestRunJiraPlanGenerateDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--output", "plan.json", "--dry-run"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("generate dry run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunJiraPlanGenerateVerboseDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--output", "plan.json", "--dry-run", "--verbose"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("verbose generate dry run = %d, stderr=%q", code, stderr.String())
	}
}

func TestRunJiraPlanCommandsRespectRemainingPhaseBoundaries(t *testing.T) {
	for _, command := range [][]string{
		{"jira", "plan", "approve", "--plan", "plan.json"},
		{"jira", "plan", "create", "--plan", "plan.json"},
	} {
		var stderr bytes.Buffer
		if code := Run(command, &bytes.Buffer{}, &stderr); code != 1 {
			t.Fatalf("Run(%v) = %d, stderr=%q", command, code, stderr.String())
		}
		if !bytes.Contains(stderr.Bytes(), []byte("unavailable until Phase")) {
			t.Fatalf("Run(%v) stderr=%q", command, stderr.String())
		}
	}
}

func TestRunJiraPlanValidateChecksCurrentSources(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "ticket-plans", "valid.json")
	var stdout bytes.Buffer
	if code := Run([]string{"jira", "plan", "validate", "--plan", plan, "--repo-root", t.TempDir()}, &stdout, &bytes.Buffer{}); code != 1 {
		t.Fatalf("validate = %d, stdout=%q", code, stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("prd source is unavailable")) {
		t.Fatalf("validate stdout=%q", stdout.String())
	}
}

func TestRunJiraPlanValidateValidFixture(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "plan.json")
	root := filepath.Join("..", "..")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"jira", "plan", "validate", "--plan", plan, "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("PASS ticket plan is valid")) {
		t.Fatalf("validate stdout=%q", stdout.String())
	}
}
