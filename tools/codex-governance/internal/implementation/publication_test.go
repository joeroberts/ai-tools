package implementation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
	"codex-governance/internal/workitem"
)

func publicationRun() Run {
	return Run{
		FormatVersion: FormatVersion,
		WorkItemKey:   "CG-42",
		State:         StateLocallyCommitted,
		Branch:        "codex/CG-42-implementation",
		CommitSHA:     "0123456789abcdef0123456789abcdef01234567",
	}
}

func TestCompletionTransitionEvidenceIsRequiredOnlyForCompletion(t *testing.T) {
	run := publicationRun()
	run.RoadmapImpact = workitem.RoadmapImpact{Mode: "required", RoadmapID: "program", CanonicalPath: "roadmaps/program.yaml", Phase: "1", Transition: "complete"}
	if err := run.RequireCompletionTransition(); err == nil {
		t.Fatal("completion run did not require transition evidence")
	}
	if err := run.BindCompletionTransition("sha256:" + strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := run.RequireCompletionTransition(); err != nil {
		t.Fatal(err)
	}
	run.RoadmapImpact.Transition = "start"
	run.TransitionDigest = ""
	if err := run.RequireCompletionTransition(); err != nil {
		t.Fatalf("non-completing run was blocked: %v", err)
	}
}

func TestSignedPublicationAuthorizationBindsRunAndConsumesOperationsSeparately(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Signing: config.Signing{
		FormatVersion: 1,
		RepositoryID:  "github.test/acme/repo",
		TrustedKeys: []config.TrustedKey{{
			KeyID: "repository-owner-1", Role: "repository-owner", Algorithm: signature.Algorithm,
			PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		}},
	}}
	run := publicationRun()
	run.ID = "run-42"
	run.BaseSHA = "fedcba9876543210fedcba9876543210fedcba98"
	remoteURL := "https://example.test/acme/repo.git"
	payload, err := json.Marshal(PublicationAuthorizationPayload{
		FormatVersion: 1, WorkItemKey: run.WorkItemKey, RunID: run.ID, RepositoryID: cfg.Signing.RepositoryID,
		Remote: "origin", RemoteFingerprint: RemoteFingerprint(remoteURL), Branch: run.Branch,
		ExpectedBaseSHA: run.BaseSHA, CommitSHA: run.CommitSHA, PRTargetBranch: "main",
		AllowedOperations: []string{"push", "create-pr"},
	})
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().Add(time.Hour)
	envelope, err := signature.Sign(payload, "repository-owner-1", "repository-owner", privateKey, time.Now(), &expiresAt)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "authorization.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	authorization, err := LoadSignedPublicationAuthorization(path, cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSignedPublication(run, authorization, "push", remoteURL, cfg.Signing.RepositoryID); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSignedPublication(run, authorization, "push", "https://example.test/acme/other.git", cfg.Signing.RepositoryID); err == nil {
		t.Fatal("accepted a different remote URL")
	}
	runtimeRoot := t.TempDir()
	if err := ConsumeSignedAuthorization(runtimeRoot, authorization, "push"); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeSignedAuthorization(runtimeRoot, authorization, "push"); err == nil {
		t.Fatal("consumed push authorization more than once")
	}
	if err := ConsumeSignedAuthorization(runtimeRoot, authorization, "create-pr"); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePublicationWorktreeRequiresExactBranchCommitAndBase(t *testing.T) {
	worktree := t.TempDir()
	publicationGit(t, worktree, "init")
	publicationGit(t, worktree, "config", "user.email", "test@example.test")
	publicationGit(t, worktree, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(worktree, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	publicationGit(t, worktree, "add", "base.txt")
	publicationGit(t, worktree, "commit", "-m", "base")
	baseSHA := strings.TrimSpace(publicationGit(t, worktree, "rev-parse", "HEAD"))
	targetBranch := strings.TrimSpace(publicationGit(t, worktree, "branch", "--show-current"))
	publicationGit(t, worktree, "remote", "add", "origin", worktree)
	publicationGit(t, worktree, "switch", "-c", "codex/CG-42-implementation")
	if err := os.WriteFile(filepath.Join(worktree, "change.txt"), []byte("change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	publicationGit(t, worktree, "add", "change.txt")
	publicationGit(t, worktree, "commit", "-m", "change")
	commitSHA := strings.TrimSpace(publicationGit(t, worktree, "rev-parse", "HEAD"))
	run := publicationRun()
	run.BaseSHA, run.CommitSHA = baseSHA, commitSHA
	if err := ValidatePublicationWorktree(worktree, run); err != nil {
		t.Fatal(err)
	}
	authorization := SignedPublicationAuthorization{Payload: PublicationAuthorizationPayload{
		Remote: "origin", PRTargetBranch: targetBranch, ExpectedBaseSHA: baseSHA,
	}}
	if err := ValidateAuthorizedRemoteBase(worktree, authorization); err != nil {
		t.Fatal(err)
	}
	authorization.Payload.ExpectedBaseSHA = commitSHA
	if err := ValidateAuthorizedRemoteBase(worktree, authorization); err == nil {
		t.Fatal("accepted a moved or mismatched target ref")
	}
	run.CommitSHA = baseSHA
	if err := ValidatePublicationWorktree(worktree, run); err == nil {
		t.Fatal("accepted a different checked-out commit")
	}
}

func TestVersion2AuthorizationValidatesIndependentLineageAndTarget(t *testing.T) {
	worktree := t.TempDir()
	publicationGit(t, worktree, "init")
	publicationGit(t, worktree, "config", "user.email", "test@example.test")
	publicationGit(t, worktree, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(worktree, "base.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	publicationGit(t, worktree, "add", "base.txt")
	publicationGit(t, worktree, "commit", "-m", "implementation base")
	implementationBase := strings.TrimSpace(publicationGit(t, worktree, "rev-parse", "HEAD"))
	targetBranch := strings.TrimSpace(publicationGit(t, worktree, "branch", "--show-current"))
	if err := os.WriteFile(filepath.Join(worktree, "target.txt"), []byte("target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	publicationGit(t, worktree, "add", "target.txt")
	publicationGit(t, worktree, "commit", "-m", "target update")
	expectedTarget := strings.TrimSpace(publicationGit(t, worktree, "rev-parse", "HEAD"))
	publicationGit(t, worktree, "remote", "add", "origin", worktree)
	publicationGit(t, worktree, "switch", "-c", "codex/CG-42-implementation")
	if err := os.WriteFile(filepath.Join(worktree, "change.txt"), []byte("change\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	publicationGit(t, worktree, "add", "change.txt")
	publicationGit(t, worktree, "commit", "-m", "change")
	commitSHA := strings.TrimSpace(publicationGit(t, worktree, "rev-parse", "HEAD"))
	authorization := SignedPublicationAuthorization{Payload: PublicationAuthorizationPayload{
		FormatVersion: 2, Remote: "origin", PRTargetBranch: targetBranch,
		ImplementationBaseSHA: implementationBase, ExpectedTargetSHA: expectedTarget, CommitSHA: commitSHA,
	}}
	if err := ValidateAuthorizedLineage(worktree, authorization); err != nil {
		t.Fatalf("validate v2 lineage: %v", err)
	}
	if err := ValidateAuthorizedRemoteBase(worktree, authorization); err != nil {
		t.Fatalf("validate v2 target: %v", err)
	}
	authorization.Payload.ExpectedTargetSHA = implementationBase
	if err := ValidateAuthorizedRemoteBase(worktree, authorization); err == nil {
		t.Fatal("accepted a moved version-2 target ref")
	}
	authorization.Payload.ExpectedTargetSHA = expectedTarget
	authorization.Payload.ImplementationBaseSHA = strings.Repeat("a", 40)
	if err := ValidateAuthorizedLineage(worktree, authorization); err == nil {
		t.Fatal("accepted a commit outside the version-2 implementation lineage")
	}
}

func TestGitHubRepositoryRequiresOwnerNameIdentity(t *testing.T) {
	if repository, err := GitHubRepository("github.com/acme/repo"); err != nil || repository != "acme/repo" {
		t.Fatalf("GitHubRepository() = %q, %v", repository, err)
	}
	if _, err := GitHubRepository("gitlab.com/acme/repo"); err == nil {
		t.Fatal("accepted a non-GitHub repository identity")
	}
}

func TestSuccessorPublicationStatePreservesPredecessorRun(t *testing.T) {
	authorization := SignedPublicationAuthorization{Digest: "sha256:" + strings.Repeat("a", 64), Payload: PublicationAuthorizationPayload{SuccessorRecordID: "adoption-" + strings.Repeat("b", 64)}}
	run := publicationRun()
	run.State = StatePushed
	run.PullRequestURL = "https://example.test/pr/1"
	root := t.TempDir()
	if err := SaveSuccessorPublicationState(root, authorization, run); err != nil {
		t.Fatal(err)
	}
	state, err := LoadSuccessorPublicationState(root, authorization)
	if err != nil {
		t.Fatal(err)
	}
	if state.State != StatePushed || state.PullRequestURL != run.PullRequestURL {
		t.Fatalf("unexpected successor state: %+v", state)
	}
}

func publicationGit(t *testing.T, worktree string, args ...string) string {
	t.Helper()
	output, err := git(worktree, args...)
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
