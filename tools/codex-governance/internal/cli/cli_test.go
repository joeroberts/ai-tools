package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/implementation"
	"codex-governance/internal/signature"
	"codex-governance/internal/ticketplan"
	"codex-governance/internal/workitem"
)

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() returned %d, want 0", code)
	}
	if got := stdout.String(); got == "" {
		t.Fatal("Run() wrote no help output")
	}
}

func TestRunRepositoryBaselineReportsViolations(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"repository", "baseline", "check", "--repo-root", t.TempDir()}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() returned %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "missing required baseline file .github/CODEOWNERS") {
		t.Fatalf("Run() stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("Run() stdout = %q, want empty", stdout.String())
	}
}

func TestImplementationCheckRequiresCompletionTransitionEvidence(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "run.json")
	run := implementation.Run{FormatVersion: implementation.FormatVersion, ID: "run-check", WorkItemKey: "REK-74", Adapter: "test", State: implementation.StatePreflight, TaskBundleDigest: "sha256:fixture", RoadmapImpact: workitem.RoadmapImpact{Mode: "required", RoadmapID: "program", CanonicalPath: "roadmaps/program.yaml", Phase: "1", Transition: "complete"}}
	if err := implementation.SaveRun(runPath, run); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"implementation", "check", "--run", runPath}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "completion transition evidence") {
		t.Fatalf("check without evidence = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if err := run.BindCompletionTransition("sha256:" + strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := implementation.SaveRun(runPath, run); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"implementation", "check", "--run", runPath}, &stdout, &stderr); code != 0 || stdout.String() != "PASS completion-transition binding\n" || stderr.Len() != 0 {
		t.Fatalf("check with evidence = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestValidateFinalizationRunMatchesIssueAndBinding(t *testing.T) {
	run := implementation.Run{WorkItemKey: "REK-74", RoadmapImpact: workitem.RoadmapImpact{Mode: "required", RoadmapID: "program", CanonicalPath: "roadmaps/program.yaml", Phase: "1", Transition: "complete"}}
	if err := validateFinalizationRun("REK-74", run); err == nil {
		t.Fatal("expected missing completion evidence to block finalization")
	}
	if err := run.BindCompletionTransition("sha256:" + strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := validateFinalizationRun("REK-75", run); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched issue error = %v", err)
	}
	if err := validateFinalizationRun("REK-74", run); err != nil {
		t.Fatalf("valid finalization run error = %v", err)
	}
}

func TestJiraFinalizationRequiresBoundRun(t *testing.T) {
	if code := Run([]string{"jira", "work", "finalize", "--issue", "REK-74", "--pr", "104"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("finalize without run = %d", code)
	}
}

func TestReportImplementationStartTerminalOutcomes(t *testing.T) {
	for _, test := range []struct {
		name        string
		run         implementation.Run
		diagnostics []string
		startErr    error
		wantCode    int
		wantOutput  string
	}{
		{name: "completed", run: implementation.Run{TaskID: "codex-42", State: implementation.StateImplementationComplete}, wantCode: 0, wantOutput: "PASS implementation completed codex-42"},
		{name: "escalated", run: implementation.Run{State: implementation.StateEscalated}, diagnostics: []string{"/private/run.stdout.log", "/private/run.stderr.log"}, wantCode: 1, wantOutput: "private diagnostic: /private/run.stderr.log"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := reportImplementationStart(test.run, test.diagnostics, test.startErr, &stdout, &stderr); code != test.wantCode {
				t.Fatalf("code = %d", code)
			}
			output := stdout.String() + stderr.String()
			if !strings.Contains(output, test.wantOutput) {
				t.Fatalf("output = %q, want %q", output, test.wantOutput)
			}
		})
	}
}

func TestImplementationLifecycleStateIsPublicSafe(t *testing.T) {
	if got := startLifecycleState(implementation.Run{State: implementation.StateRunning}, nil); got != "running" {
		t.Fatalf("running start state = %q", got)
	}
	if got := startLifecycleState(implementation.Run{State: implementation.StateEscalated}, nil); got != "failed" {
		t.Fatalf("escalated start state = %q", got)
	}
	if got := reconciledLifecycleState(implementation.Run{State: implementation.StateImplementationComplete}); got != "completed" {
		t.Fatalf("completed reconciliation state = %q", got)
	}
}

func TestOperationLifecycleDoesNotStoreOperationInputs(t *testing.T) {
	root := t.TempDir()
	recordOperationLifecycle(root, "ollama-run", "completed")
	data, err := os.ReadFile(filepath.Join(root, "lifecycle-events.jsonl"))
	if err != nil || !strings.Contains(string(data), `"phase":"operation"`) || strings.Contains(string(data), "model") {
		t.Fatalf("operation lifecycle = %q, %v", data, err)
	}
}

func TestRunInitAndConfigCheck(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"init", "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("init returned %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "governance.yml")); err != nil {
		t.Fatalf("governance.yml missing: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"config", "check", "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("config check returned %d: %s", code, stderr.String())
	}
}

func TestRunBootstrapPublishOwnerRequiresApprovalAndCreatesOwnerOnlySigner(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init returned %d", code)
	}
	signer := filepath.Join(ownerOnlyTestDir(t), "repository-owner.json")
	if code := Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", signer}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("bootstrap without approval = %d", code)
	}
	if _, err := os.Stat(signer); !os.IsNotExist(err) {
		t.Fatalf("bootstrap without approval created signer: %v", err)
	}
	inside := filepath.Join(root, "repository-owner.json")
	if code := Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", inside, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("bootstrap with repository-contained signer = %d", code)
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("bootstrap created repository-contained signer: %v", err)
	}
	if code := Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", signer, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("approved bootstrap = %d", code)
	}
	info, err := os.Stat(signer)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("signer permissions = %v, %v", info.Mode().Perm(), err)
	}
}

