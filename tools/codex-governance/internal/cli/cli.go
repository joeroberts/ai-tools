package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"codex-governance/internal/agentplan"
	"codex-governance/internal/config"
	"codex-governance/internal/initializer"
	"codex-governance/internal/jira"
	"codex-governance/internal/ollama"
	"codex-governance/internal/roadmap"
	gruntime "codex-governance/internal/runtime"
	"codex-governance/internal/syncer"
	"codex-governance/internal/ticketplan"
	"codex-governance/internal/validate"
	"codex-governance/internal/workitem"
)

const usage = `codex-governance

Usage:
  codex-governance init [--repo-root PATH]
  codex-governance config check [--repo-root PATH]
  codex-governance validate-work-item --work-item PATH --offline-export PATH [--repo-root PATH] [--runtime-root PATH] [--base-sha SHA --head-sha SHA] [--warn]
  codex-governance roadmap status --roadmap PATH [--format table|markdown|json]
  codex-governance roadmap check --roadmap PATH
  codex-governance sync --check|--dry-run --manifest PATH [--repo-root PATH]
  codex-governance jira plan generate --prd PATH --spec PATH --roadmap PATH --output PATH --policy PATH --reviewer-model NAME --verifier-model NAME [--repo-root PATH] [--runtime-root PATH] [--codex-bin PATH] [--dry-run] [--verbose]
  codex-governance jira plan validate --plan PATH [--repo-root PATH]
  codex-governance runtime agent start|complete|fail|close --work-item KEY --agent-id ID --role ROLE [--result-ref REF]
  codex-governance runtime check --work-item KEY
  codex-governance runtime cache clear [--runtime-root PATH]
  codex-governance ollama policy init [--runtime-root PATH]
  codex-governance ollama run --model NAME --role ROLE --task-type TYPE --input PATH [--policy PATH]

Governed engineering utilities for Jira-backed work items.
`

// Run executes the CLI command selected by args.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if args[0] == "init" {
		return runInit(args[1:], stdout, stderr)
	}
	if args[0] == "config" {
		return runConfig(args[1:], stdout, stderr)
	}
	if args[0] == "validate-work-item" {
		return runValidateWorkItem(args[1:], stdout, stderr)
	}
	if args[0] == "roadmap" {
		return runRoadmap(args[1:], stdout, stderr)
	}
	if args[0] == "sync" {
		return runSync(args[1:], stdout, stderr)
	}
	if args[0] == "jira" {
		return runJira(args[1:], stdout, stderr)
	}
	if args[0] == "runtime" {
		return runRuntime(args[1:], stdout, stderr)
	}
	if args[0] == "ollama" {
		return runOllama(args[1:], stdout, stderr)
	}

	fmt.Fprintf(stderr, "unknown command: %s\n\n%s", args[0], usage)
	return 2
}

