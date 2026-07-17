package agentplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"codex-governance/internal/ollama"
	gruntime "codex-governance/internal/runtime"
	"codex-governance/internal/ticketplan"
)

type Request struct {
	PRDPath, SpecPath, RoadmapPath, OutputPath, RepoRoot, RuntimeRoot, ConstraintsPath string
	Progress                                                                           func(string)
	Context                                                                            context.Context
	ManagerTimeout, ManagerWaitDelay                                                   time.Duration
}

type Result struct{ PlanPath, WorkItem string }

var marshalDecomposition = json.MarshalIndent

var recordExecutionEvent = gruntime.Record

type Runner interface {
	Run(context.Context, string, string, string) ([]byte, error)
}

// Runners makes provider ownership explicit: only Manager may be hosted.
type Runners struct{ Manager, Reviewer, Verifier Runner }

type OllamaRunner struct {
	Policy ollama.Policy
	Model  string

	setResidency func(bool) error
}

func (r OllamaRunner) Run(_ context.Context, role, prompt, _ string) ([]byte, error) {
	if role != "reviewer" && role != "verifier" {
		return nil, fmt.Errorf("local Ollama runner is restricted to reviewer and verifier roles")
	}
	output, err := ollama.Run(ollama.Client(r.Policy), r.Policy, ollama.Request{Model: r.Model, Role: role, TaskType: "ticket-plan-review", Input: []byte(prompt)})
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

func (r OllamaRunner) SetResidency(loaded bool) error {
	if r.setResidency != nil {
		return r.setResidency(loaded)
	}
	return ollama.SetResidency(ollama.Client(r.Policy), r.Policy, r.Model, loaded)
}

type CodexRunner struct {
	Binary  string
	WorkDir string
}

func (r CodexRunner) Run(ctx context.Context, role, prompt, schema string) ([]byte, error) {
	return r.run(ctx, role, prompt, schema, "", 0)
}

func (r CodexRunner) run(ctx context.Context, role, prompt, schema, diagnosticsDir string, waitDelay time.Duration) ([]byte, error) {
	if role != "manager" {
		return nil, fmt.Errorf("hosted Codex runner is restricted to the manager role")
	}
	if diagnosticsDir != "" {
		return r.runWithDiagnostics(ctx, role, prompt, schema, diagnosticsDir, waitDelay)
	}
	dir, err := os.MkdirTemp("", "codex-governance-agent-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	schemaPath, outputPath := filepath.Join(dir, "schema.json"), filepath.Join(dir, "result.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0o600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, r.Binary, "--ask-for-approval", "never", "exec", "--ephemeral", "--sandbox", "read-only", "--skip-git-repo-check", "--output-schema", schemaPath, "--output-last-message", outputPath, prompt)
	cmd.Dir = r.WorkDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("Codex %s run: %w: %s", role, err, output)
	}
	return os.ReadFile(outputPath)
}

func (r CodexRunner) runWithDiagnostics(ctx context.Context, role, prompt, schema, dir string, waitDelay time.Duration) ([]byte, error) {
	if err := makePrivateDirectory(dir); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	schemaPath := filepath.Join(dir, "schema.json")
	resultPath := filepath.Join(dir, "result.json")
	stdoutPath := filepath.Join(dir, "codex.jsonl")
	stderrPath := filepath.Join(dir, "stderr.log")
	for _, path := range []string{schemaPath, resultPath, stdoutPath, stderrPath} {
		if _, err := os.Stat(path); err == nil {
			return nil, fmt.Errorf("refusing to overwrite manager diagnostic: %s", path)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if err := writeNewPrivateFile(schemaPath, []byte(schema)); err != nil {
		return nil, err
	}
	stdout, err := openNewPrivateFile(stdoutPath)
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	stderr, err := openNewPrivateFile(stderrPath)
	if err != nil {
		return nil, err
	}
	defer stderr.Close()

	cmd := exec.CommandContext(ctx, r.Binary, "--ask-for-approval", "never", "exec", "--ephemeral", "--sandbox", "read-only", "--skip-git-repo-check", "--json", "--output-schema", schemaPath, "--output-last-message", resultPath, prompt)
	cmd.Dir = r.WorkDir
	// Non-*os.File writers make os/exec own copy pipes. WaitDelay therefore
	// bounds both a canceled process and descendants that inherit those pipes.
	cmd.Stdout, cmd.Stderr = io.MultiWriter(stdout), io.MultiWriter(stderr)
	cmd.WaitDelay = waitDelay
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Codex %s run (diagnostics: %s): %s", role, dir, gruntime.Redact(err.Error()))
	}
	if err := os.Chmod(resultPath, 0o600); err != nil {
		return nil, fmt.Errorf("secure Codex %s result (diagnostics: %s): %w", role, dir, err)
	}
	result, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("read Codex %s result (diagnostics: %s): %w", role, dir, err)
	}
	return result, nil
}

func Generate(request Request, runners Runners) (Result, error) {
	return generateAfterPhase2Approval(request, runners)
}

