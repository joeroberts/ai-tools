package agentplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/ollama"
	gruntime "codex-governance/internal/runtime"
	"codex-governance/internal/ticketplan"
)

func TestPlanSchemasUseApprovedEnumsAndBoundedArrays(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "schema-fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		AcceptedPaths               []string `json:"accepted_paths"`
		RejectedPreAssignmentPaths  []string `json:"rejected_pre_assignment_paths"`
		RejectedPostAssignmentPaths []string `json:"rejected_post_assignment_paths"`
		AcceptedStoryAcceptance     []string `json:"accepted_story_acceptance_criteria"`
		RejectedStoryAcceptance     []string `json:"rejected_story_acceptance_criteria"`
		AcceptedNonGoals            []string `json:"accepted_non_goals"`
		RejectedNonGoals            []string `json:"rejected_non_goals"`
		AcceptedAcceptance          []string `json:"accepted_acceptance_criteria"`
		RejectedAcceptance          []string `json:"rejected_acceptance_criteria"`
		AcceptedValidation          []string `json:"accepted_validation_plan"`
		RejectedValidation          []string `json:"rejected_validation_plan"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	constraints := Constraints{
		PathPool:     fixtures.AcceptedPaths,
		ReviewBudget: ticketplan.ReviewBudget{MaxChangedFiles: 12, MaxChangedLines: 900, Components: []string{"schema", "lifecycle"}},
		Story: &StoryConstraints{AcceptanceCriteria: fixtures.AcceptedStoryAcceptance, Traceability: ticketplan.TraceMap{
			"acceptance_criteria": {{Source: "prd"}, {Source: "spec"}},
		}},
		Subtasks: []SubtaskConstraints{
			{ID: "schema", Phase: "Phase 1", ChangeClass: "standard", AllowedPaths: []string{"AGENTS.md", "internal/agentplan"}, ReviewBudget: ticketplan.ReviewBudget{MaxChangedFiles: 6, MaxChangedLines: 450, Components: []string{"schema"}}, Dependencies: []string{}, SourceDerived: &SourceDerivedConstraints{NonGoals: fixtures.AcceptedNonGoals[:1], AcceptanceCriteria: fixtures.AcceptedAcceptance[:2], ValidationPlan: fixtures.AcceptedValidation}},
			{ID: "lifecycle", Phase: "Phase 2", ChangeClass: "standard", AllowedPaths: []string{"internal/agentplan"}, ReviewBudget: ticketplan.ReviewBudget{MaxChangedFiles: 5, MaxChangedLines: 400, Components: []string{"lifecycle"}}, Dependencies: []string{"schema"}, SourceDerived: &SourceDerivedConstraints{NonGoals: fixtures.AcceptedNonGoals[1:], AcceptanceCriteria: fixtures.AcceptedAcceptance[2:], ValidationPlan: fixtures.AcceptedValidation}},
		},
	}

	postSchema := planSchema(constraints)
	post := schemaProperties(t, postSchema)
	story := schemaStoryProperties(t, postSchema)
	for _, value := range fixtures.AcceptedStoryAcceptance {
		if !schemaAllowsArray(story["acceptance_criteria"].(map[string]any), []string{value, value}) {
			t.Errorf("approved story acceptance criterion rejected: %q", value)
		}
	}
	for _, value := range fixtures.RejectedStoryAcceptance {
		if schemaAllowsArray(story["acceptance_criteria"].(map[string]any), []string{value, value}) {
			t.Errorf("unapproved story acceptance criterion accepted: %q", value)
		}
	}
	if schemaAllowsArray(story["acceptance_criteria"].(map[string]any), append(fixtures.AcceptedStoryAcceptance, fixtures.AcceptedStoryAcceptance[0])) {
		t.Fatal("story acceptance criteria accepted an array over its approved bound")
	}
	if !schemaAllowsString(post["id"].(map[string]any), "schema") || schemaAllowsString(post["id"].(map[string]any), "outside") {
		t.Fatal("Subtask ID enum does not match the assigned ID set")
	}
	if !schemaAllowsString(post["phase"].(map[string]any), "Phase 2") || schemaAllowsString(post["phase"].(map[string]any), "Phase 3") {
		t.Fatal("phase enum does not match approved phases")
	}
	oversizedProtocol := strings.Repeat("tool_event:", 30) + "https://example.invalid/result"
	for _, path := range fixtures.AcceptedPaths {
		if !schemaAllowsArray(post["allowed_paths"].(map[string]any), []string{path}) {
			t.Errorf("approved path rejected: %q", path)
		}
	}
	for _, path := range append(fixtures.RejectedPostAssignmentPaths, oversizedProtocol) {
		if schemaAllowsArray(post["allowed_paths"].(map[string]any), []string{path}) {
			t.Errorf("unapproved or oversized path accepted: %q", path)
		}
	}
	if schemaAllowsArray(post["allowed_paths"].(map[string]any), []string{"AGENTS.md", "testdata", "internal/agentplan"}) {
		t.Fatal("allowed_paths accepted an array over the approved per-Subtask bound")
	}
	budget := post["review_budget"].(map[string]any)["properties"].(map[string]any)
	if !schemaAllowsInteger(budget["max_changed_files"].(map[string]any), 5) || schemaAllowsInteger(budget["max_changed_files"].(map[string]any), 7) {
		t.Fatal("review budget enum does not match approved values")
	}
	if schemaAllowsArray(budget["components"].(map[string]any), []string{"schema", "lifecycle"}) {
		t.Fatal("components accepted an array over the approved per-Subtask bound")
	}
	dependencies := post["dependencies"].(map[string]any)
	if !schemaAllowsArray(dependencies, nil) || !schemaAllowsArray(dependencies, []string{"schema"}) || schemaAllowsArray(dependencies, []string{"outside"}) || schemaAllowsArray(dependencies, []string{"schema", "lifecycle"}) {
		t.Fatal("dependency enum or zero/one-item bounds do not match approved constraints")
	}
	for _, test := range []struct {
		name     string
		schema   map[string]any
		accepted []string
		rejected []string
		over     []string
	}{
		{"non-goal", post["non_goals"].(map[string]any), fixtures.AcceptedNonGoals, fixtures.RejectedNonGoals, []string{fixtures.AcceptedNonGoals[0], fixtures.AcceptedNonGoals[1]}},
		{"acceptance criterion", post["acceptance_criteria"].(map[string]any), fixtures.AcceptedAcceptance, fixtures.RejectedAcceptance, append(fixtures.AcceptedAcceptance, fixtures.AcceptedAcceptance[0])},
		{"validation step", post["validation_plan"].(map[string]any), fixtures.AcceptedValidation, fixtures.RejectedValidation, append(fixtures.AcceptedValidation, fixtures.AcceptedValidation[0])},
	} {
		for _, value := range test.accepted {
			if !schemaAllowsArray(test.schema, []string{value}) {
				t.Errorf("approved %s rejected: %q", test.name, value)
			}
		}
		for _, value := range test.rejected {
			if schemaAllowsArray(test.schema, []string{value}) {
				t.Errorf("unapproved %s accepted: %q", test.name, value)
			}
		}
		if schemaAllowsArray(test.schema, test.over) {
			t.Errorf("%s array over approved bound accepted: %#v", test.name, test.over)
		}
	}

	preSchema := planSchemaRange(1, 8)
	pre := schemaProperties(t, preSchema)
	for _, path := range fixtures.AcceptedPaths {
		if !schemaAllowsArray(pre["allowed_paths"].(map[string]any), []string{path}) {
			t.Errorf("bounded decomposition schema rejected valid path %q", path)
		}
	}
	for _, path := range append(fixtures.RejectedPreAssignmentPaths, oversizedProtocol) {
		if schemaAllowsArray(pre["allowed_paths"].(map[string]any), []string{path}) {
			t.Errorf("bounded decomposition schema accepted invalid path %q", path)
		}
	}
	assertManagerArraysBounded(t, preSchema)
}

func schemaProperties(t *testing.T, value string) map[string]any {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal([]byte(value), &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	return schema["properties"].(map[string]any)["subtasks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
}

func schemaStoryProperties(t *testing.T, value string) map[string]any {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal([]byte(value), &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	return schema["properties"].(map[string]any)["story"].(map[string]any)["properties"].(map[string]any)
}

func schemaAllowsString(schema map[string]any, value string) bool {
	if limit, ok := schema["maxLength"].(float64); ok && len(value) > int(limit) {
		return false
	}
	if pattern, ok := schema["pattern"].(string); ok {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil || !matched {
			return false
		}
	}
	values, ok := schema["enum"].([]any)
	if !ok {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func schemaAllowsInteger(schema map[string]any, value int) bool {
	for _, candidate := range schema["enum"].([]any) {
		if candidate == float64(value) {
			return true
		}
	}
	return false
}

func schemaAllowsArray(schema map[string]any, values []string) bool {
	if minimum, ok := schema["minItems"].(float64); ok && len(values) < int(minimum) {
		return false
	}
	if maximum, ok := schema["maxItems"].(float64); ok && len(values) > int(maximum) {
		return false
	}
	items := schema["items"].(map[string]any)
	for _, value := range values {
		if !schemaAllowsString(items, value) {
			return false
		}
	}
	return true
}

func assertManagerArraysBounded(t *testing.T, value string) {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal([]byte(value), &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	subtasks := properties["subtasks"].(map[string]any)
	subtaskProperties := subtasks["items"].(map[string]any)["properties"].(map[string]any)
	arrays := []map[string]any{
		properties["story"].(map[string]any)["properties"].(map[string]any)["acceptance_criteria"].(map[string]any),
		subtasks,
		subtaskProperties["review_budget"].(map[string]any)["properties"].(map[string]any)["components"].(map[string]any),
		subtaskProperties["non_goals"].(map[string]any), subtaskProperties["acceptance_criteria"].(map[string]any),
		subtaskProperties["validation_plan"].(map[string]any), subtaskProperties["allowed_paths"].(map[string]any),
		subtaskProperties["dependencies"].(map[string]any), schema["$defs"].(map[string]any)["references"].(map[string]any),
	}
	for index, array := range arrays {
		if _, ok := array["maxItems"].(float64); !ok {
			t.Errorf("manager output array %d lacks maxItems", index)
		}
	}
}

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
	for index, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		content := fmt.Sprintf("# Scope\nApproved source %d.\n", index+1)
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
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
	if _, err := Decompose(Request{PRDPath: filepath.Join(root, "prd.md"), SpecPath: filepath.Join(root, "spec.md"), RoadmapPath: filepath.Join(root, "roadmap.md"), OutputPath: output, RepoRoot: root, RuntimeRoot: filepath.Join(root, "runtime"), ManagerTimeout: 5 * time.Second, ManagerWaitDelay: time.Second}, CodexRunner{Binary: binary, WorkDir: root}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("decomposition permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestCodexRunnerPersistsPrivateJSONLDiagnostics(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "fake-codex")
	script := `#!/bin/sh
result=""
json=false
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--json" ]; then json=true; fi
  if [ "$1" = "--output-last-message" ]; then shift; result="$1"; fi
  shift
done
[ "$json" = true ] || exit 9
[ ! -e "$result" ] || exit 8
printf '%s\n' '{"type":"thread.started"}'
printf '%s\n' '{"type":"turn.completed","usage":{"input_tokens":1}}'
printf '%s' '{"format_version":1}' > "$result"
printf '%s\n' 'manager stderr' >&2
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeRoot := filepath.Join(root, "runtime")
	result, err := runRole(Request{RuntimeRoot: runtimeRoot, ManagerTimeout: 5 * time.Second, ManagerWaitDelay: time.Second}, "ticket-plan:test", "manager", 1, CodexRunner{Binary: binary, WorkDir: root}, "prompt", `{}`)
	if err != nil || string(result) != `{"format_version":1}` {
		t.Fatalf("runRole() = %q, %v", result, err)
	}
	dir := filepath.Join(runtimeRoot, "ticket-plan-runs", "ticket-plan-test", "manager-1")
	dirInfo, err := os.Stat(dir)
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("diagnostic directory permissions = %v, %v", dirInfo, err)
	}
	for _, name := range []string{"codex.jsonl", "stderr.log", "schema.json", "result.json"} {
		info, statErr := os.Stat(filepath.Join(dir, name))
		if statErr != nil {
			t.Fatalf("diagnostic %s: %v", name, statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("diagnostic %s permissions = %v", name, info.Mode().Perm())
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "codex.jsonl"))
	if err != nil || !strings.Contains(string(data), `"turn.completed"`) || !strings.Contains(string(data), `"input_tokens":1`) {
		t.Fatalf("terminal usage event = %q, %v", data, err)
	}
	assertLedgerStates(t, runtimeRoot, "started", "completed", "closed")
}

