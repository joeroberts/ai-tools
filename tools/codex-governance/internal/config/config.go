package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

const CurrentFormatVersion = 1

type Config struct {
	FormatVersion  int            `yaml:"format_version"`
	Profile        string         `yaml:"profile"`
	Jira           JiraConfig     `yaml:"jira"`
	ReviewBudget   ReviewBudget   `yaml:"review_budget"`
	CI             CIConfig       `yaml:"ci"`
	Upstream       Upstream       `yaml:"upstream"`
	Implementation Implementation `yaml:"implementation"`
}

type JiraConfig struct {
	Project          string   `yaml:"project"`
	IssueKeyPattern  string   `yaml:"issue_key_pattern"`
	RequiredSections []string `yaml:"required_sections"`
}

type ReviewBudget struct {
	MaxChangedFiles int `yaml:"max_changed_files"`
	MaxChangedLines int `yaml:"max_changed_lines"`
	MaxComponents   int `yaml:"max_components"`
}

type CIConfig struct {
	Provider string `yaml:"provider"`
	Mode     string `yaml:"mode"`
}

type Upstream struct {
	Release       string `yaml:"release"`
	SourceCommit  string `yaml:"source_commit"`
	FormatVersion int    `yaml:"format_version"`
}

// Implementation restricts future execution providers. An omitted section is
// intentionally deny-by-default so existing adopters do not gain agent-run
// authority when they upgrade the CLI.
type Implementation struct {
	AllowedAdapters      []string `yaml:"allowed_adapters"`
	LocalCodeEditEnabled bool     `yaml:"local_code_edit_enabled"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Config{}, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse governance config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.FormatVersion != CurrentFormatVersion {
		return fmt.Errorf("unsupported format_version %d", c.FormatVersion)
	}
	if c.Profile != "generic" {
		return fmt.Errorf("unsupported profile %q", c.Profile)
	}
	if c.Jira.IssueKeyPattern == "" {
		return fmt.Errorf("jira.issue_key_pattern is required")
	}
	if _, err := regexp.Compile(c.Jira.IssueKeyPattern); err != nil {
		return fmt.Errorf("invalid jira.issue_key_pattern: %w", err)
	}
	if len(c.Jira.RequiredSections) == 0 {
		return fmt.Errorf("jira.required_sections is required")
	}
	if c.ReviewBudget.MaxChangedFiles < 1 || c.ReviewBudget.MaxChangedLines < 1 || c.ReviewBudget.MaxComponents < 1 {
		return fmt.Errorf("review_budget values must be positive")
	}
	if c.CI.Provider != "github-actions" {
		return fmt.Errorf("unsupported ci.provider %q", c.CI.Provider)
	}
	if c.CI.Mode != "warn" && c.CI.Mode != "required" {
		return fmt.Errorf("unsupported ci.mode %q", c.CI.Mode)
	}
	seenAdapters := map[string]bool{}
	for _, adapter := range c.Implementation.AllowedAdapters {
		if adapter == "" || seenAdapters[adapter] || (adapter != "fake" && adapter != "headless-codex" && adapter != "local-llm") {
			return fmt.Errorf("implementation.allowed_adapters is invalid")
		}
		seenAdapters[adapter] = true
	}
	if c.Implementation.LocalCodeEditEnabled && !seenAdapters["local-llm"] {
		return fmt.Errorf("local code edit requires the local-llm adapter")
	}
	return nil
}

func (c Config) AllowsAdapter(adapter string) bool {
	for _, allowed := range c.Implementation.AllowedAdapters {
		if allowed == adapter {
			return true
		}
	}
	return false
}
