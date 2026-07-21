package ticketplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanValidateAgainst(t *testing.T) {
	root, plan := validPlan(t)
	if issues := plan.ValidateAgainst(root); len(issues) != 0 {
		t.Fatalf("ValidateAgainst() = %v", issues)
	}
}

func TestPlanValidateAgainstRejectsChangedSourceAndTraceability(t *testing.T) {
	root, plan := validPlan(t)
	if err := os.WriteFile(filepath.Join(root, "docs", "prd.md"), []byte("# Replaced\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "prd source digest does not match") {
		t.Fatalf("issues = %q", issues)
	}
	if !strings.Contains(issues, "traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstReportsMissingSourceAsUnavailable(t *testing.T) {
	root, plan := validPlan(t)
	if err := os.Remove(filepath.Join(root, "docs", "prd.md")); err != nil {
		t.Fatal(err)
	}
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "prd source is unavailable") {
		t.Fatalf("issues = %q", issues)
	}
	if strings.Contains(issues, "prd source path is outside repository root") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateRejectsUnboundedPathsAndBadDependencies(t *testing.T) {
	_, plan := validPlan(t)
	plan.Subtasks[0].AllowedPaths = []string{"."}
	plan.Subtasks[0].Dependencies = []string{"missing"}
	issues := strings.Join(plan.Validate(), "\n")
	if !strings.Contains(issues, "invalid allowed path") || !strings.Contains(issues, "invalid dependency missing") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAcceptsNamedRootPaths(t *testing.T) {
	_, plan := validPlan(t)
	plan.Subtasks[0].AllowedPaths = []string{"Makefile", ".githooks"}
	if issues := strings.Join(plan.Validate(), "\n"); strings.Contains(issues, "invalid allowed path") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestValidateSubtasksAgainstManifestRejectsStructuralDrift(t *testing.T) {
	manifest := []ContractSlice{
		contractSlice("one", nil, []string{"internal/agentplan", "internal/ticketplan"}),
		contractSlice("two", []string{"one"}, []string{"internal/ticketplan"}),
		contractSlice("three", []string{"one", "two"}, []string{"testdata/ticket-plans"}),
	}
	matching := make([]Subtask, len(manifest))
	for index, slice := range manifest {
		matching[index] = subtaskFromContractSlice(slice)
	}
	if issues := validateSubtasksAgainstManifest(matching, manifest); len(issues) != 0 {
		t.Fatalf("matching manifest issues = %v", issues)
	}

	tests := []struct {
		name   string
		mutate func([]Subtask) []Subtask
	}{
		{"omitted", func(subtasks []Subtask) []Subtask { return subtasks[:2] }},
		{"duplicate", func(subtasks []Subtask) []Subtask { subtasks[1] = subtasks[0]; return subtasks }},
		{"reordered", func(subtasks []Subtask) []Subtask {
			subtasks[0], subtasks[1] = subtasks[1], subtasks[0]
			return subtasks
		}},
		{"extra", func(subtasks []Subtask) []Subtask { return append(subtasks, subtasks[2]) }},
		{"dependency drift", func(subtasks []Subtask) []Subtask { subtasks[1].Dependencies = nil; return subtasks }},
		{"budget drift", func(subtasks []Subtask) []Subtask { subtasks[1].ReviewBudget.MaxChangedLines++; return subtasks }},
		{"path order drift", func(subtasks []Subtask) []Subtask {
			subtasks[0].AllowedPaths[0], subtasks[0].AllowedPaths[1] = subtasks[0].AllowedPaths[1], subtasks[0].AllowedPaths[0]
			return subtasks
		}},
		{"aggregated paths", func(subtasks []Subtask) []Subtask {
			subtasks[0].AllowedPaths = []string{"internal/agentplan,internal/ticketplan"}
			return subtasks
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copyOfMatching := make([]Subtask, len(matching))
			for index, subtask := range matching {
				copyOfMatching[index] = subtask
				copyOfMatching[index].AllowedPaths = append([]string(nil), subtask.AllowedPaths...)
				copyOfMatching[index].Dependencies = append([]string(nil), subtask.Dependencies...)
			}
			if issues := validateSubtasksAgainstManifest(test.mutate(copyOfMatching), manifest); len(issues) == 0 {
				t.Fatal("manifest drift was accepted")
			}
		})
	}
}

func TestValidateSubtasksAgainstManifestRejectsRoadmapImpactDrift(t *testing.T) {
	slice := contractSlice("one", nil, []string{"internal/ticketplan"})
	slice.Assignment.RoadmapImpact = &RoadmapImpact{Mode: "required", RoadmapID: "program", CanonicalPath: "roadmaps/program.yaml", Phase: "1", Transition: "start"}
	matching := subtaskFromContractSlice(slice)
	matching.RoadmapImpact = nil
	if issues := validateSubtasksAgainstManifest([]Subtask{matching}, []ContractSlice{slice}); len(issues) == 0 {
		t.Fatal("manifest drift was accepted")
	}
}

func TestPlanValidateRejectsAggregatedAllowedPaths(t *testing.T) {
	_, plan := validPlan(t)
	plan.Subtasks[0].AllowedPaths = []string{"internal/agentplan,internal/ticketplan"}
	if issues := strings.Join(plan.Validate(), "\n"); !strings.Contains(issues, "invalid allowed path") {
		t.Fatalf("issues = %q", issues)
	}
}

func contractSlice(id string, dependencies, allowedPaths []string) ContractSlice {
	return ContractSlice{
		ID: id,
		Assignment: SliceAssignment{
			Phase: "Phase 3", ChangeClass: "standard",
			ReviewBudget: ReviewBudget{MaxChangedFiles: 2, MaxChangedLines: 100, Components: []string{"ticketplan"}},
			AllowedPaths: append([]string(nil), allowedPaths...), Dependencies: append([]string{}, dependencies...),
			ADR: "docs/decisions/0001-agent-ticket-plan-authority.md",
		},
		SourceDerived: SliceSourceDerived{
			Summary: "Implement " + id, Scope: "Bound " + id,
			NonGoals: []string{"No unrelated work"}, AcceptanceCriteria: []string{"Validate " + id}, ValidationPlan: []string{"Test " + id},
		},
	}
}

func subtaskFromContractSlice(slice ContractSlice) Subtask {
	return Subtask{
		ID: slice.ID, Summary: slice.SourceDerived.Summary, Phase: slice.Assignment.Phase, ChangeClass: slice.Assignment.ChangeClass,
		ReviewBudget: slice.Assignment.ReviewBudget, Scope: slice.SourceDerived.Scope,
		NonGoals: append([]string(nil), slice.SourceDerived.NonGoals...), AcceptanceCriteria: append([]string(nil), slice.SourceDerived.AcceptanceCriteria...),
		ValidationPlan: append([]string(nil), slice.SourceDerived.ValidationPlan...), AllowedPaths: append([]string(nil), slice.Assignment.AllowedPaths...),
		ADR: slice.Assignment.ADR, Dependencies: append([]string{}, slice.Assignment.Dependencies...), RoadmapImpact: slice.Assignment.RoadmapImpact,
	}
}

func TestContainsNormalizedPathAcceptsDirectoryPathWithTrailingSlash(t *testing.T) {
	if !containsNormalizedPath("The allowed path is internal/ticketplan/.", "internal/ticketplan") {
		t.Fatal("directory path with trailing slash was not recognized")
	}
}

func TestTraceSupportsCanonicalEvidenceGatedPublicationSummary(t *testing.T) {
	section := "# Evidence-Gated Publication Work Item"
	if !traceSupportsField("summary", section, "Evidence-Gated Publication Work Item") {
		t.Fatal("canonical summary was not supported by its verified title section")
	}
}

func TestPlanValidateAgainstRejectsAssignmentAuthorityOnNarrativeField(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Summary = "Unsupported manager narrative"
	plan.Story.Traceability["summary"][0].Authority = "assignment"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story is incomplete") || !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateRejectsNormalizedTraversalAndCycles(t *testing.T) {
	_, plan := validPlan(t)
	plan.Subtasks[0].AllowedPaths = []string{"internal/ticketplan/../README.md"}
	plan.Subtasks = append(plan.Subtasks, plan.Subtasks[0])
	plan.Subtasks[1].ID = "two"
	plan.Subtasks[0].Dependencies = []string{"two"}
	plan.Subtasks[1].Dependencies = []string{"one"}
	issues := strings.Join(plan.Validate(), "\n")
	if !strings.Contains(issues, "invalid allowed path") || !strings.Contains(issues, "contain a cycle") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsTraceWithoutEvidence(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Traceability["summary"][0].Section = "Missing source section"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsGenericTraceExcerpt(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Summary = "Unsupported manager value"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsCommonTraceExcerptForUnrelatedField(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Summary = "Source evidence supports separate work"
	plan.Story.Traceability["summary"][0].Excerpt = "Story source evidence defines the delivery objective."
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsSingleGenericTermForMultiTermField(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Summary = "Ticket plan migration"
	plan.Story.Traceability["summary"][0].Excerpt = "Story delivery objective defines the ticket plan."
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsFieldWithoutSubstantiveTokens(t *testing.T) {
	root, plan := validPlan(t)
	plan.Story.Summary = "The source evidence"
	plan.Story.Traceability["summary"][0].Excerpt = "Story source evidence defines the delivery objective."
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "story summary traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsExternalSourceSymlink(t *testing.T) {
	root, plan := validPlan(t)
	external := filepath.Join(t.TempDir(), "external-prd.md")
	if err := os.WriteFile(external, []byte("# Goal\nExternal story source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "docs", "prd.md")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, path); err != nil {
		t.Fatal(err)
	}
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "prd source path is outside repository root") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsMissingTraversalSource(t *testing.T) {
	root, plan := validPlan(t)
	plan.Sources.PRD.Path = "../missing-prd.md"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "prd source path is outside repository root") {
		t.Fatalf("issues = %q", issues)
	}
	if strings.Contains(issues, "prd source is unavailable") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsMissingDescendantOfExternalDirectorySymlink(t *testing.T) {
	root, plan := validPlan(t)
	external := t.TempDir()
	link := filepath.Join(root, "docs", "external")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	plan.Sources.PRD.Path = "docs/external/missing-prd.md"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "prd source path is outside repository root") {
		t.Fatalf("issues = %q", issues)
	}
	if strings.Contains(issues, "prd source is unavailable") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestResolvePathReturnsCanonicalInRepositorySource(t *testing.T) {
	root := t.TempDir()
	canonical := filepath.Join(root, "docs", "canonical-prd.md")
	if err := os.MkdirAll(filepath.Dir(canonical), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("# Goal\nCanonical source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("canonical-prd.md", filepath.Join(root, "docs", "prd.md")); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolvePath(root, "docs/prd.md")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != want {
		t.Fatalf("resolvePath() = %q, want %q", resolved, want)
	}
}

func TestPlanValidateAgainstRequiresEvidenceForEveryArrayElement(t *testing.T) {
	tests := []struct {
		name  string
		field string
		apply func(*Plan)
	}{
		{
			name:  "story acceptance criteria",
			field: "story acceptance_criteria",
			apply: func(plan *Plan) {
				plan.Story.AcceptanceCriteria = append(plan.Story.AcceptanceCriteria, "Independent acceptance condition")
			},
		},
		{
			name:  "review budget components",
			field: "subtask one review_budget",
			apply: func(plan *Plan) {
				plan.Subtasks[0].ReviewBudget.Components = append(plan.Subtasks[0].ReviewBudget.Components, "independent-component")
			},
		},
		{
			name:  "non goals",
			field: "subtask one non_goals",
			apply: func(plan *Plan) {
				plan.Subtasks[0].NonGoals = append(plan.Subtasks[0].NonGoals, "Independent non-goal")
			},
		},
		{
			name:  "subtask acceptance criteria",
			field: "subtask one acceptance_criteria",
			apply: func(plan *Plan) {
				plan.Subtasks[0].AcceptanceCriteria = append(plan.Subtasks[0].AcceptanceCriteria, "Independent completion condition")
			},
		},
		{
			name:  "validation plan",
			field: "subtask one validation_plan",
			apply: func(plan *Plan) {
				plan.Subtasks[0].ValidationPlan = append(plan.Subtasks[0].ValidationPlan, "independent validation")
			},
		},
		{
			name:  "allowed paths",
			field: "subtask one allowed_paths",
			apply: func(plan *Plan) {
				plan.Subtasks[0].AllowedPaths = append(plan.Subtasks[0].AllowedPaths, "cmd/independent")
			},
		},
		{
			name:  "dependencies",
			field: "subtask one dependencies",
			apply: func(plan *Plan) {
				additional := plan.Subtasks[0]
				additional.ID = "two"
				additional.Dependencies = nil
				plan.Subtasks = append(plan.Subtasks, additional)
				plan.Subtasks[0].Dependencies = []string{"two"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, plan := validPlan(t)
			test.apply(&plan)
			issues := strings.Join(plan.ValidateAgainst(root), "\n")
			want := test.field + " traceability lacks matching source evidence"
			if !strings.Contains(issues, want) {
				t.Fatalf("issues = %q, want %q", issues, want)
			}
		})
	}
}

func TestPlanValidateAgainstRequiresEvidenceForEveryReviewBudgetValue(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*Plan)
	}{
		{
			name: "max changed files",
			apply: func(plan *Plan) {
				plan.Subtasks[0].ReviewBudget.MaxChangedFiles = 6
			},
		},
		{
			name: "max changed lines",
			apply: func(plan *Plan) {
				plan.Subtasks[0].ReviewBudget.MaxChangedLines = 301
			},
		},
		{
			name: "component",
			apply: func(plan *Plan) {
				plan.Subtasks[0].ReviewBudget.Components = append(plan.Subtasks[0].ReviewBudget.Components, "independent-component")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, plan := validPlan(t)
			test.apply(&plan)
			issues := strings.Join(plan.ValidateAgainst(root), "\n")
			if !strings.Contains(issues, "subtask one review_budget traceability lacks matching source evidence") {
				t.Fatalf("issues = %q", issues)
			}
		})
	}
}

func TestPlanValidateAgainstRequiresExactNormalizedPhaseEvidence(t *testing.T) {
	root, plan := validPlan(t)
	plan.Subtasks[0].Phase = "Phase 2"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "subtask one phase traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestPlanValidateAgainstRejectsAllowedPathWithOnlySharedTokens(t *testing.T) {
	root, plan := validPlan(t)
	plan.Subtasks[0].AllowedPaths = []string{"internal/ticketplan-extra"}
	plan.Subtasks[0].Traceability["allowed_paths"] = []Reference{{
		Source: "spec", Section: "Scope", Excerpt: "The allowed path is internal/ticketplan.",
	}}
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "subtask one allowed_paths traceability lacks matching source evidence") {
		t.Fatalf("issues = %q", issues)
	}
}

func TestVerifyOpenFileRejectsSymlinkSwap(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"approved.md": "approved", "replacement.md": "replacement"} {
		if err := os.WriteFile(filepath.Join(docs, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(docs, "source.md")
	if err := os.Symlink("approved.md", link); err != nil {
		t.Fatal(err)
	}
	opened, err := os.Open(link)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("replacement.md", link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyOpenFile(root, link, opened); err == nil || !strings.Contains(err.Error(), "source changed while opening") {
		t.Fatalf("verifyOpenFile() error = %v", err)
	}
}

func TestPlanValidateAgainstRejectsADREscape(t *testing.T) {
	root, plan := validPlan(t)
	plan.Subtasks[0].ADR = "docs/decisions/../../README.md"
	issues := strings.Join(plan.ValidateAgainst(root), "\n")
	if !strings.Contains(issues, "ADR reference or rationale is invalid") {
		t.Fatalf("issues = %q", issues)
	}
}

func validPlan(t *testing.T) (string, Plan) {
	t.Helper()
	root := t.TempDir()
	write := func(path, content string) Source {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		digest, err := FileDigest(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		return Source{Path: path, Digest: digest}
	}
	prd := write("docs/prd.md", "# Goal\nStory delivery objective defines the ticket plan.\nTicket plan description explains the work.\nPlanned slice change identifies the implementation.\n# Acceptance\nExpected outcome confirms the result.\nSuccessful behavior verifies completion.\n")
	spec := write("docs/spec.md", "# Scope\nStandard change classifies the work.\nScoped change bounds the work.\nNo remote writes are excluded.\nGo test ticketplan validation verifies behavior.\nThe allowed path is internal/ticketplan.\n# Decision\nExisting ticketplan validation pattern supports the rationale.\n# Budget\n5 300 ticketplan identifies the reviewed component.\n")
	roadmap := write("docs/roadmap.md", "# Phase 1\nPhase 1 identifies the implementation stage.\n")
	trace := TraceMap{
		"summary":             {{Source: "prd", Section: "Goal", Excerpt: "Story delivery objective defines the ticket plan."}},
		"description":         {{Source: "prd", Section: "Goal", Excerpt: "Ticket plan description explains the work."}},
		"acceptance_criteria": {{Source: "prd", Section: "Acceptance", Excerpt: "Expected outcome confirms the result."}},
		"phase":               {{Source: "roadmap", Section: "Phase 1", Excerpt: "Phase 1 identifies the implementation stage."}},
		"change_class":        {{Source: "spec", Section: "Scope", Excerpt: "Standard change classifies the work."}},
		"review_budget":       {{Source: "spec", Section: "Budget", Excerpt: "5 300 ticketplan identifies the reviewed component."}},
		"scope":               {{Source: "spec", Section: "Scope", Excerpt: "Scoped change bounds the work."}},
		"non_goals":           {{Source: "spec", Section: "Scope", Excerpt: "No remote writes are excluded."}},
		"validation_plan":     {{Source: "spec", Section: "Scope", Excerpt: "Go test ticketplan validation verifies behavior."}},
		"allowed_paths":       {{Source: "spec", Section: "Scope", Excerpt: "The allowed path is internal/ticketplan."}},
		"adr":                 {{Source: "spec", Section: "Decision", Excerpt: "Existing ticketplan validation pattern supports the rationale."}},
		"dependencies":        {{Source: "roadmap", Section: "Phase 1", Excerpt: "Phase 1 identifies the implementation stage."}},
	}
	subtaskTrace := make(TraceMap, len(trace))
	for field, refs := range trace {
		subtaskTrace[field] = append([]Reference(nil), refs...)
	}
	subtaskTrace["summary"] = []Reference{{Source: "prd", Section: "Goal", Excerpt: "Planned slice change identifies the implementation."}}
	subtaskTrace["acceptance_criteria"] = []Reference{{Source: "prd", Section: "Acceptance", Excerpt: "Successful behavior verifies completion."}}
	return root, Plan{
		FormatVersion: 1,
		Status:        "draft",
		Sources:       Sources{PRD: prd, Spec: spec, Roadmap: roadmap},
		Story:         Story{Summary: "Story delivery objective", Description: "Ticket plan description", AcceptanceCriteria: []string{"Expected outcome"}, Traceability: trace},
		Subtasks: []Subtask{{
			ID: "one", Summary: "Planned slice change", Phase: "Phase 1", ChangeClass: "standard",
			ReviewBudget: ReviewBudget{MaxChangedFiles: 5, MaxChangedLines: 300, Components: []string{"ticketplan"}},
			Scope:        "Scoped change", NonGoals: []string{"No remote writes"}, AcceptanceCriteria: []string{"Successful behavior"},
			ValidationPlan: []string{"go test ticketplan validation"}, AllowedPaths: []string{"internal/ticketplan"},
			ADR: "No ADR needed: follows the existing ticket-plan validation pattern", Traceability: subtaskTrace,
		}},
	}
}