// Decompose produces a manager-only draft before per-subtask constraints are
// assigned. It deliberately does not dispatch reviewer or verifier roles.
func Decompose(request Request, manager Runner) (Result, error) {
	if !isCodexRunner(manager) {
		return Result{}, fmt.Errorf("manager runner must be a hosted CodexRunner")
	}
	if err := validateManagerLifecycle(request); err != nil {
		return Result{}, err
	}
	sources, err := loadSources(request)
	if err != nil {
		return Result{}, err
	}
	catalog, err := buildSourceCatalog(request.RepoRoot, sources)
	if err != nil {
		return Result{}, err
	}
	key := sha256.Sum256([]byte(sources.PRD.Digest + sources.Spec.Digest + sources.Roadmap.Digest))
	workItem := "ticket-plan:" + hex.EncodeToString(key[:8])
	planBytes, err := runRole(request, workItem, "manager", 1, manager, decompositionPrompt(sources, catalog), planSchemaRange(1, 8))
	if err != nil {
		return Result{}, err
	}
	var plan ticketplan.Plan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		return Result{}, fmt.Errorf("parse manager decomposition: %w", err)
	}
	plan.Sources, plan.Status = sources, "draft"
	if err := writeDecomposition(request.OutputPath, plan); err != nil {
		return Result{}, err
	}
	return Result{PlanPath: request.OutputPath, WorkItem: workItem}, nil
}

func writeDecomposition(path string, plan ticketplan.Plan) error {
	data, err := marshalDecomposition(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize manager decomposition: %w", err)
	}
	return writePrivateFile(path, append(data, '\n'))
}