func TestRunBootstrapTechnicalOwnerPreviewAndApproval(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("init failed")
	}
	policy := filepath.Join(root, "governance.yml")
	before, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	directory := ownerOnlyTestDir(t)
	signer := filepath.Join(directory, "new", "technical-owner.json")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", signer}, &stdout, &stderr); code != 0 {
		t.Fatalf("preview = %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "fixed role=technical-owner") {
		t.Fatalf("preview = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Dir(signer)); !os.IsNotExist(err) {
		t.Fatalf("preview created directory: %v", err)
	}
	after, err := os.ReadFile(policy)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("preview changed policy: %v", err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", signer, "--approve"}, &stdout, &stderr); code != 0 {
		t.Fatalf("approval = %d: %s", code, stderr.String())
	}
	key, _, err := signature.LoadLocalTechnicalOwnerSigner(signer)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(policy)
	if err != nil || !trustedSignerMatches(cfg, key) {
		t.Fatalf("read-back agreement = %v", err)
	}
}

func TestRunBootstrapTechnicalOwnerRejectsUnsafePathsAndDuplicateTrust(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("init failed")
	}
	inside := filepath.Join(root, "technical-owner.json")
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", inside}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("inside signer = %d", code)
	}
	link := filepath.Join(ownerOnlyTestDir(t), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", filepath.Join(link, "technical-owner.json")}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("symlink escape = %d", code)
	}
	unsafe := t.TempDir()
	if err := os.Chmod(unsafe, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", filepath.Join(unsafe, "technical-owner.json")}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("unsafe path = %d", code)
	}
	signer := filepath.Join(ownerOnlyTestDir(t), "technical-owner.json")
	if err := os.WriteFile(signer, []byte("exists"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", signer}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("overwrite = %d", code)
	}
	cfg, err := config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Signing.TrustedKeys = append(cfg.Signing.TrustedKeys, config.TrustedKey{KeyID: "technical-owner", Role: "technical-owner", Algorithm: signature.Algorithm, PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="})
	if err := config.Save(filepath.Join(root, "governance.yml"), cfg); err != nil {
		t.Fatal(err)
	}
	newSigner := filepath.Join(ownerOnlyTestDir(t), "technical-owner.json")
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", newSigner}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("duplicate trust = %d", code)
	}
}

func TestRunBootstrapTechnicalOwnerValidatesCompleteAncestorChain(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("init failed")
	}
	nearest := t.TempDir()
	if err := os.Chmod(nearest, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", filepath.Join(nearest, "technical-owner.json")}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("current-user 0755 ancestor = %d", code)
	}
	replaceable := t.TempDir()
	if err := os.Chmod(replaceable, 0o777); err != nil {
		t.Fatal(err)
	}
	descendant := filepath.Join(replaceable, "owner-only")
	if err := os.Mkdir(descendant, 0o700); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", filepath.Join(descendant, "technical-owner.json")}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("replaceable ancestor = %d", code)
	}
	temporary, err := os.MkdirTemp("/tmp", "technical-owner-signer-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(temporary) })
	if err := os.Chmod(temporary, 0o700); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", filepath.Join(temporary, "technical-owner.json")}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("root-owned sticky /tmp ancestry = %d", code)
	}
}

