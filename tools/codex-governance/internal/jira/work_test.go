package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkUpdateCommitRendersValidatedComment(t *testing.T) {
	update := WorkUpdate{
		Issue:    "REK-5",
		Kind:     "commit",
		Commit:   strings.Repeat("a", 40),
		Scope:    "Add work records",
		Checks:   []string{"go test ./internal/jira"},
		Evidence: []string{"/private/evidence.json"},
	}
	if err := update.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	comment := update.Comment()
	for _, want := range []string{"Commit: " + update.Commit, "Completed scope: Add work records", "- go test ./internal/jira", "- /private/evidence.json"} {
		if !strings.Contains(comment, want) {
			t.Fatalf("Comment() = %q, missing %q", comment, want)
		}
	}
}

func TestWorkUpdateRejectsIncompleteBlocker(t *testing.T) {
	err := (WorkUpdate{Issue: "REK-5", Kind: "blocker", Blocker: "Jira unavailable"}).Validate()
	if err == nil || !strings.Contains(err.Error(), "--impact") {
		t.Fatalf("Validate() error = %v, want required blocker fields", err)
	}
}

func TestWorkClientPostsAndReadsBackComment(t *testing.T) {
	const text = "Work record: blocker\n\nBlocker: test"
	var posted string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, password, ok := r.BasicAuth(); !ok || user != "owner@example.test" || password != "test-token" {
			t.Fatalf("BasicAuth() = %q, %q, %t", user, password, ok)
		}
		switch r.URL.Path {
		case "/rest/api/3/issue/REK-5/comment":
			if r.Method != http.MethodPost {
				t.Fatalf("POST path method = %s", r.Method)
			}
			var request struct {
				Body map[string]any `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			posted = adfText(request.Body)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "body": adf(text)})
		case "/rest/api/3/issue/REK-5/comment/10001":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "10001", "body": adf(text)})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := WorkClient{BaseURL: server.URL, Email: "owner@example.test", Token: "test-token"}
	created, err := client.AddComment("REK-5", text)
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if created.ID != "10001" || posted != text {
		t.Fatalf("AddComment() = %#v, posted = %q", created, posted)
	}
	readBack, err := client.ReadComment("REK-5", created.ID)
	if err != nil || readBack.Body != text {
		t.Fatalf("ReadComment() = %#v, %v", readBack, err)
	}
}

func TestWorkClientRejectsUnreadableReadBack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorMessages":["unavailable"]}`))
	}))
	defer server.Close()
	_, err := (WorkClient{BaseURL: server.URL}).ReadComment("REK-5", "10001")
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("ReadComment() error = %v", err)
	}
}
