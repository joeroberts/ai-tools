package jira

import (
	"path/filepath"
	"testing"
)

func TestLoadOfflineExport(t *testing.T) {
	export, err := LoadOfflineExport(filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"))
	if err != nil {
		t.Fatalf("LoadOfflineExport() error = %v", err)
	}
	if export.Story.Key != "CG-1" {
		t.Fatalf("story key = %q", export.Story.Key)
	}
}
