package ollama

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Endpoint              string  `yaml:"endpoint"`
	RequestTimeoutSeconds int     `yaml:"request_timeout_seconds"`
	Models                []Model `yaml:"models"`
	Fingerprint           string  `yaml:"-"`
}

type Model struct {
	Name              string   `yaml:"name"`
	ID                string   `yaml:"id"`
	BenchmarkApproved bool     `yaml:"benchmark_approved"`
	BenchmarkOnly     bool     `yaml:"benchmark_only"`
	AllowedRoles      []string `yaml:"allowed_roles"`
	AllowedTaskTypes  []string `yaml:"allowed_task_types"`
	MaxInputBytes     int      `yaml:"max_input_bytes"`
}

type Request struct {
	Model    string
	Role     string
	TaskType string
	Input    []byte
	Think    *bool
	Format   string
}

// Status reports the context allocated to an allowlisted model that is
// currently loaded by the local Ollama service. It intentionally returns only
// local runtime metadata and never starts a model or sends a prompt.
type Status struct {
	Name          string `json:"name"`
	Loaded        bool   `json:"loaded"`
	ContextLength int    `json:"context_length"`
	ContextKnown  bool   `json:"context_known"`
	SizeVRAM      int64  `json:"size_vram"`
}

// SetResidency changes only the residency of an allowlisted installed model.
// It never includes prompt content and verifies the requested final state.
func SetResidency(client *http.Client, policy Policy, modelName string, loaded bool) error {
	var allowed Model
	for _, model := range policy.Models {
		if model.Name == modelName {
			allowed = model
			break
		}
	}
	if allowed.Name == "" {
		return fmt.Errorf("model is not allowlisted")
	}
	if err := verifyInstalled(client, policy.Endpoint, allowed); err != nil {
		return err
	}
	keepAlive := any(0)
	if loaded {
		keepAlive = "10m"
	}
	payload, _ := json.Marshal(map[string]any{"model": allowed.Name, "keep_alive": keepAlive, "stream": false})
	request, err := http.NewRequest(http.MethodPost, policy.Endpoint+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama residency request returned %s", response.Status)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, err := LoadedStatus(client, policy, modelName)
		if err != nil {
			return err
		}
		if status.Loaded == loaded {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("model residency verification failed: loaded=%t", status.Loaded)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// InstalledModel is immutable local model metadata returned by a read-only
// inventory request. Inventory never authorizes a model for execution.
type InstalledModel struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func LoadPolicy(path string) (Policy, error) {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return Policy{}, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return Policy{}, fmt.Errorf("policy permissions must be owner-only")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Policy{}, err
	}
	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("parse Ollama policy: %w", err)
	}
	if policy.Endpoint == "" {
		policy.Endpoint = "http://127.0.0.1:11434"
	}
	if policy.RequestTimeoutSeconds == 0 {
		policy.RequestTimeoutSeconds = 60
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	policy.Fingerprint = InputDigest(data)
	return policy, nil
}

func Run(client *http.Client, policy Policy, request Request) (string, error) {
	model, err := policy.Authorize(request)
	if err != nil {
		return "", err
	}
	if err := verifyInstalled(client, policy.Endpoint, model); err != nil {
		return "", err
	}
	return generateWithDeadline(client, policy.Endpoint, model.Name, request.Input, request.Think, request.Format, time.Duration(policy.RequestTimeoutSeconds)*time.Second)
}

func generateWithDeadline(client *http.Client, endpoint, model string, input []byte, think *bool, format string, timeout time.Duration) (string, error) {
	deadline, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	payload, err := json.Marshal(struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
		Think  *bool  `json:"think,omitempty"`
		Format string `json:"format,omitempty"`
	}{Model: model, Prompt: string(input), Stream: true, Think: think, Format: format})
	if err != nil {
		return "", err
	}
	httpRequest, err := http.NewRequestWithContext(deadline, http.MethodPost, endpoint+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("Ollama stream stalled: policy deadline exceeded")
		}
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama generate returned %s", response.Status)
	}
	var body struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
		Error    string `json:"error"`
	}
	decoder := json.NewDecoder(response.Body)
	var output strings.Builder
	for {
		if err := decoder.Decode(&body); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(deadline.Err(), context.DeadlineExceeded) {
				return "", fmt.Errorf("Ollama stream stalled: policy deadline exceeded")
			}
			if err == io.EOF {
				return "", fmt.Errorf("Ollama stream ended before completion")
			}
			return "", err
		}
		if body.Error != "" {
			return "", fmt.Errorf("Ollama generate: %s", body.Error)
		}
		output.WriteString(body.Response)
		if body.Done {
			return output.String(), nil
		}
	}
}

