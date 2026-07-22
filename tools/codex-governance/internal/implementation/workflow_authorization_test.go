package implementation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

func TestWorkflowAuthorizationConsumptionAndRevocation(t *testing.T) {
	authorization, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	runtimeRoot := filepath.Join(root, "runtime")
	event := WorkflowAuditEvent{Operation: "jira-plan-create", PreviewDigest: testDigest("preview"), Result: "success"}
	if err := ConsumeWorkflowAuthorization(runtimeRoot, authorization, event); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeWorkflowAuthorization(runtimeRoot, authorization, event); err == nil {
		t.Fatal("replayed operation was accepted")
	}
	state, err := LoadWorkflowAuthorizationState(runtimeRoot, authorization.Digest)
	if err != nil || len(state.UsedOperations) != 1 || len(state.Audit) != 1 || state.Audit[0].ReadBackDigest != "" {
		t.Fatalf("state = %#v, %v", state, err)
	}
	if err := RevokeWorkflowAuthorization(runtimeRoot, authorization, testDigest("revoke"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeWorkflowAuthorization(runtimeRoot, authorization, WorkflowAuditEvent{Operation: "local-commit", PreviewDigest: testDigest("commit"), Result: "success"}); err == nil {
		t.Fatal("revoked authorization was accepted")
	}
}

func TestLoadSignedWorkflowAuthorizationRejectsExpiredAndTampered(t *testing.T) {
	expired, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(-time.Minute))
	_ = expired
	if _, err := LoadSignedWorkflowAuthorization(filepath.Join(root, "authorization.json"), mustConfig(t, root), time.Now()); err == nil {
		t.Fatal("expired authorization was accepted")
	}

	_, root = signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	var envelope signature.Envelope
	data, err := os.ReadFile(filepath.Join(root, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Payload = []byte(`{"format_version":1}`)
	writeJSON(t, filepath.Join(root, "authorization.json"), envelope)
	if _, err := LoadSignedWorkflowAuthorization(filepath.Join(root, "authorization.json"), mustConfig(t, root), time.Now()); err == nil {
		t.Fatal("tampered authorization was accepted")
	}
	_, root = signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	path := filepath.Join(root, "authorization.json")
	envelope = signature.Envelope{}
	data, err = os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &envelope) != nil {
		t.Fatal("read authorization fixture")
	}
	extended := time.Now().UTC().Add(2 * time.Hour)
	envelope.ExpiresAt = &extended
	writeJSON(t, path, envelope)
	if _, err := LoadSignedWorkflowAuthorization(path, mustConfig(t, root), time.Now()); err == nil {
		t.Fatal("authorization accepted an envelope expiry not bound to its payload")
	}
	_, root = signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	path = filepath.Join(root, "authorization.json")
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record["unexpected"] = true
	data, err = json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignedWorkflowAuthorization(path, mustConfig(t, root), time.Now()); err == nil {
		t.Fatal("authorization accepted an unknown envelope field")
	}
	_, root = signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	path = filepath.Join(root, "authorization.json")
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("\n{}")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignedWorkflowAuthorization(path, mustConfig(t, root), time.Now()); err == nil {
		t.Fatal("authorization accepted multiple JSON values")
	}
}

func TestConsumeAuthorizedOperationBindsJiraTargetAndPreventsReplay(t *testing.T) {
	authorization, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	cfg := mustConfig(t, root)
	cfg.Signing.RepositoryID = authorization.Payload.RepositoryID
	binding := WorkflowOperationBinding{Operation: "jira-work-update", StoryKey: authorization.Payload.StoryKey, SubtaskKey: authorization.Payload.SubtaskKey}
	if _, err := ConsumeAuthorizedOperation(cfg, filepath.Join(root, "authorization.json"), filepath.Join(root, "runtime"), binding, []byte("exact preview"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeAuthorizedOperation(cfg, filepath.Join(root, "authorization.json"), filepath.Join(root, "runtime"), binding, []byte("exact preview"), time.Now()); err == nil {
		t.Fatal("accepted replayed authorized Jira operation")
	}
	binding.SubtaskKey = "REK-other"
	if _, err := ConsumeAuthorizedOperation(cfg, filepath.Join(root, "authorization.json"), filepath.Join(root, "runtime-other"), binding, []byte("exact preview"), time.Now()); err == nil {
		t.Fatal("accepted mismatched Jira target")
	}
	binding.SubtaskKey = authorization.Payload.SubtaskKey
	binding.Operation = "local-commit"
	binding.ReviewCycle = authorization.Payload.ReviewCycleLimit + 1
	if _, err := ConsumeAuthorizedOperation(cfg, filepath.Join(root, "authorization.json"), filepath.Join(root, "runtime-cycle"), binding, []byte("exact preview"), time.Now()); err == nil {
		t.Fatal("accepted review cycle beyond signed limit")
	}
}

func TestCompleteAuthorizedOperationRecordsReadBackAndFailsClosed(t *testing.T) {
	authorization, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	cfg := mustConfig(t, root)
	cfg.Signing.RepositoryID = authorization.Payload.RepositoryID
	runtime := filepath.Join(root, "runtime")
	binding := WorkflowOperationBinding{Operation: "jira-work-update", StoryKey: authorization.Payload.StoryKey, SubtaskKey: authorization.Payload.SubtaskKey}
	if _, err := ConsumeAuthorizedOperation(cfg, filepath.Join(root, "authorization.json"), runtime, binding, []byte("preview"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := CompleteAuthorizedOperation(runtime, authorization, "jira-work-update", "completed", []byte("read-back"), time.Now()); err != nil {
		t.Fatal(err)
	}
	state, err := LoadWorkflowAuthorizationState(runtime, authorization.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Audit) != 1 || state.Audit[0].Result != "completed" || state.Audit[0].ReadBackDigest != digestBytes([]byte("read-back")) {
		t.Fatalf("completion audit = %#v", state.Audit)
	}
	if err := CompleteAuthorizedOperation(runtime, authorization, "jira-work-update", "completed", nil, time.Now()); err == nil {
		t.Fatal("accepted duplicate completion")
	}
}

func TestWorkflowAuthorizationConcurrentConsumptionAndReloadedRevocation(t *testing.T) {
	authorization, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	runtime := filepath.Join(root, "runtime")
	event := WorkflowAuditEvent{Operation: "jira-work-update", PreviewDigest: testDigest("preview"), Result: "reserved"}
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- ConsumeWorkflowAuthorization(runtime, authorization, event) }()
	}
	successes := 0
	for range 2 {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent consumptions = %d, want 1", successes)
	}
	if err := RevokeWorkflowAuthorization(runtime, authorization, testDigest("revoke"), time.Now()); err != nil {
		t.Fatal(err)
	}
	state, err := LoadWorkflowAuthorizationState(runtime, authorization.Digest)
	if err != nil || !state.Revoked {
		t.Fatalf("reloaded revocation = %#v, %v", state, err)
	}
	if err := ConsumeWorkflowAuthorization(runtime, authorization, WorkflowAuditEvent{Operation: "local-commit", PreviewDigest: testDigest("other"), Result: "reserved"}); err == nil {
		t.Fatal("accepted operation after reloaded revocation")
	}
}

func TestPlanningAuthorizationCannotReplaceLifecycleAuthorization(t *testing.T) {
	_, root := signedWorkflowAuthorization(t, time.Now().UTC().Add(time.Hour))
	cfg := mustConfig(t, root)
	cfg.Signing.RepositoryID = "github.com/acme/governance"
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePreflightConfig(t, root, base64.StdEncoding.EncodeToString(publicKey), "repository-owner", "24h")
	cfg = mustConfig(t, root)
	cfg.Signing.RepositoryID = "github.com/acme/governance"
	expires := time.Now().UTC().Add(time.Hour)
	payload := PlanningAuthorizationPayload{FormatVersion: 1, RepositoryID: cfg.Signing.RepositoryID, GitHubIssue: "22", PlanDigest: testDigest("plan"), SourceDigests: []string{testDigest("prd")}, JiraProject: "REK", ExpectedSubtasks: 2, AllowedOperations: []string{"jira-plan-create"}, AcceptanceCriteria: []string{"create only the signed plan"}, ExpiresAt: expires}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := signature.Sign(encoded, "fixture-issuer", "repository-owner", privateKey, time.Now().UTC(), &expires)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "planning-authorization.json")
	writeJSON(t, path, envelope)
	if _, err := ConsumeAuthorizedPlanningOperation(cfg, path, filepath.Join(root, "runtime"), "jira-plan-create", payload.PlanDigest, "REK", 2, []byte("preview"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeAuthorizedPlanningOperation(cfg, path, filepath.Join(root, "runtime"), "jira-plan-create", payload.PlanDigest, "REK", 2, []byte("preview"), time.Now()); err == nil {
		t.Fatal("accepted replayed planning authorization")
	}
	if _, err := LoadSignedWorkflowAuthorization(path, cfg, time.Now()); err == nil {
		t.Fatal("planning authorization was accepted as lifecycle authorization")
	}
}

func signedWorkflowAuthorization(t *testing.T, expires time.Time) (SignedWorkflowAuthorization, string) {
	t.Helper()
	root := t.TempDir()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePreflightConfig(t, root, base64.StdEncoding.EncodeToString(publicKey), "repository-owner", "24h")
	payload := WorkflowAuthorizationPayload{FormatVersion: 1, RepositoryID: "github.com/acme/governance", GitHubIssue: "22", StoryKey: "REK-94", SubtaskKey: "REK-95", PlanContractDigest: testDigest("plan"), SourceDigests: []string{testDigest("prd"), testDigest("spec"), testDigest("roadmap")}, BaseSHA: "abcdef1", AllowedPaths: []string{"internal/implementation"}, MaxChangedFiles: 8, MaxChangedLines: 700, AcceptanceCriteria: []string{"bound workflow authorization"}, ReviewCycleLimit: 2, AllowedOperations: []string{"jira-plan-create", "jira-work-update", "local-commit"}, Branch: "codex/issue-22", Remote: "origin", PRTargetBranch: "main", DerivationRules: []string{"commit-sha"}, ExpiresAt: expires}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := signature.Sign(encoded, "fixture-issuer", "repository-owner", privateKey, time.Now().UTC(), &expires)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "authorization.json")
	writeJSON(t, path, envelope)
	_, err = LoadSignedWorkflowAuthorization(path, mustConfig(t, root), time.Now())
	if err != nil && expires.After(time.Now()) {
		t.Fatal(err)
	}
	return SignedWorkflowAuthorization{Envelope: envelope, Payload: payload, Digest: signature.Digest(mustCanonical(t, envelope))}, root
}

func mustConfig(t *testing.T, root string) config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
func testDigest(value string) string {
	return signature.Digest([]byte(value + "000000000000000000000000000000000000000000000000000000000000"))
}
func mustCanonical(t *testing.T, envelope signature.Envelope) []byte {
	t.Helper()
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
