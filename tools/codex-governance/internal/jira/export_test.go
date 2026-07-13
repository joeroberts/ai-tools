package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
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

func TestCreatePlanPostsStoryAndSubtasks(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method != http.MethodPost || request.URL.Path != "/rest/api/3/issue" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if username, password, ok := request.BasicAuth(); !ok || username != "owner@example.test" || password != "test-token" {
			t.Fatalf("basic auth = %q %q %t", username, password, ok)
		}
		_, _ = io.ReadAll(request.Body)
		response.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = response.Write([]byte(`{"key":"CG-1","self":"ignored"}`))
			return
		}
		_, _ = response.Write([]byte(`{"key":"CG-2","self":"ignored"}`))
	}))
	defer server.Close()

	plan := ticketplan.Plan{Story: ticketplan.Story{Summary: "Story", Description: "Story description"}, Subtasks: []ticketplan.Subtask{{Summary: "Subtask", Scope: "bounded", NonGoals: []string{"none"}, AcceptanceCriteria: []string{"done"}, ValidationPlan: []string{"test"}, ADR: "No ADR needed: follows current design"}}}
	story, subtasks, err := (CreateClient{BaseURL: server.URL, Email: "owner@example.test", Token: "test-token"}).CreatePlan("CG", plan)
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if requests != 2 || story.Key != "CG-1" || len(subtasks) != 1 || subtasks[0].Key != "CG-2" {
		t.Fatalf("CreatePlan() = %#v, %#v; requests=%d", story, subtasks, requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
