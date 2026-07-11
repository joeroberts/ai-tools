package roadmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Roadmap struct {
	ID     string  `yaml:"id" json:"id"`
	Title  string  `yaml:"title" json:"title"`
	Status string  `yaml:"status" json:"status"`
	Phases []Phase `yaml:"phases" json:"phases"`
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
	if r.ID == "" || r.Title == "" || !oneOf(r.Status, "proposed", "in-progress", "blocked", "complete") {
		issues = append(issues, "roadmap identity or status is invalid")
	}
	active := 0
	previousID := 0
	for _, phase := range r.Phases {
		if phase.ID <= previousID || phase.Name == "" || !oneOf(phase.Status, "pending-approval", "in-progress", "blocked", "complete") {
			issues = append(issues, fmt.Sprintf("phase %d is invalid", phase.ID))
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
	return issues
}

func (r Roadmap) Render(format string) (string, error) {
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
