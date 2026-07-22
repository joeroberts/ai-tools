// Package baseline validates the checked-in repository configuration baseline.
package baseline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var requiredFiles = []string{
	".github/CODEOWNERS", ".github/dependabot.yml", "SECURITY.md",
	".github/pull_request_template.md", ".github/ISSUE_TEMPLATE/bug_report.yml",
	".github/ISSUE_TEMPLATE/config.yml", ".gitattributes", ".editorconfig",
}

// Check returns deterministic, actionable violations without changing state.
func Check(root string) []string {
	var issues []string
	read := func(path string) (string, bool) {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			issues = append(issues, fmt.Sprintf("missing required baseline file %s", path))
			return "", false
		}
		return string(data), true
	}
	files := map[string]string{}
	for _, path := range requiredFiles {
		if value, ok := read(path); ok {
			files[path] = value
		}
	}
	if value, ok := files[".github/CODEOWNERS"]; ok {
		active := map[string]string{}
		for _, line := range strings.Split(value, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && !strings.HasPrefix(fields[0], "#") {
				active[fields[0]] = fields[1]
			}
		}
		for _, rule := range []string{"*", "/tools/", "/integrations/codex/", "/.github/", "/governance.yml", "/SECURITY.md"} {
			if active[rule] != "@joeroberts" {
				issues = append(issues, "CODEOWNERS does not cover "+rule+" @joeroberts")
			}
		}
	}
	if value, ok := files["SECURITY.md"]; ok {
		for _, text := range []string{"private vulnerability reporting", "public issue"} {
			if !strings.Contains(strings.ToLower(value), text) {
				issues = append(issues, "SECURITY.md must include "+text)
			}
		}
	}
	if value, ok := files[".gitattributes"]; ok && !strings.Contains(value, "* text=auto") {
		issues = append(issues, ".gitattributes must contain * text=auto")
	}
	if value, ok := files[".editorconfig"]; ok {
		for _, text := range []string{"root = true", "end_of_line = lf", "insert_final_newline = true"} {
			if !strings.Contains(value, text) {
				issues = append(issues, ".editorconfig must contain "+text)
			}
		}
	}
	if value, ok := files[".github/pull_request_template.md"]; ok {
		for _, text := range []string{"## Scope", "## Validation", "## Tracking", "## Security considerations"} {
			if !strings.Contains(value, text) {
				issues = append(issues, "pull-request template must contain "+text)
			}
		}
	}
	if value, ok := files[".github/ISSUE_TEMPLATE/bug_report.yml"]; ok {
		checkYAML("bug-report template", value, &issues)
	}
	if value, ok := files[".github/ISSUE_TEMPLATE/config.yml"]; ok && !strings.Contains(value, "security/advisories/new") {
		issues = append(issues, "issue configuration must route private security reports")
	}
	if value, ok := files[".github/dependabot.yml"]; ok {
		checkDependabot(value, &issues)
	}
	return issues
}

func checkYAML(name, value string, issues *[]string) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(value), &data); err != nil {
		*issues = append(*issues, name+" is malformed YAML")
		return
	}
	body, ok := data["body"].([]any)
	if data["name"] == nil || !ok || len(body) == 0 {
		*issues = append(*issues, name+" must define name and body")
	}
}

func checkDependabot(value string, issues *[]string) {
	var data struct {
		Version int `yaml:"version"`
		Updates []struct {
			PackageEcosystem string `yaml:"package-ecosystem"`
			Directory        string `yaml:"directory"`
			Open             int    `yaml:"open-pull-requests-limit"`
			Schedule         struct {
				Interval string `yaml:"interval"`
			} `yaml:"schedule"`
			Groups      map[string]any    `yaml:"groups"`
			AutoMerge   any               `yaml:"auto-merge"`
			Permissions map[string]string `yaml:"permissions"`
		} `yaml:"updates"`
	}
	if err := yaml.Unmarshal([]byte(value), &data); err != nil {
		*issues = append(*issues, "dependabot configuration is malformed YAML")
		return
	}
	if data.Version != 2 || len(data.Updates) != 2 {
		*issues = append(*issues, "dependabot configuration must define exactly two version 2 updates")
		return
	}
	seen := map[string]bool{}
	for _, update := range data.Updates {
		seen[update.PackageEcosystem+":"+update.Directory] = true
		if update.Open < 1 || update.Open > 5 {
			*issues = append(*issues, "dependabot open-pull-requests-limit must be between 1 and 5")
		}
		if update.Schedule.Interval != "monthly" && update.Schedule.Interval != "weekly" {
			*issues = append(*issues, "dependabot schedule must be weekly or monthly")
		}
		if len(update.Groups) == 0 {
			*issues = append(*issues, "dependabot updates must be grouped")
		}
		if update.AutoMerge != nil {
			*issues = append(*issues, "dependabot must not grant auto-merge")
		}
		for permission, level := range update.Permissions {
			if permission == "contents" || permission == "pull-requests" || strings.Contains(permission, "workflow") || strings.Contains(level, "write") {
				*issues = append(*issues, "dependabot must not expand write permissions")
			}
		}
	}
	if !seen["gomod:/tools/codex-governance"] || !seen["github-actions:/"] {
		*issues = append(*issues, "dependabot must cover Go modules and GitHub Actions")
	}
}
