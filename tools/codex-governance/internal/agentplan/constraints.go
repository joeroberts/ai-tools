package agentplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"codex-governance/internal/ticketplan"
)

type Constraints struct {
	FormatVersion int                     `json:"format_version"`
	Sources       ticketplan.Sources      `json:"sources"`
	PathPool      []string                `json:"path_pool"`
	ReviewBudget  ticketplan.ReviewBudget `json:"review_budget"`
	Story         *StoryConstraints       `json:"story,omitempty"`
	Subtasks      []SubtaskConstraints    `json:"subtasks"`
}

// StoryConstraints is canonical source-derived story content. It is approved
// before manager dispatch and is never accepted from manager output.
type StoryConstraints struct {
	Summary            string              `json:"summary"`
	Description        string              `json:"description"`
	AcceptanceCriteria []string            `json:"acceptance_criteria"`
	Traceability       ticketplan.TraceMap `json:"traceability"`
}

type SubtaskConstraints struct {
	ID            string                    `json:"id"`
	Phase         string                    `json:"phase"`
	ChangeClass   string                    `json:"change_class"`
	AllowedPaths  []string                  `json:"allowed_paths"`
	ReviewBudget  ticketplan.ReviewBudget   `json:"review_budget"`
	Dependencies  []string                  `json:"dependencies"`
	ADR           string                    `json:"adr,omitempty"`
	RoadmapImpact *ticketplan.RoadmapImpact `json:"roadmap_impact,omitempty"`
	Traceability  ticketplan.TraceMap       `json:"traceability"`
	SourceDerived *SourceDerivedConstraints `json:"source_derived,omitempty"`
}

// SourceDerivedConstraints contains the verified-source fields which must not
// be selected or paraphrased by the manager.
type SourceDerivedConstraints struct {
	Summary            string              `json:"summary"`
	Scope              string              `json:"scope"`
	NonGoals           []string            `json:"non_goals"`
	AcceptanceCriteria []string            `json:"acceptance_criteria"`
	ValidationPlan     []string            `json:"validation_plan"`
	Traceability       ticketplan.TraceMap `json:"traceability"`
}