func TestCodexRunnerWaitDelayClosesInheritedOutputPipes(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "fake-codex")
	script := `#!/bin/sh
result=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then shift; result="$1"; fi
  shift
done
(sleep 5) &
printf '%s' '{"format_version":1}' > "$result"
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeRoot := filepath.Join(root, "runtime")
	started := time.Now()
	_, err := runRole(Request{RuntimeRoot: runtimeRoot, ManagerTimeout: time.Second, ManagerWaitDelay: 100 * time.Millisecond}, "ticket-plan:test", "manager", 1, CodexRunner{Binary: binary, WorkDir: root}, "prompt", `{}`)
	if err == nil {
		t.Fatal("manager with inherited output pipe unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("WaitDelay did not bound inherited output pipe: %s", elapsed)
	}
	assertLedgerStates(t, runtimeRoot, "started", "failed", "closed")
}

func TestCodexRunnerClosesTimeoutAndCancellationFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		context func() (context.Context, context.CancelFunc)
		timeout time.Duration
	}{
		{name: "timeout", context: func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, timeout: 150 * time.Millisecond},
		{name: "cancellation", context: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel
		}, timeout: time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			binary := filepath.Join(root, "fake-codex")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nresult=\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output-last-message\" ]; then shift; result=\"$1\"; fi\n  shift\ndone\nprintf ready > \"$(dirname \"$result\")/ready\"\nsleep 5\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := test.context()
			defer cancel()
			runtimeRoot := filepath.Join(root, "runtime")
			ready := filepath.Join(runtimeRoot, "ticket-plan-runs", "ticket-plan-test", "manager-1", "ready")
			readyObserved := make(chan bool, 1)
			if test.name == "cancellation" {
				go func() {
					readyObserved <- waitForFile(ready, 2*time.Second)
					cancel()
				}()
			}
			started := time.Now()
			_, err := runRole(Request{RuntimeRoot: runtimeRoot, Context: ctx, ManagerTimeout: test.timeout, ManagerWaitDelay: 100 * time.Millisecond}, "ticket-plan:test", "manager", 1, CodexRunner{Binary: binary, WorkDir: root}, "prompt", `{}`)
			if err == nil {
				t.Fatal("controlled manager cancellation unexpectedly succeeded")
			}
			if test.name == "cancellation" && !<-readyObserved {
				t.Fatal("manager did not publish its bounded ready marker")
			}
			if elapsed := time.Since(started); elapsed > 2*time.Second {
				t.Fatalf("controlled manager cancellation took %s", elapsed)
			}
			assertLedgerStates(t, runtimeRoot, "started", "failed", "closed")
		})
	}
}

func waitForFile(path string, deadline time.Duration) bool {
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		select {
		case <-timer.C:
			return false
		case <-ticker.C:
		}
	}
}

func TestFailureReconciliationUsesPrivateFallbackAndAttemptsClosed(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "blocked-artifact")
	if err := os.WriteFile(artifact, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalRecord := recordExecutionEvent
	defer func() { recordExecutionEvent = originalRecord }()
	var states []string
	recordExecutionEvent = func(_ string, event gruntime.Event) error {
		states = append(states, event.State)
		if event.State == "failed" {
			return errors.New("failed ledger remains unavailable")
		}
		return nil
	}
	runErr := errors.New("manager deadline exceeded")
	_, err := finishRole(root, "ticket-plan:test", "manager-ticket-plan-1", "manager", 1, artifact, nil, runErr)
	if !errors.Is(err, runErr) || !strings.Contains(err.Error(), "failed ledger remains unavailable") {
		t.Fatalf("finishRole() error = %v", err)
	}
	if got := strings.Join(states, ","); got != "failed,closed" {
		t.Fatalf("terminal attempts = %q, want failed,closed", got)
	}
	fallbacks, err := filepath.Glob(filepath.Join(root, "ticket-plan-failures", "*.txt"))
	if err != nil || len(fallbacks) != 1 {
		t.Fatalf("fallback diagnostics = %v, %v", fallbacks, err)
	}
	fallback, err := os.ReadFile(fallbacks[0])
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic := string(fallback); !strings.Contains(diagnostic, runErr.Error()) || !strings.Contains(diagnostic, "create failure diagnostics") {
		t.Fatalf("fallback diagnostic does not preserve both failures: %q", diagnostic)
	}
	info, err := os.Stat(fallbacks[0])
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("fallback mode = %v, %v", info.Mode().Perm(), err)
	}
	dirInfo, err := os.Stat(filepath.Dir(fallbacks[0]))
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("fallback directory mode = %v, %v", dirInfo.Mode().Perm(), err)
	}
}

func TestTerminalReconciliationAttemptsCompletionAndRetriesClose(t *testing.T) {
	root := t.TempDir()
	originalRecord := recordExecutionEvent
	defer func() { recordExecutionEvent = originalRecord }()
	completedErr := errors.New("completed ledger unavailable")
	closeErr := errors.New("close ledger temporarily unavailable")
	var states []string
	closeCalls := 0
	recordExecutionEvent = func(_ string, event gruntime.Event) error {
		states = append(states, event.State)
		if event.State == "completed" {
			return completedErr
		}
		if event.State == "closed" {
			closeCalls++
			if closeCalls == 1 {
				return closeErr
			}
		}
		return nil
	}
	_, err := finishRole(root, "ticket-plan:test", "reviewer-ticket-plan-1", "reviewer", 1, filepath.Join(root, "artifacts"), []byte(`{"status":"approved"}`), nil)
	if !errors.Is(err, completedErr) || !errors.Is(err, closeErr) {
		t.Fatalf("finishRole() error = %v", err)
	}
	if got := strings.Join(states, ","); got != "completed,closed,closed" {
		t.Fatalf("terminal attempts = %q, want completed,closed,closed", got)
	}
}

func TestValidateManagerLifecycleRejectsNonPositiveValues(t *testing.T) {
	for _, request := range []Request{{ManagerWaitDelay: time.Second}, {ManagerTimeout: time.Second}, {ManagerTimeout: -time.Second, ManagerWaitDelay: time.Second}, {ManagerTimeout: time.Second, ManagerWaitDelay: -time.Second}} {
		if err := validateManagerLifecycle(request); err == nil {
			t.Fatalf("validateManagerLifecycle(%+v) unexpectedly passed", request)
		}
	}
}

func assertLedgerStates(t *testing.T, root string, expected ...string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "execution-ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(expected) {
		t.Fatalf("ledger events = %q", data)
	}
	for index, line := range lines {
		var event struct {
			State     string `json:"state"`
			ResultRef string `json:"result_ref"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event.State != expected[index] {
			t.Fatalf("ledger state %d = %q, want %q", index, event.State, expected[index])
		}
		if (event.State == "completed" || event.State == "failed" || event.State == "closed") && event.ResultRef == "" {
			t.Fatalf("ledger state %q lacks diagnostic reference", event.State)
		}
	}
}

