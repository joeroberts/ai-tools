package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

// WorkflowAuthorizationPayload is the immutable owner-approved authority for
// one bounded workflow. Mutable consumption and revocation data is deliberately
// stored separately, keyed by the signed envelope digest.
type WorkflowAuthorizationPayload struct {
	FormatVersion      int      `json:"format_version"`
	RepositoryID       string   `json:"repository_id"`
	GitHubIssue        string   `json:"github_issue"`
	StoryKey           string   `json:"story_key"`
	SubtaskKey         string   `json:"subtask_key"`
	PlanContractDigest string   `json:"plan_contract_digest"`
	SourceDigests      []string `json:"source_digests"`
	BaseSHA            string   `json:"base_sha"`
	AllowedPaths       []string `json:"allowed_paths"`
	MaxChangedFiles    int      `json:"max_changed_files"`
	MaxChangedLines    int      `json:"max_changed_lines"`
	ReviewCycleLimit   int      `json:"review_cycle_limit"`
	AllowedOperations  []string `json:"allowed_operations"`
	Branch             string   `json:"branch"`
	Remote             string   `json:"remote"`
	PRTargetBranch     string   `json:"pr_target_branch"`
	DerivationRules    []string `json:"derivation_rules"`
}

type SignedWorkflowAuthorization struct {
	Envelope signature.Envelope
	Payload  WorkflowAuthorizationPayload
	Digest   string
}

type WorkflowAuthorizationState struct {
	AuthorizationDigest string               `json:"authorization_digest"`
	Revoked             bool                 `json:"revoked"`
	UsedOperations      []string             `json:"used_operations"`
	Audit               []WorkflowAuditEvent `json:"audit"`
}

// WorkflowAuditEvent intentionally retains digests and outcomes, never
// credentials, signed payloads, prompts, or external response bodies.
type WorkflowAuditEvent struct {
	Operation      string    `json:"operation"`
	PreviewDigest  string    `json:"preview_digest"`
	Result         string    `json:"result"`
	ReadBackDigest string    `json:"read_back_digest,omitempty"`
	At             time.Time `json:"at"`
}

func LoadSignedWorkflowAuthorization(path string, cfg config.Config, now time.Time) (SignedWorkflowAuthorization, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	var envelope signature.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("parse workflow authorization: %w", err)
	}
	registry, err := cfg.TrustedKeyRegistry()
	if err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	if err := registry.Verify(envelope, []string{"repository-owner"}, now); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("verify workflow authorization: %w", err)
	}
	var payload WorkflowAuthorizationPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("parse workflow authorization payload: %w", err)
	}
	if err := validateWorkflowAuthorizationPayload(payload); err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("canonicalize workflow authorization: %w", err)
	}
	return SignedWorkflowAuthorization{Envelope: envelope, Payload: payload, Digest: signature.Digest(canonical)}, nil
}

func (a SignedWorkflowAuthorization) Allows(operation string) bool {
	for _, allowed := range a.Payload.AllowedOperations {
		if allowed == operation {
			return true
		}
	}
	return false
}

// ConsumeWorkflowAuthorization atomically marks one operation consumed and
// records its privacy-safe outcome. A stale lock is deliberately blocking so a
// restart cannot duplicate an unknown external side effect.
func ConsumeWorkflowAuthorization(runtimeRoot string, authorization SignedWorkflowAuthorization, event WorkflowAuditEvent) error {
	if !authorization.Allows(event.Operation) || event.Operation == "" || event.PreviewDigest == "" || event.Result == "" {
		return fmt.Errorf("workflow authorization does not allow this operation or audit event")
	}
	return mutateWorkflowAuthorizationState(runtimeRoot, authorization.Digest, func(state *WorkflowAuthorizationState) error {
		if state.Revoked {
			return fmt.Errorf("workflow authorization is revoked")
		}
		for _, used := range state.UsedOperations {
			if used == event.Operation {
				return fmt.Errorf("workflow authorization operation has already been used")
			}
		}
		event.At = event.At.UTC()
		if event.At.IsZero() {
			event.At = time.Now().UTC()
		}
		state.UsedOperations = append(state.UsedOperations, event.Operation)
		state.Audit = append(state.Audit, event)
		return nil
	})
}

