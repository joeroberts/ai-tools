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
}

func signedWorkflowAuthorization(t *testing.T, expires time.Time) (SignedWorkflowAuthorization, string) {
	t.Helper()
	root := t.TempDir()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePreflightConfig(t, root, base64.StdEncoding.EncodeToString(publicKey), "repository-owner", "24h")
	payload := WorkflowAuthorizationPayload{FormatVersion: 1, RepositoryID: "github.com/acme/governance", GitHubIssue: "22", StoryKey: "REK-94", SubtaskKey: "REK-95", PlanContractDigest: testDigest("plan"), SourceDigests: []string{testDigest("prd"), testDigest("spec"), testDigest("roadmap")}, BaseSHA: "abcdef1", AllowedPaths: []string{"internal/implementation"}, MaxChangedFiles: 8, MaxChangedLines: 700, ReviewCycleLimit: 2, AllowedOperations: []string{"jira-plan-create", "local-commit"}, Branch: "codex/issue-22", Remote: "origin", PRTargetBranch: "main", DerivationRules: []string{"commit-sha"}}
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
