package agentplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-governance/internal/ticketplan"
)

func TestAssignConstraintsValidatesAndWritesOwnerOnlyOutput(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	output := filepath.Join(t.TempDir(), "private", "constraints.json")

	if err := AssignConstraints(filepath.Join(fixtureRoot, "plan.json"), filepath.Join(fixtureRoot, "constraints.json"), output, repoRoot); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("assigned constraints permissions = %v, %v", info.Mode().Perm(), err)
	}
	if info, err = os.Stat(filepath.Dir(output)); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("assigned constraints directory permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestWritePrivateFilePreservesExistingParentPermissions(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "constraints.json")
	if err := writePrivateFile(path, []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("existing parent permissions = %v, %v", info.Mode().Perm(), err)
	}
	info, err = os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private file permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestWritePrivateFileCreatesOwnerOnlyDirectories(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private", "nested")
	path := filepath.Join(directory, "constraints.json")
	if err := writePrivateFile(path, []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	for _, current := range []string{filepath.Dir(directory), directory} {
		info, err := os.Stat(current)
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("created directory %s permissions = %v, %v", current, info.Mode().Perm(), err)
		}
	}
}

func TestAssignConstraintsRejectsPathOutsideApprovedPool(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "constraints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var assignment Constraints
	if err := json.Unmarshal(data, &assignment); err != nil {
		t.Fatal(err)
	}
	assignment.Subtasks[0].AllowedPaths = []string{"internal/unapproved"}
	assignmentPath := filepath.Join(t.TempDir(), "assignment.json")
	data, err = json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assignmentPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	err = AssignConstraints(filepath.Join(fixtureRoot, "plan.json"), assignmentPath, filepath.Join(t.TempDir(), "constraints.json"), repoRoot)
	if err == nil || !strings.Contains(err.Error(), "outside approved pool") {
		t.Fatalf("AssignConstraints() error = %v", err)
	}
}

func TestApplyConstraintsOverridesManagerControlledPhaseAndAllowedPathEvidence(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	plan, err := ticketplan.Load(filepath.Join(fixtureRoot, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	constraints, err := LoadConstraints(filepath.Join(fixtureRoot, "constraints.json"), plan.Sources)
	if err != nil {
		t.Fatal(err)
	}

	// These emulate malformed manager output. Assignment-owned values and their
	// evidence must be replaced before deterministic plan validation runs.
	plan.Subtasks[0].Phase = "unapproved manager phase"
	plan.Subtasks[0].ChangeClass = "high-risk"
	plan.Subtasks[0].AllowedPaths = []string{"internal/unapproved"}
	plan.Subtasks[0].Traceability["phase"] = []ticketplan.Reference{{
		Source: "spec", Section: "Scope", Excerpt: "Source evidence: Add deterministic ticket-plan contract validation in internal/ticketplan.",
	}}
	plan.Subtasks[0].Traceability["change_class"] = []ticketplan.Reference{{
		Source: "spec", Section: "Scope", Excerpt: "unapproved manager change class",
	}}
	plan.Subtasks[0].Traceability["allowed_paths"] = []ticketplan.Reference{{
		Source: "spec", Section: "Scope", Excerpt: "Source evidence: Add deterministic ticket-plan contract validation in internal/ticketplan.",
	}}

	if err := ApplyConstraints(&plan, constraints); err != nil {
		t.Fatal(err)
	}
	issues := strings.Join(plan.ValidateAgainst(repoRoot), "\n")
	for _, field := range []string{"phase", "change_class", "allowed_paths"} {
		if strings.Contains(issues, "subtask contract-validation "+field+" traceability lacks matching source evidence") {
			t.Fatalf("%s evidence was not replaced before validation: %q", field, issues)
		}
	}
}

func TestAssignConstraintsRejectsMissingAssignmentPhase(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "constraints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var assignment Constraints
	if err := json.Unmarshal(data, &assignment); err != nil {
		t.Fatal(err)
	}
	assignment.Subtasks[0].Phase = ""
	assignmentPath := filepath.Join(t.TempDir(), "assignment.json")
	data, err = json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assignmentPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	err = AssignConstraints(filepath.Join(fixtureRoot, "plan.json"), assignmentPath, filepath.Join(t.TempDir(), "constraints.json"), repoRoot)
	if err == nil || !strings.Contains(err.Error(), "is missing phase") {
		t.Fatalf("AssignConstraints() error = %v", err)
	}
}

func TestLoadConstraintsRejectsMissingAssignmentPhaseBeforeManagerDispatch(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	plan, err := ticketplan.Load(filepath.Join(fixtureRoot, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "constraints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var constraints Constraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		t.Fatal(err)
	}
	constraints.Subtasks[0].Phase = ""
	path := filepath.Join(t.TempDir(), "constraints.json")
	data, err = json.Marshal(constraints)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConstraints(path, plan.Sources); err == nil || !strings.Contains(err.Error(), "is missing phase") {
		t.Fatalf("LoadConstraints() error = %v", err)
	}
}

func TestLoadConstraintsRejectsMissingAssignmentPhaseTraceability(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	plan, err := ticketplan.Load(filepath.Join(fixtureRoot, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "constraints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var constraints Constraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		t.Fatal(err)
	}
	delete(constraints.Subtasks[0].Traceability, "phase")
	path := filepath.Join(t.TempDir(), "constraints.json")
	data, err = json.Marshal(constraints)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadConstraints(path, plan.Sources); err == nil || !strings.Contains(err.Error(), "missing phase traceability") {
		t.Fatalf("LoadConstraints() error = %v", err)
	}
}