func TestOllamaRunnerRestrictsExecutionToReviewRoles(t *testing.T) {
	runner := OllamaRunner{}
	if _, err := runner.Run(context.Background(), "manager", "prompt", "schema"); err == nil || !strings.Contains(err.Error(), "restricted to reviewer and verifier roles") {
		t.Fatalf("OllamaRunner.Run(manager) error = %v", err)
	}
}

func TestHandoffReviewerToVerifierOrdersResidencyChanges(t *testing.T) {
	var changes []string
	reviewer := OllamaRunner{Model: "reviewer-model", setResidency: func(loaded bool) error {
		changes = append(changes, "reviewer=false")
		if loaded {
			t.Fatal("reviewer was loaded during handoff")
		}
		return nil
	}}
	verifier := OllamaRunner{Model: "verifier-model", setResidency: func(loaded bool) error {
		changes = append(changes, "verifier=true")
		if !loaded {
			t.Fatal("verifier was unloaded during handoff")
		}
		return nil
	}}
	if err := handoffReviewerToVerifier(reviewer, verifier); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(changes, ","); got != "reviewer=false,verifier=true" {
		t.Fatalf("residency changes = %q", got)
	}
}

func TestHandoffReviewerToVerifierStopsOnUnloadFailure(t *testing.T) {
	verifierCalled := false
	reviewer := OllamaRunner{Model: "reviewer-model", setResidency: func(bool) error { return errors.New("status verification failed") }}
	verifier := OllamaRunner{Model: "verifier-model", setResidency: func(bool) error {
		verifierCalled = true
		return nil
	}}
	err := handoffReviewerToVerifier(reviewer, verifier)
	if err == nil || !strings.Contains(err.Error(), `unload reviewer model "reviewer-model": status verification failed`) {
		t.Fatalf("handoffReviewerToVerifier() error = %v", err)
	}
	if verifierCalled {
		t.Fatal("verifier residency changed after reviewer unload failure")
	}
}

