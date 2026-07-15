package agentplan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-governance/internal/ticketplan"
)

func TestWriteDecompositionReturnsMarshalError(t *testing.T) {
	original := marshalDecomposition
	marshalDecomposition = func(any, string, string) ([]byte, error) { return nil, errors.New("marshal failed") }
	t.Cleanup(func() { marshalDecomposition = original })
	if err := writeDecomposition(filepath.Join(t.TempDir(), "plan.json"), ticketplan.Plan{}); err == nil || !strings.Contains(err.Error(), "serialize manager decomposition") {
		t.Fatalf("writeDecomposition() error = %v", err)
	}
}

type fakeRunner struct {
	roles []string
}

func (r *fakeRunner) Run(_ context.Context, role, _, _ string) ([]byte, error) {
	r.roles = append(r.roles, role)
	if role == "manager" {
		return []byte(`{
  "format_version": 1,
  "status": "draft",
  "sources": {},
	  "story": {"summary": "Discovery", "description": "Validate the design", "acceptance_criteria": ["Reviewed"], "traceability":{"summary":[{"source":"prd","section":"Goal","excerpt":"Discovery source evidence defines the goal."}],"description":[{"source":"prd","section":"Goal","excerpt":"Validate design source evidence defines the work."}],"acceptance_criteria":[{"source":"prd","section":"Acceptance","excerpt":"Reviewed source evidence confirms acceptance."}]}},
	  "subtasks": [{"id": "P0-1", "summary": "Spike", "phase":"Phase 1", "change_class":"trivial", "review_budget":{"max_changed_files":2,"max_changed_lines":100,"components":["docs"]}, "scope": "Document a bounded spike", "non_goals": ["No production code"], "acceptance_criteria": ["Decision recorded"], "validation_plan": ["Review output"], "allowed_paths": ["docs/spike.md"], "adr": "No ADR needed: spike follows the current plan", "dependencies": [], "traceability":{"summary":[{"source":"prd","section":"Goal","excerpt":"Spike source evidence identifies the task."}],"phase":[{"source":"roadmap","section":"Phase 1","excerpt":"Phase 1 source evidence identifies the stage."}],"change_class":[{"source":"spec","section":"Scope","excerpt":"Trivial source evidence classifies the task."}],"review_budget":[{"source":"spec","section":"Scope","excerpt":"Review budget limits docs change to 2 files and 100 lines."}],"scope":[{"source":"spec","section":"Scope","excerpt":"Document bounded spike source evidence limits scope."}],"non_goals":[{"source":"spec","section":"Scope","excerpt":"No production code source evidence is excluded."}],"acceptance_criteria":[{"source":"prd","section":"Acceptance","excerpt":"Decision recorded source evidence confirms completion."}],"validation_plan":[{"source":"spec","section":"Scope","excerpt":"Review output source evidence validates the change."}],"allowed_paths":[{"source":"spec","section":"Scope","excerpt":"The allowed path is docs/spike.md."}],"adr":[{"source":"spec","section":"Decision","excerpt":"Spike follows current plan source evidence supports the rationale."}],"dependencies":[{"source":"roadmap","section":"Phase 1","excerpt":"Phase 1 source evidence identifies the stage."}]}}]
	}`), nil
	}
	return []byte(`{"status":"approved","summary":"ready"}`), nil
}

func TestGenerateAfterPhase2ApprovalRejectsUnownedRunners(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"prd.md":     "# Goal\nDiscovery source evidence defines the goal.\nValidate design source evidence defines the work.\nSpike source evidence identifies the task.\n# Acceptance\nReviewed source evidence confirms acceptance.\nDecision recorded source evidence confirms completion.\n",
		"spec.md":    "# Scope\nTrivial source evidence classifies the task.\nReview budget limits docs change to 2 files and 100 lines.\nDocument bounded spike source evidence limits scope.\nNo production code source evidence is excluded.\nReview output source evidence validates the change.\nThe allowed path is docs/spike.md.\n# Decision\nSpike follows current plan source evidence supports the rationale.\n",
		"roadmap.md": "# Phase 1\nPhase 1 source evidence identifies the stage.\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runner := &fakeRunner{}
	output := filepath.Join(root, "plans", "generated.json")
	_, err := generateAfterPhase2Approval(Request{
		PRDPath: filepath.Join(root, "prd.md"), SpecPath: filepath.Join(root, "spec.md"),
		RoadmapPath: filepath.Join(root, "roadmap.md"), OutputPath: output,
		RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime"),
	}, Runners{Manager: runner, Reviewer: runner, Verifier: runner})
	if err == nil || !strings.Contains(err.Error(), "manager runner must be a hosted CodexRunner") {
		t.Fatalf("generateAfterPhase2Approval() error = %v", err)
	}
	if len(runner.roles) != 0 {
		t.Fatalf("generateAfterPhase2Approval dispatched unowned runner: %#v", runner.roles)
	}
}

func TestGenerateAfterPhase2ApprovalRejectsSharedLocalWorker(t *testing.T) {
	worker := &OllamaRunner{}
	_, err := generateAfterPhase2Approval(Request{}, Runners{
		Manager: CodexRunner{}, Reviewer: worker, Verifier: worker,
	})
	if err == nil || !strings.Contains(err.Error(), "must be independent instances") {
		t.Fatalf("generateAfterPhase2Approval() error = %v", err)
	}
}

func TestCodexRunnerRestrictsExecutionToManagerRole(t *testing.T) {
	runner := CodexRunner{}
	if _, err := runner.Run(context.Background(), "reviewer", "prompt", "schema"); err == nil || !strings.Contains(err.Error(), "restricted to the manager role") {
		t.Fatalf("CodexRunner.Run(reviewer) error = %v", err)
	}
}

