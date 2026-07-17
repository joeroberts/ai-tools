package gitdiff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangesUsesSuppliedNestedRoot(t *testing.T) {
	repo, product := nestedRepository(t)
	base := gitOutput(t, repo, "rev-parse", "HEAD")
	writeFile(t, filepath.Join(product, "tracked.txt"), "changed\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "change tracked file")
	head := gitOutput(t, repo, "rev-parse", "HEAD")

	changes, err := Changes(product, base, head)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, changes, "tracked.txt")

	changes, err = Changes(repo, base, head)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, changes, "tools/product/tracked.txt")
}

func TestWorkingChangesUsesSuppliedNestedRootAndRejectsUntracked(t *testing.T) {
	_, product := nestedRepository(t)
	writeFile(t, filepath.Join(product, "tracked.txt"), "working change\n")

	changes, err := WorkingChanges(product)
	if err != nil {
		t.Fatal(err)
	}
	assertPaths(t, changes, "tracked.txt")

	writeFile(t, filepath.Join(product, "untracked.txt"), "untracked\n")
	if _, err := WorkingChanges(product); err == nil || !strings.Contains(err.Error(), "untracked files") {
		t.Fatalf("WorkingChanges() error = %v, want untracked-file failure", err)
	}
}

func nestedRepository(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	product := filepath.Join(repo, "tools", "product")
	writeFile(t, filepath.Join(product, "tracked.txt"), "initial\n")
	git(t, repo, "init")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test User")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "initial")
	return repo, product
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

func assertPaths(t *testing.T, changes []Change, want ...string) {
	t.Helper()
	if len(changes) != len(want) {
		t.Fatalf("changes = %#v, want paths %v", changes, want)
	}
	for index := range want {
		if changes[index].Path != want[index] {
			t.Fatalf("changes[%d].Path = %q, want %q", index, changes[index].Path, want[index])
		}
	}
}