func TestRunBootstrapTechnicalOwnerFailsClosedOnSaveCleanupAndReadback(t *testing.T) {
	for _, test := range []struct {
		name       string
		save       func(string, config.Config) error
		remove     func(string) error
		load       func(string) (config.Config, error)
		create     func(string) (signature.TrustedKey, error)
		restore    func(string, config.Config) error
		wantSigner bool
		wantOutput string
	}{
		{name: "policy failure removes signer", save: failFirstBootstrapSave()},
		{name: "cleanup failure blocks", save: failFirstBootstrapSave(), remove: func(string) error { return errors.New("cleanup failure") }, wantSigner: true, wantOutput: "cleanup failure"},
		{name: "restore failure blocks", load: mismatchedBootstrapConfigLoad(), restore: func(string, config.Config) error { return errors.New("restore failure") }, wantSigner: false, wantOutput: "restore failure"},
		{name: "read-back mismatch", load: mismatchedBootstrapConfigLoad()},
		{name: "duplicate role read-back", load: duplicateBootstrapConfigLoad()},
		{name: "configuration reload failure", load: failingBootstrapConfigReload()},
		{name: "signer reload failure", create: unreadableTechnicalOwnerCreate},
		{name: "role mismatch", create: mismatchedTechnicalOwnerCreate},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetBootstrapHooks(t)
			root := t.TempDir()
			if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
				t.Fatal("init failed")
			}
			if test.save != nil {
				saveBootstrapConfig = test.save
			}
			if test.remove != nil {
				removeBootstrapSigner = test.remove
			}
			if test.load != nil {
				loadBootstrapConfig = test.load
			}
			if test.create != nil {
				createBootstrapSigner = test.create
			}
			if test.restore != nil {
				restoreBootstrapConfig = test.restore
			}
			before, err := config.Load(filepath.Join(root, "governance.yml"))
			if err != nil {
				t.Fatal(err)
			}
			signer := filepath.Join(ownerOnlyTestDir(t), "technical-owner.json")
			var stderr bytes.Buffer
			if code := Run([]string{"implementation", "bootstrap-technical-owner", "--repo-root", root, "--signer", signer, "--approve"}, &bytes.Buffer{}, &stderr); code != 1 {
				t.Fatalf("bootstrap = %d", code)
			}
			if test.wantOutput != "" && !strings.Contains(stderr.String(), test.wantOutput) {
				t.Fatalf("blocking cleanup output = %q", stderr.String())
			}
			_, err = os.Stat(signer)
			if test.wantSigner != !os.IsNotExist(err) {
				t.Fatalf("signer exists = %t, err=%v", !os.IsNotExist(err), err)
			}
			after, err := config.Load(filepath.Join(root, "governance.yml"))
			if test.restore == nil && (err != nil || !reflect.DeepEqual(before, after)) {
				t.Fatalf("config restored = %t, err=%v", reflect.DeepEqual(before, after), err)
			}
		})
	}
}

func resetBootstrapHooks(t *testing.T) {
	t.Helper()
	loadBootstrapConfig = config.Load
	saveBootstrapConfig = config.Save
	restoreBootstrapConfig = config.Save
	createBootstrapSigner = signature.CreateLocalTechnicalOwnerSigner
	removeBootstrapSigner = os.Remove
	t.Cleanup(func() {
		loadBootstrapConfig = config.Load
		saveBootstrapConfig = config.Save
		restoreBootstrapConfig = config.Save
		createBootstrapSigner = signature.CreateLocalTechnicalOwnerSigner
		removeBootstrapSigner = os.Remove
	})
}

