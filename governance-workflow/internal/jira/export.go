package jira

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type OfflineExport struct {
	CapturedAt string `json:"captured_at"`
	Story      Issue  `json:"story"`
	Subtask    Issue  `json:"subtask"`
}

type Issue struct {
	Key                string `json:"key"`
	URL                string `json:"url"`
	UpdatedAt          string `json:"updated_at"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
}

func LoadOfflineExport(path string) (OfflineExport, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return OfflineExport{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var export OfflineExport
	if err := decoder.Decode(&export); err != nil {
		return OfflineExport{}, fmt.Errorf("parse offline Jira export: %w", err)
	}
	if export.Story.Key == "" || export.Subtask.Key == "" || export.Story.URL == "" || export.Subtask.URL == "" || export.Story.Description == "" || export.Subtask.Description == "" || export.Story.AcceptanceCriteria == "" || export.Subtask.AcceptanceCriteria == "" {
		return OfflineExport{}, fmt.Errorf("offline Jira export is incomplete")
	}
	for _, value := range []string{export.CapturedAt, export.Story.UpdatedAt, export.Subtask.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return OfflineExport{}, fmt.Errorf("offline Jira export timestamp is invalid")
		}
	}
	return export, nil
}

func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