func runJira(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || args[0] != "plan" {
		fmt.Fprintln(stderr, "usage: codex-governance jira plan generate|validate")
		return 2
	}
	command := args[1]
	if command == "generate" {
		return runJiraPlanGenerate(args[2:], stdout, stderr)
	}
	if command == "approve" {
		fmt.Fprintln(stderr, "jira plan approve is unavailable until Phase 3 is approved")
		return 1
	}
	if command == "create" {
		fmt.Fprintln(stderr, "jira plan create is unavailable until Phase 4 is approved")
		return 1
	}
	if command != "validate" && command != "create" && command != "approve" {
		fmt.Fprintln(stderr, "usage: codex-governance jira plan generate|validate|approve|create")
		return 2
	}
	flags := flag.NewFlagSet("jira plan "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("plan", "", "ticket plan JSON")
	repoRoot := flags.String("repo-root", ".", "repository root")
	dryRun := flags.Bool("dry-run", false, "report Jira writes without sending them")
	approve := flags.Bool("approve", false, "authorize Jira issue creation")
	resultPath := flags.String("result", "", "creation result JSON")
	workflowPath := flags.String("workflow", "", "persisted ticket-plan workflow state")
	if err := flags.Parse(args[2:]); err != nil || *path == "" || flags.NArg() != 0 {
		return 2
	}
	plan, err := ticketplan.Load(*path)
	if err != nil {
		fmt.Fprintf(stderr, "load ticket plan: %v\n", err)
		return 2
	}
	issues := plan.ValidateAgainst(*repoRoot)
	if len(issues) != 0 {
		for _, issue := range issues {
			fmt.Fprintf(stdout, "FAIL %s\n", issue)
		}
		return 1
	}
	if command == "validate" {
		fmt.Fprintln(stdout, "PASS ticket plan is valid")
		return 0
	}
	if command == "approve" {
		if !*approve || *workflowPath == "" {
			fmt.Fprintln(stderr, "jira plan approve requires --workflow and --approve")
			return 2
		}
		if err := ticketplan.Approve(*path, *workflowPath); err != nil {
			fmt.Fprintf(stderr, "approve ticket plan: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "PASS ticket plan approved by stakeholder")
		return 0
	}
	if plan.Status != "approved" {
		fmt.Fprintln(stdout, "FAIL ticket plan must have status approved before creation")
		return 1
	}
	if *workflowPath == "" {
		fmt.Fprintln(stderr, "jira plan create requires --workflow")
		return 2
	}
	state, err := ticketplan.LoadWorkflow(*workflowPath)
	if err != nil {
		fmt.Fprintf(stderr, "load workflow state: %v\n", err)
		return 2
	}
	planDigest, err := ticketplan.FileDigest(*path)
	if err != nil {
		fmt.Fprintf(stderr, "digest ticket plan: %v\n", err)
		return 2
	}
	if state.Status != "approved" || state.PlanDigest != planDigest {
		fmt.Fprintln(stdout, "FAIL workflow state is not approved for this exact plan")
		return 1
	}
	if *dryRun == *approve {
		fmt.Fprintln(stderr, "jira plan create requires exactly one of --dry-run or --approve")
		return 2
	}
	if *dryRun {
		fmt.Fprintf(stdout, "DRY RUN would create Story %q and %d subtasks\n", plan.Story.Summary, len(plan.Subtasks))
		return 0
	}
	cfg, err := config.Load(filepath.Join(*repoRoot, "governance.yml"))
	if err != nil || cfg.Jira.Project == "" {
		fmt.Fprintln(stderr, "jira plan create requires governance.yml with jira.project")
		return 2
	}
	if *resultPath == "" {
		*resultPath = *path + ".result.json"
	}
	if _, err := os.Stat(*resultPath); err == nil {
		fmt.Fprintf(stderr, "refusing to overwrite existing creation result: %s\n", *resultPath)
		return 1
	}
	baseURL, email, token := os.Getenv("JIRA_BASE_URL"), os.Getenv("JIRA_EMAIL"), os.Getenv("JIRA_API_TOKEN")
	if baseURL == "" || email == "" || token == "" {
		fmt.Fprintln(stderr, "jira plan create requires JIRA_BASE_URL, JIRA_EMAIL, and JIRA_API_TOKEN")
		return 2
	}
	story, subtasks, err := (jira.CreateClient{BaseURL: baseURL, Email: email, Token: token}).CreatePlan(cfg.Jira.Project, plan)
	if err != nil {
		fmt.Fprintf(stderr, "create Jira issues: %v\n", err)
		return 1
	}
	digest, _ := ticketplan.FileDigest(*path)
	result, _ := json.MarshalIndent(map[string]any{"plan_digest": digest, "story": story, "subtasks": subtasks}, "", "  ")
	if err := os.WriteFile(*resultPath, result, 0o600); err != nil {
		fmt.Fprintf(stderr, "write creation result: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "PASS created Story %s and %d subtasks\n", story.Key, len(subtasks))
	return 0
}

func runJiraPlanGenerate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("jira plan generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	prd := flags.String("prd", "", "approved PRD Markdown")
	spec := flags.String("spec", "", "approved specification Markdown")
	roadmapPath := flags.String("roadmap", "", "approved roadmap Markdown")
	output := flags.String("output", "", "generated ticket plan JSON")
	repoRoot := flags.String("repo-root", ".", "repository root")
	runtimeRoot := flags.String("runtime-root", "", "owner-only runtime directory")
	codexBin := flags.String("codex-bin", "codex", "hosted Codex executable")
	policyPath := flags.String("policy", "", "owner-only local Ollama policy")
	reviewerModel := flags.String("reviewer-model", "", "allowlisted local Ollama reviewer model")
	verifierModel := flags.String("verifier-model", "", "allowlisted local Ollama verifier model")
	dryRun := flags.Bool("dry-run", false, "show the governed dispatch without running agents")
	verbose := flags.Bool("verbose", false, "report orchestration progress without printing agent content")
	if err := flags.Parse(args); err != nil || *prd == "" || *spec == "" || *roadmapPath == "" || *output == "" || flags.NArg() != 0 {
		return 2
	}
	if *dryRun {
		fmt.Fprintf(stdout, "DRY RUN would dispatch hosted manager and local reviewer/verifier for %q\n", *output)
		return 0
	}
	if *policyPath == "" || *reviewerModel == "" || *verifierModel == "" {
		fmt.Fprintln(stderr, "jira plan generate requires --policy, --reviewer-model, and --verifier-model")
		return 2
	}
	policy, err := ollama.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "load Ollama policy: %v\n", err)
		return 2
	}
	if *runtimeRoot == "" {
		*runtimeRoot, err = gruntime.DefaultRoot()
		if err != nil {
			fmt.Fprintf(stderr, "resolve runtime root: %v\n", err)
			return 2
		}
	}
	request := agentplan.Request{
		PRDPath: *prd, SpecPath: *spec, RoadmapPath: *roadmapPath, OutputPath: *output,
		RepoRoot: *repoRoot, RuntimeRoot: *runtimeRoot,
	}
	if *verbose {
		request.Progress = func(message string) { fmt.Fprintln(stderr, message) }
	}
	result, err := agentplan.Generate(request, agentplan.Runners{
		Manager:  agentplan.CodexRunner{Binary: *codexBin, WorkDir: *repoRoot},
		Reviewer: agentplan.OllamaRunner{Policy: policy, Model: *reviewerModel},
		Verifier: agentplan.OllamaRunner{Policy: policy, Model: *verifierModel},
	})
	if err != nil {
		fmt.Fprintf(stderr, "generate ticket plan: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "PASS ticket plan generated: %s (work item %s)\n", result.PlanPath, result.WorkItem)
	return 0
}

func runInit(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("repo-root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "init accepts no positional arguments")
		return 2
	}
	created, err := initializer.Initialize(*root)
	if err != nil {
		fmt.Fprintf(stderr, "init failed: %v\n", err)
		return 1
	}
	for _, path := range created {
		fmt.Fprintln(stdout, path)
	}
	return 0
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(stderr, "usage: codex-governance config check [--repo-root PATH]")
		return 2
	}
	flags := flag.NewFlagSet("config check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("repo-root", ".", "repository root")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "config check accepts no positional arguments")
		return 2
	}
	if _, err := config.Load(filepath.Join(*root, "governance.yml")); err != nil {
		fmt.Fprintf(stderr, "config check failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "governance.yml is valid")
	return 0
}

func runValidateWorkItem(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate-work-item", flag.ContinueOnError)
	flags.SetOutput(stderr)
	workItemPath := flags.String("work-item", "", "normalized work-item JSON")
	offlineExportPath := flags.String("offline-export", "", "offline Jira export JSON")
	repoRoot := flags.String("repo-root", ".", "repository root")
	runtimeRootPath := flags.String("runtime-root", "", "runtime ledger root")
	baseSHA := flags.String("base-sha", "", "Git base SHA")
	headSHA := flags.String("head-sha", "", "Git head SHA")
	warnOnly := flags.Bool("warn", false, "report violations without a failing exit code")
	strict := flags.Bool("strict", false, "enforce violations with a failing exit code")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *warnOnly && *strict {
		fmt.Fprintln(stderr, "--warn and --strict cannot be combined")
		return 2
	}
	if *workItemPath == "" || *offlineExportPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "validate-work-item requires --work-item and --offline-export")
		return 2
	}
	item, err := workitem.Load(*workItemPath)
	if err != nil {
		fmt.Fprintf(stderr, "load work item: %v\n", err)
		return 2
	}
	export, err := jira.LoadOfflineExport(*offlineExportPath)
	if err != nil {
		fmt.Fprintf(stderr, "load offline export: %v\n", err)
		return 2
	}
	violations, err := validate.Evaluate(item, export, *repoRoot, *baseSHA, *headSHA)
	if err != nil {
		fmt.Fprintf(stderr, "validate work item: %v\n", err)
		return 2
	}
	resolvedRuntimeRoot, err := runtimeRoot(*runtimeRootPath)
	if err != nil {
		fmt.Fprintf(stderr, "resolve runtime root: %v\n", err)
		return 2
	}
	open, err := gruntime.OpenAgents(resolvedRuntimeRoot, item.Source.SubtaskKey)
	if err != nil {
		fmt.Fprintf(stderr, "check runtime: %v\n", err)
		return 2
	}
	if len(open) != 0 && item.Links.AgentException == nil {
		violations = append(violations, validate.Violation{Code: "open-agents", Message: "open agents block finalization without an approved agent exception"})
	}
	if len(violations) == 0 {
		fmt.Fprintln(stdout, "PASS work item is valid")
		return 0
	}
	for _, violation := range violations {
		fmt.Fprintf(stdout, "FAIL %s: %s\n", violation.Code, violation.Message)
	}
	if *warnOnly {
		return 0
	}
	return 1
}

func runRoadmap(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "status" && args[0] != "check") {
		fmt.Fprintln(stderr, "usage: codex-governance roadmap status|check --roadmap PATH")
		return 2
	}
	command := args[0]
	flags := flag.NewFlagSet("roadmap "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("roadmap", "", "structured roadmap YAML")
	format := flags.String("format", "table", "table, markdown, or json")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *path == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "roadmap command requires --roadmap")
		return 2
	}
	value, err := roadmap.Load(*path)
	if err != nil {
		fmt.Fprintf(stderr, "load roadmap: %v\n", err)
		return 2
	}
	if command == "check" {
		issues := value.Check()
		if len(issues) == 0 {
			fmt.Fprintln(stdout, "PASS roadmap is valid")
			return 0
		}
		for _, issue := range issues {
			fmt.Fprintf(stdout, "FAIL %s\n", issue)
		}
		return 1
	}
	output, err := value.Render(*format)
	if err != nil {
		fmt.Fprintf(stderr, "render roadmap: %v\n", err)
		return 2
	}
	fmt.Fprint(stdout, output)
	return 0
}

func runSync(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "verify adopted release matches manifest")
	dryRun := flags.Bool("dry-run", false, "report required migration changes")
	manifestPath := flags.String("manifest", "", "release manifest JSON")
	repoRoot := flags.String("repo-root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *check == *dryRun || *manifestPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "sync requires exactly one of --check or --dry-run and --manifest")
		return 2
	}
	cfg, err := config.Load(filepath.Join(*repoRoot, "governance.yml"))
	if err != nil {
		fmt.Fprintf(stderr, "load governance config: %v\n", err)
		return 2
	}
	manifest, err := syncer.LoadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "load release manifest: %v\n", err)
		return 2
	}
	changes := syncer.Compare(syncer.Adoption{Release: cfg.Upstream.Release, SourceCommit: cfg.Upstream.SourceCommit, FormatVersion: cfg.Upstream.FormatVersion}, manifest)
	artifactIssues := syncer.VerifyArtifacts(manifest, *repoRoot)
	if *dryRun {
		fmt.Fprintf(stdout, "Current adopted release: %s\nTarget release: %s\n", cfg.Upstream.Release, manifest.Release)
		if len(changes) == 0 {
			fmt.Fprintln(stdout, "No migration changes required.")
			return 0
		}
		for _, change := range changes {
			fmt.Fprintf(stdout, "- %s\n", change)
		}
		return 0
	}
	if len(changes) != 0 {
		for _, change := range changes {
			fmt.Fprintf(stdout, "FAIL %s\n", change)
		}
		return 1
	}
	if len(artifactIssues) != 0 {
		for _, issue := range artifactIssues {
			fmt.Fprintf(stdout, "FAIL %s\n", issue)
		}
		return 1
	}
	fmt.Fprintln(stdout, "PASS adopted release matches manifest")
	return 0
}