// generateAfterPhase2Approval performs orchestration after the public phase
// gate has confirmed that Phase 2 is approved.
func generateAfterPhase2Approval(request Request, runners Runners) (Result, error) {
	if err := validateRunners(runners); err != nil {
		return Result{}, err
	}
	if err := validateManagerLifecycle(request); err != nil {
		return Result{}, err
	}
	progress := func(message string) {
		if request.Progress != nil {
			request.Progress(message)
		}
	}
	progress("Loading approved product sources")
	sources, err := loadSources(request)
	if err != nil {
		return Result{}, err
	}
	constraints, err := LoadConstraints(request.ConstraintsPath, sources)
	if err != nil {
		return Result{}, fmt.Errorf("load approved constraints: %w", err)
	}
	// Build and validate the authority contract before catalog construction or
	// manager dispatch. Its ordered Slices are the declared slice manifest.
	contract, err := buildAuthorityContract(constraints)
	if err != nil {
		return Result{}, err
	}
	if err := contract.ValidateAgainst(request.RepoRoot); err != nil {
		return Result{}, fmt.Errorf("validate authority contract: %w", err)
	}
	catalog, err := buildSourceCatalog(request.RepoRoot, sources)
	if err != nil {
		return Result{}, err
	}
	key := sha256.Sum256([]byte(sources.PRD.Digest + sources.Spec.Digest + sources.Roadmap.Digest))
	workItem := "ticket-plan:" + hex.EncodeToString(key[:8])
	contractPath := filepath.Join(request.RuntimeRoot, "ticket-plan-runs", strings.ReplaceAll(workItem, ":", "-"), "authority-contract.json")
	contractDigest, err := contract.Digest()
	if existing, loadErr := ticketplan.LoadAuthorityContract(contractPath, request.RepoRoot); loadErr == nil {
		existingDigest, _ := existing.Digest()
		if existingDigest != contractDigest {
			return Result{}, fmt.Errorf("persisted authority contract does not match approved constraints")
		}
	} else if os.IsNotExist(loadErr) {
		if _, err = ticketplan.SaveAuthorityContract(contractPath, request.RepoRoot, contract); err != nil {
			return Result{}, fmt.Errorf("persist authority contract: %w", err)
		}
	} else {
		return Result{}, fmt.Errorf("load persisted authority contract: %w", loadErr)
	}
	var plan ticketplan.Plan
	feedback := ""
	for cycle := 1; cycle <= 2; cycle++ {
		progress(fmt.Sprintf("Dispatching Codex manager (cycle %d/2)", cycle))
		prompt := masterPrompt(sources, catalog, constraints)
		if feedback != "" {
			prompt = remediationPrompt(sources, catalog, constraints, feedback)
		}
		planBytes, err := runRole(request, workItem, "manager", cycle, runners.Manager, prompt, planSchema(constraints))
		if err != nil {
			return Result{}, err
		}
		if err := json.Unmarshal(planBytes, &plan); err != nil {
			return Result{}, fmt.Errorf("parse manager plan: %w", err)
		}
		plan.Sources, plan.Status, plan.ContractDigest = sources, "draft", contractDigest
		if err := ApplyConstraints(&plan, constraints); err != nil {
			return Result{}, err
		}
		if issues := plan.ValidateAgainstContract(request.RepoRoot, contract); len(issues) != 0 {
			if err := saveValidationFindings(request.RuntimeRoot, workItem, cycle, issues); err != nil {
				return Result{}, fmt.Errorf("save manager validation findings: %w", err)
			}
			return Result{}, fmt.Errorf("ticket plan contains unsupported manager narrative or invalid canonical content: %v", issues)
		}
		serialized, _ := json.Marshal(plan)
		progress(fmt.Sprintf("Dispatching independent reviewer (cycle %d/2)", cycle))
		result, err := runRole(request, workItem, "reviewer", cycle, runners.Reviewer, reviewPrompt("reviewer", serialized), reviewSchema())
		if err != nil {
			return Result{}, err
		}
		review, err := parseReviewResult(result)
		if err != nil {
			return Result{}, fmt.Errorf("parse reviewer result: %w", err)
		}
		if review.Status == "approved" {
			progress("Reviewer approved the ticket plan")
			break
		}
		progress("Reviewer requested changes; returning findings to the manager")
		if cycle == 2 {
			if err := saveEscalation(request.RuntimeRoot, workItem, cycle, "review did not converge", []string{review.Summary}); err != nil {
				return Result{}, fmt.Errorf("save stakeholder escalation: %w", err)
			}
			return Result{}, fmt.Errorf("ticket plan requires stakeholder escalation after two review cycles: %s", review.Summary)
		}
		feedback = review.Summary
	}
	serialized, _ := json.Marshal(plan)
	progress("Unloading approved reviewer before verifier handoff")
	if err := handoffReviewerToVerifier(runners.Reviewer, runners.Verifier); err != nil {
		if saveErr := saveEscalation(request.RuntimeRoot, workItem, 1, "reviewer-to-verifier residency handoff failed", []string{err.Error()}); saveErr != nil {
			return Result{}, fmt.Errorf("save residency handoff escalation: %w", saveErr)
		}
		return Result{}, err
	}
	progress("Reviewer unloaded and verifier residency verified")
	progress("Dispatching independent verifier")
	result, err := runRole(request, workItem, "verifier", 1, runners.Verifier, reviewPrompt("verifier", serialized), reviewSchema())
	if err != nil {
		return Result{}, err
	}
	verification, err := parseReviewResult(result)
	if err != nil {
		return Result{}, fmt.Errorf("parse verifier result: %w", err)
	}
	if verification.Status != "approved" {
		if err := saveEscalation(request.RuntimeRoot, workItem, 1, "verifier did not approve", []string{verification.Summary}); err != nil {
			return Result{}, fmt.Errorf("save stakeholder escalation: %w", err)
		}
		return Result{}, fmt.Errorf("verifier did not approve ticket plan: %s", verification.Summary)
	}
	progress("Verifier approved the ticket plan; writing plan for stakeholder approval")
	plan.Status = "ready-for-approval"
	if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
		return Result{}, err
	}
	data, _ := json.MarshalIndent(plan, "", "  ")
	if err := os.WriteFile(request.OutputPath, append(data, '\n'), 0o644); err != nil {
		return Result{}, err
	}
	digest, err := ticketplan.FileDigest(request.OutputPath)
	if err != nil {
		return Result{}, err
	}
	statePath := filepath.Join(request.RuntimeRoot, "ticket-plan-runs", strings.ReplaceAll(workItem, ":", "-"), "workflow.json")
	state, err := ticketplan.NewWorkflowState(request.RepoRoot, request.OutputPath, digest, "ready-for-approval", plan.Sources)
	if err != nil {
		return Result{}, err
	}
	state.ContractPath, state.ContractDigest = contractPath, contractDigest
	if err := ticketplan.SaveWorkflow(statePath, state); err != nil {
		return Result{}, err
	}
	return Result{PlanPath: request.OutputPath, WorkItem: workItem}, nil
}

func saveValidationFindings(root, workItem string, cycle int, issues []string) error {
	dir := filepath.Join(root, "ticket-plan-runs", strings.ReplaceAll(workItem, ":", "-"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{"cycle": cycle, "findings": issues}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("manager-%d-validation.json", cycle)), append(data, '\n'), 0o600)
}

