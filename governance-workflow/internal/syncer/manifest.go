package syncer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Adoption struct {
	Release       string
	SourceCommit  string
	FormatVersion int
}

type Manifest struct {
	Release            string            `json:"release"`
	SourceCommit       string            `json:"source_commit"`
	FormatVersion      int               `json:"format_version"`
	CompatibilityRange string            `json:"compatibility_range"`
	Artifacts          map[string]string `json:"artifacts"`
	Changelog          string            `json:"changelog"`
	MigrationNotes     string            `json:"migration_notes"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse release manifest: %w", err)
	}
	if manifest.Release == "" || manifest.SourceCommit == "" || manifest.FormatVersion < 1 || manifest.CompatibilityRange == "" || manifest.Changelog == "" || manifest.MigrationNotes == "" || len(manifest.Artifacts) == 0 {
		return Manifest{}, fmt.Errorf("release manifest is incomplete")
	}
	for path, digest := range manifest.Artifacts {
		if path == "" || !strings.HasPrefix(digest, "sha256:") || len(digest) != 71 {
			return Manifest{}, fmt.Errorf("release manifest artifact is invalid")
		}
	}
	return manifest, nil
}

func VerifyArtifacts(manifest Manifest, root string) []string {
	var issues []string
	for path, expected := range manifest.Artifacts {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("artifact unavailable: %s", path))
			continue
		}
		sum := sha256.Sum256(data)
		if "sha256:"+fmt.Sprintf("%x", sum) != expected {
			issues = append(issues, fmt.Sprintf("artifact digest mismatch: %s", path))
		}
	}
	return issues
}

func Compare(adoption Adoption, manifest Manifest) []string {
	var changes []string
	if adoption.Release != manifest.Release {
		changes = append(changes, fmt.Sprintf("adopt release %s", manifest.Release))
	}
	if adoption.SourceCommit != manifest.SourceCommit {
		changes = append(changes, "update upstream source commit")
	}
	if adoption.FormatVersion != manifest.FormatVersion {
		changes = append(changes, fmt.Sprintf("migrate format version to %d", manifest.FormatVersion))
	}
	return changes
}
