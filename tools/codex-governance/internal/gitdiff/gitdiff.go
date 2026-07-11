package gitdiff

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Change struct {
	Path    string
	Added   int
	Deleted int
}

func Changes(repoRoot, baseSHA, headSHA string) ([]Change, error) {
	command := exec.Command("git", "diff", "--numstat", baseSHA+"..."+headSHA)
	command.Dir = filepath.Clean(repoRoot)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("read git diff %s...%s: %w", baseSHA, headSHA, err)
	}
	if len(output) == 0 {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	changes := make([]Change, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("parse git diff output: %q", line)
		}
		added, deleted := 0, 0
		if parts[0] != "-" {
			added, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, err
			}
			deleted, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, err
			}
		}
		changes = append(changes, Change{Path: parts[2], Added: added, Deleted: deleted})
	}
	return changes, nil
}
