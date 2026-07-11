package ollama

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Endpoint    string  `yaml:"endpoint"`
	Models      []Model `yaml:"models"`
	Fingerprint string  `yaml:"-"`
}

type Model struct {
	Name              string   `yaml:"name"`
	ID                string   `yaml:"id"`
	BenchmarkApproved bool     `yaml:"benchmark_approved"`
	AllowedRoles      []string `yaml:"allowed_roles"`
	AllowedTaskTypes  []string `yaml:"allowed_task_types"`
	MaxInputBytes     int      `yaml:"max_input_bytes"`
}

type Request struct {
	Model    string
	Role     string
	TaskType string
	Input    []byte
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
	if err := validateEndpoint(policy.Endpoint); err != nil {
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
	payload := fmt.Sprintf(`{"model":%q,"prompt":%q,"stream":false}`, model.Name, string(request.Input))
	response, err := client.Post(policy.Endpoint+"/api/generate", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama generate returned %s", response.Status)
	}
	var body struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Response, nil
}

func (p Policy) Authorize(request Request) (Model, error) { return p.authorize(request) }

func VerifyInstalled(client *http.Client, policy Policy, model Model) error {
	return verifyInstalled(client, policy.Endpoint, model)
}

func (p Policy) authorize(request Request) (Model, error) {
	if request.TaskType == "code-edit" {
		return Model{}, fmt.Errorf("code-edit tasks are disabled")
	}
	for _, model := range p.Models {
		if model.Name != request.Model {
			continue
		}
		if !model.BenchmarkApproved || !contains(model.AllowedRoles, request.Role) || !contains(model.AllowedTaskTypes, request.TaskType) {
			return Model{}, fmt.Errorf("model is not approved for this role and task")
		}
		if model.MaxInputBytes < 1 || len(request.Input) > model.MaxInputBytes {
			return Model{}, fmt.Errorf("input exceeds policy limit")
		}
		return model, nil
	}
	return Model{}, fmt.Errorf("model is not allowlisted")
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
