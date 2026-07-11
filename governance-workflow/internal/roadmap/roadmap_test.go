package roadmap

import (
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
