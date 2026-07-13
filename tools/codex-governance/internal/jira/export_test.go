package jira

import (
	"net/http"
	"path/filepath"
	"testing"

	"codex-governance/internal/ticketplan"
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

func TestCreatePlanIsUnavailableBeforePhase4WithoutSendingRequest(t *testing.T) {
	called := false
	client := CreateClient{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}}
	_, _, err := client.CreatePlan("CG", ticketplan.Plan{})
	if err == nil || err.Error() != phase4PendingMessage {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if called {
		t.Fatal("CreatePlan sent an HTTP request before Phase 4 approval")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
