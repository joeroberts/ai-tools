package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/jira"
	gruntime "codex-governance/internal/runtime"
	"codex-governance/internal/validate"
	"codex-governance/internal/workitem"
	"codex-governance/internal/worktree"
)

const FormatVersion = 1

const (
	StatePreflight              = "preflight"
	StateQueued                 = "queued"
	StateRunning                = "running"
	StateImplementationComplete = "implementation-complete"
	StateReview                 = "review"
	StateVerification           = "verification"
	StateRemediation            = "remediation"
	StateReadyToCommit          = "ready-to-commit"
	StateLocallyCommitted       = "locally-committed"
	StateReadyForRemoteApproval = "ready-for-remote-approval"
	StatePushed                 = "pushed"
	StatePRCreated              = "PR-created"
	StateEscalated              = "escalated"
	StateClosed                 = "closed"
)

type Run struct {
	FormatVersion    int       `json:"format_version"`
	ID               string    `json:"id"`
	WorkItemKey      string    `json:"work_item_key"`
	Adapter          string    `json:"adapter"`
	State            string    `json:"state"`
	BaseSHA          string    `json:"base_sha"`
	Branch           string    `json:"branch"`
	TaskBundleDigest string    `json:"task_bundle_digest"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Attempts         int       `json:"attempts"`
	ReviewCycles     int       `json:"review_cycles"`
	TaskID           string    `json:"task_id"`
	ProcessID        int       `json:"process_id"`
	ResultRef        string    `json:"result_ref"`
	CommitSHA        string    `json:"commit_sha"`
	PullRequestURL   string    `json:"pull_request_url"`
}

type AdapterStatus string

const (
	AdapterRunning   AdapterStatus = "running"
	AdapterCompleted AdapterStatus = "completed"
	AdapterFailed    AdapterStatus = "failed"
	AdapterUnknown   AdapterStatus = "unknown"
)

// Adapter is deliberately read-only in Phase 3. Later phases add a disposable
// worktree to the adapter contract before any real execution provider exists.
type Adapter interface {
	Start(TaskBundle) (string, error)
	Status(string) (AdapterStatus, error)
	Cancel(string) error
	Result(string) ([]byte, error)
}

type ProcessMetadata interface{ ProcessID(string) int }

type ResultReference interface{ ResultReference(string) string }

type FakeAdapter struct {
	mu    sync.Mutex
	next  int
	tasks map[string]fakeTask
}

type fakeTask struct {
	status AdapterStatus
	result []byte
}

func NewFakeAdapter() *FakeAdapter { return &FakeAdapter{tasks: map[string]fakeTask{}} }

func (a *FakeAdapter) Start(_ TaskBundle) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.next++
	id := fmt.Sprintf("fake-%d", a.next)
	a.tasks[id] = fakeTask{status: AdapterRunning}
	return id, nil
}

func (a *FakeAdapter) Status(taskID string) (AdapterStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	if !ok {
		return AdapterUnknown, nil
	}
	return task.status, nil
}

func (a *FakeAdapter) Result(taskID string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	if !ok || task.status != AdapterCompleted {
		return nil, fmt.Errorf("fake task result is unavailable")
	}
	return append([]byte(nil), task.result...), nil
}

func (a *FakeAdapter) Cancel(taskID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	if !ok {
		return fmt.Errorf("unknown fake task")
	}
	task.status = AdapterFailed
	a.tasks[taskID] = task
	return nil
}

func (a *FakeAdapter) Complete(taskID string, result []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	if !ok {
		return fmt.Errorf("unknown fake task")
	}
	task.status, task.result = AdapterCompleted, append([]byte(nil), result...)
	a.tasks[taskID] = task
	return nil
}

func Launch(run *Run, bundle TaskBundle, adapter Adapter) error {
	if run.State != StateQueued || run.Attempts >= 1 {
		return fmt.Errorf("implementation run cannot be dispatched")
	}
	taskID, err := adapter.Start(bundle)
	if err != nil {
		return err
	}
	if taskID == "" {
		return fmt.Errorf("adapter returned an empty task ID")
	}
	run.TaskID, run.Attempts = taskID, run.Attempts+1
	if metadata, ok := adapter.(ProcessMetadata); ok {
		run.ProcessID = metadata.ProcessID(taskID)
	}
	if metadata, ok := adapter.(ResultReference); ok {
		run.ResultRef = metadata.ResultReference(taskID)
	}
	return run.Transition(StateRunning)
}

func Reconcile(run *Run, adapter Adapter, resultPath string) error {
	if run.State != StateRunning || run.TaskID == "" {
		return fmt.Errorf("implementation run is not reconcilable")
	}
	status, err := adapter.Status(run.TaskID)
	if err != nil {
		return err
	}
	switch status {
	case AdapterRunning:
		return nil
	case AdapterCompleted:
		result, err := adapter.Result(run.TaskID)
		if err != nil {
			return err
		}
		if err := writePrivate(resultPath, result); err != nil {
			return err
		}
		run.ResultRef = resultPath
		return run.Transition(StateImplementationComplete)
	case AdapterFailed, AdapterUnknown:
		return run.Transition(StateEscalated)
	default:
		return fmt.Errorf("adapter returned invalid status %q", status)
	}
}

type TaskBundle struct {
	FormatVersion  int                `json:"format_version"`
	WorkItem       workitem.Item      `json:"work_item"`
	TicketBaseline jira.OfflineExport `json:"ticket_baseline"`
	AllowedPaths   []string           `json:"allowed_paths"`
	Commands       []string           `json:"commands"`
	ADR            string             `json:"adr"`
	Guidance       string             `json:"guidance"`
}

type PreflightRequest struct {
	WorkItemPath      string
	OfflineExportPath string
	RepoRoot          string
	RuntimeRoot       string
	Adapter           string
	BundlePath        string
	RunPath           string
}

type PreflightResult struct {
	Run        Run
	BundlePath string
}

func NewRun(item workitem.Item, adapter, bundleDigest string) (Run, error) {
	if item.Source.SubtaskKey == "" || item.GitRange.BaseSHA == "" || adapter == "" || bundleDigest == "" {
		return Run{}, fmt.Errorf("implementation run inputs are incomplete")
	}
	now := time.Now().UTC()
	idSource := strings.Join([]string{item.Source.SubtaskKey, item.GitRange.BaseSHA, adapter, bundleDigest}, "\x00")
	sum := sha256.Sum256([]byte(idSource))
	return Run{FormatVersion: FormatVersion, ID: "run-" + hex.EncodeToString(sum[:8]), WorkItemKey: item.Source.SubtaskKey, Adapter: adapter, State: StatePreflight, BaseSHA: item.GitRange.BaseSHA, Branch: "", TaskBundleDigest: bundleDigest, CreatedAt: now, UpdatedAt: now}, nil
}

func (r *Run) Transition(next string) error {
	if !allowedTransition(r.State, next) {
		return fmt.Errorf("invalid implementation-run transition %q -> %q", r.State, next)
	}
	r.State, r.UpdatedAt = next, time.Now().UTC()
	return nil
}

func allowedTransition(current, next string) bool {
	if next == StateEscalated {
		return current != StateClosed && current != StateEscalated
	}
	switch current {
	case StatePreflight:
		return next == StateQueued
	case StateQueued:
		return next == StateRunning
	case StateRunning:
		return next == StateImplementationComplete
	case StateImplementationComplete:
		return next == StateReview
	case StateReview, StateVerification:
		return next == StateRemediation || next == StateVerification || next == StateReadyToCommit
	case StateRemediation:
		return next == StateReview
	case StateReadyToCommit:
		return next == StateLocallyCommitted
	case StateLocallyCommitted:
		return next == StateReadyForRemoteApproval || next == StateClosed
	case StateReadyForRemoteApproval:
		return next == StatePushed
	case StatePushed:
		return next == StatePRCreated
	case StatePRCreated:
		return next == StateClosed
	default:
		return false
	}
}

func BuildTaskBundle(item workitem.Item, baseline jira.OfflineExport, repoRoot string) (TaskBundle, error) {
	guidance, err := readGuidance(repoRoot)
	if err != nil {
		return TaskBundle{}, err
	}
	return TaskBundle{FormatVersion: FormatVersion, WorkItem: item, TicketBaseline: baseline, AllowedPaths: append([]string(nil), item.Scope.AllowedPaths...), Commands: append([]string(nil), item.Scope.ValidationPlan...), ADR: item.Decision.ADR, Guidance: guidance}, nil
}

func WriteTaskBundle(path string, bundle TaskBundle) (string, error) {
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return digest(data), nil
}

func SaveRun(path string, run Run) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func LoadRun(path string) (Run, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Run{}, err
	}
	var run Run
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&run); err != nil {
		return Run{}, fmt.Errorf("parse implementation run: %w", err)
	}
	if run.FormatVersion != FormatVersion || run.ID == "" || run.WorkItemKey == "" || run.Adapter == "" || run.State == "" || run.TaskBundleDigest == "" {
		return Run{}, fmt.Errorf("implementation run is incomplete")
	}
	return run, nil
}

func LoadTaskBundle(path string) (TaskBundle, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return TaskBundle{}, err
	}
	var bundle TaskBundle
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return TaskBundle{}, fmt.Errorf("parse task bundle: %w", err)
	}
	if bundle.FormatVersion != FormatVersion || bundle.WorkItem.Source.SubtaskKey == "" {
		return TaskBundle{}, fmt.Errorf("task bundle is incomplete")
	}
	return bundle, nil
}

// StartHeadless creates the disposable worktree and starts an explicitly
// approved headless Codex process. It does not commit, push, or create a PR.
func StartHeadless(run *Run, bundle TaskBundle, repoRoot, runtimeRoot, codexBinary string) error {
	if run.Adapter != "headless-codex" || run.State != StatePreflight {
		return fmt.Errorf("run is not ready for headless Codex execution")
	}
	worktreePath := filepath.Join(runtimeRoot, "worktrees", run.ID)
	if err := worktree.Create(repoRoot, run.BaseSHA, worktreePath); err != nil {
		return err
	}
	if err := run.Transition(StateQueued); err != nil {
		return err
	}
	adapter := NewHeadlessCodexAdapter(codexBinary, worktreePath, filepath.Join(runtimeRoot, "runs", run.ID, "results"))
	return Launch(run, bundle, adapter)
}

// ReconcilePersisted is deliberately conservative: an unavailable process or
// result is escalation, never a new dispatch.
func ReconcilePersisted(run *Run) error {
	if run.State != StateRunning || run.ProcessID < 1 || run.ResultRef == "" {
		return fmt.Errorf("run lacks persisted process evidence")
	}
	process, err := os.FindProcess(run.ProcessID)
	if err == nil && process.Signal(syscall.Signal(0)) == nil {
		return nil
	}
	if info, err := os.Stat(run.ResultRef); err == nil && !info.IsDir() && info.Size() > 0 {
		return run.Transition(StateImplementationComplete)
	}
	return run.Transition(StateEscalated)
}

func Preflight(request PreflightRequest) (PreflightResult, error) {
	if request.Adapter == "" || request.BundlePath == "" || request.RunPath == "" {
		return PreflightResult{}, fmt.Errorf("preflight requires adapter, bundle path, and run path")
	}
	cfg, err := config.Load(filepath.Join(request.RepoRoot, "governance.yml"))
	if err != nil {
		return PreflightResult{}, fmt.Errorf("load governance config: %w", err)
	}
	if !cfg.AllowsAdapter(request.Adapter) {
		return PreflightResult{}, fmt.Errorf("execution adapter %q is not allowed by governance config", request.Adapter)
	}
	item, err := workitem.Load(request.WorkItemPath)
	if err != nil {
		return PreflightResult{}, err
	}
	baseline, err := jira.LoadOfflineExport(request.OfflineExportPath)
	if err != nil {
		return PreflightResult{}, err
	}
	violations, err := validate.Evaluate(item, baseline, request.RepoRoot, "", "")
	if err != nil {
		return PreflightResult{}, err
	}
	open, err := gruntime.OpenAgents(request.RuntimeRoot, item.Source.SubtaskKey)
	if err != nil {
		return PreflightResult{}, err
	}
	if len(open) != 0 {
		violations = append(violations, validate.Violation{Code: "open-agents", Message: "open agents block implementation preflight"})
	}
	if len(violations) != 0 {
		messages := make([]string, 0, len(violations))
		for _, violation := range violations {
			messages = append(messages, violation.Code+": "+violation.Message)
		}
		return PreflightResult{}, fmt.Errorf("preflight failed: %s", strings.Join(messages, "; "))
	}
	bundle, err := BuildTaskBundle(item, baseline, request.RepoRoot)
	if err != nil {
		return PreflightResult{}, err
	}
	bundleDigest, err := WriteTaskBundle(request.BundlePath, bundle)
	if err != nil {
		return PreflightResult{}, err
	}
	run, err := NewRun(item, request.Adapter, bundleDigest)
	if err != nil {
		return PreflightResult{}, err
	}
	if err := SaveRun(request.RunPath, run); err != nil {
		return PreflightResult{}, err
	}
	return PreflightResult{Run: run, BundlePath: request.BundlePath}, nil
}

func readGuidance(repoRoot string) (string, error) {
	path := filepath.Join(repoRoot, "AGENTS.md")
	data, err := os.ReadFile(filepath.Clean(path))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(data) > 64*1024 {
		return "", fmt.Errorf("repository guidance exceeds task-bundle limit")
	}
	return string(data), nil
}

func writePrivate(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), data, 0o600)
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
