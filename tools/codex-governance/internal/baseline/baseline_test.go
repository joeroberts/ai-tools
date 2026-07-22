package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string)
		want   string
	}{
		{name: "valid"},
		{
			name:   "missing required file",
			mutate: func(root string) { _ = os.Remove(filepath.Join(root, "SECURITY.md")) },
			want:   "missing required baseline file SECURITY.md",
		},
		{
			name:   "malformed YAML",
			mutate: func(root string) { write(t, root, ".github/dependabot.yml", "updates: [") },
			want:   "dependabot configuration is malformed YAML",
		},
		{
			name:   "empty bug form",
			mutate: func(root string) { write(t, root, ".github/ISSUE_TEMPLATE/bug_report.yml", "name: Bug\nbody: []\n") },
			want:   "bug-report template must define name and body",
		},
		{
			name: "commented ownership",
			mutate: func(root string) {
				write(t, root, ".github/CODEOWNERS", "# * @joeroberts\n# /tools/ @joeroberts\n")
			},
			want: "CODEOWNERS does not cover * @joeroberts",
		},
		{
			name: "permission expansion",
			mutate: func(root string) {
				value := strings.Replace(dependabot, "    groups:\n      go:", "    permissions:\n      contents: write\n    groups:\n      go:", 1)
				write(t, root, ".github/dependabot.yml", value)
			},
			want: "dependabot must not expand write permissions",
		},
		{
			name: "conflicting schedule",
			mutate: func(root string) {
				write(t, root, ".github/dependabot.yml", strings.Replace(dependabot, "interval: monthly", "interval: daily", 1))
			},
			want: "dependabot schedule must be weekly or monthly",
		},
		{
			name: "unexpected update",
			mutate: func(root string) {
				write(t, root, ".github/dependabot.yml", dependabot+`  - package-ecosystem: npm
    directory: /
    schedule:
      interval: monthly
    open-pull-requests-limit: 5
    groups:
      packages:
        patterns: ["*"]
`)
			},
			want: "dependabot configuration must define exactly two version 2 updates",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t)
			if test.mutate != nil {
				test.mutate(root)
			}
			issues := Check(root)
			if test.want == "" && len(issues) != 0 {
				t.Fatalf("Check() = %v", issues)
			}
			if test.want != "" && !strings.Contains(strings.Join(issues, "\n"), test.want) {
				t.Fatalf("Check() = %v, want %q", issues, test.want)
			}
		})
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, ".github/CODEOWNERS", "* @joeroberts\n/tools/ @joeroberts\n/integrations/codex/ @joeroberts\n/.github/ @joeroberts\n/governance.yml @joeroberts\n/SECURITY.md @joeroberts\n")
	write(t, root, ".github/dependabot.yml", dependabot)
	write(t, root, "SECURITY.md", "private vulnerability reporting\nDo not include details in a public issue\n")
	write(t, root, ".github/pull_request_template.md", "## Scope\n## Validation\n## Tracking\n## Security considerations\n")
	write(t, root, ".github/ISSUE_TEMPLATE/bug_report.yml", "name: Bug\nbody:\n  - type: textarea\n    id: problem\n")
	write(t, root, ".github/ISSUE_TEMPLATE/config.yml", "contact_links:\n  - url: https://github.com/joeroberts/ai-tools/security/advisories/new\n")
	write(t, root, ".gitattributes", "* text=auto\n")
	write(t, root, ".editorconfig", "root = true\nend_of_line = lf\ninsert_final_newline = true\n")
	return root
}

func write(t *testing.T, root, path, value string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

const dependabot = `version: 2
updates:
  - package-ecosystem: gomod
    directory: /tools/codex-governance
    schedule:
      interval: monthly
    open-pull-requests-limit: 5
    groups:
      go:
        patterns: ["*"]
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: monthly
    open-pull-requests-limit: 5
    groups:
      actions:
        patterns: ["*"]
`
