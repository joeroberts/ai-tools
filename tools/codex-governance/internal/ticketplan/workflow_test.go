package ticketplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWorkflowRejectsInvalidState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(path, []byte(`{"plan_path":"plan.json","plan_digest":"bad","status":"ready-for-approval"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkflow(path); err == nil {
		t.Fatal("LoadWorkflow accepted an invalid digest")
	}
}

func TestApproveDoesNotMutatePlanOrWorkflow(t *testing.T) {
	root, plan := validPlan(t)
	plan.Status = "ready-for-approval"
	planPath := filepath.Join(root, "plan.json")
	planData, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, planData, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "workflow.json")
	if err := SaveWorkflow(workflowPath, newWorkflowState(t, root, planPath, digest, "ready-for-approval")); err != nil {
		t.Fatal(err)
	}
	beforePlan, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeWorkflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Approve(planPath, workflowPath); err == nil || !strings.Contains(err.Error(), "unavailable until Phase 3") {
		t.Fatalf("Approve() error = %v", err)
	}
	afterPlan, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	afterWorkflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterPlan) != string(beforePlan) || string(afterWorkflow) != string(beforeWorkflow) {
		t.Fatal("Approve() mutated the plan or workflow state")
	}
}

func TestApproveRejectsWorkflowForDifferentPlan(t *testing.T) {
	root, plan := validPlan(t)
	plan.Status = "ready-for-approval"
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	otherPath := filepath.Join(root, "other.json")
	if err := os.WriteFile(otherPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "workflow.json")
	if err := SaveWorkflow(workflowPath, newWorkflowState(t, root, otherPath, digest, "ready-for-approval")); err != nil {
		t.Fatal(err)
	}
	if err := Approve(planPath, workflowPath); err == nil {
		t.Fatal("Approve accepted a workflow state for a different plan")
	}
}

func TestSaveWorkflowRejectsInvalidTransition(t *testing.T) {
	root, plan := validPlan(t)
	plan.Status = "ready-for-approval"
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "workflow.json")
	if err := SaveWorkflow(workflowPath, newWorkflowState(t, root, planPath, digest, "ready-for-approval")); err != nil {
		t.Fatal(err)
	}
	if err := SaveWorkflow(workflowPath, newWorkflowState(t, root, planPath, digest, "draft")); err == nil {
		t.Fatal("SaveWorkflow accepted a backward transition")
	}
}

func TestSaveWorkflowRejectsPlanWithDifferentStatus(t *testing.T) {
	root, plan := validPlan(t)
	planPath := filepath.Join(root, "plan.json")
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	err = SaveWorkflow(filepath.Join(root, "workflow.json"), newWorkflowState(t, root, planPath, digest, "ready-for-approval"))
	if err == nil || !strings.Contains(err.Error(), "does not match plan status") {
		t.Fatalf("SaveWorkflow() error = %v", err)
	}
}

func TestLoadWorkflowRejectsChangedSource(t *testing.T) {
	root, plan := validPlan(t)
	plan.Status = "ready-for-approval"
	planPath := filepath.Join(root, "plan.json")
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "workflow.json")
	if err := SaveWorkflow(workflowPath, newWorkflowState(t, root, planPath, digest, "ready-for-approval")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "prd.md"), []byte("# Replaced\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkflow(workflowPath); err == nil || !strings.Contains(err.Error(), "source verification failed") {
		t.Fatalf("LoadWorkflow() error = %v", err)
	}
}

func TestSaveWorkflowRejectsMismatchedPersistedSources(t *testing.T) {
	root, plan := validPlan(t)
	planPath := filepath.Join(root, "plan.json")
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	state := newWorkflowState(t, root, planPath, digest, "draft")
	state.Sources.PRD.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := SaveWorkflow(filepath.Join(root, "workflow.json"), state); err == nil || !strings.Contains(err.Error(), "sources do not match") {
		t.Fatalf("SaveWorkflow() error = %v", err)
	}
}

func newWorkflowState(t *testing.T, root, planPath, digest, status string) WorkflowState {
	t.Helper()
	plan, err := Load(planPath)
	if err != nil {
		t.Fatal(err)
	}
	state, err := NewWorkflowState(root, planPath, digest, status, plan.Sources)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestLoadWorkflowRejectsInvalidReferencedPlan(t *testing.T) {
	root := t.TempDir()
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, []byte(`{"format_version":1,"status":"draft"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(root, "workflow.json")
	workflow, err := NewWorkflowState(root, planPath, digest, "draft", Sources{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := json.Marshal(workflow)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflowPath, state, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkflow(workflowPath); err == nil || !strings.Contains(err.Error(), "does not reference a valid plan") {
		t.Fatalf("LoadWorkflow() error = %v", err)
	}
}