func runRuntime(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: codex-governance runtime agent|check")
		return 2
	}
	if len(args) >= 2 && args[0] == "cache" && args[1] == "clear" {
		flags := flag.NewFlagSet("runtime cache clear", flag.ContinueOnError)
		flags.SetOutput(stderr)
		root := flags.String("runtime-root", "", "runtime root")
		if err := flags.Parse(args[2:]); err != nil || flags.NArg() != 0 {
			return 2
		}
		resolvedRoot, err := runtimeRoot(*root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if err := gruntime.ClearCache(resolvedRoot); err != nil {
			fmt.Fprintf(stderr, "clear runtime cache: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, "PASS runtime cache cleared")
		return 0
	}
	if args[0] == "check" {
		flags := flag.NewFlagSet("runtime check", flag.ContinueOnError)
		flags.SetOutput(stderr)
		workItem := flags.String("work-item", "", "work item key")
		root := flags.String("runtime-root", "", "runtime root")
		if err := flags.Parse(args[1:]); err != nil || *workItem == "" {
			return 2
		}
		resolvedRoot, err := runtimeRoot(*root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		open, err := gruntime.OpenAgents(resolvedRoot, *workItem)
		if err != nil {
			fmt.Fprintf(stderr, "check runtime: %v\n", err)
			return 2
		}
		if len(open) == 0 {
			fmt.Fprintln(stdout, "PASS no open agents")
			return 0
		}
		for _, event := range open {
			fmt.Fprintf(stdout, "OPEN %s %s %s\n", event.AgentID, event.Role, event.State)
		}
		return 1
	}
	if args[0] != "agent" || len(args) < 2 || !oneOf(args[1], "start", "complete", "fail", "close") {
		fmt.Fprintln(stderr, "usage: codex-governance runtime agent start|complete|fail|close")
		return 2
	}
	flags := flag.NewFlagSet("runtime agent", flag.ContinueOnError)
	flags.SetOutput(stderr)
	workItem := flags.String("work-item", "", "work item key")
	agentID := flags.String("agent-id", "", "agent ID")
	role := flags.String("role", "", "agent role")
	resultRef := flags.String("result-ref", "", "result reference")
	inputRef := flags.String("input-ref", "", "input reference")
	root := flags.String("runtime-root", "", "runtime root")
	if err := flags.Parse(args[2:]); err != nil || *workItem == "" || *agentID == "" || *role == "" {
		return 2
	}
	resolvedRoot, err := runtimeRoot(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	states := map[string]string{"start": "started", "complete": "completed", "fail": "failed", "close": "closed"}
	if err := gruntime.Record(resolvedRoot, gruntime.Event{WorkItem: *workItem, AgentID: *agentID, Role: *role, State: states[args[1]], ResultRef: *resultRef, InputRef: *inputRef}); err != nil {
		fmt.Fprintf(stderr, "record agent: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, "PASS agent event recorded")
	return 0
}

func runOllama(args []string, stdout, stderr io.Writer) int {
	if len(args) >= 2 && args[0] == "policy" && args[1] == "init" {
		flags := flag.NewFlagSet("ollama policy init", flag.ContinueOnError)
		root := flags.String("runtime-root", "", "runtime root")
		if err := flags.Parse(args[2:]); err != nil {
			return 2
		}
		resolvedRoot, err := runtimeRoot(*root)
		if err != nil {
			return 2
		}
		if err := os.MkdirAll(resolvedRoot, 0o700); err != nil {
			return 2
		}
		path := ollama.PolicyPath(resolvedRoot)
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintln(stderr, "refusing to overwrite existing policy")
			return 1
		}
		if err := os.WriteFile(path, ollama.DefaultPolicy(), 0o600); err != nil {
			return 2
		}
		fmt.Fprintln(stdout, path)
		return 0
	}
	if len(args) == 0 || args[0] != "run" {
		fmt.Fprintln(stderr, "usage: codex-governance ollama policy init|run")
		return 2
	}
	flags := flag.NewFlagSet("ollama run", flag.ContinueOnError)
	model := flags.String("model", "", "allowlisted model")
	role := flags.String("role", "", "approved role")
	task := flags.String("task-type", "", "approved task type")
	input := flags.String("input", "", "input file")
	policyPath := flags.String("policy", "", "policy path")
	root := flags.String("runtime-root", "", "runtime root")
	if err := flags.Parse(args[1:]); err != nil || *model == "" || *role == "" || *task == "" || *input == "" {
		return 2
	}
	resolvedRoot, err := runtimeRoot(*root)
	if err != nil {
		return 2
	}
	if *policyPath == "" {
		*policyPath = ollama.PolicyPath(resolvedRoot)
	}
	policy, err := ollama.LoadPolicy(*policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "load Ollama policy: %v\n", err)
		return 2
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		return 2
	}
	request := ollama.Request{Model: *model, Role: *role, TaskType: *task, Input: data}
	allowedModel, err := policy.Authorize(request)
	if err != nil {
		fmt.Fprintf(stderr, "authorize Ollama job: %v\n", err)
		return 1
	}
	if err := ollama.VerifyInstalled(ollama.Client(policy), policy, allowedModel); err != nil {
		fmt.Fprintf(stderr, "verify Ollama model: %v\n", err)
		return 1
	}
	key := gruntime.CacheKey(*model, allowedModel.ID, policy.Fingerprint, *role, *task, ollama.InputDigest(data))
	if entry, ok, err := gruntime.LoadCache(resolvedRoot, key); err == nil && ok {
		fmt.Fprintln(stdout, entry.Summary)
		return 0
	}
	output, err := ollama.Run(ollama.Client(policy), policy, request)
	if err != nil {
		fmt.Fprintf(stderr, "run Ollama job: %v\n", err)
		return 1
	}
	if err := gruntime.StoreCache(resolvedRoot, key, output); err != nil {
		return 2
	}
	fmt.Fprintln(stdout, gruntime.Redact(output))
	return 0
}

func runtimeRoot(value string) (string, error) {
	if value != "" {
		return value, nil
	}
	return gruntime.DefaultRoot()
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
