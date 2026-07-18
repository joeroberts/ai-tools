package implementation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/jira"
	"codex-governance/internal/signature"
	"codex-governance/internal/workitem"
)

func TestPreflightWritesPrivateBundleAndRun(t *testing.T) {
	root, base, head := testRepository(t)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\nStay in scope.\n")
	exportPath, publicKey := signedFixtureExport(t, "export-issuer")
	writePreflightConfig(t, root, publicKey, "export-issuer", "8760h")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	itemPath := filepath.Join(root, "work-item.json")
	writeJSON(t, itemPath, item)
	result, err := Preflight(PreflightRequest{
		WorkItemPath: itemPath, OfflineExportPath: exportPath, RepoRoot: root,
		RuntimeRoot: filepath.Join(root, "runtime"), Adapter: "fake", BundlePath: filepath.Join(root, "runtime", "bundle.json"), RunPath: filepath.Join(root, "runtime", "run.json"),
	})
	if err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if result.Run.State != StatePreflight || !strings.HasPrefix(result.Run.TaskBundleDigest, "sha256:") {
		t.Fatalf("unexpected run: %#v", result.Run)
	}
	if result.Run.SourceEvidence.IssuerKeyID != "fixture-issuer" || result.Run.SourceEvidence.AppliedMaxAge != "8760h0m0s" || !strings.HasPrefix(result.Run.SourceEvidence.EnvelopeDigest, "sha256:") {
		t.Fatalf("unexpected source evidence: %#v", result.Run.SourceEvidence)
	}
	bundle, err := LoadTaskBundle(result.BundlePath)
	if err != nil || bundle.SourceEvidence != result.Run.SourceEvidence {
		t.Fatalf("bundle source evidence = %#v, %v", bundle.SourceEvidence, err)
	}
	for _, path := range []string{result.BundlePath, filepath.Join(root, "runtime", "run.json")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("private artifact %s = %v, %v", path, info, err)
		}
	}
}

func TestPreflightRejectsSourceMismatchBeforeArtifacts(t *testing.T) {
	root, base, head := testRepository(t)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
	exportPath, publicKey := signedFixtureExport(t, "export-issuer")
	writePreflightConfig(t, root, publicKey, "export-issuer", "8760h")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	item.Source.StoryKey = "CG-99"
	itemPath := filepath.Join(root, "mismatched-work-item.json")
	writeJSON(t, itemPath, item)
	bundlePath := filepath.Join(root, "runtime", "bundle.json")
	runPath := filepath.Join(root, "runtime", "run.json")
	if _, err := Preflight(PreflightRequest{WorkItemPath: itemPath, OfflineExportPath: exportPath, RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime"), Adapter: "fake", BundlePath: bundlePath, RunPath: runPath}); err == nil {
		t.Fatal("Preflight() accepted mismatched source evidence")
	}
	for _, path := range []string{bundlePath, runPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Preflight() wrote artifact %s after source mismatch: %v", path, err)
		}
	}
}

func TestPreflightRequiresPrimarySubtaskInProgressBeforeArtifacts(t *testing.T) {
	for _, status := range []string{"To Do", "Blocked", "Done", "in progress", "In progress"} {
		t.Run(status, func(t *testing.T) {
			root, base, head := testRepository(t)
			writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
			exportPath, publicKey := signedFixtureExportWithSubtaskStatus(t, status)
			writePreflightConfig(t, root, publicKey, "export-issuer", "8760h")
			item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
			if err != nil {
				t.Fatal(err)
			}
			item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
			itemPath := filepath.Join(root, "work-item.json")
			writeJSON(t, itemPath, item)
			runtimeRoot := filepath.Join(root, "runtime")
			bundlePath := filepath.Join(runtimeRoot, "bundle.json")
			runPath := filepath.Join(runtimeRoot, "run.json")

			if _, err := Preflight(PreflightRequest{WorkItemPath: itemPath, OfflineExportPath: exportPath, RepoRoot: root, RuntimeRoot: runtimeRoot, Adapter: "fake", BundlePath: bundlePath, RunPath: runPath}); err == nil {
				t.Fatalf("Preflight() accepted primary subtask status %q", status)
			}
			for _, path := range []string{bundlePath, runPath, filepath.Join(runtimeRoot, "worktrees")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("Preflight() created implementation artifact %s for status %q: %v", path, status, err)
				}
			}
		})
	}
}