func saveEscalation(root, workItem string, cycle int, reason string, findings []string) error {
	dir := filepath.Join(root, "ticket-plan-runs", strings.ReplaceAll(workItem, ":", "-"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	redacted := make([]string, 0, len(findings))
	for _, finding := range findings {
		redacted = append(redacted, gruntime.Redact(finding))
	}
	data, err := json.MarshalIndent(map[string]any{
		"cycle":    cycle,
		"reason":   gruntime.Redact(reason),
		"findings": redacted,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "stakeholder-escalation.json"), append(data, '\n'), 0o600)
}

func validateRunners(runners Runners) error {
	if !isCodexRunner(runners.Manager) {
		return fmt.Errorf("manager runner must be a hosted CodexRunner")
	}
	if !isOllamaRunner(runners.Reviewer) {
		return fmt.Errorf("reviewer runner must be a local OllamaRunner")
	}
	if !isOllamaRunner(runners.Verifier) {
		return fmt.Errorf("verifier runner must be a local OllamaRunner")
	}
	if sameRunnerInstance(runners.Reviewer, runners.Verifier) {
		return fmt.Errorf("reviewer and verifier runners must be independent instances")
	}
	reviewerModel, err := authorizedLocalModel(runners.Reviewer, "reviewer")
	if err != nil {
		return fmt.Errorf("reviewer worker policy: %w", err)
	}
	verifierModel, err := authorizedLocalModel(runners.Verifier, "verifier")
	if err != nil {
		return fmt.Errorf("verifier worker policy: %w", err)
	}
	reviewer, _ := localWorker(runners.Reviewer)
	verifier, _ := localWorker(runners.Verifier)
	if reviewerModel.Name == verifierModel.Name || reviewerModel.ID == verifierModel.ID {
		return fmt.Errorf("reviewer and verifier models must have distinct identities")
	}
	if reviewer.Policy.Endpoint != verifier.Policy.Endpoint || reviewer.Policy.Fingerprint != verifier.Policy.Fingerprint {
		return fmt.Errorf("reviewer and verifier runners must use the same local model policy")
	}
	return nil
}

func authorizedLocalModel(runner Runner, role string) (ollama.Model, error) {
	worker, err := localWorker(runner)
	if err != nil {
		return ollama.Model{}, err
	}
	return worker.Policy.Authorize(ollama.Request{Model: worker.Model, Role: role, TaskType: "ticket-plan-review"})
}

func localWorker(runner Runner) (OllamaRunner, error) {
	switch value := runner.(type) {
	case OllamaRunner:
		return value, nil
	case *OllamaRunner:
		if value == nil {
			return OllamaRunner{}, fmt.Errorf("local Ollama runner is nil")
		}
		return *value, nil
	default:
		return OllamaRunner{}, fmt.Errorf("local Ollama runner is invalid")
	}
}

func handoffReviewerToVerifier(reviewerRunner, verifierRunner Runner) error {
	reviewer, err := localWorker(reviewerRunner)
	if err != nil {
		return fmt.Errorf("resolve reviewer residency worker: %w", err)
	}
	verifier, err := localWorker(verifierRunner)
	if err != nil {
		return fmt.Errorf("resolve verifier residency worker: %w", err)
	}
	if err := reviewer.SetResidency(false); err != nil {
		return fmt.Errorf("unload reviewer model %q: %w", reviewer.Model, err)
	}
	if err := verifier.SetResidency(true); err != nil {
		return fmt.Errorf("load verifier model %q: %w", verifier.Model, err)
	}
	return nil
}

func sameRunnerInstance(left, right Runner) bool {
	leftOllama, leftOK := left.(*OllamaRunner)
	rightOllama, rightOK := right.(*OllamaRunner)
	return leftOK && rightOK && leftOllama != nil && leftOllama == rightOllama
}

func isCodexRunner(runner Runner) bool {
	switch value := runner.(type) {
	case CodexRunner:
		return true
	case *CodexRunner:
		return value != nil
	default:
		return false
	}
}

func isOllamaRunner(runner Runner) bool {
	switch value := runner.(type) {
	case OllamaRunner:
		return true
	case *OllamaRunner:
		return value != nil
	default:
		return false
	}
}

func loadSources(request Request) (ticketplan.Sources, error) {
	load := func(path string) (ticketplan.Source, error) {
		file, err := ticketplan.ReadVerifiedFile(request.RepoRoot, path)
		if err != nil {
			return ticketplan.Source{}, fmt.Errorf("approved source must be inside repository root: %s", path)
		}
		return ticketplan.Source{Path: file.RelativePath, Digest: file.Digest}, nil
	}
	prd, err := load(request.PRDPath)
	if err != nil {
		return ticketplan.Sources{}, err
	}
	spec, err := load(request.SpecPath)
	if err != nil {
		return ticketplan.Sources{}, err
	}
	roadmap, err := load(request.RoadmapPath)
	if err != nil {
		return ticketplan.Sources{}, err
	}
	return ticketplan.Sources{PRD: prd, Spec: spec, Roadmap: roadmap}, nil
}

var catalogHeadingPattern = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t]*#*[ \t]*$`)

func buildSourceCatalog(repoRoot string, sources ticketplan.Sources) (string, error) {
	var builder strings.Builder
	canonicalPaths, digests := map[string]bool{}, map[string]bool{}
	for _, entry := range []struct {
		Name   string
		Source ticketplan.Source
	}{
		{Name: "prd", Source: sources.PRD},
		{Name: "spec", Source: sources.Spec},
		{Name: "roadmap", Source: sources.Roadmap},
	} {
		file, err := ticketplan.ReadVerifiedSource(repoRoot, entry.Source.Path)
		if err != nil {
			return "", fmt.Errorf("load %s source catalog: %w", entry.Name, err)
		}
		if file.RelativePath != entry.Source.Path || file.Digest != entry.Source.Digest || canonicalPaths[file.CanonicalPath] || digests[file.Digest] {
			return "", fmt.Errorf("source catalog requires three distinct canonical source identities")
		}
		canonicalPaths[file.CanonicalPath], digests[file.Digest] = true, true
		matches := catalogHeadingPattern.FindAllSubmatchIndex(file.Data, -1)
		if len(matches) == 0 {
			return "", fmt.Errorf("%s source catalog requires Markdown headings", entry.Name)
		}
		fmt.Fprintf(&builder, "SOURCE %s (%s)\n", entry.Name, entry.Source.Path)
		for index, match := range matches {
			end := len(file.Data)
			if index+1 < len(matches) {
				end = matches[index+1][0]
			}
			section := strings.TrimSpace(string(file.Data[match[2]:match[3]]))
			content := strings.TrimSpace(string(file.Data[match[0]:end]))
			fmt.Fprintf(&builder, "SECTION %s\n%s\n", section, content)
		}
	}
	return builder.String(), nil
}

func validateManagerLifecycle(request Request) error {
	if request.ManagerTimeout <= 0 {
		return fmt.Errorf("manager timeout must be positive")
	}
	if request.ManagerWaitDelay <= 0 {
		return fmt.Errorf("manager wait delay must be positive")
	}
	return nil
}

func runRole(request Request, workItem, role string, cycle int, runner Runner, prompt, schema string) ([]byte, error) {
	root := request.RuntimeRoot
	id := fmt.Sprintf("%s-ticket-plan-%d", role, cycle)
	if err := recordExecutionEvent(root, gruntime.Event{WorkItem: workItem, AgentID: id, Role: role, State: "started"}); err != nil {
		return nil, err
	}
	ctx := request.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if role == "manager" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.ManagerTimeout)
		defer cancel()
	}
	artifactDir := filepath.Join(root, "ticket-plan-runs", strings.ReplaceAll(workItem, ":", "-"))
	if managed, ok := runner.(CodexRunner); ok && role == "manager" {
		artifactDir = filepath.Join(artifactDir, fmt.Sprintf("%s-%d", role, cycle))
		result, err := managed.run(ctx, role, prompt, schema, artifactDir, request.ManagerWaitDelay)
		return finishRole(root, workItem, id, role, cycle, artifactDir, result, err)
	}
	result, err := runner.Run(ctx, role, prompt, schema)
	return finishRole(root, workItem, id, role, cycle, artifactDir, result, err)
}

