package jira

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinalizationPlanRejectsIncompleteSibling(t *testing.T) {
	server := finalizationServer(t, true)
	defer server.Close()
	_, err := (FinalizationClient{BaseURL: server.URL}).Plan("REK-6", PullRequest{URL: "https://example.test/pr/1", MergeCommit: strings.Repeat("a", 40), Merged: true})
	if err == nil || !strings.Contains(err.Error(), "incomplete child REK-7") {
		t.Fatalf("Plan() error = %v", err)
	}
}

func TestFinalizationPlanFindsOrderedDoneTransitions(t *testing.T) {
	server := finalizationServer(t, false)
	defer server.Close()
	plan, err := (FinalizationClient{BaseURL: server.URL}).Plan("REK-6", PullRequest{URL: "https://example.test/pr/1", MergeCommit: strings.Repeat("a", 40), Merged: true})
	if err != nil || plan.SubtaskTransitionID != "11" || plan.StoryTransitionID != "12" || !strings.Contains(plan.Comment, "Merged commit") {
		t.Fatalf("Plan() = %#v, %v", plan, err)
	}
}

func TestFinalizationVerifyClosedRequiresResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"key":"REK-6","fields":{"status":{"name":"Done","statusCategory":{"key":"done"}},"resolution":null,"subtasks":[]}}`))
	}))
	defer server.Close()
	err := (FinalizationClient{BaseURL: server.URL}).VerifyClosed("REK-6")
	if err == nil || !strings.Contains(err.Error(), "resolution") {
		t.Fatalf("VerifyClosed() error = %v", err)
	}
}

func finalizationServer(t *testing.T, incomplete bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issue/REK-6":
			_, _ = w.Write([]byte(`{"key":"REK-6","fields":{"parent":{"key":"REK-4"},"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},"resolution":null,"subtasks":[]}}`))
		case "/rest/api/3/issue/REK-4":
			status := "done"
			resolution := `{"name":"Done"}`
			if incomplete {
				status = "indeterminate"
				resolution = "null"
			}
			_, _ = w.Write([]byte(`{"key":"REK-4","fields":{"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},"resolution":null,"subtasks":[{"key":"REK-6","fields":{"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},"resolution":null}},{"key":"REK-7","fields":{"status":{"name":"Done","statusCategory":{"key":"` + status + `"}},"resolution":` + resolution + `}}]}}`))
		case "/rest/api/3/issue/REK-6/transitions":
			_, _ = w.Write([]byte(`{"transitions":[{"id":"11","to":{"statusCategory":{"key":"done"}}}]}`))
		case "/rest/api/3/issue/REK-4/transitions":
			_, _ = w.Write([]byte(`{"transitions":[{"id":"12","to":{"statusCategory":{"key":"done"}}}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
}