func failFirstBootstrapSave() func(string, config.Config) error {
	calls := 0
	return func(path string, cfg config.Config) error {
		calls++
		if calls == 1 {
			return errors.New("policy failure")
		}
		return config.Save(path, cfg)
	}
}

func mismatchedTechnicalOwnerCreate(path string) (signature.TrustedKey, error) {
	key, err := signature.CreateLocalTechnicalOwnerSigner(path)
	key.Role = "repository-owner"
	return key, err
}

func unreadableTechnicalOwnerCreate(path string) (signature.TrustedKey, error) {
	key, err := signature.CreateLocalTechnicalOwnerSigner(path)
	if err != nil {
		return signature.TrustedKey{}, err
	}
	return key, os.Chmod(path, 0o644)
}

func mismatchedBootstrapConfigLoad() func(string) (config.Config, error) {
	loads := 0
	return func(requestedPath string) (config.Config, error) {
		cfg, err := config.Load(requestedPath)
		loads++
		if err == nil && loads > 1 {
			for index := range cfg.Signing.TrustedKeys {
				if cfg.Signing.TrustedKeys[index].Role == "technical-owner" {
					cfg.Signing.TrustedKeys[index].KeyID = "sha256:readback-mismatch"
				}
			}
		}
		return cfg, err
	}
}

func failingBootstrapConfigReload() func(string) (config.Config, error) {
	loads := 0
	return func(requestedPath string) (config.Config, error) {
		loads++
		if loads > 1 {
			return config.Config{}, errors.New("configuration reload failure")
		}
		return config.Load(requestedPath)
	}
}

func duplicateBootstrapConfigLoad() func(string) (config.Config, error) {
	loads := 0
	return func(requestedPath string) (config.Config, error) {
		cfg, err := config.Load(requestedPath)
		loads++
		if err == nil && loads > 1 {
			for _, key := range cfg.Signing.TrustedKeys {
				if key.Role == "technical-owner" {
					key.KeyID = "sha256:duplicate-technical-owner"
					cfg.Signing.TrustedKeys = append(cfg.Signing.TrustedKeys, key)
					break
				}
			}
		}
		return cfg, err
	}
}