func LoadedStatus(client *http.Client, policy Policy, modelName string) (Status, error) {
	if err := policy.Validate(); err != nil {
		return Status{}, err
	}
	var allowed Model
	found := false
	for _, model := range policy.Models {
		if model.Name == modelName {
			allowed = model
			found = true
			break
		}
	}
	if !found {
		return Status{}, fmt.Errorf("model is not allowlisted")
	}
	if err := verifyInstalled(client, policy.Endpoint, allowed); err != nil {
		return Status{}, err
	}
	response, err := client.Get(policy.Endpoint + "/api/ps")
	if err != nil {
		return Status{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("Ollama process status returned %s", response.Status)
	}
	var body struct {
		Models []struct {
			Name          string `json:"name"`
			Digest        string `json:"digest"`
			ContextLength int    `json:"context_length"`
			SizeVRAM      int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return Status{}, err
	}
	for _, loaded := range body.Models {
		if loaded.Name == allowed.Name && loaded.Digest == allowed.ID {
			return Status{Name: allowed.Name, Loaded: true, ContextLength: loaded.ContextLength, ContextKnown: loaded.ContextLength > 0, SizeVRAM: loaded.SizeVRAM}, nil
		}
	}
	return Status{Name: allowed.Name}, nil
}

// Inventory returns only locally installed model names and immutable digests.
// The policy still validates that the endpoint is local, but no allowlist entry
// is required because this operation neither loads nor invokes a model.
func Inventory(client *http.Client, policy Policy) ([]InstalledModel, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	response, err := client.Get(policy.Endpoint + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama tags returned %s", response.Status)
	}
	var body struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, err
	}
	models := make([]InstalledModel, 0, len(body.Models))
	for _, model := range body.Models {
		if model.Name == "" || !modelIDPattern.MatchString(model.Digest) {
			return nil, fmt.Errorf("Ollama inventory contains invalid model metadata")
		}
		models = append(models, InstalledModel{Name: model.Name, ID: model.Digest})
	}
	return models, nil
}

func (p Policy) Authorize(request Request) (Model, error) { return p.authorize(request) }

func VerifyInstalled(client *http.Client, policy Policy, model Model) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if !policy.includes(model) {
		return fmt.Errorf("model is not allowlisted by policy")
	}
	return verifyInstalled(client, policy.Endpoint, model)
}

func (p Policy) authorize(request Request) (Model, error) {
	if err := p.Validate(); err != nil {
		return Model{}, err
	}
	if request.TaskType == "code-edit" {
		return Model{}, fmt.Errorf("code-edit tasks are disabled")
	}
	for _, model := range p.Models {
		if model.Name != request.Model {
			continue
		}
		benchmarkRun := model.BenchmarkOnly && request.TaskType == "ticket-plan-benchmark"
		if (!model.BenchmarkApproved && !benchmarkRun) || !contains(model.AllowedRoles, request.Role) || !contains(model.AllowedTaskTypes, request.TaskType) {
			return Model{}, fmt.Errorf("model is not approved for this role and task")
		}
		if model.MaxInputBytes < 1 || len(request.Input) > model.MaxInputBytes {
			return Model{}, fmt.Errorf("input exceeds policy limit")
		}
		return model, nil
	}
	return Model{}, fmt.Errorf("model is not allowlisted")
}

// Validate applies policy invariants at every execution boundary so callers
// cannot bypass local-endpoint and allowlist checks with a constructed Policy.
func (p Policy) Validate() error {
	if err := validateEndpoint(p.Endpoint); err != nil {
		return err
	}
	if p.RequestTimeoutSeconds < 10 || p.RequestTimeoutSeconds > 600 {
		return fmt.Errorf("Ollama request timeout must be between 10 and 600 seconds")
	}
	seen := map[string]bool{}
	for _, model := range p.Models {
		if model.Name == "" || !modelIDPattern.MatchString(model.ID) || seen[model.Name] ||
			len(model.AllowedRoles) == 0 || len(model.AllowedTaskTypes) == 0 || model.MaxInputBytes < 1 {
			return fmt.Errorf("Ollama policy contains an invalid model")
		}
		seen[model.Name] = true
	}
	return nil
}

func (p Policy) includes(target Model) bool {
	for _, model := range p.Models {
		if model.Name == target.Name && model.ID == target.ID {
			return true
		}
	}
	return false
}

func verifyInstalled(client *http.Client, endpoint string, model Model) error {
	response, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama tags returned %s", response.Status)
	}
	var body struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return err
	}
	for _, installed := range body.Models {
		if installed.Name == model.Name && installed.Digest == model.ID {
			return nil
		}
	}
	return fmt.Errorf("allowlisted model is not installed with the pinned ID")
}

func validateEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost") {
		return fmt.Errorf("Ollama endpoint must be local HTTP")
	}
	return nil
}

var modelIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func PolicyPath(root string) string { return filepath.Join(root, "policy.yaml") }

func DefaultPolicy() []byte {
	return []byte("endpoint: http://127.0.0.1:11434\nmodels: []\n")
}

func InputDigest(input []byte) string {
	sum := sha256.Sum256(input)
	return hex.EncodeToString(sum[:])
}

func DefaultClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func Client(policy Policy) *http.Client {
	return &http.Client{Timeout: time.Duration(policy.RequestTimeoutSeconds) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
