package workitem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFixture(t *testing.T) {
	item, err := Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if issues := item.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %v", issues)
	}
}

func TestLoadNormalizesLegacySourceProvider(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy-work-item.json")
	legacy := strings.Replace(string(data), `"mode": "offline-export"`, `"provider": "offline-export"`, 1)
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	item, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if item.Source.Mode != "offline-export" || len(item.Validate()) != 0 {
		t.Fatalf("legacy source = %#v, issues=%v", item.Source, item.Validate())
	}
}

func TestRoadmapImpactRequiresAnExplicitDeclaration(t *testing.T) {
	item, err := Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.RoadmapImpact = RoadmapImpact{}
	if issues := strings.Join(item.Validate(), "\n"); !strings.Contains(issues, "roadmap_impact declaration is required") {
		t.Fatalf("missing declaration issues = %q", issues)
	}
}

func TestRoadmapImpactRejectsUnboundedNotApplicableReason(t *testing.T) {
	item, err := Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.RoadmapImpact = RoadmapImpact{Mode: "not-applicable", Reason: "short"}
	if issues := strings.Join(item.Validate(), "\n"); !strings.Contains(issues, "bounded reason") {
		t.Fatalf("unbounded reason issues = %q", issues)
	}
}

func TestRoadmapImpactRejectsEscapingCanonicalPath(t *testing.T) {
	item, err := Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.RoadmapImpact = RoadmapImpact{Mode: "required", RoadmapID: "repo-roadmap", CanonicalPath: "../roadmap.yaml", Phase: "phase-1", Transition: "start"}
	if issues := strings.Join(item.Validate(), "\n"); !strings.Contains(issues, "incomplete or invalid") {
		t.Fatalf("escaping path issues = %q", issues)
	}
}