func finishRole(root, workItem, id, role string, cycle int, artifactDir string, result []byte, runErr error) ([]byte, error) {
	if runErr != nil {
		ref, reconciliation := persistFailureDiagnostic(root, artifactDir, role, cycle, runErr)
		reconciliation = append(reconciliation, reconcileTerminalEvents(root, workItem, id, role, "failed", ref)...)
		return nil, joinReconciliation(runErr, reconciliation)
	}
	var reconciliation []error
	if err := makePrivateDirectory(artifactDir); err != nil {
		reconciliation = append(reconciliation, fmt.Errorf("create result diagnostics: %w", err))
	}
	if err := os.Chmod(artifactDir, 0o700); err != nil {
		reconciliation = append(reconciliation, fmt.Errorf("secure result diagnostics: %w", err))
	}
	ref := filepath.Join(artifactDir, fmt.Sprintf("%s-%d.json", role, cycle))
	if role == "manager" && strings.HasSuffix(artifactDir, fmt.Sprintf("%s-%d", role, cycle)) {
		ref = filepath.Join(artifactDir, "result.json")
	} else if err := writeNewPrivateFile(ref, result); err != nil {
		reconciliation = append(reconciliation, fmt.Errorf("persist result diagnostics: %w", err))
	}
	if len(reconciliation) != 0 {
		fallback, fallbackErrors := persistFallbackDiagnostic(root, role, cycle, errors.Join(reconciliation...))
		reconciliation = append(reconciliation, fallbackErrors...)
		if fallback != "" {
			ref = fallback
		}
	}
	reconciliation = append(reconciliation, reconcileTerminalEvents(root, workItem, id, role, "completed", ref)...)
	if err := joinReconciliation(nil, reconciliation); err != nil {
		return nil, err
	}
	return result, nil
}

// persistFailureDiagnostic records the original run error. If the primary
// manager artifact cannot be written, it uses a real, private fallback rather
// than inventing a diagnostic reference.
func persistFailureDiagnostic(root, artifactDir, role string, cycle int, runErr error) (string, []error) {
	ref := filepath.Join(artifactDir, fmt.Sprintf("%s-%d-error-%d.txt", role, cycle, time.Now().UnixNano()))
	if err := makePrivateDirectory(artifactDir); err != nil {
		return failureFallback(root, role, cycle, runErr, fmt.Errorf("create failure diagnostics: %w", err))
	}
	if err := os.Chmod(artifactDir, 0o700); err != nil {
		return failureFallback(root, role, cycle, runErr, fmt.Errorf("secure failure diagnostics: %w", err))
	}
	if err := writeNewPrivateFile(ref, []byte(gruntime.Redact(runErr.Error())+"\n")); err != nil {
		return failureFallback(root, role, cycle, runErr, fmt.Errorf("persist failure diagnostics: %w", err))
	}
	return ref, nil
}

func failureFallback(root, role string, cycle int, runErr, primary error) (string, []error) {
	ref, fallbackErrors := persistFallbackDiagnostic(root, role, cycle, errors.Join(runErr, primary))
	return ref, append([]error{primary}, fallbackErrors...)
}

