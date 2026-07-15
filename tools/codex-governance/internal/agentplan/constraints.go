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
	Subtasks      []SubtaskConstraints    `json:"subtasks"`
}

type SubtaskConstraints struct {
	ID           string                  `json:"id"`
	Phase        string                  `json:"phase"`
	ChangeClass  string                  `json:"change_class"`
	AllowedPaths []string                `json:"allowed_paths"`
	ReviewBudget ticketplan.ReviewBudget `json:"review_budget"`
	Dependencies []string                `json:"dependencies"`
	ADR          string                  `json:"adr,omitempty"`
	Traceability ticketplan.TraceMap     `json:"traceability"`
}

var (
	constraintPathPattern   = regexp.MustCompile(`(?:[A-Za-z0-9._-]+/)+[A-Za-z0-9._-]+`)
	constraintBudgetPattern = regexp.MustCompile(`(?i)(\d+) changed files,?\s*(\d+) changed lines,?\s*(?:and )?([^\.\n]+)`)
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
	paths := constraintPathPattern.FindAllString(string(spec.Data), -1)
	unique := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSuffix(path, ".")
		if strings.Contains(path, "..") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "./") {
			continue
		}
		unique[path] = true
	}
	pathPool := make([]string, 0, len(unique))
	for path := range unique {
		pathPool = append(pathPool, path)
	}
	sort.Strings(pathPool)
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
	if err := ApplyConstraints(&plan, constraints); err != nil {
		return err
	}
	// The manager decomposition is intentionally unconstrained. Its remaining
	// narrative fields are validated after the assignment is applied to the
	// newly generated plan, not while assigning the approved boundaries.
	return writePrivateConstraints(outputPath, constraints)
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
	if len(plan.Subtasks) != len(constraints.Subtasks) {
		return fmt.Errorf("manager decomposition has %d subtasks but constraints assign %d", len(plan.Subtasks), len(constraints.Subtasks))
	}
	for index := range plan.Subtasks {
		assignment := constraints.Subtasks[index]
		if assignment.ID == "" || assignment.Phase == "" || assignment.ChangeClass == "" || len(assignment.AllowedPaths) == 0 || assignment.ReviewBudget.MaxChangedFiles < 1 || assignment.ReviewBudget.MaxChangedLines < 1 || len(assignment.ReviewBudget.Components) == 0 || assignment.Traceability == nil {
			return fmt.Errorf("constraints subtask %d is incomplete", index+1)
		}
		subtask := &plan.Subtasks[index]
		subtask.ID, subtask.Phase, subtask.ChangeClass, subtask.AllowedPaths, subtask.ReviewBudget, subtask.Dependencies = assignment.ID, assignment.Phase, assignment.ChangeClass, assignment.AllowedPaths, assignment.ReviewBudget, assignment.Dependencies
		if assignment.ADR != "" {
			subtask.ADR = assignment.ADR
		}
		for _, field := range []string{"phase", "change_class", "review_budget", "allowed_paths", "dependencies", "adr"} {
			if len(assignment.Traceability[field]) == 0 && field == "adr" && assignment.ADR == "" {
				continue
			}
			subtask.Traceability[field] = assignment.Traceability[field]
		}
	}
	return nil
}
