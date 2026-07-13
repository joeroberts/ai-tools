package ollama

import (
	"testing"
	"time"
)

func TestClientUsesPolicyTimeout(t *testing.T) {
	client := Client(Policy{RequestTimeoutSeconds: 90})
	if client.Timeout != 90*time.Second {
		t.Fatalf("timeout = %s, want 90s", client.Timeout)
	}
}

func TestPolicyRejectsUnbenchmarkedModel(t *testing.T) {
	policy := testPolicy(Model{
		Name:              "local-model",
		BenchmarkApproved: false,
		AllowedRoles:      []string{"ticket-analyst"},
		AllowedTaskTypes:  []string{"ticket-summary"},
		MaxInputBytes:     100,
	})
	if _, err := policy.authorize(Request{Model: "local-model", Role: "ticket-analyst", TaskType: "ticket-summary", Input: []byte("input")}); err == nil {
		t.Fatal("unbenchmarked model was authorized")
	}
}

func TestPolicyAllowsBenchmarkOnlyModelForSyntheticBenchmark(t *testing.T) {
	policy := testPolicy(Model{
		Name:             "local-model",
		BenchmarkOnly:    true,
		AllowedRoles:     []string{"reviewer"},
		AllowedTaskTypes: []string{"ticket-plan-benchmark"},
		MaxInputBytes:    100,
	})
	if _, err := policy.authorize(Request{Model: "local-model", Role: "reviewer", TaskType: "ticket-plan-benchmark", Input: []byte("input")}); err != nil {
		t.Fatalf("benchmark-only model was not authorized: %v", err)
	}
	if _, err := policy.authorize(Request{Model: "local-model", Role: "reviewer", TaskType: "ticket-plan-review", Input: []byte("input")}); err == nil {
		t.Fatal("benchmark-only model was authorized for production review")
	}
}

func TestPolicyRejectsCodeEditTask(t *testing.T) {
	policy := testPolicy(Model{
		Name:              "local-model",
		BenchmarkApproved: true,
		AllowedRoles:      []string{"implementer"},
		AllowedTaskTypes:  []string{"code-edit"},
		MaxInputBytes:     100,
	})
	if _, err := policy.authorize(Request{Model: "local-model", Role: "implementer", TaskType: "code-edit", Input: []byte("input")}); err == nil {
		t.Fatal("code-edit task was authorized")
	}
}

func TestAuthorizeRejectsConstructedRemotePolicy(t *testing.T) {
	policy := testPolicy(Model{
		Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"},
		AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100,
	})
	policy.Endpoint = "http://example.com:11434"
	if _, err := policy.Authorize(Request{Model: "local-model", Role: "reviewer", TaskType: "ticket-plan-review", Input: []byte("input")}); err == nil || err.Error() != "Ollama endpoint must be local HTTP" {
		t.Fatalf("Authorize() error = %v", err)
	}
}

func TestVerifyInstalledRejectsModelOutsideValidatedPolicy(t *testing.T) {
	policy := testPolicy(Model{
		Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"},
		AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100,
	})
	if err := VerifyInstalled(nil, policy, Model{Name: "other-model", ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}); err == nil || err.Error() != "model is not allowlisted by policy" {
		t.Fatalf("VerifyInstalled() error = %v", err)
	}
}

func testPolicy(model Model) Policy {
	model.ID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return Policy{Endpoint: "http://127.0.0.1:11434", RequestTimeoutSeconds: 60, Models: []Model{model}}
}
