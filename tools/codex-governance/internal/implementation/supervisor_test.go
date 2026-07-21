package implementation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReconcileSupervisorRejectsStaleProcessIdentity(t *testing.T) {
	root := t.TempDir()
	path := supervisorPath(root, "run-supervisor-test")
	record := SupervisorRecord{FormatVersion: 1, RunID: "run-supervisor-test", State: "running", PID: os.Getpid(), ProcessStart: "original", ResultPath: filepath.Join(root, "result.json")}
	if err := writeSupervisor(path, record); err != nil {
		t.Fatal(err)
	}
	previous := supervisorProcessStart
	supervisorProcessStart = func(int) (string, error) { return "reused", nil }
	t.Cleanup(func() { supervisorProcessStart = previous })
	run := Run{State: StateRunning, ID: record.RunID, SupervisorRef: path}
	if err := reconcileSupervisor(&run); err != nil {
		t.Fatal(err)
	}
	if run.State != StateEscalated {
		t.Fatalf("state = %s", run.State)
	}
	updated, err := loadSupervisor(path)
	if err != nil || updated.State != "failed" {
		t.Fatalf("record = %#v, %v", updated, err)
	}
}

func TestReconcileSupervisorPublishesValidTerminalResult(t *testing.T) {
	root := t.TempDir()
	result := filepath.Join(root, "result.json")
	if err := os.WriteFile(result, []byte(`{"status":"complete"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	path := supervisorPath(root, "run-terminal-test")
	record := SupervisorRecord{FormatVersion: 1, RunID: "run-terminal-test", State: "running", PID: 999999, ProcessStart: "original", ResultPath: result, StartedAt: time.Now().UTC()}
	if err := writeSupervisor(path, record); err != nil {
		t.Fatal(err)
	}
	run := Run{State: StateRunning, ID: record.RunID, SupervisorRef: path}
	if err := reconcileSupervisor(&run); err != nil {
		t.Fatal(err)
	}
	if run.State != StateImplementationComplete {
		t.Fatalf("state = %s", run.State)
	}
	updated, err := loadSupervisor(path)
	if err != nil || updated.State != "complete" {
		t.Fatalf("record = %#v, %v", updated, err)
	}
}
