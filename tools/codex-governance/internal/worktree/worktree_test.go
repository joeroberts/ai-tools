package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateUsesDetachedBase(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "user.email", "test@example.test")
	if err := os.WriteFile(filepath.Join(root, "value.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	path := filepath.Join(t.TempDir(), "run")
	if err := Create(root, base, path); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(runGit(t, path, "rev-parse", "HEAD")); got != base {
		t.Fatalf("worktree HEAD = %s, want %s", got, base)
	}
	if branch := strings.TrimSpace(runGit(t, path, "rev-parse", "--abbrev-ref", "HEAD")); branch != "HEAD" {
		t.Fatalf("worktree branch = %q, want detached", branch)
	}
	if err := Create(root, base, path); err == nil {
		t.Fatal("existing worktree path was accepted")
	}
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}