func TestHandoffReviewerToVerifierStopsOnLoadFailure(t *testing.T) {
	reviewerCalled := false
	reviewer := OllamaRunner{Model: "reviewer-model", setResidency: func(bool) error {
		reviewerCalled = true
		return nil
	}}
	verifier := OllamaRunner{Model: "verifier-model", setResidency: func(bool) error { return errors.New("load verification failed") }}
	err := handoffReviewerToVerifier(reviewer, verifier)
	if err == nil || !strings.Contains(err.Error(), `load verifier model "verifier-model": load verification failed`) {
		t.Fatalf("handoffReviewerToVerifier() error = %v", err)
	}
	if !reviewerCalled {
		t.Fatal("reviewer was not unloaded before verifier load failure")
	}
}

func TestValidateRunnersRejectsSharedModelIdentity(t *testing.T) {
	policy := ticketPlanPolicy("same-model", "same-model", "same-policy")
	err := validateRunners(Runners{
		Manager:  CodexRunner{},
		Reviewer: OllamaRunner{Policy: policy, Model: "same-model"},
		Verifier: OllamaRunner{Policy: policy, Model: "same-model"},
	})
	if err == nil || !strings.Contains(err.Error(), "models must have distinct identities") {
		t.Fatalf("validateRunners() error = %v", err)
	}
}

