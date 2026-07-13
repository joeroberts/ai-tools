package ticketplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WorkflowState struct {
	RepositoryRoot string    `json:"repository_root"`
	PlanPath       string    `json:"plan_path"`
	PlanDigest     string    `json:"plan_digest"`
	Sources        Sources   `json:"sources"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updated_at"`
	Findings       []string  `json:"findings,omitempty"`
	ApprovedBy     string    `json:"approved_by,omitempty"`
	ApprovedAt     time.Time `json:"approved_at,omitempty"`
}

func NewWorkflowState(repoRoot, planPath, planDigest, status string, sources Sources) (WorkflowState, error) {
	canonicalRoot, err := canonicalRepositoryRoot(repoRoot)
	if err != nil {
		return WorkflowState{}, err
	}
	return WorkflowState{RepositoryRoot: canonicalRoot, PlanPath: planPath, PlanDigest: planDigest, Sources: sources, Status: status}, nil
}

func LoadWorkflow(path string) (WorkflowState, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return WorkflowState{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var state WorkflowState
	if err := decoder.Decode(&state); err != nil {
		return WorkflowState{}, fmt.Errorf("parse workflow state: %w", err)
	}
	if decoder.More() {
		return WorkflowState{}, fmt.Errorf("parse workflow state: multiple JSON values")
	}
	if err := state.Validate(); err != nil {
		return WorkflowState{}, err
	}
	return state, nil
}
func SaveWorkflow(path string, state WorkflowState) error {
	canonicalRoot, err := canonicalRepositoryRoot(state.RepositoryRoot)
	if err != nil {
		return err
	}
	state.RepositoryRoot = canonicalRoot
	if err := state.Validate(); err != nil {
		return err
	}
	if existing, err := LoadWorkflow(path); err == nil && !validTransition(existing.Status, state.Status) {
		return fmt.Errorf("workflow state transition from %s to %s is invalid", existing.Status, state.Status)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func (s WorkflowState) Validate() error {
	canonicalRoot, err := canonicalRepositoryRoot(s.RepositoryRoot)
	if err != nil || s.RepositoryRoot != canonicalRoot {
		return fmt.Errorf("workflow state requires a canonical repository root")
	}
	if s.PlanPath == "" || s.PlanDigest == "" || !digestPattern.MatchString(s.PlanDigest) {
		return fmt.Errorf("workflow state requires a plan path and sha256 plan digest")
	}
	if !oneOf(s.Status, "draft", "needs-remediation", "ready-for-approval", "approved", "escalated") {
		return fmt.Errorf("workflow state status is invalid")
	}
	if s.Status == "approved" && (s.ApprovedBy == "" || s.ApprovedAt.IsZero()) {
		return fmt.Errorf("approved workflow state requires approver and approval time")
	}
	if s.Status != "approved" && (s.ApprovedBy != "" || !s.ApprovedAt.IsZero()) {
		return fmt.Errorf("only an approved workflow state may include approval metadata")
	}
	plan, err := Load(s.PlanPath)
	if err != nil {
		return fmt.Errorf("workflow state does not reference a loadable plan: %w", err)
	}
	if issues := plan.Validate(); len(issues) != 0 {
		return fmt.Errorf("workflow state does not reference a valid plan: %v", issues)
	}
	if plan.Status != s.Status {
		return fmt.Errorf("workflow state status does not match plan status")
	}
	digest, err := FileDigest(s.PlanPath)
	if err != nil || digest != s.PlanDigest {
		return fmt.Errorf("workflow state does not match the current plan")
	}
	if plan.Sources != s.Sources {
		return fmt.Errorf("workflow state sources do not match the plan")
	}
	if issues := plan.ValidateAgainst(s.RepositoryRoot); len(issues) != 0 {
		return fmt.Errorf("workflow state source verification failed: %v", issues)
	}
	return nil
}

func canonicalRepositoryRoot(repoRoot string) (string, error) {
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required")
	}
	root, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root must be a directory")
	}
	return root, nil
}

func validTransition(from, to string) bool {
	return from == to || (from == "draft" && oneOf(to, "needs-remediation", "ready-for-approval", "escalated")) || (from == "needs-remediation" && oneOf(to, "ready-for-approval", "escalated")) || (from == "ready-for-approval" && oneOf(to, "approved", "escalated"))
}
func Approve(planPath, workflowPath, approvedBy string) error {
	approvedBy = strings.TrimSpace(approvedBy)
	if approvedBy == "" {
		return fmt.Errorf("approver is required")
	}
	plan, err := Load(planPath)
	if err != nil {
		return fmt.Errorf("load ticket plan: %w", err)
	}
	state, err := LoadWorkflow(workflowPath)
	if err != nil {
		return fmt.Errorf("load workflow state: %w", err)
	}
	if !samePath(planPath, state.PlanPath) {
		return fmt.Errorf("workflow state references a different ticket plan")
	}
	if plan.Status != "ready-for-approval" || state.Status != "ready-for-approval" {
		return fmt.Errorf("ticket plan must be ready-for-approval before approval")
	}
	if err := writeApprovedPlan(planPath, &plan); err != nil {
		return err
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		return fmt.Errorf("digest approved ticket plan: %w", err)
	}
	state.Status = "approved"
	state.PlanDigest = digest
	state.ApprovedBy = approvedBy
	state.ApprovedAt = time.Now().UTC()
	return writeWorkflow(workflowPath, state)
}

func writeApprovedPlan(path string, plan *Plan) error {
	plan.Status = "approved"
	if issues := plan.Validate(); len(issues) != 0 {
		return fmt.Errorf("approved ticket plan is invalid: %v", issues)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write approved ticket plan: %w", err)
	}
	return nil
}

// writeWorkflow validates the final state without rereading the prior state.
// Approval updates the plan digest and workflow status together, so a normal
// SaveWorkflow transition check would observe an intentionally stale peer.
func writeWorkflow(path string, state WorkflowState) error {
	state.UpdatedAt = time.Now().UTC()
	if err := state.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(filepath.Clean(left))
	rightAbs, rightErr := filepath.Abs(filepath.Clean(right))
	return leftErr == nil && rightErr == nil && leftAbs == rightAbs
}
