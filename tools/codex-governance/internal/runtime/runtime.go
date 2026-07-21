package runtime

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Event struct {
	At        time.Time `json:"at"`
	WorkItem  string    `json:"work_item"`
	AgentID   string    `json:"agent_id"`
	Role      string    `json:"role"`
	State     string    `json:"state"`
	ResultRef string    `json:"result_ref,omitempty"`
	InputRef  string    `json:"input_ref,omitempty"`
}

// LifecycleEvent is the small, public-safe record used to report governed
// execution progress. It deliberately excludes task prompts, logs, artifacts,
// and filesystem references; those remain in the owner-only run record.
type LifecycleEvent struct {
	At       time.Time `json:"at"`
	RunID    string    `json:"run_id"`
	WorkItem string    `json:"work_item"`
	Phase    string    `json:"phase"`
	State    string    `json:"state"`
}

func RecordLifecycle(root string, event LifecycleEvent) error {
	if event.RunID == "" || event.WorkItem == "" || event.Phase != "implementation" || !oneOf(event.State, "dispatched", "running", "completed", "failed") {
		return fmt.Errorf("invalid lifecycle event")
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	path := filepath.Join(root, "lifecycle-events.jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return err
}

func LoadLifecycle(root, runID string) ([]LifecycleEvent, error) {
	path := filepath.Join(root, "lifecycle-events.jsonl")
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var events []LifecycleEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event LifecycleEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse lifecycle ledger: %w", err)
		}
		if event.RunID == runID {
			events = append(events, event)
		}
	}
	return events, scanner.Err()
}

func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex-governance-runtime"), nil
}

func Record(root string, event Event) error {
	if event.WorkItem == "" || event.AgentID == "" || !oneOf(event.Role, "manager", "ticket-analyst", "implementer", "reviewer", "verifier", "remediation-editor", "ollama") || !oneOf(event.State, "started", "completed", "failed", "closed") {
		return fmt.Errorf("invalid execution event")
	}
	open, err := OpenAgents(root, event.WorkItem)
	if err != nil {
		return err
	}
	latest := map[string]Event{}
	for _, previous := range open {
		latest[previous.AgentID] = previous
	}
	if event.State == "started" && latest[event.AgentID].State != "" {
		return fmt.Errorf("agent already open")
	}
	if event.State == "completed" && latest[event.AgentID].State != "started" {
		return fmt.Errorf("agent must be started before completion")
	}
	if event.State == "completed" && event.ResultRef == "" {
		return fmt.Errorf("completed agent requires result reference")
	}
	if event.State == "failed" && latest[event.AgentID].State != "started" {
		return fmt.Errorf("agent must be started before failure")
	}
	if event.State == "failed" && event.ResultRef == "" {
		return fmt.Errorf("failed agent requires result reference")
	}
	if event.State == "closed" && !oneOf(latest[event.AgentID].State, "completed", "failed") {
		return fmt.Errorf("agent must complete or fail before closure")
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	path := filepath.Join(root, "execution-ledger.jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return err
}

func OpenAgents(root, workItem string) ([]Event, error) {
	path := filepath.Join(root, "execution-ledger.jsonl")
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	latest := map[string]Event{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse execution ledger: %w", err)
		}
		if event.WorkItem == workItem {
			latest[event.AgentID] = event
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	open := make([]Event, 0)
	for _, event := range latest {
		if event.State != "closed" {
			open = append(open, event)
		}
	}
	return open, nil
}

type CacheEntry struct {
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Summary   string    `json:"summary"`
}

func CacheKey(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(hash[:])
}

func LoadCache(root, key string) (CacheEntry, bool, error) {
	cacheDir := filepath.Join(root, "cache")
	if err := ensurePrivate(cacheDir); err != nil && !os.IsNotExist(err) {
		return CacheEntry{}, false, err
	}
	path := filepath.Join(cacheDir, key+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return CacheEntry{}, false, nil
	}
	if err != nil {
		return CacheEntry{}, false, err
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return CacheEntry{}, false, err
	}
	if entry.Key != key {
		return CacheEntry{}, false, fmt.Errorf("cache entry key mismatch")
	}
	if err := ensurePrivate(path); err != nil {
		return CacheEntry{}, false, err
	}
	if time.Now().After(entry.ExpiresAt) {
		return CacheEntry{}, false, nil
	}
	return entry, true, nil
}

func StoreCache(root, key, summary string) error {
	cacheDir := filepath.Join(root, "cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return err
	}
	if err := ensurePrivate(cacheDir); err != nil {
		return err
	}
	entry := CacheEntry{Key: key, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour), Summary: Redact(summary)}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, key+".json"), data, 0o600)
}

func ClearCache(root string) error {
	cacheDir := filepath.Join(root, "cache")
	if err := ensurePrivate(cacheDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.RemoveAll(cacheDir)
}

func ensurePrivate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("runtime path is not owner-only: %s", path)
	}
	return nil
}

func Redact(value string) string {
	return secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
}

var secretPattern = regexp.MustCompile(`(?i)(token|password|secret|api_key)=\S+`)

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
