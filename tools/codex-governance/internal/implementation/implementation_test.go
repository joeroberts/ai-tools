package implementation

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codex-governance/internal/jira"
	"codex-governance/internal/workitem"
)

func TestPreflightWritesPrivateBundleAndRun(t *testing.T) {
	root, base, head := testRepository(t)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\nStay in scope.\n")
	writeFile(t, filepath.Join(root, "governance.yml"), "format_version: 1\nprofile: generic\njira:\n  issue_key_pattern: '^[A-Z]+-[0-9]+$'\n  required_sections: [Scope]\nreview_budget:\n  max_changed_files: 1\n  max_changed_lines: 1\n  max_components: 1\nci:\n  provider: github-actions\n  mode: warn\nupstream: {}\nimplementation:\n  allowed_adapters: [fake]\n  local_code_edit_enabled: false\n")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	itemPath := filepath.Join(root, "work-item.json")
	writeJSON(t, itemPath, item)
	result, err := Preflight(PreflightRequest{
		WorkItemPath: itemPath, OfflineExportPath: filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"), RepoRoot: root,
		RuntimeRoot: filepath.Join(root, "runtime"), Adapter: "fake", BundlePath: filepath.Join(root, "runtime", "bundle.json"), RunPath: filepath.Join(root, "runtime", "run.json"),
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if result.Run.State != StatePreflight || !strings.HasPrefix(result.Run.TaskBundleDigest, "sha256:") {
		t.Fatalf("unexpected run: %#v", result.Run)
	}
	for _, path := range []string{result.BundlePath, filepath.Join(root, "runtime", "run.json")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private artifact %s = %v, %v", path, info, err)
		}
	}
}

func TestTransitionRejectsSkippedState(t *testing.T) {
	run := Run{State: StatePreflight}
	if err := run.Transition(StateRunning); err == nil {
		t.Fatal("skipped transition was accepted")
	}
	if err := run.Transition(StateQueued); err != nil {
		t.Fatalf("expected queued transition: %v", err)
	}
}

func TestFakeAdapterReconcilesWithoutRedispatch(t *testing.T) {
	run := Run{State: StateQueued}
	adapter := NewFakeAdapter()
	if err := Launch(&run, TaskBundle{}, adapter); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(&run, adapter, filepath.Join(t.TempDir(), "result.json")); err != nil || run.State != StateRunning {
		t.Fatalf("running reconciliation = %v, %#v", err, run)
	}
	if err := Launch(&run, TaskBundle{}, adapter); err == nil {
		t.Fatal("duplicate dispatch was accepted")
	}
	if err := adapter.Complete(run.TaskID, []byte(`{"status":"complete"}`)); err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(t.TempDir(), "result.json")
	if err := Reconcile(&run, adapter, resultPath); err != nil || run.State != StateImplementationComplete || run.ResultRef != resultPath {
		t.Fatalf("completion reconciliation = %v, %#v", err, run)
	}
}

func TestUnknownAdapterTaskEscalates(t *testing.T) {
	run := Run{State: StateRunning, TaskID: "missing"}
	if err := Reconcile(&run, NewFakeAdapter(), filepath.Join(t.TempDir(), "result.json")); err != nil {
		t.Fatal(err)
	}
	if run.State != StateEscalated {
		t.Fatalf("unknown task state = %q", run.State)
	}
}

func TestCodexThreadID(t *testing.T) {
	id, ok := codexThreadID([]byte(`{"type":"thread.started","thread_id":"019f5c26-6390-70b2-b2ba-3b747156c0dd"}`))
	if !ok || id != "019f5c26-6390-70b2-b2ba-3b747156c0dd" {
		t.Fatalf("codexThreadID() = %q, %v", id, ok)
	}
	if _, ok := codexThreadID([]byte(`{"type":"turn.started"}`)); ok {
		t.Fatal("non-thread event was accepted")
	}
}

func TestHeadlessPromptConstrainsRemoteActions(t *testing.T) {
	prompt := headlessPrompt(TaskBundle{AllowedPaths: []string{"internal"}})
	for _, denied := range []string{"Do not push", "create a pull request", "access secrets", "modify remote state"} {
		if !strings.Contains(prompt, denied) {
			t.Fatalf("prompt does not contain %q", denied)
		}
	}
}

func TestBuildTaskBundleCopiesApprovedInputs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
	item := workitem.Item{Scope: workitem.Scope{AllowedPaths: []string{"internal"}, ValidationPlan: []string{"go test ./..."}}, Decision: workitem.Decision{ADR: "No ADR needed: test"}}
	bundle, err := BuildTaskBundle(item, jira.OfflineExport{}, root)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Guidance != "# Guidance\n" || bundle.AllowedPaths[0] != "internal" || bundle.Commands[0] != "go test ./..." {
		t.Fatalf("bundle did not preserve approved inputs: %#v", bundle)
	}
}

func testRepository(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "user.email", "test@example.test")
	writeFile(t, filepath.Join(root, "internal", "value.go"), "package internal\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(root, "internal", "value.go"), "package internal\n\nconst Value = 1\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "head")
	return root, base, strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}