func TestRunIssuePublishBindsSignerTargetAndLineageWithoutRemoteMutation(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init returned %d", code)
	}
	signer := filepath.Join(ownerOnlyTestDir(t), "repository-owner.json")
	if code := Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", signer, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("bootstrap returned %d", code)
	}
	cfg, err := config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Signing.RepositoryID = "github.test/acme/repo"
	localKey, _, err := signature.LoadLocalRepositoryOwnerSigner(signer)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := signature.CreateLocalRepositoryOwnerSigner(filepath.Join(ownerOnlyTestDir(t), "other-owner.json"))
	if err != nil {
		t.Fatal(err)
	}
	for index := range cfg.Signing.TrustedKeys {
		if cfg.Signing.TrustedKeys[index].Role == "repository-owner" {
			cfg.Signing.TrustedKeys[index].PublicKey = otherKey.PublicKey
		}
	}
	if err := config.Save(filepath.Join(root, "governance.yml"), cfg); err != nil {
		t.Fatal(err)
	}

	remote := t.TempDir()
	cliGit(t, remote, "init", "--bare")
	worktree := t.TempDir()
	cliGit(t, worktree, "init")
	cliGit(t, worktree, "config", "user.email", "test@example.test")
	cliGit(t, worktree, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(worktree, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cliGit(t, worktree, "add", "base.txt")
	cliGit(t, worktree, "commit", "-m", "base")
	baseSHA := strings.TrimSpace(cliGit(t, worktree, "rev-parse", "HEAD"))
	cliGit(t, worktree, "remote", "add", "origin", remote)
	cliGit(t, worktree, "push", "origin", "HEAD:refs/heads/main")
	cliGit(t, worktree, "switch", "-c", "codex/REK-30-implementation")
	if err := os.WriteFile(filepath.Join(worktree, "change.txt"), []byte("change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cliGit(t, worktree, "add", "change.txt")
	cliGit(t, worktree, "commit", "-m", "change")
	commitSHA := strings.TrimSpace(cliGit(t, worktree, "rev-parse", "HEAD"))
	run := implementation.Run{
		FormatVersion: implementation.FormatVersion, ID: "run-issue-publish", WorkItemKey: "REK-30",
		Adapter: "headless-codex", State: implementation.StateLocallyCommitted, BaseSHA: baseSHA,
		Branch: "codex/REK-30-implementation", CommitSHA: commitSHA, TaskBundleDigest: "sha256:fixture",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	runPath := filepath.Join(t.TempDir(), "run.json")
	if err := implementation.SaveRun(runPath, run); err != nil {
		t.Fatal(err)
	}
	remoteBefore := cliGit(t, worktree, "ls-remote", "origin", "refs/heads/main")
	issue := func(output string) int {
		return Run([]string{
			"implementation", "issue-publish", "--run", runPath, "--signer", signer,
			"--output", output, "--worktree", worktree, "--remote", "origin",
			"--target-branch", "main", "--repo-root", root, "--approve",
		}, &bytes.Buffer{}, &bytes.Buffer{})
	}
	rejectedOutput := filepath.Join(t.TempDir(), "rejected.json")
	if code := issue(rejectedOutput); code != 1 {
		t.Fatalf("issuance with mismatched trusted public key = %d", code)
	}
	if _, err := os.Stat(rejectedOutput); !os.IsNotExist(err) {
		t.Fatalf("mismatched signer wrote authorization: %v", err)
	}
	for index := range cfg.Signing.TrustedKeys {
		if cfg.Signing.TrustedKeys[index].Role == "repository-owner" {
			cfg.Signing.TrustedKeys[index].PublicKey = localKey.PublicKey
		}
	}
	if err := config.Save(filepath.Join(root, "governance.yml"), cfg); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(ownerOnlyTestDir(t), "authorization.json")
	if code := issue(output); code != 0 {
		t.Fatalf("approved issuance = %d", code)
	}
	if remoteAfter := cliGit(t, worktree, "ls-remote", "origin", "refs/heads/main"); remoteAfter != remoteBefore {
		t.Fatalf("issuance mutated remote ref: before=%q after=%q", remoteBefore, remoteAfter)
	}
	authorization, err := implementation.LoadSignedPublicationAuthorization(output, cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if authorization.Payload.FormatVersion != 2 || authorization.Payload.ImplementationBaseSHA != baseSHA || authorization.Payload.ExpectedTargetSHA != baseSHA || authorization.Payload.CommitSHA != commitSHA {
		t.Fatalf("issued payload = %#v", authorization.Payload)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("authorization permissions = %v, %v", info.Mode().Perm(), err)
	}
	tooLong := filepath.Join(t.TempDir(), "too-long.json")
	code := Run([]string{
		"implementation", "issue-publish", "--run", runPath, "--signer", signer,
		"--output", tooLong, "--worktree", worktree, "--remote", "origin",
		"--target-branch", "main", "--repo-root", root, "--expires-in", "25h", "--approve",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 2 {
		t.Fatalf("issuance with unbounded expiry = %d", code)
	}
	if _, err := os.Stat(tooLong); !os.IsNotExist(err) {
		t.Fatalf("unbounded expiry wrote authorization: %v", err)
	}
}

func TestAuthorizeWorkflowIssuesReloadableOwnerSignedEnvelope(t *testing.T) {
	root := t.TempDir()
	if Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}) != 0 {
		t.Fatal("init failed")
	}
	signer := filepath.Join(ownerOnlyTestDir(t), "repository-owner.json")
	if Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", signer, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}) != 0 {
		t.Fatal("bootstrap signer failed")
	}
	cfg, err := config.Load(filepath.Join(root, "governance.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Signing.RepositoryID = "github.test/acme/repo"
	if err := config.Save(filepath.Join(root, "governance.yml"), cfg); err != nil {
		t.Fatal(err)
	}
	digest := signature.Digest([]byte("fixture"))
	payload := implementation.WorkflowAuthorizationPayload{FormatVersion: 1, RepositoryID: cfg.Signing.RepositoryID, GitHubIssue: "22", StoryKey: "REK-94", SubtaskKey: "REK-97", PlanContractDigest: digest, SourceDigests: []string{digest, signature.Digest([]byte("source-2")), signature.Digest([]byte("source-3"))}, BaseSHA: "abcdef1", AllowedPaths: []string{"internal/cli"}, MaxChangedFiles: 9, MaxChangedLines: 800, AcceptanceCriteria: []string{"bound authority"}, ReviewCycleLimit: 2, AllowedOperations: []string{"jira-work-start"}, Branch: "codex/issue-22", Remote: "origin", PRTargetBranch: "main", DerivationRules: []string{"commit-sha"}, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	payloadPath := filepath.Join(ownerOnlyTestDir(t), "payload.json")
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(payloadPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(ownerOnlyTestDir(t), "authorization.json")
	if code := Run([]string{"implementation", "authorize-workflow", "--repo-root", root, "--payload", payloadPath, "--signer", signer, "--output", output, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("authorize workflow = %d", code)
	}
	if _, err := implementation.LoadSignedWorkflowAuthorization(output, cfg, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizeWorkflowRejectsConcatenatedPayload(t *testing.T) {
	root := t.TempDir()
	if Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}) != 0 {
		t.Fatal("init failed")
	}
	signer := filepath.Join(ownerOnlyTestDir(t), "repository-owner.json")
	if Run([]string{"implementation", "bootstrap-publish-owner", "--repo-root", root, "--signer", signer, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}) != 0 {
		t.Fatal("bootstrap signer failed")
	}
	payloadPath := filepath.Join(ownerOnlyTestDir(t), "payload.json")
	if err := os.WriteFile(payloadPath, []byte("{}{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(ownerOnlyTestDir(t), "authorization.json")
	if code := Run([]string{"implementation", "authorize-workflow", "--repo-root", root, "--payload", payloadPath, "--signer", signer, "--output", output, "--approve"}, &bytes.Buffer{}, &bytes.Buffer{}); code == 0 {
		t.Fatal("concatenated payload was accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("authorization output exists after rejected payload: %v", err)
	}
}

func cliGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func ownerOnlyTestDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestRunValidateHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"validate-work-item", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate-work-item --help returned %d", code)
	}
}

func TestRunValidateWorkItemRejectsUnsignedExport(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init returned %d", code)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{
		"validate-work-item",
		"--work-item", filepath.Join("..", "..", "testdata", "work-items", "valid.json"),
		"--offline-export", filepath.Join("..", "..", "testdata", "jira-exports", "valid.json"),
		"--repo-root", root,
	}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "load signed offline export") {
		t.Fatalf("validate unsigned export = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunRoadmapStatus(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	path := filepath.Join("..", "..", "docs", "roadmaps", "go-cli-migration.yaml")

	if code := Run([]string{"roadmap", "status", "--roadmap", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("roadmap status returned %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Adoption And Synchronization")) {
		t.Fatalf("roadmap status output = %q", stdout.String())
	}
}

func TestRunRoadmapCommandsRejectInconsistentStates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roadmap.yaml")
	if err := os.WriteFile(path, []byte("id: test\ntitle: Test\nstatus: complete\nphases:\n  - id: 1\n    name: Phase 1\n    status: in-progress\n    evidence: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"check", "status"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run([]string{"roadmap", command, "--roadmap", path}, &stdout, &stderr); code == 0 {
			t.Fatalf("roadmap %s unexpectedly passed: stdout=%q stderr=%q", command, stdout.String(), stderr.String())
		}
	}
}

func TestRunSyncDryRun(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init returned %d", code)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	manifest := filepath.Join("..", "..", "testdata", "releases", "1.0.0.json")

	if code := Run([]string{"sync", "--dry-run", "--manifest", manifest, "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("sync dry-run returned %d: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Target release: 1.0.0")) {
		t.Fatalf("sync output = %q", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() returned %d, want 2", code)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("Run() wrote no error output")
	}
}

func TestRunJiraPlanGenerateDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--constraints", "constraints.json", "--output", "plan.json", "--manager-timeout", "1m", "--manager-wait-delay", "5s", "--dry-run"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("generate dry run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunJiraWorkUpdatePreviewsCommitWithoutCredentials(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	commit := strings.Repeat("a", 40)
	code := Run([]string{"jira", "work", "update", "--issue", "REK-5", "--kind", "commit", "--commit", commit, "--scope", "Add work records", "--check", "go test ./internal/jira", "--evidence", "/private/evidence.json"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "PREVIEW Jira comment for REK-5") || !strings.Contains(stdout.String(), "Commit: "+commit) {
		t.Fatalf("work preview = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunJiraWorkUpdateRequiresCompleteBlocker(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"jira", "work", "update", "--issue", "REK-5", "--kind", "blocker", "--blocker", "Jira unavailable"}, &bytes.Buffer{}, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--impact") {
		t.Fatalf("incomplete blocker = %d, stderr=%q", code, stderr.String())
	}
}

func TestRunJiraWorkUpdateRendersEvidenceSummaryNotPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.json")
	if err := os.WriteFile(path, []byte(`[{"kind":"reviewer","executor":"gemma","outcome":"passed"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	code := Run([]string{"jira", "work", "update", "--issue", "REK-10", "--kind", "commit", "--commit", strings.Repeat("a", 40), "--scope", "Render evidence", "--check", "make test: passed", "--evidence", "placeholder", "--evidence-summary", path}, &stdout, &bytes.Buffer{})
	if code != 0 || !strings.Contains(stdout.String(), "reviewer: passed") || strings.Contains(stdout.String(), path) {
		t.Fatalf("summary preview = %d, %q", code, stdout.String())
	}
}

func TestRunJiraWorkUpdateApproveRequiresSignedAuthorization(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	var stderr bytes.Buffer
	commit := strings.Repeat("a", 40)
	code := Run([]string{"jira", "work", "update", "--issue", "REK-5", "--kind", "commit", "--commit", commit, "--scope", "Add work records", "--check", "go test ./internal/jira", "--evidence", "/private/evidence.json", "--approve"}, &bytes.Buffer{}, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "signed --authorization") {
		t.Fatalf("approve without authorization = %d, stderr=%q", code, stderr.String())
	}
}

func TestRunJiraPlanGenerateVerboseDryRun(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--constraints", "constraints.json", "--output", "plan.json", "--manager-timeout", "1m", "--manager-wait-delay", "5s", "--dry-run", "--verbose"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would dispatch hosted manager and local reviewer/verifier")) {
		t.Fatalf("verbose generate dry run = %d, stderr=%q", code, stderr.String())
	}
}

func TestRunJiraPlanCommandsRejectInvalidManagerTiming(t *testing.T) {
	generate := []string{"jira", "plan", "generate", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--constraints", "constraints.json", "--output", "plan.json", "--dry-run"}
	decompose := []string{"jira", "plan", "decompose", "--prd", "prd.md", "--spec", "spec.md", "--roadmap", "roadmap.md", "--output", "plan.json"}
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "generate missing", args: generate},
		{name: "generate zero timeout", args: append(append([]string{}, generate...), "--manager-timeout", "0s", "--manager-wait-delay", "1s")},
		{name: "generate negative wait", args: append(append([]string{}, generate...), "--manager-timeout", "1s", "--manager-wait-delay=-1s")},
		{name: "decompose missing", args: decompose},
		{name: "decompose negative timeout", args: append(append([]string{}, decompose...), "--manager-timeout=-1s", "--manager-wait-delay", "1s")},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(test.args, &stdout, &stderr); code != 2 {
				t.Fatalf("Run() = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), "requires positive --manager-timeout and --manager-wait-delay") {
				t.Fatalf("timing error is not actionable: %q", stderr.String())
			}
		})
	}
}

func TestRunJiraConstraintsAssignRequiresDecompositionAndAssignment(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"jira", "constraints", "assign", "--output", "constraints.json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("constraints assign = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestWriteJiraPublicationRecordIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if err := writeJiraPublicationRecord(path, jiraPublicationRecord{PlanDigest: "sha256:abc", Status: "creating"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("publication record permissions = %v, %v", info.Mode().Perm(), err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(`"status": "creating"`)) {
		t.Fatalf("publication record = %q, %v", data, err)
	}
}

func TestRunJiraPlanCreateDryRunUsesApprovedWorkflowWithoutWritingRecord(t *testing.T) {
	root := t.TempDir()
	if code := Run([]string{"init", "--repo-root", root}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("init = %d", code)
	}
	for _, name := range []string{"prd.md", "spec.md", "roadmap.md"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "valid", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "testdata", "ticket-plans", "valid", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan ticketplan.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Status = "approved"
	planPath := filepath.Join(root, "plan.json")
	data, err = json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := ticketplan.FileDigest(planPath)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := ticketplan.NewWorkflowState(root, planPath, digest, "approved", plan.Sources)
	if err != nil {
		t.Fatal(err)
	}
	contractData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(root, "contract.json")
	if err := os.WriteFile(contractPath, contractData, 0o600); err != nil {
		t.Fatal(err)
	}
	workflow.ContractPath, workflow.ContractDigest = contractPath, plan.ContractDigest
	workflow.ApprovedBy, workflow.ApprovedAt = "stakeholder@example.test", time.Now().UTC()
	workflowPath := filepath.Join(root, "workflow.json")
	if err := ticketplan.SaveWorkflow(workflowPath, workflow); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "governance.yml")
	if err := os.WriteFile(configPath, []byte(strings.Replace(string(mustReadFile(t, configPath)), "project: \"\"", "project: \"CG\"", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(root, "result.json")
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"jira", "plan", "create", "--plan", planPath, "--workflow", workflowPath, "--repo-root", root, "--result", resultPath, "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("create dry run = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("DRY RUN would create Story")) {
		t.Fatalf("create dry-run output = %q", stdout.String())
	}
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote a publication record: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRunJiraPlanValidateChecksCurrentSources(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "ticket-plans", "valid.json")
	var stdout bytes.Buffer
	if code := Run([]string{"jira", "plan", "validate", "--plan", plan, "--repo-root", t.TempDir()}, &stdout, &bytes.Buffer{}); code != 1 {
		t.Fatalf("validate = %d, stdout=%q", code, stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("prd source is unavailable")) {
		t.Fatalf("validate stdout=%q", stdout.String())
	}
}

func TestRunJiraPlanValidateValidFixture(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "plan.json")
	contract := filepath.Join("..", "..", "testdata", "ticket-plans", "valid", "contract.json")
	root := filepath.Join("..", "..")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"jira", "plan", "validate", "--plan", plan, "--contract", contract, "--repo-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("validate = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("PASS ticket plan is valid")) {
		t.Fatalf("validate stdout=%q", stdout.String())
	}
}

func TestRunJiraPlanValidateRejectsContractDriftAndLegacyInvocation(t *testing.T) {
	root := filepath.Join("..", "..")
	planPath := filepath.Join(root, "testdata", "ticket-plans", "valid", "plan.json")
	contractPath := filepath.Join(root, "testdata", "ticket-plans", "valid", "contract.json")
	var stdout bytes.Buffer
	if code := Run([]string{"jira", "plan", "validate", "--plan", planPath, "--repo-root", root}, &stdout, &bytes.Buffer{}); code != 1 || !strings.Contains(stdout.String(), "unsupported legacy ticket plan") {
		t.Fatalf("legacy validate = %d, output=%q", code, stdout.String())
	}
	plan := ticketplan.Plan{}
	data := mustReadFile(t, planPath)
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Subtasks[0].Phase = "Changed"
	tampered := filepath.Join(t.TempDir(), "plan.json")
	data, _ = json.Marshal(plan)
	if err := os.WriteFile(tampered, data, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := Run([]string{"jira", "plan", "validate", "--plan", tampered, "--contract", contractPath, "--repo-root", root}, &stdout, &bytes.Buffer{}); code != 1 || !strings.Contains(stdout.String(), "does not match authority contract") {
		t.Fatalf("contract drift validate = %d, output=%q", code, stdout.String())
	}
}

func TestRunImplementationAdoptRequiresAuditEvidence(t *testing.T) {
	args := []string{"implementation", "adopt", "--run", "run.json", "--bundle", "bundle.json", "--candidate-worktree", ".", "--review-evidence", "review.json", "--check-evidence", "checks.json", "--registry", filepath.Join(t.TempDir(), "registry"), "--reason", "reason", "--issued-at", "2026-07-20T00:00:00Z", "--expires-at", "2026-07-20T01:00:00Z"}
	if code := Run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Fatalf("implementation adopt without audit evidence = %d", code)
	}
}

func TestRunImplementationAdoptHelpSucceeds(t *testing.T) {
	if code := Run([]string{"implementation", "adopt", "--help"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("implementation adopt help = %d", code)
	}
}
