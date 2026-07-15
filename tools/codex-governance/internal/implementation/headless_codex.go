package implementation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// HeadlessCodexAdapter supervises a non-ephemeral Codex process in one
// disposable worktree. It intentionally returns unknown after a process-map
// loss; the control plane then escalates instead of starting another agent.
type HeadlessCodexAdapter struct {
	Binary    string
	WorkDir   string
	ResultDir string

	mu    sync.Mutex
	tasks map[string]*codexTask
}

// codexThreadID remains available for parsing archived Codex JSON event logs.
func codexThreadID(line []byte) (string, bool) {
	var event struct {
		Type     string `json:"type"`
		ThreadID string `json:"thread_id"`
	}
	if err := json.Unmarshal(line, &event); err != nil || event.Type != "thread.started" || event.ThreadID == "" {
		return "", false
	}
	return event.ThreadID, true
}

type codexTask struct {
	command    *exec.Cmd
	resultPath string
	stderrPath string
	stderr     *os.File
	stdout     *os.File
	done       bool
	err        error
}

func NewHeadlessCodexAdapter(binary, workDir, resultDir string) *HeadlessCodexAdapter {
	return &HeadlessCodexAdapter{Binary: binary, WorkDir: workDir, ResultDir: resultDir, tasks: map[string]*codexTask{}}
}

func (a *HeadlessCodexAdapter) Start(bundle TaskBundle) (string, error) {
	if a.Binary == "" || a.WorkDir == "" || a.ResultDir == "" {
		return "", fmt.Errorf("headless Codex adapter is incomplete")
	}
	if err := os.MkdirAll(a.ResultDir, 0o700); err != nil {
		return "", err
	}
	resultPath := filepath.Join(a.ResultDir, fmt.Sprintf("codex-%d-result.json", time.Now().UTC().UnixNano()))
	stderrPath := resultPath + ".stderr.log"
	schemaPath := filepath.Join(a.ResultDir, fmt.Sprintf("codex-%d-schema.json", time.Now().UTC().UnixNano()))
	if err := os.WriteFile(schemaPath, []byte(codexResultSchema), 0o600); err != nil {
		return "", err
	}
	command := exec.Command(a.Binary, "--ask-for-approval", "never", "exec", "--ephemeral", "--sandbox", "workspace-write", "--output-schema", schemaPath, "--output-last-message", resultPath, headlessPrompt(bundle))
	command.Dir = filepath.Clean(a.WorkDir)
	stderr, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	command.Stderr = stderr
	stdout, err := os.OpenFile(resultPath+".stdout.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = stderr.Close()
		return "", err
	}
	command.Stdout = stdout
	if err := command.Start(); err != nil {
		_ = stderr.Close()
		_ = stdout.Close()
		return "", err
	}
	taskID := fmt.Sprintf("codex-%d", command.Process.Pid)
	a.mu.Lock()
	a.tasks[taskID] = &codexTask{command: command, resultPath: resultPath, stderrPath: stderrPath, stderr: stderr, stdout: stdout}
	a.mu.Unlock()
	go a.wait(taskID, command)
	return taskID, nil
}

func (a *HeadlessCodexAdapter) Status(taskID string) (AdapterStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	if !ok {
		return AdapterUnknown, nil
	}
	if !task.done {
		return AdapterRunning, nil
	}
	if task.err != nil {
		return AdapterFailed, nil
	}
	return AdapterCompleted, nil
}

func (a *HeadlessCodexAdapter) ProcessID(taskID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if task, ok := a.tasks[taskID]; ok && task.command.Process != nil {
		return task.command.Process.Pid
	}
	return 0
}

func (a *HeadlessCodexAdapter) ResultReference(taskID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if task, ok := a.tasks[taskID]; ok {
		return task.resultPath
	}
	return ""
}

func (a *HeadlessCodexAdapter) Cancel(taskID string) error {
	a.mu.Lock()
	task, ok := a.tasks[taskID]
	a.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown Codex task")
	}
	if task.done {
		return nil
	}
	return task.command.Process.Kill()
}

func (a *HeadlessCodexAdapter) Result(taskID string) ([]byte, error) {
	a.mu.Lock()
	task, ok := a.tasks[taskID]
	a.mu.Unlock()
	if !ok || !task.done || task.err != nil {
		return nil, fmt.Errorf("Codex task result is unavailable")
	}
	return os.ReadFile(filepath.Clean(task.resultPath))
}

func (a *HeadlessCodexAdapter) wait(taskID string, command *exec.Cmd) {
	err := command.Wait()
	a.mu.Lock()
	defer a.mu.Unlock()
	if task, ok := a.tasks[taskID]; ok {
		_ = task.stderr.Close()
		_ = task.stdout.Close()
		task.done, task.err = true, err
	}
}

func headlessPrompt(bundle TaskBundle) string {
	data, _ := json.Marshal(bundle)
	return "Perform only the approved implementation task in this bundle. Remain within allowed paths. Do not push, create a pull request, access secrets, or modify remote state. Return only the required JSON result.\n\nTASK BUNDLE:\n" + string(data)
}

const codexResultSchema = `{"type":"object","properties":{"status":{"enum":["complete","blocked","escalated"]},"summary":{"type":"string"},"changed_paths":{"type":"array","items":{"type":"string"}},"validation_results":{"type":"array","items":{"type":"string"}},"blockers":{"type":"array","items":{"type":"string"}}},"required":["status","summary","changed_paths","validation_results","blockers"],"additionalProperties":false}`
