package implementation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

// PlanningAuthorizationPayload is deliberately distinct from
// WorkflowAuthorizationPayload. Jira planning happens before Jira assigns a
// Story or Subtask key, while lifecycle authority must always bind those keys.
type PlanningAuthorizationPayload struct {
	FormatVersion      int       `json:"format_version"`
	RepositoryID       string    `json:"repository_id"`
	GitHubIssue        string    `json:"github_issue"`
	PlanDigest         string    `json:"plan_digest"`
	SourceDigests      []string  `json:"source_digests"`
	JiraProject        string    `json:"jira_project"`
	ExpectedSubtasks   int       `json:"expected_subtasks"`
	AllowedOperations  []string  `json:"allowed_operations"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	ExpiresAt          time.Time `json:"expires_at"`
}

type SignedPlanningAuthorization struct {
	Envelope signature.Envelope
	Payload  PlanningAuthorizationPayload
	Digest   string
}

func LoadSignedPlanningAuthorization(path string, cfg config.Config, now time.Time) (SignedPlanningAuthorization, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SignedPlanningAuthorization{}, err
	}
	var envelope signature.Envelope
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("parse planning authorization: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("parse planning authorization: %w", err)
	}
	registry, err := cfg.TrustedKeyRegistry()
	if err != nil {
		return SignedPlanningAuthorization{}, err
	}
	if err := registry.Verify(envelope, []string{"repository-owner"}, now); err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("verify planning authorization: %w", err)
	}
	var payload PlanningAuthorizationPayload
	payloadDecoder := json.NewDecoder(strings.NewReader(string(envelope.Payload)))
	payloadDecoder.DisallowUnknownFields()
	if err := payloadDecoder.Decode(&payload); err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("parse planning authorization payload: %w", err)
	}
	if err := requireJSONEOF(payloadDecoder); err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("parse planning authorization payload: %w", err)
	}
	if err := validatePlanningAuthorizationPayload(payload); err != nil || envelope.ExpiresAt == nil || !payload.ExpiresAt.Equal(*envelope.ExpiresAt) {
		if err == nil {
			err = fmt.Errorf("planning authorization expiry is not bound to the signed payload")
		}
		return SignedPlanningAuthorization{}, err
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		return SignedPlanningAuthorization{}, fmt.Errorf("canonicalize planning authorization: %w", err)
	}
	return SignedPlanningAuthorization{Envelope: envelope, Payload: payload, Digest: signature.Digest(canonical)}, nil
}

func (a SignedPlanningAuthorization) Allows(operation string) bool {
	for _, allowed := range a.Payload.AllowedOperations {
		if allowed == operation {
			return true
		}
	}
	return false
}

// ConsumeAuthorizedPlanningOperation reserves the one pre-key Jira action.
// It shares the durable, fail-closed state store with lifecycle authority but
// cannot be substituted for it because its signed claims are different.
func ConsumeAuthorizedPlanningOperation(cfg config.Config, authorizationPath, runtimeRoot, operation, planDigest, jiraProject string, expectedSubtasks int, preview []byte, now time.Time) (SignedPlanningAuthorization, error) {
	if operation == "" || len(preview) == 0 || expectedSubtasks < 1 {
		return SignedPlanningAuthorization{}, fmt.Errorf("planning authorization operation is incomplete")
	}
	authorization, err := LoadSignedPlanningAuthorization(authorizationPath, cfg, now)
	if err != nil {
		return SignedPlanningAuthorization{}, err
	}
	payload := authorization.Payload
	if payload.RepositoryID != cfg.Signing.RepositoryID || payload.PlanDigest != planDigest || payload.JiraProject != jiraProject || payload.ExpectedSubtasks != expectedSubtasks || !authorization.Allows(operation) {
		return SignedPlanningAuthorization{}, fmt.Errorf("planning authorization does not match the operation target")
	}
	event := WorkflowAuditEvent{Operation: operation, PreviewDigest: digestBytes(preview), Result: "reserved", At: now.UTC()}
	if err := consumePlanningAuthorization(runtimeRoot, authorization, event); err != nil {
		return SignedPlanningAuthorization{}, err
	}
	return authorization, nil
}

func CompleteAuthorizedPlanningOperation(runtimeRoot string, authorization SignedPlanningAuthorization, operation, result string, readBack []byte, now time.Time) error {
	return CompleteAuthorizedOperation(runtimeRoot, SignedWorkflowAuthorization{Digest: authorization.Digest}, operation, result, readBack, now)
}

func consumePlanningAuthorization(runtimeRoot string, authorization SignedPlanningAuthorization, event WorkflowAuditEvent) error {
	if !authorization.Allows(event.Operation) || event.Operation == "" || event.PreviewDigest == "" || event.Result == "" {
		return fmt.Errorf("planning authorization does not allow this operation or audit event")
	}
	return mutateWorkflowAuthorizationState(runtimeRoot, authorization.Digest, func(state *WorkflowAuthorizationState) error {
		if state.Revoked {
			return fmt.Errorf("planning authorization is revoked")
		}
		for _, used := range state.UsedOperations {
			if used == event.Operation {
				return fmt.Errorf("planning authorization operation has already been used")
			}
		}
		state.UsedOperations = append(state.UsedOperations, event.Operation)
		state.Audit = append(state.Audit, event)
		return nil
	})
}

func validatePlanningAuthorizationPayload(payload PlanningAuthorizationPayload) error {
	if payload.FormatVersion != 1 || payload.RepositoryID == "" || payload.GitHubIssue == "" || !validDigest(payload.PlanDigest) || len(payload.SourceDigests) == 0 || payload.JiraProject == "" || payload.ExpectedSubtasks < 1 || len(payload.AllowedOperations) == 0 || len(payload.AcceptanceCriteria) == 0 || payload.ExpiresAt.IsZero() {
		return fmt.Errorf("planning authorization payload is invalid")
	}
	seen := map[string]bool{}
	for _, digest := range payload.SourceDigests {
		if !validDigest(digest) || seen[digest] {
			return fmt.Errorf("planning authorization source digests are invalid")
		}
		seen[digest] = true
	}
	for _, value := range append(append([]string{}, payload.AllowedOperations...), payload.AcceptanceCriteria...) {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("planning authorization payload is invalid")
		}
	}
	return nil
}
