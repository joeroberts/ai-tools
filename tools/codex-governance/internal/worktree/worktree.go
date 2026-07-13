package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Create prepares a detached, disposable worktree at an empty path. Callers
// own cleanup; this package never removes a worktree automatically.
func Create(repoRoot, baseSHA, path string) error {
	if baseSHA == "" || path == "" {
		return fmt.Errorf("worktree base SHA and path are required")
	}
	_, err := os.Stat(path)
	if err == nil || !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("worktree path already exists: %s", path)
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	command := exec.Command("git", "worktree", "add", "--detach", path, baseSHA)
	command.Dir = filepath.Clean(repoRoot)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("create disposable worktree: %w: %s", err, output)
	}
	return nil
}
