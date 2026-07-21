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
	plan, err := ticketplan.Load(filepath.Join(fixtureRoot, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}

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
	constraints, err := LoadConstraints(output, plan.Sources)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := buildAuthorityContract(constraints)
	if err != nil || contract.ValidateAgainst(repoRoot) != nil {
		t.Fatalf("assigned constraints did not preserve canonical source-derived handoff: %v", err)
	}
	if constraints.Story == nil || constraints.Story.Summary != "Validate ticket plans" || constraints.Subtasks[0].SourceDerived == nil || constraints.Subtasks[0].SourceDerived.Summary != "Deterministic contract validation" {
		t.Fatalf("assigned constraints omitted canonical narrative: %#v", constraints)
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

func TestAssignConstraintsRejectsAssignmentCanonicalNarrative(t *testing.T) {
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
	assignment.Story = &StoryConstraints{Summary: "Injected narrative"}
	assignmentPath := filepath.Join(t.TempDir(), "assignment.json")
	data, err = json.Marshal(assignment)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assignmentPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	err = AssignConstraints(filepath.Join(fixtureRoot, "plan.json"), assignmentPath, filepath.Join(t.TempDir(), "constraints.json"), repoRoot)
	if err == nil || !strings.Contains(err.Error(), "must not provide canonical source-derived narrative") {
		t.Fatalf("AssignConstraints() error = %v", err)
	}
}

func TestAssignConstraintsRejectsUntraceableManagerNarrative(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	fixtureRoot := filepath.Join(repoRoot, "testdata", "ticket-plans", "valid")
	plan, err := ticketplan.Load(filepath.Join(fixtureRoot, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	plan.Subtasks[0].Scope = "Untraceable manager scope"
	decomposition := filepath.Join(t.TempDir(), "decomposition.json")
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(decomposition, data, 0o600); err != nil {
		t.Fatal(err)
	}
	err = AssignConstraints(decomposition, filepath.Join(fixtureRoot, "constraints.json"), filepath.Join(t.TempDir(), "constraints.json"), repoRoot)
	if err == nil || !strings.Contains(err.Error(), "invalid source-derived narrative or traceability") {
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
		for _, ref := range plan.Subtasks[0].Traceability[field] {
			if ref.Authority != "" {
				t.Fatalf("%s trace retained temporary assignment authority %q", field, ref.Authority)
			}
		}
		if strings.Contains(issues, "subtask contract-validation "+field+" traceability lacks matching source evidence") {
			t.Fatalf("%s evidence was not replaced before validation: %q", field, issues)
		}
	}
}

func TestApplyConstraintsOverridesMalformedManagerSourceDerivedFields(t *testing.T) {
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
	constraints.Story = &StoryConstraints{Summary: "Canonical story", Description: "Canonical description", AcceptanceCriteria: []string{"Canonical acceptance"}, Traceability: plan.Story.Traceability}
	constraints.Subtasks[0].SourceDerived = &SourceDerivedConstraints{Summary: "Canonical subtask", Scope: "Canonical scope", NonGoals: []string{"Canonical non-goal"}, AcceptanceCriteria: []string{"Canonical acceptance"}, ValidationPlan: []string{"Canonical validation"}, Traceability: ticketplan.TraceMap{"summary": plan.Subtasks[0].Traceability["summary"], "scope": plan.Subtasks[0].Traceability["scope"], "non_goals": plan.Subtasks[0].Traceability["non_goals"], "acceptance_criteria": plan.Subtasks[0].Traceability["acceptance_criteria"], "validation_plan": plan.Subtasks[0].Traceability["validation_plan"]}}
	plan.Story.Summary, plan.Subtasks[0].Scope = "Malformed manager story", "Malformed manager scope"
	plan.Subtasks[0].AcceptanceCriteria = []string{"Malformed manager acceptance"}
	if err := ApplyConstraints(&plan, constraints); err != nil {
		t.Fatal(err)
	}
	if plan.Story.Summary != "Canonical story" || plan.Subtasks[0].Summary != "Canonical subtask" || plan.Subtasks[0].Scope != "Canonical scope" || plan.Subtasks[0].AcceptanceCriteria[0] != "Canonical acceptance" {
		t.Fatalf("manager fields were not replaced: %#v", plan)
	}
}

func TestBuildAuthorityContractFromCanonicalConstraints(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	plan, err := ticketplan.Load(filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	constraints, err := LoadConstraints(filepath.Join(repoRoot, "testdata", "ticket-plans", "valid", "constraints.json"), plan.Sources)
	if err != nil {
		t.Fatal(err)
	}
	constraints.Story = &StoryConstraints{Summary: plan.Story.Summary, Description: plan.Story.Description, AcceptanceCriteria: plan.Story.AcceptanceCriteria, Traceability: plan.Story.Traceability}
	subtask := plan.Subtasks[0]
	constraints.Subtasks[0].ADR = subtask.ADR
	constraints.Subtasks[0].SourceDerived = &SourceDerivedConstraints{Summary: subtask.Summary, Scope: subtask.Scope, NonGoals: subtask.NonGoals, AcceptanceCriteria: subtask.AcceptanceCriteria, ValidationPlan: subtask.ValidationPlan, Traceability: subtask.Traceability}
	contract, err := buildAuthorityContract(constraints)
	if err != nil || contract.ValidateAgainst(repoRoot) != nil {
		t.Fatalf("buildAuthorityContract() error = %v", err)
	}
	if contract.Roles != (ticketplan.SourceRoleBindings{PRD: "prd", Spec: "spec", Roadmap: "roadmap"}) {
		t.Fatalf("authority contract role bindings = %#v", contract.Roles)
	}
	evidenceRoles := map[string]bool{}
	for _, evidence := range contract.Evidence {
		evidenceRoles[evidence.Role] = true
	}
	for _, role := range []string{"prd", "spec", "roadmap"} {
		if !evidenceRoles[role] {
			t.Fatalf("authority contract evidence lost %s role binding", role)
		}
	}
	again, err := buildAuthorityContract(constraints)
	firstDigest, _ := contract.Digest()
	secondDigest, _ := again.Digest()
	if err != nil || firstDigest != secondDigest {
		t.Fatalf("authority contract digest is not deterministic: %q != %q", firstDigest, secondDigest)
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

func TestDraftConstraintsReadsOnlyAllowedPathsSection(t *testing.T) {
	constraints, err := draftConstraintsForSpec(t, "# Overview\n`outside/path` must not be included.\n\n## Allowed Paths\n\nImplementation is limited to `AGENTS.md`, `testdata`, and `internal/agentplan`.\n- `docs/design`\n\n## Review Budget\n\n4 changed files, 350 changed lines, and parser tests. `also/outside`\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"AGENTS.md", "docs/design", "internal/agentplan", "testdata"}
	if strings.Join(constraints.PathPool, ",") != strings.Join(want, ",") {
		t.Fatalf("PathPool = %#v, want %#v", constraints.PathPool, want)
	}
}

func TestDraftConstraintsReadsCanonicalProseAllowedPaths(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	constraints, err := DraftConstraints(Request{
		PRDPath:     "docs/design/constraint-path-parser-prd.md",
		SpecPath:    "docs/design/constraint-path-parser-spec.md",
		RoadmapPath: "docs/roadmaps/go-cli-migration.md",
		RepoRoot:    repoRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs/design", "internal/agentplan", "testdata/ticket-plans"}
	if strings.Join(constraints.PathPool, ",") != strings.Join(want, ",") {
		t.Fatalf("PathPool = %#v, want %#v", constraints.PathPool, want)
	}
}

func TestDraftConstraintsRejectsInvalidAndDuplicateAllowedPaths(t *testing.T) {
	for name, entry := range map[string]string{
		"absolute":  "`/outside`",
		"traversal": "`internal/../outside`",
		"wildcard":  "`internal/*`",
		"duplicate": "`docs/design`\n- `docs/design`",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := draftConstraintsForSpec(t, "## Allowed Paths\n\n"+entry+"\n\n## Review Budget\n\n4 changed files, 350 changed lines, and parser tests.\n")
			if err == nil || !strings.Contains(err.Error(), "Allowed Paths") {
				t.Fatalf("DraftConstraints() error = %v", err)
			}
		})
	}
}

func draftConstraintsForSpec(t *testing.T, spec string) (Constraints, error) {
	t.Helper()
	root := t.TempDir()
	for path, data := range map[string]string{
		"prd.md":     "# Goal\nUnique PRD source.\n",
		"spec.md":    spec,
		"roadmap.md": "# Phase 1\nUnique roadmap source.\n",
	} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return DraftConstraints(Request{PRDPath: "prd.md", SpecPath: "spec.md", RoadmapPath: "roadmap.md", RepoRoot: root})
}
