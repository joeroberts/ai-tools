package implementation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportAuditRedactsSensitiveRunReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	run := Run{ID: "run-1234567890abcdef", WorkItemKey: "CG-2", Adapter: "headless-codex", State: StateRunning, ResultRef: "/private/result.json", TaskID: "secret-thread"}
	if err := ExportAudit(path, run); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-thread") || strings.Contains(string(data), "/private") {
		t.Fatalf("audit leaked sensitive reference: %s", data)
	}
	if err := ExportAudit(path, run); !os.IsExist(err) {
		t.Fatalf("overwrite error = %v", err)
	}
}

func TestRunMetricsRedactsRunReferences(t *testing.T) {
	run := Run{ID: "run-1234567890abcdef", State: StateRunning, TaskID: "secret-thread", ResultRef: "/private/result", CreatedAt: time.Unix(100, 0)}
	metrics := RunMetrics(run, time.Unix(105, 0))
	if metrics.Elapsed != 5*time.Second || metrics.RunID != run.ID || metrics.Attempts != 0 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}
