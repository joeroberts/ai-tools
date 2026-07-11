package ollama

import "testing"

func TestPolicyRejectsUnbenchmarkedModel(t *testing.T) {
	policy := Policy{Models: []Model{{
		Name:              "local-model",
		ID:                "abc",
		BenchmarkApproved: false,
		AllowedRoles:      []string{"ticket-analyst"},
		AllowedTaskTypes:  []string{"ticket-summary"},
		MaxInputBytes:     100,
	}}}
	if _, err := policy.authorize(Request{Model: "local-model", Role: "ticket-analyst", TaskType: "ticket-summary", Input: []byte("input")}); err == nil {
		t.Fatal("unbenchmarked model was authorized")
	}
}

func TestPolicyRejectsCodeEditTask(t *testing.T) {
	policy := Policy{Models: []Model{{
		Name:              "local-model",
		ID:                "abc",
		BenchmarkApproved: true,
		AllowedRoles:      []string{"implementer"},
		AllowedTaskTypes:  []string{"code-edit"},
		MaxInputBytes:     100,
	}}}
	if _, err := policy.authorize(Request{Model: "local-model", Role: "implementer", TaskType: "code-edit", Input: []byte("input")}); err == nil {
		t.Fatal("code-edit task was authorized")
	}
}
