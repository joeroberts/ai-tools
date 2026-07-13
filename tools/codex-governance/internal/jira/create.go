package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"codex-governance/internal/ticketplan"
)

type CreateClient struct {
	BaseURL, Email, Token string
	HTTPClient            *http.Client
}
type CreatedIssue struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

func (c CreateClient) CreatePlan(project string, plan ticketplan.Plan) (CreatedIssue, []CreatedIssue, error) {
	return c.createPlanAfterPhase4Approval(project, plan)
}

func (c CreateClient) createPlanAfterPhase4Approval(project string, plan ticketplan.Plan) (CreatedIssue, []CreatedIssue, error) {
	story, err := c.create(project, "Story", plan.Story.Summary, plan.Story.Description, "")
	if err != nil {
		return CreatedIssue{}, nil, err
	}
	subtasks := make([]CreatedIssue, 0, len(plan.Subtasks))
	for _, subtask := range plan.Subtasks {
		description := "## Scope\n" + subtask.Scope + "\n\n## Non-Goals\n- " + strings.Join(subtask.NonGoals, "\n- ") + "\n\n## Technical Acceptance Criteria\n- " + strings.Join(subtask.AcceptanceCriteria, "\n- ") + "\n\n## Validation Plan\n- " + strings.Join(subtask.ValidationPlan, "\n- ") + "\n\n## ADR\n" + subtask.ADR
		created, err := c.create(project, "Sub-task", subtask.Summary, description, story.Key)
		if err != nil {
			return story, subtasks, err
		}
		subtasks = append(subtasks, created)
	}
	return story, subtasks, nil
}

func (c CreateClient) create(project, issueType, summary, description, parent string) (CreatedIssue, error) {
	fields := map[string]any{"project": map[string]string{"key": project}, "summary": summary, "issuetype": map[string]string{"name": issueType}, "description": adf(description)}
	if parent != "" {
		fields["parent"] = map[string]string{"key": parent}
	}
	body, _ := json.Marshal(map[string]any{"fields": fields})
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue", bytes.NewReader(body))
	if err != nil {
		return CreatedIssue{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := client.Do(req)
	if err != nil {
		return CreatedIssue{}, err
	}
	defer resp.Body.Close()
	var result struct {
		Key    string            `json:"key"`
		Self   string            `json:"self"`
		Errors map[string]string `json:"errors"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CreatedIssue{}, fmt.Errorf("Jira create %s: HTTP %d: %v", issueType, resp.StatusCode, result.Errors)
	}
	return CreatedIssue{Key: result.Key, URL: strings.TrimRight(c.BaseURL, "/") + "/browse/" + result.Key}, nil
}

func adf(text string) map[string]any {
	return map[string]any{"type": "doc", "version": 1, "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]string{"type": "text", "text": text}}}}}
}