func TestValidateRunnersRejectsAliasedModelIdentity(t *testing.T) {
	policy := ticketPlanPolicy("reviewer-model", "verifier-alias", "same-policy")
	policy.Models[1].ID = policy.Models[0].ID
	err := validateRunners(Runners{
		Manager:  CodexRunner{},
		Reviewer: OllamaRunner{Policy: policy, Model: "reviewer-model"},
		Verifier: OllamaRunner{Policy: policy, Model: "verifier-alias"},
	})
	if err == nil || !strings.Contains(err.Error(), "models must have distinct identities") {
		t.Fatalf("validateRunners() error = %v", err)
	}
}

func TestValidateRunnersRejectsDifferentPolicies(t *testing.T) {
	reviewerPolicy := ticketPlanPolicy("reviewer-model", "verifier-model", "reviewer-policy")
	verifierPolicy := ticketPlanPolicy("reviewer-model", "verifier-model", "verifier-policy")
	err := validateRunners(Runners{
		Manager:  CodexRunner{},
		Reviewer: OllamaRunner{Policy: reviewerPolicy, Model: "reviewer-model"},
		Verifier: OllamaRunner{Policy: verifierPolicy, Model: "verifier-model"},
	})
	if err == nil || !strings.Contains(err.Error(), "must use the same local model policy") {
		t.Fatalf("validateRunners() error = %v", err)
	}
}

