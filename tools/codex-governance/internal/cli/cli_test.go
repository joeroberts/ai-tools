package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/ticketplan"
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
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--constraints", "constraints.json", "--output", "plan.json", "--dry-run"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("generate dry run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunJiraPlanGenerateVerboseDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--constraints", "constraints.json", "--output", "plan.json", "--dry-run", "--verbose"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("verbose generate dry run = %d, stderr=%q", code, stderr.String())
	}
}

func TestWriteJiraPublicationRecordIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if err := writeJiraPublicationRecord(path, jiraPublicationRecord{PlanDigest: "sha256:abc", Status: "creating"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("publication record permissions = %v, %v", info.Mode().Perm(), err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(`"status": "creating"`)) {
		t.Fatalf("publication record = %q, %v", data, err)
	}
}

func TestRunJiraPlanCreateDryRunUsesApprovedWorkflowWithoutWritingRecord(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init = %d", code)
	}
	for _, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "valid", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "testdata", "ticket-plans", "valid", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan ticketplan.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Status = "approved"
	planPath := filepath.Join(root, "plan.json")
	data, err = json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := ticketplan.FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := ticketplan.NewWorkflowState(root, planPath, digest, "approved", plan.Sources)
	if err != nil {
		t.Fatal(err)
	}
	workflow.ApprovedBy, workflow.ApprovedAt = "stakeholder@example.test", time.Now().UTC()
	workflowPath := filepath.Join(root, "workflow.json")
	if err := ticketplan.SaveWorkflow(workflowPath, workflow); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "governance.yml")
	if err := os.WriteFile(configPath, []byte(strings.Replace(string(mustReadFile(t, configPath)), "project: \"\"", "project: \"CG\"", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(root, "result.json")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"jira", "plan", "create", "--plan", planPath, "--workflow", workflowPath, "--repo-root", root, "--result", resultPath, "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("create dry run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would create Story")) {
		t.Fatalf("create dry-run output = %q", stdout.String())
	}
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote a publication record: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
