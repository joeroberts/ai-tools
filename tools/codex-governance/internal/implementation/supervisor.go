package implementation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// SupervisorRecord is the durable, owner-only lifecycle record for one
// headless Codex child. It deliberately contains no prompt or result body.
type SupervisorRecord struct {
	FormatVersion int       `json:"format_version"`
	RunID         string    `json:"run_id"`
	State         string    `json:"state"`
	PID           int       `json:"pid"`
	ProcessStart  string    `json:"process_start"`
	StartedAt     time.Time `json:"started_at"`
	ResultPath    string    `json:"result_path"`
	StdoutPath    string    `json:"stdout_path"`
	StderrPath    string    `json:"stderr_path"`
	Failure       string    `json:"failure,omitempty"`
}

var supervisorProcessStart = func(pid int) (string, error) {
	output, err := exec.Command("ps", "-o", "lstart=", "-p", fmt.Sprint(pid)).Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return "", fmt.Errorf("read process start identity")
	}
	return strings.TrimSpace(string(output)), nil
}

func supervisorPath(runtimeRoot, runID string) string {
	return filepath.Join(runtimeRoot, "runs", runID, "supervisor", "state.json")
}

func writeSupervisor(path string, record SupervisorRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func loadSupervisor(path string) (SupervisorRecord, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SupervisorRecord{}, err
	}
	var record SupervisorRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return SupervisorRecord{}, err
	}
	if record.FormatVersion != 1 || record.RunID == "" || record.State == "" || record.ResultPath == "" {
		return SupervisorRecord{}, fmt.Errorf("supervisor record is incomplete")
	}
	return record, nil
}

func launchSupervisor(run *Run, bundle TaskBundle, workDir, runtimeRoot, binary string) ([]string, error) {
	path := supervisorPath(runtimeRoot, run.ID)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("supervisor record already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	resultDir := filepath.Join(runtimeRoot, "runs", run.ID, "results")
	if err := os.MkdirAll(resultDir, 0o700); err != nil {
		return nil, err
	}
	result := filepath.Join(resultDir, "result.json")
	stdout, stderr := result+".stdout.log", result+".stderr.log"
	schema := filepath.Join(resultDir, "schema.json")
	if err := os.WriteFile(schema, []byte(codexResultSchema), 0o600); err != nil {
		return nil, err
	}
	record := SupervisorRecord{FormatVersion: 1, RunID: run.ID, State: "launching", ResultPath: result, StdoutPath: stdout, StderrPath: stderr}
	if err := writeSupervisor(path, record); err != nil {
		return nil, err
	}
	out, err := os.OpenFile(stdout, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	errout, err := os.OpenFile(stderr, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = out.Close()
		return nil, err
	}
	command := exec.Command(binary, "--ask-for-approval", "never", "exec", "--ephemeral", "--sandbox", "workspace-write", "--output-schema", schema, "--output-last-message", result, headlessPrompt(bundle))
	command.Dir, command.Stdout, command.Stderr = filepath.Clean(workDir), out, errout
	if err := command.Start(); err != nil {
		_ = out.Close()
		_ = errout.Close()
		record.State, record.Failure = "failed", err.Error()
		_ = writeSupervisor(path, record)
		return []string{stdout, stderr}, err
	}
	_ = out.Close()
	_ = errout.Close()
	start, err := supervisorProcessStart(command.Process.Pid)
	if err != nil {
		_ = command.Process.Kill()
		record.State, record.Failure = "failed", "cannot capture child process identity"
		_ = writeSupervisor(path, record)
		return []string{stdout, stderr}, err
	}
	record.State, record.PID, record.ProcessStart, record.StartedAt = "running", command.Process.Pid, start, time.Now().UTC()
	if err := writeSupervisor(path, record); err != nil {
		_ = command.Process.Kill()
		return []string{stdout, stderr}, err
	}
	run.TaskID, run.ProcessID, run.ResultRef, run.SupervisorRef = fmt.Sprintf("supervisor-%d", record.PID), record.PID, result, path
	if err := run.Transition(StateRunning); err != nil {
		return []string{stdout, stderr}, err
	}
	return []string{stdout, stderr}, nil
}

func reconcileSupervisor(run *Run) error {
	record, err := loadSupervisor(run.SupervisorRef)
	if err != nil {
		return err
	}
	if record.RunID != run.ID {
		return fmt.Errorf("supervisor record run mismatch")
	}
	if record.State == "complete" {
		return run.Transition(StateImplementationComplete)
	}
	if record.State == "failed" {
		return run.Transition(StateEscalated)
	}
	if record.State != "running" || record.PID < 1 || record.ProcessStart == "" {
		return run.Transition(StateEscalated)
	}
	process, err := os.FindProcess(record.PID)
	if err == nil && process.Signal(syscall.Signal(0)) == nil {
		identity, identityErr := supervisorProcessStart(record.PID)
		if identityErr == nil && identity == record.ProcessStart {
			return nil
		}
		record.State, record.Failure = "failed", "stale or reused process identity"
		if err := writeSupervisor(run.SupervisorRef, record); err != nil {
			return err
		}
		return run.Transition(StateEscalated)
	}
	data, err := os.ReadFile(record.ResultPath)
	if err == nil && len(data) > 0 {
		var result codexResult
		if json.Unmarshal(data, &result) == nil && result.Status == "complete" {
			record.State = "complete"
			if err := writeSupervisor(run.SupervisorRef, record); err != nil {
				return err
			}
			return run.Transition(StateImplementationComplete)
		}
	}
	record.State, record.Failure = "failed", "child exited without a valid complete result"
	if err := writeSupervisor(run.SupervisorRef, record); err != nil {
		return err
	}
	return run.Transition(StateEscalated)
}
