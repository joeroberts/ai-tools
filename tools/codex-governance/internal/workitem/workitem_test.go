package workitem

import (
	"path/filepath"
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
