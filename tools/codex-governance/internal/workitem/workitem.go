package workitem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type Item struct {
	FormatVersion    int      `json:"format_version"`
	GovernanceStatus string   `json:"governance_status"`
	Profile          string   `json:"profile"`
	Source           Source   `json:"source"`
	Scope            Scope    `json:"scope"`
	GitRange         GitRange `json:"git_range"`
	Decision         Decision `json:"decision"`
	Links            Links    `json:"links"`
	Handoff          Handoff  `json:"handoff"`
}

type Source struct {
	Mode                            string `json:"mode"`
	Provider                        string `json:"provider,omitempty"`
	StoryKey                        string `json:"story_key"`
	StoryURL                        string `json:"story_url"`
	SubtaskKey                      string `json:"subtask_key"`
	SubtaskURL                      string `json:"subtask_url"`
	CapturedAt                      string `json:"captured_at"`
	StoryUpdatedAt                  string `json:"story_updated_at"`
	SubtaskUpdatedAt                string `json:"subtask_updated_at"`
	StoryDescriptionDigest          string `json:"story_description_digest"`
	StoryAcceptanceCriteriaDigest   string `json:"story_acceptance_criteria_digest"`
	SubtaskDescriptionDigest        string `json:"subtask_description_digest"`
	SubtaskAcceptanceCriteriaDigest string `json:"subtask_acceptance_criteria_digest"`
}

type Scope struct {
	Phase                       string             `json:"phase"`
	ChangeClass                 string             `json:"change_class"`
	AllowedPaths                []string           `json:"allowed_paths"`
	NonGoals                    []string           `json:"non_goals"`
	TechnicalAcceptanceCriteria []string           `json:"technical_acceptance_criteria"`
	ValidationPlan              []string           `json:"validation_plan"`
	FileClassification          FileClassification `json:"file_classification"`
	ReviewBudget                ReviewBudget       `json:"review_budget"`
}

type FileClassification struct {
	GeneratedPaths []string `json:"generated_paths"`
	LockfilePaths  []string `json:"lockfile_paths"`
}

type ReviewBudget struct {
	MaxChangedFiles int `json:"max_changed_files"`
	MaxChangedLines int `json:"max_changed_lines"`
	MaxComponents   int `json:"max_components"`
}

type GitRange struct {
	BaseSHA string `json:"base_sha"`
	HeadSHA string `json:"head_sha"`
}

type Decision struct {
	ADR                 string `json:"adr"`
	ADRPreflightPending bool   `json:"adr_preflight_pending,omitempty"`
}

type Links struct {
	PullRequest     *string          `json:"pull_request"`
	CIRun           *string          `json:"ci_run"`
	ReviewException *ReviewException `json:"review_exception"`
	AgentException  *AgentException  `json:"agent_exception"`
}

type AgentException struct {
	JiraReference string `json:"jira_reference"`
	Reason        string `json:"reason"`
	Approver      string `json:"approver"`
	ApprovedAt    string `json:"approved_at"`
}

type ReviewException struct {
	JiraReference   string   `json:"jira_reference"`
	Reason          string   `json:"reason"`
	Approver        string   `json:"approver"`
	ReviewPlan      []string `json:"review_plan"`
	ApprovedAt      string   `json:"approved_at"`
	ContainmentPlan *string  `json:"containment_plan"`
}

type Handoff struct {
	Status        string  `json:"status"`
	CompletedWork string  `json:"completed_work"`
	Blocker       *string `json:"blocker"`
	NextAction    string  `json:"next_action"`
}

