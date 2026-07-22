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

func TestLaunchSupervisorWaitsForTerminalResult(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "codex")
	script := "#!/bin/sh\nresult=\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--output-last-message\" ]; then shift; result=$1; break; fi\n  shift\ndone\nsleep 0.15\nprintf '%s' '{\"status\":\"complete\"}' > \"$result\"\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	run := Run{ID: "run-foreground-wait", State: StateQueued}
	started := time.Now()
	if _, err := launchSupervisor(&run, TaskBundle{}, t.TempDir(), t.TempDir(), binary); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 100*time.Millisecond {
		t.Fatalf("launchSupervisor returned before child result: %s", elapsed)
	}
	if run.State != StateImplementationComplete {
		t.Fatalf("state = %s", run.State)
	}
	if data, err := os.ReadFile(run.ResultRef); err != nil || string(data) != `{"status":"complete"}` {
		t.Fatalf("result = %q, %v", data, err)
	}
}
