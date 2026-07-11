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
	FormatVersion int          `yaml:"format_version"`
	Profile       string       `yaml:"profile"`
	Jira          JiraConfig   `yaml:"jira"`
	ReviewBudget  ReviewBudget `yaml:"review_budget"`
	CI            CIConfig     `yaml:"ci"`
	Upstream      Upstream     `yaml:"upstream"`
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
	return nil
}