func Load(path string) (Item, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Item{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var item Item
	if err := decoder.Decode(&item); err != nil {
		return Item{}, fmt.Errorf("parse work item: %w", err)
	}
	if item.Source.Mode == "" && item.Source.Provider != "" {
		item.Source.Mode = legacySourceMode(item.Source.Provider)
	}
	if item.Source.Provider != "" && item.Source.Mode != legacySourceMode(item.Source.Provider) {
		return Item{}, fmt.Errorf("parse work item: source.provider conflicts with source.mode")
	}
	return item, nil
}

func (i Item) Validate() []string {
	var issues []string
	if i.FormatVersion != 1 {
		issues = append(issues, "format_version must be 1")
	}
	if i.Profile != "generic" {
		issues = append(issues, "profile must be generic")
	}
	if !oneOf(i.GovernanceStatus, "ready", "in-implementation", "pending-review", "pending-verification", "source-drift-blocked", "blocked", "closed") {
		issues = append(issues, "governance_status is invalid")
	}
	if !oneOf(i.Source.Mode, "live-jira", "offline-export") || i.Source.StoryKey == "" || i.Source.SubtaskKey == "" || i.Source.StoryURL == "" || i.Source.SubtaskURL == "" {
		issues = append(issues, "source identity is incomplete")
	}
	for _, value := range []string{i.Source.CapturedAt, i.Source.StoryUpdatedAt, i.Source.SubtaskUpdatedAt} {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			issues = append(issues, "source timestamps must use RFC3339")
			break
		}
	}
	for _, digest := range []string{i.Source.StoryDescriptionDigest, i.Source.StoryAcceptanceCriteriaDigest, i.Source.SubtaskDescriptionDigest, i.Source.SubtaskAcceptanceCriteriaDigest} {
		if !digestPattern.MatchString(digest) {
			issues = append(issues, "source digests must use sha256:<hex>")
			break
		}
	}
	if i.Scope.Phase == "" || len(i.Scope.AllowedPaths) == 0 || len(i.Scope.NonGoals) == 0 || len(i.Scope.TechnicalAcceptanceCriteria) == 0 || len(i.Scope.ValidationPlan) == 0 {
		issues = append(issues, "scope is incomplete")
	}
	if !oneOf(i.Scope.ChangeClass, "trivial", "standard", "high-risk") {
		issues = append(issues, "scope.change_class is invalid")
	}
	if i.Scope.ReviewBudget.MaxChangedFiles < 1 || i.Scope.ReviewBudget.MaxChangedLines < 1 || i.Scope.ReviewBudget.MaxComponents < 1 {
		issues = append(issues, "review budget values must be positive")
	}
	if i.GitRange.BaseSHA == "" || i.GitRange.HeadSHA == "" {
		issues = append(issues, "git_range is incomplete")
	}
	if i.Decision.ADR == "" || !(strings.HasPrefix(i.Decision.ADR, "docs/decisions/") || strings.HasPrefix(i.Decision.ADR, "No ADR needed: ")) {
		issues = append(issues, "decision.adr is invalid")
	}
	if i.Decision.ADRPreflightPending && (!strings.HasPrefix(i.Decision.ADR, "docs/decisions/") || !allowedPath(i.Decision.ADR, i.Scope.AllowedPaths)) {
		issues = append(issues, "decision.adr_preflight_pending requires an allowed docs/decisions ADR path")
	}
	if i.Handoff.Status == "" || i.Handoff.CompletedWork == "" || i.Handoff.NextAction == "" {
		issues = append(issues, "handoff is incomplete")
	}
	if i.Links.ReviewException != nil {
		exception := i.Links.ReviewException
		if exception.JiraReference == "" || !validHTTPURL(exception.JiraReference) || exception.Reason == "" || exception.Approver == "" || len(exception.ReviewPlan) == 0 || exception.ApprovedAt == "" {
			issues = append(issues, "review exception is incomplete")
		}
		if _, err := time.Parse(time.RFC3339, exception.ApprovedAt); err != nil {
			issues = append(issues, "review exception approval timestamp must use RFC3339")
		}
		if i.Scope.ChangeClass == "high-risk" && (exception.ContainmentPlan == nil || *exception.ContainmentPlan == "") {
			issues = append(issues, "high-risk review exception requires containment_plan")
		}
	}
	if exception := i.Links.AgentException; exception != nil {
		if exception.JiraReference == "" || !validHTTPURL(exception.JiraReference) || exception.Reason == "" || exception.Approver == "" {
			issues = append(issues, "agent exception is incomplete")
		}
		if _, err := time.Parse(time.RFC3339, exception.ApprovedAt); err != nil {
			issues = append(issues, "agent exception approval timestamp must use RFC3339")
		}
	}
	return issues
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func allowedPath(path string, roots []string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	if cleanPath != path || cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
		return false
	}
	for _, root := range roots {
		cleanRoot := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(root)), "/")
		if cleanRoot != "." && (cleanPath == cleanRoot || strings.HasPrefix(cleanPath, cleanRoot+"/")) {
			return true
		}
	}
	return false
}

func legacySourceMode(provider string) string {
	switch provider {
	case "jira":
		return "live-jira"
	case "offline-export":
		return "offline-export"
	default:
		return ""
	}
}
