package implementation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// AuditRecord is intentionally compact and excludes task prompts, result
// content, local paths, credentials, and model output.
type AuditRecord struct {
	FormatVersion int       `json:"format_version"`
	ExportedAt    time.Time `json:"exported_at"`
	RunID         string    `json:"run_id"`
	WorkItemKey   string    `json:"work_item_key"`
	Adapter       string    `json:"adapter"`
	State         string    `json:"state"`
	BaseSHA       string    `json:"base_sha"`
	Branch        string    `json:"branch"`
	Attempts      int       `json:"attempts"`
	ReviewCycles  int       `json:"review_cycles"`
}

// Metrics is a redacted, per-run operational snapshot suitable for local
// dashboards or aggregation without exposing prompts or agent result content.
type Metrics struct {
	RunID        string        `json:"run_id"`
	State        string        `json:"state"`
	Attempts     int           `json:"attempts"`
	ReviewCycles int           `json:"review_cycles"`
	Elapsed      time.Duration `json:"elapsed_nanoseconds"`
}

func RunMetrics(run Run, now time.Time) Metrics {
	elapsed := now.Sub(run.CreatedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	return Metrics{RunID: run.ID, State: run.State, Attempts: run.Attempts, ReviewCycles: run.ReviewCycles, Elapsed: elapsed}
}

func ExportAudit(path string, run Run) error {
	record := AuditRecord{FormatVersion: FormatVersion, ExportedAt: time.Now().UTC(), RunID: run.ID, WorkItemKey: run.WorkItemKey, Adapter: run.Adapter, State: run.State, BaseSHA: run.BaseSHA, Branch: run.Branch, Attempts: run.Attempts, ReviewCycles: run.ReviewCycles}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}