func persistFallbackDiagnostic(root, role string, cycle int, diagnostic error) (string, []error) {
	dir := filepath.Join(root, "ticket-plan-failures")
	if err := makePrivateDirectory(dir); err != nil {
		return "", []error{fmt.Errorf("create fallback diagnostics: %w", err)}
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", []error{fmt.Errorf("secure fallback diagnostics: %w", err)}
	}
	ref := filepath.Join(dir, fmt.Sprintf("%s-%d-error-%d.txt", role, cycle, time.Now().UnixNano()))
	if err := writeNewPrivateFile(ref, []byte(gruntime.Redact(diagnostic.Error())+"\n")); err != nil {
		return "", []error{fmt.Errorf("persist fallback diagnostics: %w", err)}
	}
	return ref, nil
}

// reconcileTerminalEvents makes all terminal attempts even when a preceding
// diagnostic or ledger write fails. A failed close gets one bounded retry; no
// work is re-dispatched during reconciliation.
func reconcileTerminalEvents(root, workItem, id, role, state, ref string) []error {
	var reconciliation []error
	if err := recordExecutionEvent(root, gruntime.Event{WorkItem: workItem, AgentID: id, Role: role, State: state, ResultRef: ref}); err != nil {
		reconciliation = append(reconciliation, fmt.Errorf("record %s state: %w", state, err))
	}
	closeEvent := gruntime.Event{WorkItem: workItem, AgentID: id, Role: role, State: "closed", ResultRef: ref}
	if err := recordExecutionEvent(root, closeEvent); err != nil {
		reconciliation = append(reconciliation, fmt.Errorf("record closed state: %w", err))
		if retryErr := recordExecutionEvent(root, closeEvent); retryErr != nil {
			reconciliation = append(reconciliation, fmt.Errorf("retry closed state: %w", retryErr))
		}
	}
	return reconciliation
}

func joinReconciliation(original error, reconciliation []error) error {
	if len(reconciliation) == 0 {
		return original
	}
	errorsToJoin := make([]error, 0, len(reconciliation)+1)
	if original != nil {
		errorsToJoin = append(errorsToJoin, original)
	}
	errorsToJoin = append(errorsToJoin, reconciliation...)
	return errors.Join(errorsToJoin...)
}

func openNewPrivateFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

func writeNewPrivateFile(path string, data []byte) error {
	file, err := openNewPrivateFile(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

type reviewResult struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

func parseReviewResult(data []byte) (reviewResult, error) {
	cleaned := strings.TrimSpace(string(data))
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		if len(lines) < 3 || !strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
			return reviewResult{}, fmt.Errorf("review result code fence is incomplete")
		}
		cleaned = strings.Join(lines[1:len(lines)-1], "\n")
	}
	var result reviewResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return reviewResult{}, err
	}
	if !validReviewStatus(result.Status) || strings.TrimSpace(result.Summary) == "" {
		return reviewResult{}, fmt.Errorf("review result is invalid")
	}
	return result, nil
}

func validReviewStatus(status string) bool {
	return status == "approved" || status == "changes_requested" || status == "blocked"
}