func TestPreflightRejectsUnsignedOrUntrustedSourceBeforeArtifacts(t *testing.T) {
	root, base, head := testRepository(t)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
	exportPath, publicKey := signedFixtureExport(t, "export-issuer")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	itemPath := filepath.Join(root, "work-item.json")
	writeJSON(t, itemPath, item)

	for _, test := range []struct {
		name       string
		exportPath string
		role       string
		maxAge     string
	}{
		{name: "unsigned", exportPath: filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"), role: "export-issuer", maxAge: "8760h"},
		{name: "wrong-role", exportPath: exportPath, role: "technical-owner", maxAge: "8760h"},
		{name: "expired", exportPath: exportPath, role: "export-issuer", maxAge: "1ns"},
	} {
		t.Run(test.name, func(t *testing.T) {
			writePreflightConfig(t, root, publicKey, test.role, test.maxAge)
			bundlePath := filepath.Join(root, "runtime", test.name, "bundle.json")
			runPath := filepath.Join(root, "runtime", test.name, "run.json")
			_, err := Preflight(PreflightRequest{WorkItemPath: itemPath, OfflineExportPath: test.exportPath, RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime"), Adapter: "fake", BundlePath: bundlePath, RunPath: runPath})
			if err == nil {
				t.Fatal("Preflight() accepted invalid source evidence")
			}
			for _, path := range []string{bundlePath, runPath} {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("Preflight() wrote artifact %s after source failure: %v", path, statErr)
				}
			}
		})
	}
}

func TestVerifyDispatchReadinessRechecksPolicyAndBundle(t *testing.T) {
	root, base, head := testRepository(t)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
	exportPath, publicKey := signedFixtureExport(t, "export-issuer")
	writePreflightConfig(t, root, publicKey, "export-issuer", "8760h")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, head
	itemPath := filepath.Join(root, "work-item.json")
	writeJSON(t, itemPath, item)
	bundlePath := filepath.Join(root, "runtime", "bundle.json")
	runPath := filepath.Join(root, "runtime", "run.json")
	result, err := Preflight(PreflightRequest{WorkItemPath: itemPath, OfflineExportPath: exportPath, RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime"), Adapter: "fake", BundlePath: bundlePath, RunPath: runPath})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadTaskBundle(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDispatchReadiness(result.Run, bundle, bundlePath, cfg, time.Now().UTC()); err != nil {
		t.Fatalf("VerifyDispatchReadiness() error = %v", err)
	}

	writePreflightConfig(t, root, publicKey, "technical-owner", "8760h")
	cfg, err = config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDispatchReadiness(result.Run, bundle, bundlePath, cfg, time.Now().UTC()); err == nil {
		t.Fatal("VerifyDispatchReadiness() accepted a revoked export issuer")
	}

	writePreflightConfig(t, root, publicKey, "export-issuer", "1ns")
	cfg, err = config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDispatchReadiness(result.Run, bundle, bundlePath, cfg, time.Now().UTC()); err == nil {
		t.Fatal("VerifyDispatchReadiness() accepted an expired export")
	}

	writePreflightConfig(t, root, publicKey, "export-issuer", "8760h")
	cfg, err = config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	bundle.Guidance = "tampered"
	writeJSON(t, bundlePath, bundle)
	if err := VerifyDispatchReadiness(result.Run, bundle, bundlePath, cfg, time.Now().UTC()); err == nil {
		t.Fatal("VerifyDispatchReadiness() accepted a tampered task bundle")
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

func TestWaitAndReconcileTerminalOutcomes(t *testing.T) {
	for _, test := range []struct {
		name      string
		script    string
		wantState string
	}{
		{name: "completed", script: "#!/bin/sh\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output-last-message\" ]; then shift; result=$1; break; fi\n  shift\ndone\nprintf '%s' '{\"status\":\"complete\"}' > \"$result\"\n", wantState: StateImplementationComplete},
		{name: "failed", script: "#!/bin/sh\necho adapter failed >&2\nexit 1\n", wantState: StateEscalated},
	} {
		t.Run(test.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "codex")
			if err := os.WriteFile(binary, []byte(test.script), 0o700); err != nil {
				t.Fatal(err)
			}
			run := Run{State: StateQueued}
			adapter := NewHeadlessCodexAdapter(binary, t.TempDir(), t.TempDir())
			if err := Launch(&run, TaskBundle{}, adapter); err != nil {
				t.Fatal(err)
			}
			if err := WaitAndReconcile(&run, adapter, run.ResultRef); err != nil {
				t.Fatalf("WaitAndReconcile() error = %v", err)
			}
			if run.State != test.wantState || run.Attempts != 1 {
				t.Fatalf("run = %#v", run)
			}
			if test.wantState == StateImplementationComplete {
				if data, err := os.ReadFile(run.ResultRef); err != nil || string(data) != `{"status":"complete"}` {
					t.Fatalf("result = %q, %v", data, err)
				}
			} else if refs := adapter.DiagnosticReferences(run.TaskID); len(refs) != 2 {
				t.Fatalf("diagnostic references = %#v", refs)
			}
		})
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
	bundle, err := BuildTaskBundle(item, jira.OfflineExport{}, signature.Envelope{}, jira.OfflineExportEvidence{}, root)
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

func signedFixtureExport(t *testing.T, role string) (string, string) {
	return signedFixtureExportWithSubtaskStatusAndRole(t, "In Progress", role)
}

func signedFixtureExportWithSubtaskStatus(t *testing.T, status string) (string, string) {
	return signedFixtureExportWithSubtaskStatusAndRole(t, status, "export-issuer")
}

func signedFixtureExportWithSubtaskStatusAndRole(t *testing.T, status, role string) (string, string) {
	t.Helper()
	export, err := jira.LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	export.Subtask.Status = status
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(export)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	envelope, err := signature.Sign(payload, "fixture-issuer", role, privateKey, now, ptrTime(now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signed-export.json")
	writeJSON(t, path, envelope)
	return path, base64.StdEncoding.EncodeToString(publicKey)
}

func writePreflightConfig(t *testing.T, root, publicKey, role, maxAge string) {
	t.Helper()
	content := fmt.Sprintf("format_version: 1\nprofile: generic\njira:\n  issue_key_pattern: '^[A-Z]+-[0-9]+$'\n  required_sections: [Scope]\nreview_budget:\n  max_changed_files: 1\n  max_changed_lines: 1\n  max_components: 1\nci:\n  provider: github-actions\n  mode: warn\nupstream: {}\nimplementation:\n  allowed_adapters: [fake]\n  local_code_edit_enabled: false\nsigning:\n  format_version: 1\n  offline_export_max_age: %s\n  trusted_keys:\n    - key_id: fixture-issuer\n      role: %s\n      algorithm: ed25519\n      public_key: %s\n", maxAge, role, publicKey)
	writeFile(t, filepath.Join(root, "governance.yml"), content)
}

func ptrTime(value time.Time) *time.Time { return &value }

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
