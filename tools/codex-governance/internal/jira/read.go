package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ReadClient is restricted to retrieving the Jira fields needed for a signed
// offline export. It has no methods that mutate Jira state.
type ReadClient struct {
	BaseURL, Email, Token string
	HTTPClient            *http.Client
}

func (c ReadClient) ReadIssue(key string) (Issue, error) {
	if key == "" {
		return Issue{}, fmt.Errorf("Jira issue key is required")
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+key+"?fields=description,updated", nil)
	if err != nil {
		return Issue{}, err
	}
	req.SetBasicAuth(c.Email, c.Token)
	response, err := client.Do(req)
	if err != nil {
		return Issue{}, err
	}
	defer response.Body.Close()
	var payload struct {
		Key    string `json:"key"`
		Fields struct {
			Updated     string         `json:"updated"`
			Description map[string]any `json:"description"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Issue{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Issue{}, fmt.Errorf("Jira read %s: HTTP %d", key, response.StatusCode)
	}
	updated, err := parseJiraTimestamp(payload.Fields.Updated)
	if err != nil {
		return Issue{}, fmt.Errorf("parse Jira issue update time: %w", err)
	}
	description := adfText(payload.Fields.Description)
	if payload.Key == "" || description == "" {
		return Issue{}, fmt.Errorf("Jira issue %s is missing required export fields", key)
	}
	return Issue{Key: payload.Key, URL: strings.TrimRight(c.BaseURL, "/") + "/browse/" + payload.Key, UpdatedAt: updated.UTC().Format(time.RFC3339), Description: description, AcceptanceCriteria: description}, nil
}

func parseJiraTimestamp(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000-0700", "2006-01-02T15:04:05-0700"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported Jira timestamp %q", value)
}

func adfText(value any) string {
	var parts []string
	var visit func(any)
	visit = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			if text, ok := typed["text"].(string); ok {
				parts = append(parts, text)
			}
			if content, ok := typed["content"].([]any); ok {
				for _, child := range content {
					visit(child)
				}
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