func masterPrompt(s ticketplan.Sources, catalog string, constraints Constraints) string {
	template, _ := json.Marshal(constraints)
	return fmt.Sprintf("You are the Codex manager. Create a story and exactly %d independently reviewable subtasks from this verified source catalog. The subtasks must be in the same order as the approved constraints template; constrained IDs, budgets, paths, dependencies, and their traceability will be applied locally. Return only a JSON ticket plan. For each remaining traceability reference, select source exactly from prd, spec, or roadmap; select section exactly from the catalog; and copy its excerpt verbatim from that section. Do not write files, Jira, or code.\n\nAPPROVED CONSTRAINTS:\n%s\n\nSOURCE CATALOG:\n%s", len(constraints.Subtasks), template, catalog)
}
func decompositionPrompt(s ticketplan.Sources, catalog string) string {
	return fmt.Sprintf("You are the Codex manager. Create a story and between one and eight independently reviewable subtasks from this verified source catalog. This is a decomposition draft: do not claim approval and do not write files, Jira, or code. Return only a JSON ticket plan. For every traceability reference, select source exactly from prd, spec, or roadmap; and copy its excerpt verbatim. Every traced field value must be explicitly supported by its cited excerpt; use only allowed-path strings and review-budget components from the catalog, use a phase value present in its cited excerpt, and copy the Architecture Decision wording exactly for an ADR rationale without inserting or changing whitespace.\n\nSOURCE CATALOG:\n%s", catalog)
}
func remediationPrompt(s ticketplan.Sources, catalog string, constraints Constraints, feedback string) string {
	template, _ := json.Marshal(constraints)
	return fmt.Sprintf("You are the Codex manager revising a ticket plan. Resolve these deterministic validation findings:\n%s\nReturn exactly %d subtasks in the same order as the approved constraints template. Apply the same catalog rules for narrative fields. Do not write files, Jira, or code.\n\nAPPROVED CONSTRAINTS:\n%s\n\nSOURCE CATALOG:\n%s", feedback, len(constraints.Subtasks), template, catalog)
}
func reviewPrompt(role string, plan []byte) string {
	return fmt.Sprintf("You are an independent local %s. Review this ticket plan for source traceability, bounded scope, acceptance criteria, validation, allowed paths, and ADR references. Return only JSON matching {\"status\":\"approved|changes_requested|blocked\",\"summary\":\"concise finding summary\"}; use status approved only if it is ready. Do not write files or Jira. Plan: %s", role, plan)
}
func planSchema(constraints Constraints) string {
	var schema map[string]any
	if err := json.Unmarshal([]byte(planSchemaRange(len(constraints.Subtasks), len(constraints.Subtasks))), &schema); err != nil {
		panic(fmt.Sprintf("parse static ticket-plan schema: %v", err))
	}
	properties := schema["properties"].(map[string]any)
	subtaskProperties := properties["subtasks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	budgetProperties := subtaskProperties["review_budget"].(map[string]any)["properties"].(map[string]any)

	ids, phases, classes := []string{}, []string{}, []string{}
	files, lines := []int{}, []int{}
	maxPaths, maxComponents, maxDependencies := 0, 0, 0
	maxNonGoals, maxAcceptance, maxValidation, maxReferences := 1, 1, 1, 1
	nonGoals, acceptanceCriteria, validationPlan := []string{}, []string{}, []string{}
	for _, subtask := range constraints.Subtasks {
		ids = append(ids, subtask.ID)
		phases = append(phases, subtask.Phase)
		classes = append(classes, subtask.ChangeClass)
		files = append(files, subtask.ReviewBudget.MaxChangedFiles)
		lines = append(lines, subtask.ReviewBudget.MaxChangedLines)
		maxPaths = max(maxPaths, len(subtask.AllowedPaths))
		maxComponents = max(maxComponents, len(subtask.ReviewBudget.Components))
		maxDependencies = max(maxDependencies, len(subtask.Dependencies))
		maxReferences = max(maxReferences, largestTraceSet(subtask.Traceability))
		if source := subtask.SourceDerived; source != nil {
			nonGoals = append(nonGoals, source.NonGoals...)
			acceptanceCriteria = append(acceptanceCriteria, source.AcceptanceCriteria...)
			validationPlan = append(validationPlan, source.ValidationPlan...)
			maxNonGoals = max(maxNonGoals, len(source.NonGoals))
			maxAcceptance = max(maxAcceptance, len(source.AcceptanceCriteria))
			maxValidation = max(maxValidation, len(source.ValidationPlan))
			maxReferences = max(maxReferences, largestTraceSet(source.Traceability))
		}
	}
	if constraints.Story != nil {
		count := len(constraints.Story.AcceptanceCriteria)
		properties["story"].(map[string]any)["properties"].(map[string]any)["acceptance_criteria"] = boundedStringArray(constraints.Story.AcceptanceCriteria, count, count)
		maxReferences = max(maxReferences, largestTraceSet(constraints.Story.Traceability))
	}

	subtaskProperties["id"] = enumStringSchema(ids)
	subtaskProperties["phase"] = enumStringSchema(phases)
	subtaskProperties["change_class"] = enumStringSchema(classes)
	budgetProperties["max_changed_files"] = enumIntegerSchema(files)
	budgetProperties["max_changed_lines"] = enumIntegerSchema(lines)
	budgetProperties["components"] = boundedStringArray(constraints.ReviewBudget.Components, 1, maxComponents)
	subtaskProperties["allowed_paths"] = boundedStringArray(constraints.PathPool, 1, maxPaths)
	// Dependency items may name any assigned Subtask, while the maximum comes
	// from the approved dependency arrays. minItems remains zero so an initial
	// slice stays valid when a later slice has dependencies.
	subtaskProperties["dependencies"] = boundedStringArray(ids, 0, maxDependencies)
	subtaskProperties["non_goals"] = boundedStringArray(nonGoals, 1, maxNonGoals)
	subtaskProperties["acceptance_criteria"] = boundedStringArray(acceptanceCriteria, 1, maxAcceptance)
	subtaskProperties["validation_plan"] = boundedStringArray(validationPlan, 1, maxValidation)
	boundArray(schema["$defs"].(map[string]any)["references"], maxReferences)

	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("serialize ticket-plan schema: %v", err))
	}
	return string(encoded)
}

func enumStringSchema(values []string) map[string]any {
	return map[string]any{"type": "string", "enum": uniqueSorted(values)}
}

