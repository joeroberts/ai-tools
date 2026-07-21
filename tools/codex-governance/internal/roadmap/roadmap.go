package roadmap

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Roadmap struct {
	ID     string  `yaml:"id" json:"id"`
	Title  string  `yaml:"title" json:"title"`
	Status string  `yaml:"status" json:"status"`
	Phases []Phase `yaml:"phases" json:"phases"`
}

type Transition struct {
	RoadmapID   string
	Phase       int
	Action      string
	PriorDigest string
	EvidenceID  string
}

// PreviewTransition validates a bound transition and returns the resulting
// roadmap and digest without writing the roadmap or any external system.
func PreviewTransition(current Roadmap, transition Transition, usedEvidence map[string]bool) (Roadmap, string, error) {
	if issues := current.Check(); len(issues) != 0 {
		return Roadmap{}, "", fmt.Errorf("invalid roadmap: %s", strings.Join(issues, "; "))
	}
	prior, err := Digest(current)
	if err != nil {
		return Roadmap{}, "", err
	}
	if transition.RoadmapID != current.ID || transition.PriorDigest != prior {
		return Roadmap{}, "", fmt.Errorf("roadmap transition binding is stale or mismatched")
	}
	if transition.EvidenceID == "" || usedEvidence == nil || usedEvidence[transition.EvidenceID] {
		return Roadmap{}, "", fmt.Errorf("roadmap transition evidence is missing or replayed")
	}
	result := current
	result.Phases = make([]Phase, len(current.Phases))
	for index, phase := range current.Phases {
		result.Phases[index] = phase
		result.Phases[index].Evidence = append([]string(nil), phase.Evidence...)
	}
	found := false
	for index := range result.Phases {
		phase := &result.Phases[index]
		if phase.ID != transition.Phase {
			continue
		}
		found = true
		if !validTransition(phase.Status, transition.Action) {
			return Roadmap{}, "", fmt.Errorf("roadmap phase %d cannot %s from %s", phase.ID, transition.Action, phase.Status)
		}
		switch transition.Action {
		case "start", "resume":
			phase.Status = "in-progress"
		case "block":
			phase.Status = "blocked"
		case "complete":
			phase.Status, phase.Evidence = "complete", append(phase.Evidence, transition.EvidenceID)
			if phase.ApprovedBy == "" {
				phase.ApprovedBy = "transition-evidence"
			}
			if phase.CompletedAt == "" {
				phase.CompletedAt = "evidence-bound"
			}
		}
	}
	if !found {
		return Roadmap{}, "", fmt.Errorf("roadmap phase %d is not present", transition.Phase)
	}
	result.Status = aggregateStatus(result)
	if issues := result.Check(); len(issues) != 0 {
		return Roadmap{}, "", fmt.Errorf("resulting roadmap is invalid: %s", strings.Join(issues, "; "))
	}
	digest, err := Digest(result)
	if err != nil {
		return Roadmap{}, "", err
	}
	usedEvidence[transition.EvidenceID] = true
	return result, digest, nil
}

func Digest(roadmap Roadmap) (string, error) {
	data, err := json.Marshal(roadmap)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum), nil
}

func validTransition(status, action string) bool {
	return (action == "start" && status == "pending-approval") || (action == "block" && status == "in-progress") || (action == "resume" && status == "blocked") || (action == "complete" && status == "in-progress")
}

func aggregateStatus(roadmap Roadmap) string {
	allComplete, blocked := true, false
	for _, phase := range roadmap.Phases {
		allComplete = allComplete && phase.Status == "complete"
		blocked = blocked || phase.Status == "blocked"
	}
	if allComplete {
		return "complete"
	}
	if blocked {
		return "blocked"
	}
	return "in-progress"
}

// ValidateImpact checks that an enforced work-item declaration still names the
// configured roadmap and a phase in the state required for entry.
func ValidateImpact(repoRoot, canonicalPath, roadmapID, phase, transition string) error {
	if canonicalPath == "" || roadmapID == "" || phase == "" {
		return fmt.Errorf("roadmap impact declaration is incomplete")
	}
	value, err := strconv.Atoi(phase)
	if err != nil || value < 1 {
		return fmt.Errorf("roadmap impact phase must be a positive integer")
	}
	loaded, err := Load(filepath.Join(repoRoot, filepath.FromSlash(canonicalPath)))
	if err != nil {
		return fmt.Errorf("load configured roadmap: %w", err)
	}
	if issues := loaded.Check(); len(issues) != 0 {
		return fmt.Errorf("configured roadmap is invalid: %s", strings.Join(issues, "; "))
	}
	if loaded.ID != roadmapID {
		return fmt.Errorf("roadmap impact identity %q does not match configured roadmap %q", roadmapID, loaded.ID)
	}
	for _, candidate := range loaded.Phases {
		if candidate.ID != value {
			continue
		}
		if transition == "start" || transition == "resume" {
			if candidate.Status != "in-progress" {
				return fmt.Errorf("roadmap phase %d must be in-progress for %s, got %s", value, transition, candidate.Status)
			}
		}
		return nil
	}
	return fmt.Errorf("roadmap impact phase %d is not present", value)
}

