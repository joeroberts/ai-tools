package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	FormatVersion      int       `json:"format_version"`
	RepositoryID       string    `json:"repository_id"`
	GitHubIssue        string    `json:"github_issue"`
	StoryKey           string    `json:"story_key"`
	SubtaskKey         string    `json:"subtask_key"`
	PlanContractDigest string    `json:"plan_contract_digest"`
	SourceDigests      []string  `json:"source_digests"`
	BaseSHA            string    `json:"base_sha"`
	AllowedPaths       []string  `json:"allowed_paths"`
	MaxChangedFiles    int       `json:"max_changed_files"`
	MaxChangedLines    int       `json:"max_changed_lines"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	ReviewCycleLimit   int       `json:"review_cycle_limit"`
	AllowedOperations  []string  `json:"allowed_operations"`
	Branch             string    `json:"branch"`
	Remote             string    `json:"remote"`
	PRTargetBranch     string    `json:"pr_target_branch"`
	DerivationRules    []string  `json:"derivation_rules"`
	ExpiresAt          time.Time `json:"expires_at"`
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

// WorkflowOperationBinding is the non-secret context that must match a signed
// workflow authorization before an operation may cross a local or Jira side
// effect boundary.
type WorkflowOperationBinding struct {
	Operation   string
	StoryKey    string
	SubtaskKey  string
	BaseSHA     string
	Branch      string
	Remote      string
	PRTarget    string
	ReviewCycle int
}

// ConsumeAuthorizedOperation verifies the live signed authorization against
// the operation context, then atomically reserves the operation before its
// side effect. A caller must render its exact preview before invoking this
// function; a failed or ambiguous side effect remains consumed and blocking,
// preventing a restart from silently duplicating it.
func ConsumeAuthorizedOperation(cfg config.Config, authorizationPath, runtimeRoot string, binding WorkflowOperationBinding, preview []byte, now time.Time) (SignedWorkflowAuthorization, error) {
	if binding.Operation == "" || binding.StoryKey == "" || binding.SubtaskKey == "" || len(preview) == 0 {
		return SignedWorkflowAuthorization{}, fmt.Errorf("workflow operation binding is incomplete")
	}
	authorization, err := LoadSignedWorkflowAuthorization(authorizationPath, cfg, now)
	if err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	payload := authorization.Payload
	if payload.RepositoryID != cfg.Signing.RepositoryID || payload.StoryKey != binding.StoryKey || payload.SubtaskKey != binding.SubtaskKey || (binding.BaseSHA != "" && payload.BaseSHA != binding.BaseSHA) || (binding.Branch != "" && payload.Branch != binding.Branch) || (binding.Remote != "" && payload.Remote != binding.Remote) || (binding.PRTarget != "" && payload.PRTargetBranch != binding.PRTarget) || binding.ReviewCycle < 0 || binding.ReviewCycle > payload.ReviewCycleLimit {
		return SignedWorkflowAuthorization{}, fmt.Errorf("workflow authorization does not match the operation target")
	}
	event := WorkflowAuditEvent{Operation: binding.Operation, PreviewDigest: digestBytes(preview), Result: "reserved", At: now.UTC()}
	if err := ConsumeWorkflowAuthorization(runtimeRoot, authorization, event); err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	return authorization, nil
}

// CompleteAuthorizedOperation records the deterministic outcome and read-back
// after a previously reserved operation. Failed and ambiguous operations remain
// consumed, so recovery requires an explicit reconciliation instead of a retry
// that could duplicate an external side effect.
func CompleteAuthorizedOperation(runtimeRoot string, authorization SignedWorkflowAuthorization, operation, result string, readBack []byte, now time.Time) error {
	if operation == "" || (result != "completed" && result != "failed" && result != "ambiguous") {
		return fmt.Errorf("workflow authorization completion is invalid")
	}
	return mutateWorkflowAuthorizationState(runtimeRoot, authorization.Digest, func(state *WorkflowAuthorizationState) error {
		for index := len(state.Audit) - 1; index >= 0; index-- {
			event := &state.Audit[index]
			if event.Operation != operation {
				continue
			}
			if event.Result != "reserved" {
				return fmt.Errorf("workflow authorization operation is already completed")
			}
			event.Result = result
			if len(readBack) > 0 {
				event.ReadBackDigest = digestBytes(readBack)
			}
			event.At = now.UTC()
			return nil
		}
		return fmt.Errorf("workflow authorization operation was not reserved")
	})
}

func LoadSignedWorkflowAuthorization(path string, cfg config.Config, now time.Time) (SignedWorkflowAuthorization, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SignedWorkflowAuthorization{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var envelope signature.Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("parse workflow authorization: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
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
	payloadDecoder := json.NewDecoder(strings.NewReader(string(envelope.Payload)))
	payloadDecoder.DisallowUnknownFields()
	if err := payloadDecoder.Decode(&payload); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("parse workflow authorization payload: %w", err)
	}
	if err := requireJSONEOF(payloadDecoder); err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("parse workflow authorization payload: %w", err)
	}
	if err := validateWorkflowAuthorizationPayload(payload); err != nil || envelope.ExpiresAt == nil || !payload.ExpiresAt.Equal(*envelope.ExpiresAt) {
		if err == nil {
			err = fmt.Errorf("workflow authorization expiry is not bound to the signed payload")
		}
		return SignedWorkflowAuthorization{}, err
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		return SignedWorkflowAuthorization{}, fmt.Errorf("canonicalize workflow authorization: %w", err)
	}
	return SignedWorkflowAuthorization{Envelope: envelope, Payload: payload, Digest: signature.Digest(canonical)}, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
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
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func workflowAuthorizationStatePath(runtimeRoot, digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || len(strings.TrimPrefix(digest, "sha256:")) != sha256.Size*2 {
		return "", fmt.Errorf("workflow authorization digest is invalid")
	}
	return filepath.Join(filepath.Clean(runtimeRoot), "workflow-authorization-state", strings.TrimPrefix(digest, "sha256:")+".json"), nil
}

func validateWorkflowAuthorizationPayload(payload WorkflowAuthorizationPayload) error {
	if payload.FormatVersion != 1 || payload.RepositoryID == "" || payload.GitHubIssue == "" || payload.StoryKey == "" || payload.SubtaskKey == "" || !validDigest(payload.PlanContractDigest) || len(payload.SourceDigests) != 3 || len(payload.BaseSHA) < 7 || len(payload.AllowedPaths) == 0 || payload.MaxChangedFiles < 1 || payload.MaxChangedLines < 1 || len(payload.AcceptanceCriteria) == 0 || payload.ReviewCycleLimit < 1 || len(payload.AllowedOperations) == 0 || payload.Branch == "" || payload.Remote == "" || payload.PRTargetBranch == "" || len(payload.DerivationRules) == 0 || payload.ExpiresAt.IsZero() {
		return fmt.Errorf("workflow authorization payload is invalid")
	}
	seen := map[string]bool{}
	for _, digest := range payload.SourceDigests {
		if !validDigest(digest) || seen[digest] {
			return fmt.Errorf("workflow authorization source digests are invalid")
		}
		seen[digest] = true
	}
	for _, value := range append(append(append([]string{}, payload.AllowedPaths...), append(payload.AcceptanceCriteria, payload.AllowedOperations...)...), payload.DerivationRules...) {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("workflow authorization payload is invalid")
		}
	}
	paths := append([]string(nil), payload.AllowedPaths...)
	sort.Strings(paths)
	return nil
}

// ValidateWorkflowAuthorizationPayload exposes the same fail-closed claim
// validation used by signed authorization loading to owner-only issuers.
func ValidateWorkflowAuthorizationPayload(payload WorkflowAuthorizationPayload) error {
	return validateWorkflowAuthorizationPayload(payload)
}

func validDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(strings.TrimPrefix(value, "sha256:")) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