func enumIntegerSchema(values []int) map[string]any {
	seen := map[int]bool{}
	unique := []int{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	sort.Ints(unique)
	return map[string]any{"type": "integer", "enum": unique}
}

func boundedStringArray(values []string, minimum, maximum int) map[string]any {
	items := map[string]any{"type": "string"}
	if len(values) > 0 {
		items["enum"] = uniqueSorted(values)
	}
	return map[string]any{"type": "array", "minItems": minimum, "maxItems": maximum, "items": items}
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func largestTraceSet(trace ticketplan.TraceMap) int {
	maximum := 0
	for _, references := range trace {
		maximum = max(maximum, len(references))
	}
	return maximum
}

func boundArray(value any, maximum int) {
	value.(map[string]any)["maxItems"] = maximum
}

func planSchemaRange(minSubtasks, maxSubtasks int) string {
	return fmt.Sprintf(`{
  "type":"object",
  "properties":{
    "format_version":{"type":"integer"},
    "status":{"type":"string","enum":["draft"]},
    "sources":{"type":"object","properties":{
      "prd":{"$ref":"#/$defs/source"},
      "spec":{"$ref":"#/$defs/source"},
      "roadmap":{"$ref":"#/$defs/source"}
    },"required":["prd","spec","roadmap"],"additionalProperties":false},
    "story":{"type":"object","properties":{
      "summary":{"type":"string"},
      "description":{"type":"string"},
      "acceptance_criteria":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string"}},
      "traceability":{"$ref":"#/$defs/story_traceability"}
    },"required":["summary","description","acceptance_criteria","traceability"],"additionalProperties":false},
    "subtasks":{"type":"array","minItems":%d,"maxItems":%d,"items":{"type":"object","properties":{
      "id":{"type":"string"},
      "summary":{"type":"string"},
	      "phase":{"type":"string"},
	      "change_class":{"type":"string","enum":["trivial","standard","high-risk"]},
	      "review_budget":{"type":"object","properties":{"max_changed_files":{"type":"integer"},"max_changed_lines":{"type":"integer"},"components":{"type":"array","minItems":1,"maxItems":16,"items":{"type":"string"}}},"required":["max_changed_files","max_changed_lines","components"],"additionalProperties":false},
      "scope":{"type":"string"},
	      "non_goals":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string"}},
	      "acceptance_criteria":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string"}},
	      "validation_plan":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string"}},
	      "allowed_paths":{"type":"array","minItems":1,"maxItems":64,"items":{"type":"string","minLength":1,"maxLength":256,"pattern":"^(?:[^.*?\\[\\],/\\r\\n][^*?\\[\\],/\\r\\n]*|\\.[^.*?\\[\\],/\\r\\n][^*?\\[\\],/\\r\\n]*|\\.\\.[^*?\\[\\],/\\r\\n]+)(?:/(?:[^.*?\\[\\],/\\r\\n][^*?\\[\\],/\\r\\n]*|\\.[^.*?\\[\\],/\\r\\n][^*?\\[\\],/\\r\\n]*|\\.\\.[^*?\\[\\],/\\r\\n]+))*$"}},
	      "adr":{"type":"string","pattern":"^No ADR needed: .{10,}$"},
	      "dependencies":{"type":"array","minItems":0,"maxItems":8,"items":{"type":"string"}},
      "traceability":{"$ref":"#/$defs/subtask_traceability"}
    },"required":["id","summary","phase","change_class","review_budget","scope","non_goals","acceptance_criteria","validation_plan","allowed_paths","adr","dependencies","traceability"],"additionalProperties":false}}
  },
  "required":["format_version","status","sources","story","subtasks"],
  "additionalProperties":false,
  "$defs":{"source":{"type":"object","properties":{
    "path":{"type":"string"},
    "digest":{"type":"string"}
  },"required":["path","digest"],"additionalProperties":false},
  "reference":{"type":"object","properties":{"source":{"type":"string","enum":["prd","spec","roadmap"]},"section":{"type":"string"},"excerpt":{"type":"string"}},"required":["source","section","excerpt"],"additionalProperties":false},
	  "references":{"type":"array","minItems":1,"maxItems":32,"items":{"$ref":"#/$defs/reference"}},
  "story_traceability":{"type":"object","properties":{"summary":{"$ref":"#/$defs/references"},"description":{"$ref":"#/$defs/references"},"acceptance_criteria":{"$ref":"#/$defs/references"}},"required":["summary","description","acceptance_criteria"],"additionalProperties":false},
  "subtask_traceability":{"type":"object","properties":{"summary":{"$ref":"#/$defs/references"},"phase":{"$ref":"#/$defs/references"},"change_class":{"$ref":"#/$defs/references"},"review_budget":{"$ref":"#/$defs/references"},"scope":{"$ref":"#/$defs/references"},"non_goals":{"$ref":"#/$defs/references"},"acceptance_criteria":{"$ref":"#/$defs/references"},"validation_plan":{"$ref":"#/$defs/references"},"allowed_paths":{"$ref":"#/$defs/references"},"adr":{"$ref":"#/$defs/references"},"dependencies":{"$ref":"#/$defs/references"}},"required":["summary","phase","change_class","review_budget","scope","non_goals","acceptance_criteria","validation_plan","allowed_paths","adr","dependencies"],"additionalProperties":false}}
}`, minSubtasks, maxSubtasks)
}
func reviewSchema() string {
	return `{"type":"object","properties":{"status":{"type":"string","enum":["approved","changes_requested","blocked"]},"summary":{"type":"string"}},"required":["status","summary"],"additionalProperties":false}`
}