var (
	constraintBudgetPattern = regexp.MustCompile(`(?i)(\d+) changed files,?\s*(\d+) changed lines,?\s*(?:and )?([^\.\n]+)`)
	markdownHeadingPattern  = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$`)
	inlineCodePattern       = regexp.MustCompile("`([^`\\r\\n]*)`")
)

func DraftConstraints(request Request) (Constraints, error) {
	sources, err := loadSources(request)
	if err != nil {
		return Constraints{}, err
	}
	spec, err := ticketplan.ReadVerifiedSource(request.RepoRoot, sources.Spec.Path)
	if err != nil {
		return Constraints{}, fmt.Errorf("read specification constraints: %w", err)
	}
	pathPool, err := parseAllowedPaths(string(spec.Data))
	if err != nil {
		return Constraints{}, fmt.Errorf("parse specification allowed paths: %w", err)
	}
	match := constraintBudgetPattern.FindStringSubmatch(string(spec.Data))
	if len(match) != 4 {
		return Constraints{}, fmt.Errorf("specification must state a review budget as '<files> changed files, <lines> changed lines, and <components>'")
	}
	files, _ := strconv.Atoi(match[1])
	lines, _ := strconv.Atoi(match[2])
	components := strings.Split(strings.TrimSpace(match[3]), ",")
	for index := range components {
		components[index] = strings.TrimSpace(components[index])
	}
	return Constraints{FormatVersion: 1, Sources: sources, PathPool: pathPool, ReviewBudget: ticketplan.ReviewBudget{MaxChangedFiles: files, MaxChangedLines: lines, Components: components}}, nil
}

// parseAllowedPaths reads only inline-code entries in the Allowed Paths
// Markdown section. The section may declare paths as prose or list entries.
func parseAllowedPaths(markdown string) ([]string, error) {
	lines := strings.Split(markdown, "\n")
	sectionLevel := 0
	inSection, inFence := false, false
	seen := map[string]bool{}
	var paths []string

	for _, line := range lines {
		if heading := markdownHeadingPattern.FindStringSubmatch(line); len(heading) == 3 {
			level := len(heading[1])
			title := strings.TrimSpace(heading[2])
			if !inSection && title == "Allowed Paths" {
				inSection, sectionLevel = true, level
				continue
			}
			if inSection && level <= sectionLevel {
				break
			}
			if inSection {
				continue
			}
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		matches := inlineCodePattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			path := match[1]
			if !validConstraintPath(path) {
				return nil, fmt.Errorf("Allowed Paths contains invalid entry %q", path)
			}
			if seen[path] {
				return nil, fmt.Errorf("Allowed Paths contains duplicate entry %q", path)
			}
			seen[path] = true
			paths = append(paths, path)
		}
	}
	if !inSection {
		return nil, fmt.Errorf("specification has no Allowed Paths section")
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("Allowed Paths section contains no entries")
	}
	sort.Strings(paths)
	return paths, nil
}

func validConstraintPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.ContainsAny(path, "*?[,\r\n") {
		return false
	}
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	cleaned := filepath.Clean(filepath.FromSlash(path))
	return cleaned == filepath.FromSlash(path) && cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
}

func WriteConstraints(path string, constraints Constraints) error {
	if constraints.FormatVersion != 1 || len(constraints.PathPool) == 0 || constraints.ReviewBudget.MaxChangedFiles < 1 || constraints.ReviewBudget.MaxChangedLines < 1 || len(constraints.ReviewBudget.Components) == 0 {
		return fmt.Errorf("constraints draft is incomplete")
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing constraints: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(constraints, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func LoadConstraints(path string, sources ticketplan.Sources) (Constraints, error) {
	constraints, err := loadConstraints(path)
	if err != nil {
		return Constraints{}, err
	}
	if constraints.FormatVersion != 1 || constraints.Sources != sources || len(constraints.Subtasks) == 0 {
		return Constraints{}, fmt.Errorf("constraints do not match current sources or contain no subtask assignments")
	}
	if err := validateAssignment(constraints); err != nil {
		return Constraints{}, err
	}
	return constraints, nil
}

// AssignConstraints validates a manager decomposition and a human-approved
// assignment before producing an owner-only constraints file for plan review.
func AssignConstraints(decompositionPath, assignmentPath, outputPath, repoRoot string) error {
	plan, err := ticketplan.Load(decompositionPath)
	if err != nil {
		return fmt.Errorf("load manager decomposition: %w", err)
	}
	constraints, err := loadConstraints(assignmentPath)
	if err != nil {
		return err
	}
	if constraints.FormatVersion != 1 || constraints.Sources != plan.Sources || len(constraints.Subtasks) == 0 {
		return fmt.Errorf("assignment does not match manager decomposition sources or contains no subtask assignments")
	}
	if err := validateAssignment(constraints); err != nil {
		return err
	}
	if err := validateDecompositionAssignments(plan, constraints); err != nil {
		return err
	}
	if err := ApplyConstraints(&plan, constraints); err != nil {
		return err
	}
	if issues := plan.ValidateAgainst(repoRoot); len(issues) != 0 {
		return fmt.Errorf("manager decomposition contains invalid source-derived narrative or traceability: %v", issues)
	}
	if err := captureSourceDerivedConstraints(&constraints, plan); err != nil {
		return err
	}
	if _, err := buildAuthorityContract(constraints); err != nil {
		return fmt.Errorf("build source-derived authority contract: %w", err)
	}
	return writePrivateConstraints(outputPath, constraints)
}

// validateDecompositionAssignments ensures approved assignments remain bound
// to the manager's declared slices before assignment-owned IDs replace them.
// Without this check, a reordered decomposition could promote valid narrative
// and traceability under the wrong approved subtask ID.
func validateDecompositionAssignments(plan ticketplan.Plan, constraints Constraints) error {
	if len(plan.Subtasks) != len(constraints.Subtasks) {
		return fmt.Errorf("manager decomposition has %d subtasks but constraints assign %d", len(plan.Subtasks), len(constraints.Subtasks))
	}
	for index, subtask := range plan.Subtasks {
		if subtask.ID != constraints.Subtasks[index].ID {
			return fmt.Errorf("manager decomposition subtask %d has ID %q, but constraints assign %q", index+1, subtask.ID, constraints.Subtasks[index].ID)
		}
	}
	return nil
}

// captureSourceDerivedConstraints promotes the validated manager narrative to
// the approved handoff. Assignment-owned fields have already replaced the
// manager values, so the resulting constraints are a complete authority input
// for generation.
func captureSourceDerivedConstraints(constraints *Constraints, plan ticketplan.Plan) error {
	if len(constraints.Subtasks) != len(plan.Subtasks) {
		return fmt.Errorf("manager decomposition has %d subtasks but constraints assign %d", len(plan.Subtasks), len(constraints.Subtasks))
	}
	constraints.Story = &StoryConstraints{
		Summary:            plan.Story.Summary,
		Description:        plan.Story.Description,
		AcceptanceCriteria: append([]string(nil), plan.Story.AcceptanceCriteria...),
		Traceability:       traceFields(plan.Story.Traceability, "summary", "description", "acceptance_criteria"),
	}
	for index := range constraints.Subtasks {
		subtask := plan.Subtasks[index]
		if constraints.Subtasks[index].ADR == "" {
			constraints.Subtasks[index].ADR = subtask.ADR
			constraints.Subtasks[index].Traceability["adr"] = append([]ticketplan.Reference(nil), subtask.Traceability["adr"]...)
		}
		constraints.Subtasks[index].SourceDerived = &SourceDerivedConstraints{
			Summary:            subtask.Summary,
			Scope:              subtask.Scope,
			NonGoals:           append([]string(nil), subtask.NonGoals...),
			AcceptanceCriteria: append([]string(nil), subtask.AcceptanceCriteria...),
			ValidationPlan:     append([]string(nil), subtask.ValidationPlan...),
			Traceability:       traceFields(subtask.Traceability, "summary", "scope", "non_goals", "acceptance_criteria", "validation_plan"),
		}
	}
	return nil
}

func traceFields(source ticketplan.TraceMap, fields ...string) ticketplan.TraceMap {
	result := make(ticketplan.TraceMap, len(fields))
	for _, field := range fields {
		result[field] = append([]ticketplan.Reference(nil), source[field]...)
	}
	return result
}

// PromoteConstraints preserves the legacy CLI contract until the assignment
// command is wired in a follow-up change. New callers must use
// AssignConstraints, which binds constraints to a manager decomposition.
func PromoteConstraints(draftPath, outputPath string) error {
	constraints, err := loadConstraints(draftPath)
	if err != nil {
		return err
	}
	return writePrivateConstraints(outputPath, constraints)
}

func loadConstraints(path string) (Constraints, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Constraints{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var constraints Constraints
	if err := decoder.Decode(&constraints); err != nil {
		return Constraints{}, fmt.Errorf("parse constraints: %w", err)
	}
	if decoder.More() {
		return Constraints{}, fmt.Errorf("parse constraints: multiple JSON values")
	}
	return constraints, nil
}

func validateAssignment(constraints Constraints) error {
	if len(constraints.PathPool) == 0 || constraints.ReviewBudget.MaxChangedFiles < 1 || constraints.ReviewBudget.MaxChangedLines < 1 || len(constraints.ReviewBudget.Components) == 0 {
		return fmt.Errorf("assignment is missing the approved path pool or review budget")
	}
	paths, components := map[string]bool{}, map[string]bool{}
	for _, path := range constraints.PathPool {
		if path == "" || paths[path] {
			return fmt.Errorf("assignment path pool is invalid")
		}
		paths[path] = true
	}
	for _, component := range constraints.ReviewBudget.Components {
		if component == "" || components[component] {
			return fmt.Errorf("assignment review-budget components are invalid")
		}
		components[component] = true
	}
	files, lines := 0, 0
	for _, assignment := range constraints.Subtasks {
		files += assignment.ReviewBudget.MaxChangedFiles
		lines += assignment.ReviewBudget.MaxChangedLines
		if assignment.Phase == "" || assignment.ChangeClass == "" {
			return fmt.Errorf("assignment for subtask %q is missing phase or change class", assignment.ID)
		}
		if assignment.ReviewBudget.MaxChangedFiles < 1 || assignment.ReviewBudget.MaxChangedLines < 1 || len(assignment.AllowedPaths) == 0 || assignment.Traceability == nil {
			return fmt.Errorf("assignment for subtask %q is incomplete", assignment.ID)
		}
		for _, field := range []string{"phase", "change_class", "review_budget", "allowed_paths", "dependencies"} {
			if len(assignment.Traceability[field]) == 0 {
				return fmt.Errorf("assignment for subtask %q is missing %s traceability", assignment.ID, field)
			}
		}
		if assignment.ADR != "" && len(assignment.Traceability["adr"]) == 0 {
			return fmt.Errorf("assignment for subtask %q is missing adr traceability", assignment.ID)
		}
		for _, path := range assignment.AllowedPaths {
			if !paths[path] {
				return fmt.Errorf("assignment for subtask %q uses path outside approved pool: %s", assignment.ID, path)
			}
		}
		for _, component := range assignment.ReviewBudget.Components {
			if !components[component] {
				return fmt.Errorf("assignment for subtask %q uses component outside approved budget: %s", assignment.ID, component)
			}
		}
	}
	if files > constraints.ReviewBudget.MaxChangedFiles || lines > constraints.ReviewBudget.MaxChangedLines {
		return fmt.Errorf("assigned review budgets exceed the approved review budget")
	}
	return nil
}

func writePrivateConstraints(path string, constraints Constraints) error {
	data, err := json.MarshalIndent(constraints, "", "  ")
	if err != nil {
		return err
	}
	return writePrivateFile(path, append(data, '\n'))
}

func writePrivateFile(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("refusing to overwrite existing constraints: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	directory := filepath.Dir(path)
	if err := makePrivateDirectory(directory); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// makePrivateDirectory creates missing path components with owner-only
// permissions without changing an existing operator-selected directory.
func makePrivateDirectory(directory string) error {
	directory = filepath.Clean(directory)
	missing := []string{}
	for current := directory; ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("artifact parent is not a directory: %s", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("artifact parent does not exist: %s", directory)
		}
		missing = append(missing, current)
	}
	for index := len(missing) - 1; index >= 0; index-- {
		created := false
		if err := os.Mkdir(missing[index], 0o700); err == nil {
			created = true
		} else if !os.IsExist(err) {
			return err
		}
		info, err := os.Stat(missing[index])
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("artifact parent is not a directory: %s", missing[index])
		}
		if created {
			if err := os.Chmod(missing[index], 0o700); err != nil {
				return err
			}
		}
	}
	return nil
}

func ApplyConstraints(plan *ticketplan.Plan, constraints Constraints) error {
	if constraints.Story != nil {
		plan.Story.Summary = constraints.Story.Summary
		plan.Story.Description = constraints.Story.Description
		plan.Story.AcceptanceCriteria = append([]string(nil), constraints.Story.AcceptanceCriteria...)
		plan.Story.Traceability = cloneTrace(constraints.Story.Traceability)
	}
	if len(plan.Subtasks) != len(constraints.Subtasks) {
		return fmt.Errorf("manager decomposition has %d subtasks but constraints assign %d", len(plan.Subtasks), len(constraints.Subtasks))
	}
	for index := range plan.Subtasks {
		assignment := constraints.Subtasks[index]
		if assignment.ID == "" || assignment.Phase == "" || assignment.ChangeClass == "" || len(assignment.AllowedPaths) == 0 || assignment.ReviewBudget.MaxChangedFiles < 1 || assignment.ReviewBudget.MaxChangedLines < 1 || len(assignment.ReviewBudget.Components) == 0 || assignment.Traceability == nil {
			return fmt.Errorf("constraints subtask %d is incomplete", index+1)
		}
		subtask := &plan.Subtasks[index]
		subtask.ID, subtask.Phase, subtask.ChangeClass, subtask.AllowedPaths, subtask.ReviewBudget, subtask.Dependencies, subtask.RoadmapImpact = assignment.ID, assignment.Phase, assignment.ChangeClass, assignment.AllowedPaths, assignment.ReviewBudget, assignment.Dependencies, assignment.RoadmapImpact
		if assignment.ADR != "" {
			subtask.ADR = assignment.ADR
		}
		for _, field := range []string{"phase", "change_class", "review_budget", "allowed_paths", "dependencies", "adr"} {
			if len(assignment.Traceability[field]) == 0 && field == "adr" && assignment.ADR == "" {
				continue
			}
			subtask.Traceability[field] = append([]ticketplan.Reference(nil), assignment.Traceability[field]...)
		}
		if source := assignment.SourceDerived; source != nil {
			subtask.Summary = source.Summary
			subtask.Scope = source.Scope
			subtask.NonGoals = append([]string(nil), source.NonGoals...)
			subtask.AcceptanceCriteria = append([]string(nil), source.AcceptanceCriteria...)
			subtask.ValidationPlan = append([]string(nil), source.ValidationPlan...)
			for _, field := range []string{"summary", "scope", "non_goals", "acceptance_criteria", "validation_plan"} {
				subtask.Traceability[field] = append([]ticketplan.Reference(nil), source.Traceability[field]...)
			}
		}
	}
	return nil
}

func cloneTrace(source ticketplan.TraceMap) ticketplan.TraceMap {
	result := make(ticketplan.TraceMap, len(source))
	for field, refs := range source {
		result[field] = append([]ticketplan.Reference(nil), refs...)
	}
	return result
}

func buildAuthorityContract(constraints Constraints) (ticketplan.AuthorityContract, error) {
	if constraints.Story == nil {
		return ticketplan.AuthorityContract{}, fmt.Errorf("unsupported legacy constraints: canonical story is required")
	}
	contract := ticketplan.AuthorityContract{
		FormatVersion: ticketplan.AuthorityContractFormatVersion,
		Sources: []ticketplan.ContractSource{
			{ID: "prd", Path: constraints.Sources.PRD.Path, Digest: constraints.Sources.PRD.Digest},
			{ID: "spec", Path: constraints.Sources.Spec.Path, Digest: constraints.Sources.Spec.Digest},
			{ID: "roadmap", Path: constraints.Sources.Roadmap.Path, Digest: constraints.Sources.Roadmap.Digest},
		},
		Roles:          ticketplan.SourceRoleBindings{PRD: "prd", Spec: "spec", Roadmap: "roadmap"},
		Story:          ticketplan.ContractStory{Summary: constraints.Story.Summary, Description: constraints.Story.Description, AcceptanceCriteria: append([]string(nil), constraints.Story.AcceptanceCriteria...)},
		NarrativeRules: []ticketplan.NarrativeRule{},
	}
	appendEvidence := func(prefix string, trace ticketplan.TraceMap) {
		fields := make([]string, 0, len(trace))
		for field := range trace {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			refs := trace[field]
			for _, ref := range refs {
				contract.Evidence = append(contract.Evidence, ticketplan.ContractEvidence{Field: prefix + field, Role: ref.Source, Section: ref.Section, Excerpt: ref.Excerpt})
			}
		}
	}
	appendEvidence("story.", constraints.Story.Traceability)
	for _, assignment := range constraints.Subtasks {
		if assignment.SourceDerived == nil {
			return ticketplan.AuthorityContract{}, fmt.Errorf("unsupported legacy constraints: subtask %q lacks canonical source-derived values", assignment.ID)
		}
		source := assignment.SourceDerived
		contract.Slices = append(contract.Slices, ticketplan.ContractSlice{
			ID:            assignment.ID,
			Assignment:    ticketplan.SliceAssignment{Phase: assignment.Phase, ChangeClass: assignment.ChangeClass, ReviewBudget: assignment.ReviewBudget, AllowedPaths: append([]string(nil), assignment.AllowedPaths...), Dependencies: append([]string{}, assignment.Dependencies...), ADR: assignment.ADR, RoadmapImpact: assignment.RoadmapImpact},
			SourceDerived: ticketplan.SliceSourceDerived{Summary: source.Summary, Scope: source.Scope, NonGoals: append([]string(nil), source.NonGoals...), AcceptanceCriteria: append([]string(nil), source.AcceptanceCriteria...), ValidationPlan: append([]string(nil), source.ValidationPlan...)},
		})
		appendEvidence("slices[].", assignment.Traceability)
		appendEvidence("slices[].", source.Traceability)
	}
	return contract, contract.Validate()
}
