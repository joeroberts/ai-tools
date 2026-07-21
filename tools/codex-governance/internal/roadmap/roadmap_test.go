package roadmap

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndRender(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "roadmaps", "go-cli-migration.yaml")
	roadmap, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if issues := roadmap.Check(); len(issues) != 0 {
		t.Fatalf("Check() issues = %v", issues)
	}
	output, err := roadmap.Render("table")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Adoption And Synchronization") {
		t.Fatalf("table output = %q", output)
	}
}

func TestCheckAggregatePhaseStates(t *testing.T) {
	tests := []struct {
		name, status string
		phases       []string
		wantIssue    string
	}{
		{"proposed accepts pending approval", "proposed", []string{"pending-approval", "pending-approval"}, ""},
		{"proposed rejects active phase", "proposed", []string{"pending-approval", "in-progress"}, "requires every phase to be pending-approval"},
		{"in progress accepts incomplete phase", "in-progress", []string{"complete", "in-progress"}, ""},
		{"in progress rejects all complete", "in-progress", []string{"complete", "complete"}, "requires at least one incomplete phase"},
		{"in progress rejects blocked phase", "in-progress", []string{"in-progress", "blocked"}, "requires at least one incomplete phase and no blocked phase"},
		{"blocked accepts blocked phase", "blocked", []string{"complete", "blocked"}, ""},
		{"blocked rejects no blocked phase", "blocked", []string{"complete", "in-progress"}, "requires at least one blocked phase"},
		{"complete accepts all complete", "complete", []string{"complete", "complete"}, ""},
		{"complete rejects incomplete phase", "complete", []string{"complete", "in-progress"}, "requires every phase to be complete"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			roadmap := Roadmap{ID: "test", Title: "Test", Status: test.status}
			for index, status := range test.phases {
				phase := Phase{ID: index + 1, Name: fmt.Sprintf("Phase %d", index+1), Status: status}
				if status == "complete" {
					phase.ApprovedBy, phase.CompletedAt, phase.Evidence = "owner", "2026-07-21", []string{"test"}
				}
				roadmap.Phases = append(roadmap.Phases, phase)
			}
			issues := strings.Join(roadmap.Check(), "\n")
			if test.wantIssue == "" && issues != "" {
				t.Fatalf("Check() issues = %q", issues)
			}
			if test.wantIssue != "" && !strings.Contains(issues, test.wantIssue) {
				t.Fatalf("Check() issues = %q, want %q", issues, test.wantIssue)
			}
		})
	}
}

func TestRenderRejectsInvalidRoadmap(t *testing.T) {
	roadmap := Roadmap{ID: "test", Title: "Test", Status: "complete", Phases: []Phase{{ID: 1, Name: "Phase 1", Status: "in-progress"}}}
	for _, format := range []string{"table", "markdown", "json"} {
		if _, err := roadmap.Render(format); err == nil || !strings.Contains(err.Error(), "roadmap status complete") {
			t.Fatalf("Render(%q) error = %v", format, err)
		}
	}
}