func ticketPlanPolicy(reviewer, verifier, fingerprint string) ollama.Policy {
	const reviewerID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	verifierID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if reviewer == verifier {
		verifierID = reviewerID
	}
	models := []ollama.Model{
		{Name: reviewer, ID: reviewerID, BenchmarkApproved: true, AllowedRoles: []string{"reviewer", "verifier"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 1024},
	}
	if verifier != reviewer {
		models = append(models, ollama.Model{Name: verifier, ID: verifierID, BenchmarkApproved: true, AllowedRoles: []string{"verifier"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 1024})
	}
	return ollama.Policy{Endpoint: "http://127.0.0.1:11434", RequestTimeoutSeconds: 60, Models: models, Fingerprint: fingerprint}
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
		"prd.md":     "# Goal\nPRD-BODY delivers a ticket plan.\n",
		"spec.md":    "# Scope\nSPEC-BODY uses deterministic validation.\n",
		"roadmap.md": "# Phase 1\nROADMAP-BODY sequences the plan contract.\n",
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
	for _, marker := range []string{"SOURCE prd", "SOURCE spec", "SOURCE roadmap", "PRD-BODY", "SPEC-BODY", "ROADMAP-BODY"} {
		if count := strings.Count(catalog, marker); count != 1 {
			t.Fatalf("catalog contains %q %d times, want once: %q", marker, count, catalog)
		}
	}
}

func TestBuildSourceCatalogRejectsDuplicateSourceIdentity(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("# Scope\nShared source body.\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := loadSources(Request{PRDPath: "prd.md", SpecPath: "spec.md", RoadmapPath: "roadmap.md", RepoRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildSourceCatalog(root, sources); err == nil || !strings.Contains(err.Error(), "three distinct canonical source identities") {
		t.Fatalf("buildSourceCatalog() error = %v", err)
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
