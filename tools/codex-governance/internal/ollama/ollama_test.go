package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunStreamsResponseUntilDone(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	think := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/generate":
			var payload struct {
				Stream bool   `json:"stream"`
				Think  *bool  `json:"think"`
				Format string `json:"format"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil || !payload.Stream || payload.Think == nil || *payload.Think || payload.Format != "json" {
				t.Fatalf("stream payload = %#v, %v", payload, err)
			}
			_, _ = writer.Write([]byte("{\"response\":\"first \"}\n{\"response\":\"second\",\"done\":true}\n"))
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	output, err := Run(Client(policy), policy, Request{Model: "local-model", Role: "reviewer", TaskType: "ticket-plan-review", Input: []byte("input"), Think: &think, Format: "json"})
	if err != nil || output != "first second" {
		t.Fatalf("Run() = %q, %v", output, err)
	}
}

func TestGenerateWithDeadlineCancelsStalledStream(t *testing.T) {
	disconnected := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/x-ndjson")
		writer.WriteHeader(http.StatusOK)
		writer.(http.Flusher).Flush()
		<-request.Context().Done()
		close(disconnected)
	}))
	defer server.Close()
	start := time.Now()
	_, err := generateWithDeadline(&http.Client{}, server.URL, "local-model", []byte("input"), nil, "", 100*time.Millisecond)
	if err == nil || err.Error() != "Ollama stream stalled: policy deadline exceeded" {
		t.Fatalf("generateWithDeadline() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("stalled stream exceeded deadline: %s", elapsed)
	}
	select {
	case <-disconnected:
	case <-time.After(time.Second):
		t.Fatal("server did not observe client cancellation")
	}
}

func TestLoadedStatusReportsContextForAllowlistedLoadedModel(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/ps":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","context_length":32768,"size_vram":1234}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	status, err := LoadedStatus(Client(policy), policy, "local-model")
	if err != nil {
		t.Fatalf("LoadedStatus() error = %v", err)
	}
	if !status.Loaded || !status.ContextKnown || status.ContextLength != 32768 || status.SizeVRAM != 1234 {
		t.Fatalf("LoadedStatus() = %#v", status)
	}
}

func TestLoadedStatusMarksZeroContextUnknown(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/ps":
			_, _ = writer.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","context_length":0}]}`))
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	status, err := LoadedStatus(Client(policy), policy, "local-model")
	if err != nil {
		t.Fatalf("LoadedStatus() error = %v", err)
	}
	if !status.Loaded || status.ContextKnown {
		t.Fatalf("LoadedStatus() = %#v", status)
	}
}

func TestSetResidencySendsNoPromptAndVerifiesStopped(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/generate":
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if _, ok := payload["prompt"]; ok || payload["keep_alive"] != float64(0) {
				t.Fatalf("stop payload = %#v", payload)
			}
			_, _ = w.Write([]byte(`{"done":true}`))
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[]}`))
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	if err := SetResidency(Client(policy), policy, "local-model", false); err != nil {
		t.Fatalf("SetResidency() = %v", err)
	}
}

func TestSetResidencyLoadsWithoutPromptAndVerifiesLoaded(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	installedVerified, loadRequested := false, false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			installedVerified = true
			_, _ = w.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/generate":
			if !installedVerified {
				t.Fatal("load requested before installed-model identity verification")
			}
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if _, ok := payload["prompt"]; ok || payload["model"] != "local-model" || payload["keep_alive"] != "10m" || payload["stream"] != false {
				t.Fatalf("load payload = %#v", payload)
			}
			loadRequested = true
			_, _ = w.Write([]byte(`{"done":true}`))
		case "/api/ps":
			if !loadRequested {
				t.Fatal("residency checked before load request")
			}
			_, _ = w.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","context_length":4096,"size_vram":1}]}`))
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	if err := SetResidency(Client(policy), policy, "local-model", true); err != nil {
		t.Fatalf("SetResidency() = %v", err)
	}
}

func TestSetResidencyRejectsUnknownModel(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	if err := SetResidency(Client(policy), policy, "unknown", false); err == nil {
		t.Fatal("unknown model was accepted")
	}
}

func TestSetResidencyFailsWhenStatusDoesNotReachRequestedState(t *testing.T) {
	policy := testPolicy(Model{Name: "local-model", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		case "/api/generate":
			_, _ = w.Write([]byte(`{"done":true}`))
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"local-model","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
		}
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	err := setResidency(Client(policy), policy, "local-model", false, 10*time.Millisecond)
	if err == nil || err.Error() != "model residency verification failed: loaded=true" {
		t.Fatalf("setResidency() error = %v", err)
	}
}

func TestInventoryReturnsInstalledModelsWithoutAllowlistingThem(t *testing.T) {
	policy := testPolicy(Model{Name: "allowed", BenchmarkApproved: true, AllowedRoles: []string{"reviewer"}, AllowedTaskTypes: []string{"ticket-plan-review"}, MaxInputBytes: 100})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/tags" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"models":[{"name":"devstral:24b","digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}`))
	}))
	defer server.Close()
	policy.Endpoint = server.URL
	models, err := Inventory(Client(policy), policy)
	if err != nil || len(models) != 1 || models[0].Name != "devstral:24b" || models[0].ID != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("Inventory() = %#v, %v", models, err)
	}
}

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
