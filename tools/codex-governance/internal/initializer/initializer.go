package initializer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"codex-governance/internal/assets"
)

const templateDestination = "docs/governance/templates"

type Outcome struct {
	Path   string
	Action string
}

// Preview reports the non-destructive initialization result without changing
// the repository. Existing repository-owned paths are merge-required.
func Preview(root string) ([]Outcome, error) {
	root, err := validRoot(root)
	if err != nil {
		return nil, err
	}
	files, err := templateFiles()
	if err != nil {
		return nil, err
	}
	outcomes := make([]Outcome, 0, len(files)+1)
	for destination := range files {
		action := "create"
		exists, err := destinationExists(root, destination)
		if err != nil {
			return nil, err
		}
		if exists {
			action = "merge-required"
		}
		outcomes = append(outcomes, Outcome{Path: destination, Action: action})
	}
	decisions, err := destinationInfo(root, "docs/decisions")
	if err != nil {
		return nil, err
	}
	if decisions == nil {
		outcomes = append(outcomes, Outcome{Path: "docs/decisions/", Action: "create"})
	} else if decisions.IsDir() {
		outcomes = append(outcomes, Outcome{Path: "docs/decisions/", Action: "existing"})
	} else {
		outcomes = append(outcomes, Outcome{Path: "docs/decisions/", Action: "conflict"})
	}
	sort.Slice(outcomes, func(i, j int) bool { return outcomes[i].Path < outcomes[j].Path })
	return outcomes, nil
}

func Initialize(root string) ([]string, error) {
	root, err := validRoot(root)
	if err != nil {
		return nil, err
	}
	preview, err := Preview(root)
	if err != nil {
		return nil, err
	}
	for _, outcome := range preview {
		if outcome.Action == "merge-required" || outcome.Action == "conflict" {
			return nil, fmt.Errorf("merge required; refusing to overwrite %s", outcome.Path)
		}
	}
	files, err := templateFiles()
	if err != nil {
		return nil, err
	}
	created := make([]string, 0, len(files)+1)
	for destination, source := range files {
		path := filepath.Join(root, destination)
		if err := ensureParentDirectories(root, destination); err != nil {
			return nil, err
		}
		data, err := fs.ReadFile(assets.Templates, source)
		if err != nil {
			return nil, err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return nil, err
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		created = append(created, destination)
	}

	if err := ensureDirectory(root, "docs/decisions"); err != nil {
		return nil, err
	}
	created = append(created, "docs/decisions/")
	sort.Strings(created)
	return created, nil
}

func validRoot(root string) (string, error) {
	resolved, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repo root is not a directory: %s", resolved)
	}
	return resolved, nil
}

func destinationExists(root, destination string) (bool, error) {
	info, err := destinationInfo(root, destination)
	return info != nil, err
}

func destinationInfo(root, destination string) (os.FileInfo, error) {
	current := root
	parts := splitPath(destination)
	for index, component := range parts {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing symlink in initialization path: %s", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("initialization path component is not a directory: %s", current)
		}
	}
	return os.Lstat(filepath.Join(root, destination))
}

func ensureParentDirectories(root, destination string) error {
	parts := splitPath(destination)
	if len(parts) < 2 {
		return nil
	}
	return ensureDirectory(root, filepath.Join(parts[:len(parts)-1]...))
}

func ensureDirectory(root, destination string) error {
	current := root
	for _, component := range splitPath(destination) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o755); err != nil && !os.IsExist(err) {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("initialization path component is not a directory: %s", current)
		}
	}
	return nil
}

func splitPath(path string) []string {
	return strings.FieldsFunc(filepath.Clean(path), func(r rune) bool { return r == filepath.Separator })
}

func templateFiles() (map[string]string, error) {
	files := make(map[string]string)
	err := fs.WalkDir(assets.Templates, "templates", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}
		destination := filepath.Join(templateDestination, relative)
		if relative == "governance.yml" {
			destination = "governance.yml"
		}
		files[destination] = path
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
