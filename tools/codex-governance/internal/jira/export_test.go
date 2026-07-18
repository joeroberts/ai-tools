package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	if export.Story.Status != "To Do" || export.Subtask.Status != "In Progress" {
		t.Fatalf("source statuses = %q, %q", export.Story.Status, export.Subtask.Status)
	}
}

func TestReadIssueCapturesSourceStatus(t *testing.T) {
	client := ReadClient{
		BaseURL: "https://jira.example.test",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Query().Get("fields") != "description,status,updated" {
				t.Fatalf("requested fields = %q", request.URL.Query().Get("fields"))
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"key":"CG-2","fields":{"updated":"2026-07-11T10:00:00Z","status":{"name":"In Progress"},"description":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Implement validation."}]}]}}}`)), Header: make(http.Header)}, nil
		})},
	}
	issue, err := client.ReadIssue("CG-2")
	if err != nil {
		t.Fatalf("ReadIssue() error = %v", err)
	}
	if issue.Status != "In Progress" {
		t.Fatalf("ReadIssue() status = %q", issue.Status)
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

func TestResumePlanCreatesOnlyRemainingSubtasksWithConfiguredType(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		var payload struct {
			Fields struct {
				IssueType struct {
					Name string `json:"name"`
				} `json:"issuetype"`
				Parent struct {
					Key string `json:"key"`
				} `json:"parent"`
			} `json:"fields"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Fields.IssueType.Name != "Subtask" || payload.Fields.Parent.Key != "CG-1" {
			t.Fatalf("resume payload = %#v", payload.Fields)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"key":"CG-3","self":"ignored"}`))
	}))
	defer server.Close()

	plan := ticketplan.Plan{Subtasks: []ticketplan.Subtask{
		{Summary: "First"},
		{Summary: "Second", Scope: "bounded", NonGoals: []string{"none"}, AcceptanceCriteria: []string{"done"}, ValidationPlan: []string{"test"}, ADR: "No ADR needed: follows current design"},
	}}
	created, err := (CreateClient{BaseURL: server.URL}).ResumePlan("CG", plan, CreatedIssue{Key: "CG-1"}, []CreatedIssue{{Key: "CG-2"}})
	if err != nil {
		t.Fatalf("ResumePlan() error = %v", err)
	}
	if requests != 1 || len(created) != 2 || created[1].Key != "CG-3" {
		t.Fatalf("ResumePlan() = %#v; requests=%d", created, requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