func RevokeWorkflowAuthorization(runtimeRoot string, authorization SignedWorkflowAuthorization, previewDigest string, now time.Time) error {
	if previewDigest == "" {
		return fmt.Errorf("workflow authorization revocation requires a preview digest")
	}
	return mutateWorkflowAuthorizationState(runtimeRoot, authorization.Digest, func(state *WorkflowAuthorizationState) error {
		if state.Revoked {
			return fmt.Errorf("workflow authorization is already revoked")
		}
		state.Revoked = true
		state.Audit = append(state.Audit, WorkflowAuditEvent{Operation: "revoke", PreviewDigest: previewDigest, Result: "revoked", At: now.UTC()})
		return nil
	})
}

func LoadWorkflowAuthorizationState(runtimeRoot, digest string) (WorkflowAuthorizationState, error) {
	path, err := workflowAuthorizationStatePath(runtimeRoot, digest)
	if err != nil {
		return WorkflowAuthorizationState{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return WorkflowAuthorizationState{}, err
	}
	var state WorkflowAuthorizationState
	if err := json.Unmarshal(data, &state); err != nil || state.AuthorizationDigest != digest {
		return WorkflowAuthorizationState{}, fmt.Errorf("workflow authorization state is invalid")
	}
	return state, nil
}

func mutateWorkflowAuthorizationState(runtimeRoot, digest string, mutate func(*WorkflowAuthorizationState) error) error {
	path, err := workflowAuthorizationStatePath(runtimeRoot, digest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lock, err := os.OpenFile(path+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("workflow authorization state is already in progress or requires operator recovery")
		}
		return err
	}
	if err := lock.Close(); err != nil {
		_ = os.Remove(path + ".lock")
		return err
	}
	defer os.Remove(path + ".lock")
	state := WorkflowAuthorizationState{AuthorizationDigest: digest}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &state); err != nil || state.AuthorizationDigest != digest {
			return fmt.Errorf("workflow authorization state is invalid")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := mutate(&state); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".workflow-authorization-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func workflowAuthorizationStatePath(runtimeRoot, digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || len(strings.TrimPrefix(digest, "sha256:")) != sha256.Size*2 {
		return "", fmt.Errorf("workflow authorization digest is invalid")
	}
	return filepath.Join(filepath.Clean(runtimeRoot), "workflow-authorization-state", strings.TrimPrefix(digest, "sha256:")+".json"), nil
}

func validateWorkflowAuthorizationPayload(payload WorkflowAuthorizationPayload) error {
	if payload.FormatVersion != 1 || payload.RepositoryID == "" || payload.GitHubIssue == "" || payload.StoryKey == "" || payload.SubtaskKey == "" || !validDigest(payload.PlanContractDigest) || len(payload.SourceDigests) != 3 || len(payload.BaseSHA) < 7 || len(payload.AllowedPaths) == 0 || payload.MaxChangedFiles < 1 || payload.MaxChangedLines < 1 || payload.ReviewCycleLimit < 1 || len(payload.AllowedOperations) == 0 || payload.Branch == "" || payload.Remote == "" || payload.PRTargetBranch == "" || len(payload.DerivationRules) == 0 {
		return fmt.Errorf("workflow authorization payload is invalid")
	}
	seen := map[string]bool{}
	for _, digest := range payload.SourceDigests {
		if !validDigest(digest) || seen[digest] {
			return fmt.Errorf("workflow authorization source digests are invalid")
		}
		seen[digest] = true
	}
	for _, value := range append(append([]string{}, payload.AllowedPaths...), append(payload.AllowedOperations, payload.DerivationRules...)...) {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("workflow authorization payload is invalid")
		}
	}
	sort.Strings(payload.AllowedPaths)
	return nil
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(strings.TrimPrefix(value, "sha256:")) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
