package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	issueKeyRE  = regexp.MustCompile(`^[A-Z][A-Z0-9]+-[1-9][0-9]*$`)
	commitSHARE = regexp.MustCompile(`^[0-9a-fA-F]{40,64}$`)
)

// WorkUpdate is a factual Jira work-record event. Commit and blocker events
// have distinct required fields and are validated before a comment is sent.
type WorkUpdate struct {
	Issue          string
	Kind           string
	Commit         string
	Scope          string
	Checks         []string
	Evidence       []string
	Blocker        string
	Impact         string
	DecisionNeeded string
	NextAction     string
}

func (u WorkUpdate) Validate() error {
	if !issueKeyRE.MatchString(u.Issue) {
		return fmt.Errorf("invalid Jira issue key %q", u.Issue)
	}
	switch u.Kind {
	case "commit":
		if !commitSHARE.MatchString(u.Commit) || strings.TrimSpace(u.Scope) == "" || len(nonBlank(u.Checks)) == 0 || len(nonBlank(u.Evidence)) == 0 {
			return fmt.Errorf("commit update requires a full commit SHA, --scope, --check, and --evidence")
		}
	case "blocker":
		if strings.TrimSpace(u.Blocker) == "" || strings.TrimSpace(u.Impact) == "" || strings.TrimSpace(u.DecisionNeeded) == "" || strings.TrimSpace(u.NextAction) == "" {
			return fmt.Errorf("blocker update requires --blocker, --impact, --decision-needed, and --next-action")
		}
	default:
		return fmt.Errorf("work update kind must be commit or blocker")
	}
	return nil
}

func (u WorkUpdate) Comment() string {
	if u.Kind == "commit" {
		return "Work record: commit\n\nCommit: " + u.Commit + "\nCompleted scope: " + u.Scope + "\nChecks:\n- " + strings.Join(nonBlank(u.Checks), "\n- ") + "\nEvidence:\n- " + strings.Join(nonBlank(u.Evidence), "\n- ")
	}
	return "Work record: blocker\n\nBlocker: " + u.Blocker + "\nImpact: " + u.Impact + "\nOwner decision needed: " + u.DecisionNeeded + "\nNext action: " + u.NextAction
}

type WorkClient struct {
	BaseURL, Email, Token string
	HTTPClient            *http.Client
}

type Comment struct {
	ID   string
	Body string
}

func (c WorkClient) AddComment(issue, text string) (Comment, error) {
	client := c.client()
	body, err := json.Marshal(map[string]any{"body": adf(text)})
	if err != nil {
		return Comment{}, fmt.Errorf("serialize Jira comment: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+issue+"/comment", bytes.NewReader(body))
	if err != nil {
		return Comment{}, fmt.Errorf("create Jira comment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return Comment{}, fmt.Errorf("post Jira comment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Comment{}, fmt.Errorf("Jira comment create: HTTP %d", resp.StatusCode)
	}
	comment, err := decodeComment(resp)
	if err != nil {
		return Comment{}, err
	}
	if comment.ID == "" {
		return Comment{}, fmt.Errorf("Jira comment create response is missing comment ID")
	}
	return comment, nil
}

func (c WorkClient) ReadComment(issue, id string) (Comment, error) {
	client := c.client()
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+issue+"/comment/"+id, nil)
	if err != nil {
		return Comment{}, fmt.Errorf("create Jira comment read request: %w", err)
	}
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return Comment{}, fmt.Errorf("read Jira comment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Comment{}, fmt.Errorf("Jira comment read: HTTP %d", resp.StatusCode)
	}
	comment, err := decodeComment(resp)
	if err != nil {
		return Comment{}, err
	}
	return comment, nil
}

func (c WorkClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func decodeComment(resp *http.Response) (Comment, error) {
	var payload struct {
		ID   string         `json:"id"`
		Body map[string]any `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Comment{}, fmt.Errorf("decode Jira comment response: %w", err)
	}
	return Comment{ID: payload.ID, Body: adfText(payload.Body)}, nil
}

func nonBlank(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
