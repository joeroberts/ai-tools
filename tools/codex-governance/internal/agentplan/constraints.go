package agentplan

import (
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
	AllowedPaths []string                `json:"allowed_paths"`
	ReviewBudget ticketplan.ReviewBudget `json:"review_budget"`
	Dependencies []string                `json:"dependencies"`
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

func PromoteConstraints(draftPath, outputPath string) error {
	data, err := os.ReadFile(filepath.Clean(draftPath))
	if err != nil {
		return err
	}
	var constraints Constraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		return fmt.Errorf("parse constraints draft: %w", err)
	}
	return WriteConstraints(outputPath, constraints)
}

func LoadConstraints(path string, sources ticketplan.Sources) (Constraints, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Constraints{}, err
	}
	var constraints Constraints
	if err := json.Unmarshal(data, &constraints); err != nil {
		return Constraints{}, fmt.Errorf("parse constraints: %w", err)
	}
	if constraints.FormatVersion != 1 || constraints.Sources != sources || len(constraints.Subtasks) == 0 {
		return Constraints{}, fmt.Errorf("constraints do not match current sources or contain no subtask assignments")
	}
	return constraints, nil
}

func ApplyConstraints(plan *ticketplan.Plan, constraints Constraints) error {
	if len(plan.Subtasks) != len(constraints.Subtasks) {
		return fmt.Errorf("manager decomposition has %d subtasks but constraints assign %d", len(plan.Subtasks), len(constraints.Subtasks))
	}
	for index := range plan.Subtasks {
		assignment := constraints.Subtasks[index]
		if assignment.ID == "" || len(assignment.AllowedPaths) == 0 || assignment.ReviewBudget.MaxChangedFiles < 1 || assignment.ReviewBudget.MaxChangedLines < 1 || len(assignment.ReviewBudget.Components) == 0 || assignment.Traceability == nil {
			return fmt.Errorf("constraints subtask %d is incomplete", index+1)
		}
		subtask := &plan.Subtasks[index]
		subtask.ID, subtask.AllowedPaths, subtask.ReviewBudget, subtask.Dependencies = assignment.ID, assignment.AllowedPaths, assignment.ReviewBudget, assignment.Dependencies
		for _, field := range []string{"review_budget", "allowed_paths", "dependencies"} {
			subtask.Traceability[field] = assignment.Traceability[field]
		}
	}
	return nil
}
