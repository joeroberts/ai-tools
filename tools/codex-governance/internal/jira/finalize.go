package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type PullRequest struct {
	URL, MergeCommit string
	Merged           bool
}

type PullRequestReader interface {
	ReadMerged(reference string) (PullRequest, error)
}

type GitHubCLI struct{}

func (GitHubCLI) ReadMerged(reference string) (PullRequest, error) {
	if strings.TrimSpace(reference) == "" {
		return PullRequest{}, fmt.Errorf("pull request reference is required")
	}
	output, err := exec.Command("gh", "pr", "view", reference, "--json", "url,state,mergedAt,mergeCommit").Output()
	if err != nil {
		return PullRequest{}, fmt.Errorf("read pull request %q: %w", reference, err)
	}
	var payload struct {
		URL         string `json:"url"`
		State       string `json:"state"`
		MergedAt    string `json:"mergedAt"`
		MergeCommit struct {
			OID string `json:"oid"`
		} `json:"mergeCommit"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return PullRequest{}, fmt.Errorf("parse pull request %q: %w", reference, err)
	}
	if payload.URL == "" || payload.State != "MERGED" || payload.MergedAt == "" || payload.MergeCommit.OID == "" {
		return PullRequest{}, fmt.Errorf("pull request %q is not merged with a merge commit", reference)
	}
	return PullRequest{URL: payload.URL, MergeCommit: payload.MergeCommit.OID, Merged: true}, nil
}

type FinalizationClient struct {
	BaseURL, Email, Token string
	HTTPClient            *http.Client
}

type FinalizationPlan struct {
	Subtask, Story      IssueState
	SubtaskTransitionID string
	StoryTransitionID   string
	Comment             string
}

type IssueState struct {
	Key, Parent, Status, Resolution string
	Done                            bool
	Children                        []IssueState
}

func (c FinalizationClient) Plan(subtask string, pr PullRequest) (FinalizationPlan, error) {
	if !issueKeyRE.MatchString(subtask) || !pr.Merged || pr.URL == "" || pr.MergeCommit == "" {
		return FinalizationPlan{}, fmt.Errorf("finalization requires a merged pull request and valid subtask key")
	}
	child, err := c.readIssue(subtask)
	if err != nil {
		return FinalizationPlan{}, err
	}
	if child.Parent == "" {
		return FinalizationPlan{}, fmt.Errorf("subtask %s has no parent Story", subtask)
	}
	if child.Done {
		return FinalizationPlan{}, fmt.Errorf("subtask %s is already complete", subtask)
	}
	story, err := c.readIssue(child.Parent)
	if err != nil {
		return FinalizationPlan{}, err
	}
	if story.Done {
		return FinalizationPlan{}, fmt.Errorf("parent Story %s is already complete", story.Key)
	}
	for _, sibling := range story.Children {
		if sibling.Key != child.Key && !sibling.Done {
			return FinalizationPlan{}, fmt.Errorf("parent Story %s has incomplete child %s", story.Key, sibling.Key)
		}
	}
	childTransition, err := c.doneTransition(child.Key)
	if err != nil {
		return FinalizationPlan{}, err
	}
	storyTransition, err := c.doneTransition(story.Key)
	if err != nil {
		return FinalizationPlan{}, err
	}
	return FinalizationPlan{Subtask: child, Story: story, SubtaskTransitionID: childTransition, StoryTransitionID: storyTransition, Comment: "Work record: merged pull request\n\nPull request: " + pr.URL + "\nMerged commit: " + pr.MergeCommit}, nil
}

func (c FinalizationClient) Transition(issue, transitionID string) error {
	body, _ := json.Marshal(map[string]any{"transition": map[string]string{"id": transitionID}})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+issue+"/transitions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Jira transition request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := c.client().Do(req)
	if err != nil {
		return fmt.Errorf("transition Jira issue %s: %w", issue, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("transition Jira issue %s: HTTP %d", issue, resp.StatusCode)
	}
	return nil
}

func (c FinalizationClient) VerifyClosed(issue string) error {
	state, err := c.readIssue(issue)
	if err != nil {
		return err
	}
	if !state.Done || strings.TrimSpace(state.Resolution) == "" {
		return fmt.Errorf("Jira issue %s is not complete with a resolution after transition", issue)
	}
	return nil
}

func (c FinalizationClient) readIssue(key string) (IssueState, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+key+"?fields=parent,status,resolution,subtasks", nil)
	if err != nil {
		return IssueState{}, err
	}
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := c.client().Do(req)
	if err != nil {
		return IssueState{}, fmt.Errorf("read Jira issue %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return IssueState{}, fmt.Errorf("read Jira issue %s: HTTP %d", key, resp.StatusCode)
	}
	var payload struct {
		Key    string `json:"key"`
		Fields struct {
			Parent struct {
				Key string `json:"key"`
			} `json:"parent"`
			Status struct {
				Name     string `json:"name"`
				Category struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"status"`
			Resolution *struct {
				Name string `json:"name"`
			} `json:"resolution"`
			Subtasks []struct {
				Key    string `json:"key"`
				Fields struct {
					Status struct {
						Name     string `json:"name"`
						Category struct {
							Key string `json:"key"`
						} `json:"statusCategory"`
					} `json:"status"`
					Resolution *struct {
						Name string `json:"name"`
					} `json:"resolution"`
				} `json:"fields"`
			} `json:"subtasks"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return IssueState{}, fmt.Errorf("decode Jira issue %s: %w", key, err)
	}
	state := IssueState{Key: payload.Key, Parent: payload.Fields.Parent.Key, Status: payload.Fields.Status.Name, Done: payload.Fields.Status.Category.Key == "done"}
	if payload.Fields.Resolution != nil {
		state.Resolution = payload.Fields.Resolution.Name
	}
	for _, child := range payload.Fields.Subtasks {
		item := IssueState{Key: child.Key, Status: child.Fields.Status.Name, Done: child.Fields.Status.Category.Key == "done"}
		if child.Fields.Resolution != nil {
			item.Resolution = child.Fields.Resolution.Name
		}
		state.Children = append(state.Children, item)
	}
	if state.Key == "" {
		return IssueState{}, fmt.Errorf("Jira issue %s response is incomplete", key)
	}
	return state, nil
}

func (c FinalizationClient) doneTransition(issue string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/rest/api/3/issue/"+issue+"/transitions", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.Email, c.Token)
	resp, err := c.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("read Jira transitions for %s: %w", issue, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("read Jira transitions for %s: HTTP %d", issue, resp.StatusCode)
	}
	var payload struct {
		Transitions []struct {
			ID string `json:"id"`
			To struct {
				Category struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Jira transitions for %s: %w", issue, err)
	}
	for _, transition := range payload.Transitions {
		if transition.To.Category.Key == "done" && transition.ID != "" {
			return transition.ID, nil
		}
	}
	return "", fmt.Errorf("Jira issue %s has no available transition to a done status", issue)
}

func (c FinalizationClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return (&WorkClient{}).client()
}
