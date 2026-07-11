package initializer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"codex-governance/internal/assets"
)

const templateDestination = "docs/governance/templates"

func Initialize(root string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repo root is not a directory: %s", root)
	}

	files, err := templateFiles()
	if err != nil {
		return nil, err
	}
	for destination := range files {
		if _, err := os.Stat(filepath.Join(root, destination)); err == nil {
			return nil, fmt.Errorf("refusing to overwrite %s", destination)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	created := make([]string, 0, len(files)+1)
	for destination, source := range files {
		path := filepath.Join(root, destination)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		data, err := fs.ReadFile(assets.Templates, source)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
		created = append(created, destination)
	}

	decisions := filepath.Join(root, "docs", "decisions")
	if err := os.MkdirAll(decisions, 0o755); err != nil {
		return nil, err
	}
	created = append(created, "docs/decisions/")
	sort.Strings(created)
	return created, nil
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
