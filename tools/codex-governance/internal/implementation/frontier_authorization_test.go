package implementation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

func TestFrontierAssessmentAuthorizationIsSignedPolicyBoundAndSingleUse(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := signature.NewRegistry(signature.FormatVersion, []signature.TrustedKey{{KeyID: "owner", Role: "repository-owner", Algorithm: signature.Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	payload := FrontierAssessmentAuthorization{FormatVersion: 1, Provider: "frontier-subagent", Role: "reviewer", Model: "frontier-model", ReasoningEffort: "high", WorkItem: "REK-100", DiffDigest: "sha256:diff", ScopeDigest: "sha256:scope", IssuedAt: now, ExpiresAt: now.Add(time.Hour), ConsumptionID: "once"}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	expires := now.Add(time.Hour)
	envelope, err := signature.Sign(data, "owner", "repository-owner", privateKey, now, &expires)
	if err != nil {
		t.Fatal(err)
	}
	policy := config.FrontierSubagentPolicy{Enabled: true, AllowedModels: []string{"frontier-model"}, MaxReasoningEffort: "high"}
	authorization, err := ValidateFrontierAssessmentAuthorization(envelope, registry, policy, "reviewer", "REK-100", "sha256:diff", "sha256:scope", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeFrontierAssessmentAuthorization(filepath.Join(t.TempDir(), "ledger"), authorization, envelope); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "replay")
	if _, err := ConsumeFrontierAssessmentAuthorization(directory, authorization, envelope); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeFrontierAssessmentAuthorization(directory, authorization, envelope); err == nil {
		t.Fatal("accepted replayed frontier authorization")
	}
	policy.AllowedModels = []string{"other-model"}
	if _, err := ValidateFrontierAssessmentAuthorization(envelope, registry, policy, "reviewer", "REK-100", "sha256:diff", "sha256:scope", now); err == nil {
		t.Fatal("accepted unconfigured frontier model")
	}
}