func TestCodexRunnerPermitsSyntheticNonGitWorkdir(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "fake-codex")
	script := `#!/bin/sh
output=""
found_skip=false
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--skip-git-repo-check" ]; then found_skip=true; fi
  if [ "$1" = "--output-last-message" ]; then shift; output="$1"; fi
  shift
done
[ "$found_skip" = true ] || exit 9
printf '%s' '{"format_version":1}' > "$output"
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := (CodexRunner{Binary: binary, WorkDir: dir}).Run(context.Background(), "manager", "prompt", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != `{"format_version":1}` {
		t.Fatalf("CodexRunner.Run() output = %q", output)
	}
}

func TestDecomposeWritesOwnerOnlyArtifact(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# Scope\nApproved source.\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(root, "fake-codex")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then shift; printf '%s' '{"format_version":1}' > "$1"; exit 0; fi
  shift
done
exit 1
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "private", "decomposition.json")
	if _, err := Decompose(Request{PRDPath: filepath.Join(root, "prd.md"), SpecPath: filepath.Join(root, "spec.md"), RoadmapPath: filepath.Join(root, "roadmap.md"), OutputPath: output, RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime")}, CodexRunner{Binary: binary, WorkDir: root}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("decomposition permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestOllamaRunnerRestrictsExecutionToReviewRoles(t *testing.T) {
	runner := OllamaRunner{}
	if _, err := runner.Run(context.Background(), "manager", "prompt", "schema"); err == nil || !strings.Contains(err.Error(), "restricted to reviewer and verifier roles") {
		t.Fatalf("OllamaRunner.Run(manager) error = %v", err)
	}
}

func TestReviewPromptRequiresStructuredResult(t *testing.T) {
	prompt := reviewPrompt("reviewer", []byte(`{"format_version":1}`))
	if !strings.Contains(prompt, `Return only JSON matching`) || !strings.Contains(prompt, `"status":"approved|changes_requested|blocked"`) {
		t.Fatalf("reviewPrompt() = %q", prompt)
	}
}

func TestParseReviewResultAcceptsFencedJSON(t *testing.T) {
	result, err := parseReviewResult([]byte("```json\n{\"status\":\"approved\",\"summary\":\"ready\"}\n```"))
	if err != nil || result.Status != "approved" || result.Summary != "ready" {
		t.Fatalf("parseReviewResult() = %#v, %v", result, err)
	}
}

func TestSaveValidationFindingsUsesPrivateArtifact(t *testing.T) {
	root := t.TempDir()
	if err := saveValidationFindings(root, "ticket-plan:1234", 1, []string{"invalid dependency"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "ticket-plan-runs", "ticket-plan-1234", "manager-1-validation.json")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "invalid dependency") {
		t.Fatalf("validation artifact = %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("validation artifact permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestSaveEscalationUsesPrivateRedactedArtifact(t *testing.T) {
	root := t.TempDir()
	if err := saveEscalation(root, "ticket-plan:1234", 2, "review did not converge", []string{"token=secret-value"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "ticket-plan-runs", "ticket-plan-1234", "stakeholder-escalation.json")
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "secret-value") || !strings.Contains(string(data), "[REDACTED]") {
		t.Fatalf("escalation artifact = %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("escalation artifact permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestBuildSourceCatalogUsesVerifiedNamedSections(t *testing.T) {
	root := t.TempDir()
	sources := ticketplan.Sources{}
	for name, content := range map[string]string{
		"prd.md":     "# Goal\nDeliver a ticket plan.\n",
		"spec.md":    "# Scope\nUse deterministic validation.\n",
		"roadmap.md": "# Phase 1\nPlan contract.\n",
	} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		digest, err := ticketplan.FileDigest(path)
		if err != nil {
			t.Fatal(err)
		}
		source := ticketplan.Source{Path: name, Digest: digest}
		switch name {
		case "prd.md":
			sources.PRD = source
		case "spec.md":
			sources.Spec = source
		default:
			sources.Roadmap = source
		}
	}
	catalog, err := buildSourceCatalog(root, sources)
	if err != nil || !strings.Contains(catalog, "SOURCE prd (prd.md)") || !strings.Contains(catalog, "SECTION Scope") {
		t.Fatalf("buildSourceCatalog() = %q, %v", catalog, err)
	}
}

func TestGenerateRejectsHostedReviewer(t *testing.T) {
	_, err := Generate(Request{}, Runners{
		Manager:  CodexRunner{},
		Reviewer: CodexRunner{},
		Verifier: OllamaRunner{},
	})
	if err == nil || !strings.Contains(err.Error(), "reviewer runner must be a local OllamaRunner") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateRejectsHostedVerifier(t *testing.T) {
	_, err := Generate(Request{}, Runners{
		Manager:  CodexRunner{},
		Reviewer: OllamaRunner{},
		Verifier: CodexRunner{},
	})
	if err == nil || !strings.Contains(err.Error(), "verifier runner must be a local OllamaRunner") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateLoadsApprovedSourcesBeforeDispatch(t *testing.T) {
	_, err := Generate(Request{}, Runners{
		Manager:  CodexRunner{},
		Reviewer: OllamaRunner{},
		Verifier: OllamaRunner{},
	})
	if err == nil || !strings.Contains(err.Error(), "reviewer worker policy: Ollama endpoint must be local HTTP") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestLoadSourcesRejectsExternalTargetThroughInternalSymlink(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "prd.md")
	if err := os.WriteFile(external, []byte("external source"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "prd.md")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	_, err := loadSources(Request{PRDPath: link, SpecPath: link, RoadmapPath: link, RepoRoot: root})
	if err == nil || !strings.Contains(err.Error(), "approved source must be inside repository root") {
		t.Fatalf("loadSources() error = %v", err)
	}
}