type Phase struct {
	ID          int      `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Status      string   `yaml:"status" json:"status"`
	ApprovedBy  string   `yaml:"approved_by" json:"approved_by,omitempty"`
	CompletedAt string   `yaml:"completed_at" json:"completed_at,omitempty"`
	Evidence    []string `yaml:"evidence" json:"evidence"`
}

func Load(path string) (Roadmap, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Roadmap{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var roadmap Roadmap
	if err := decoder.Decode(&roadmap); err != nil {
		return Roadmap{}, fmt.Errorf("parse roadmap: %w", err)
	}
	return roadmap, nil
}

func (r Roadmap) Check() []string {
	var issues []string
	validStatus := oneOf(r.Status, "proposed", "in-progress", "blocked", "complete")
	if r.ID == "" || r.Title == "" || !validStatus {
		issues = append(issues, "roadmap identity or status is invalid")
	}
	active := 0
	previousID := 0
	validPhases := true
	for _, phase := range r.Phases {
		if phase.ID <= previousID || phase.Name == "" || !oneOf(phase.Status, "pending-approval", "in-progress", "blocked", "complete") {
			issues = append(issues, fmt.Sprintf("phase %d is invalid", phase.ID))
			validPhases = false
		}
		previousID = phase.ID
		if phase.Status == "in-progress" {
			active++
		}
		if phase.Status == "complete" && (phase.ApprovedBy == "" || phase.CompletedAt == "" || len(phase.Evidence) == 0) {
			issues = append(issues, fmt.Sprintf("phase %d is complete without approval, completion date, or evidence", phase.ID))
		}
	}
	if active > 1 {
		issues = append(issues, "only one phase may be in progress")
	}
	if validStatus && validPhases {
		if issue := r.aggregatePhaseIssue(); issue != "" {
			issues = append(issues, issue)
		}
	}
	return issues
}

func (r Roadmap) aggregatePhaseIssue() string {
	statuses := make([]string, 0, len(r.Phases))
	hasBlocked, hasIncomplete, allPendingApproval, allComplete := false, false, true, true
	for _, phase := range r.Phases {
		statuses = append(statuses, fmt.Sprintf("%d=%s", phase.ID, phase.Status))
		hasBlocked = hasBlocked || phase.Status == "blocked"
		hasIncomplete = hasIncomplete || phase.Status != "complete"
		allPendingApproval = allPendingApproval && phase.Status == "pending-approval"
		allComplete = allComplete && phase.Status == "complete"
	}
	found := strings.Join(statuses, ", ")
	switch r.Status {
	case "proposed":
		if !allPendingApproval {
			return fmt.Sprintf("roadmap status proposed requires every phase to be pending-approval; found phase statuses: %s; set every phase to pending-approval or update the roadmap status", found)
		}
	case "in-progress":
		if !hasIncomplete || hasBlocked {
			return fmt.Sprintf("roadmap status in-progress requires at least one incomplete phase and no blocked phase; found phase statuses: %s; clear blocked phases or update the roadmap status", found)
		}
	case "blocked":
		if !hasBlocked {
			return fmt.Sprintf("roadmap status blocked requires at least one blocked phase; found phase statuses: %s; mark the blocked phase or update the roadmap status", found)
		}
	case "complete":
		if !allComplete {
			return fmt.Sprintf("roadmap status complete requires every phase to be complete; found phase statuses: %s; complete every phase or update the roadmap status", found)
		}
	}
	return ""
}

func (r Roadmap) Render(format string) (string, error) {
	if issues := r.Check(); len(issues) != 0 {
		return "", fmt.Errorf("invalid roadmap: %s", strings.Join(issues, "; "))
	}
	switch format {
	case "table":
		var builder strings.Builder
		fmt.Fprintln(&builder, "PHASE  STATUS            NAME")
		for _, phase := range r.Phases {
			fmt.Fprintf(&builder, "%5d  %-16s %s\n", phase.ID, phase.Status, phase.Name)
		}
		return builder.String(), nil
	case "markdown":
		var builder strings.Builder
		fmt.Fprintln(&builder, "| Phase | Status | Name |")
		fmt.Fprintln(&builder, "| --- | --- | --- |")
		for _, phase := range r.Phases {
			fmt.Fprintf(&builder, "| %d | %s | %s |\n", phase.ID, phase.Status, phase.Name)
		}
		return builder.String(), nil
	case "json":
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	default:
		return "", fmt.Errorf("unsupported roadmap format %q", format)
	}
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
