package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

// PublicationAuthorizationPayload is the signed, externally issued authority
// for one run's remote publication. The CLI verifies it; it never signs it.
type PublicationAuthorizationPayload struct {
	FormatVersion         int      `json:"format_version"`
	WorkItemKey           string   `json:"work_item_key"`
	RunID                 string   `json:"run_id"`
	RepositoryID          string   `json:"repository_id"`
	Remote                string   `json:"remote"`
	RemoteFingerprint     string   `json:"remote_fingerprint"`
	Branch                string   `json:"branch"`
	ExpectedBaseSHA       string   `json:"expected_base_sha"`
	ImplementationBaseSHA string   `json:"implementation_base_sha,omitempty"`
	ExpectedTargetSHA     string   `json:"expected_target_sha,omitempty"`
	CommitSHA             string   `json:"commit_sha"`
	PRTargetBranch        string   `json:"pr_target_branch"`
	AllowedOperations     []string `json:"allowed_operations"`
}

type SignedPublicationAuthorization struct {
	Envelope signature.Envelope
	Payload  PublicationAuthorizationPayload
	Digest   string
}

type AuthorizationConsumption struct {
	AuthorizationDigest string   `json:"authorization_digest"`
	UsedOperations      []string `json:"used_operations"`
}

// LoadSignedPublicationAuthorization verifies an externally signed
// repository-owner authorization against current repository policy.
func LoadSignedPublicationAuthorization(path string, cfg config.Config, now time.Time) (SignedPublicationAuthorization, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return SignedPublicationAuthorization{}, err
	}
	var envelope signature.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return SignedPublicationAuthorization{}, fmt.Errorf("parse signed publication authorization: %w", err)
	}
	registry, err := cfg.TrustedKeyRegistry()
	if err != nil {
		return SignedPublicationAuthorization{}, err
	}
	if err := registry.Verify(envelope, []string{"repository-owner"}, now); err != nil {
		return SignedPublicationAuthorization{}, fmt.Errorf("verify signed publication authorization: %w", err)
	}
	var payload PublicationAuthorizationPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return SignedPublicationAuthorization{}, fmt.Errorf("parse publication authorization payload: %w", err)
	}
	if err := validatePublicationPayload(payload); err != nil {
		return SignedPublicationAuthorization{}, err
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return SignedPublicationAuthorization{}, err
	}
	canonical, err := signature.Canonicalize(encoded)
	if err != nil {
		return SignedPublicationAuthorization{}, err
	}
	return SignedPublicationAuthorization{Envelope: envelope, Payload: payload, Digest: signature.Digest(canonical)}, nil
}

func ValidateSignedPublication(run Run, authorization SignedPublicationAuthorization, operation, remoteURL, repositoryID string) error {
	payload := authorization.Payload
	implementationBase := payload.ExpectedBaseSHA
	if payload.FormatVersion == 2 {
		implementationBase = payload.ImplementationBaseSHA
	}
	if repositoryID == "" || payload.RepositoryID != repositoryID || payload.WorkItemKey != run.WorkItemKey || payload.RunID != run.ID || payload.Branch != run.Branch || implementationBase != run.BaseSHA || payload.CommitSHA != run.CommitSHA || payload.RemoteFingerprint != RemoteFingerprint(remoteURL) || !allowsOperation(payload.AllowedOperations, operation) {
		return fmt.Errorf("signed remote publication authorization does not match this run")
	}
	if operation == "push" && run.State != StateLocallyCommitted {
		return fmt.Errorf("implementation run is not ready for authorized push")
	}
	if operation == "create-pr" && (run.State != StatePushed || payload.PRTargetBranch == "") {
		return fmt.Errorf("implementation run is not ready for authorized pull request creation")
	}
	return nil
}

func ConsumeSignedAuthorization(runtimeRoot string, authorization SignedPublicationAuthorization, operation string) error {
	if !allowsOperation(authorization.Payload.AllowedOperations, operation) {
		return fmt.Errorf("authorization does not allow %s", operation)
	}
	path := filepath.Join(runtimeRoot, "publication-consumption", strings.TrimPrefix(authorization.Digest, "sha256:")+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	lockPath := path + ".lock"
	lock, err := os.OpenFile(filepath.Clean(lockPath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("publication authorization consumption is already in progress or requires operator recovery")
		}
		return err
	}
	defer os.Remove(lockPath)
	if err := lock.Close(); err != nil {
		return err
	}
	var record AuthorizationConsumption
	if data, err := os.ReadFile(filepath.Clean(path)); err == nil {
		if err := json.Unmarshal(data, &record); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		record.AuthorizationDigest = authorization.Digest
	}
	if record.AuthorizationDigest != authorization.Digest || allowsOperation(record.UsedOperations, operation) {
		return fmt.Errorf("publication authorization operation has already been used")
	}
	record.UsedOperations = append(record.UsedOperations, operation)
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func validatePublicationPayload(payload PublicationAuthorizationPayload) error {
	if (payload.FormatVersion != 1 && payload.FormatVersion != 2) || payload.WorkItemKey == "" || payload.RunID == "" || payload.RepositoryID == "" || payload.Remote == "" || payload.RemoteFingerprint == "" || !strings.HasPrefix(payload.Branch, "codex/") || len(payload.CommitSHA) < 7 || len(payload.AllowedOperations) == 0 {
		return fmt.Errorf("signed publication authorization payload is invalid")
	}
	if payload.FormatVersion == 1 && len(payload.ExpectedBaseSHA) < 7 {
		return fmt.Errorf("signed publication authorization payload is invalid")
	}
	if payload.FormatVersion == 2 && (len(payload.ImplementationBaseSHA) < 7 || len(payload.ExpectedTargetSHA) < 7 || payload.ExpectedBaseSHA != "") {
		return fmt.Errorf("signed publication authorization version-2 lineage is invalid")
	}
	seen := map[string]bool{}
	for _, operation := range payload.AllowedOperations {
		if (operation != "push" && operation != "create-pr") || seen[operation] {
			return fmt.Errorf("signed publication authorization operations are invalid")
		}
		seen[operation] = true
	}
	if payload.PRTargetBranch == "" {
		return fmt.Errorf("signed publication authorization pull request target is invalid")
	}
	return nil
}

func allowsOperation(operations []string, wanted string) bool {
	for _, operation := range operations {
		if operation == wanted {
			return true
		}
	}
	return false
}

// ValidatePublicationWorktree confirms that the local branch and checked-out
// commit are exactly the records bound by the signed authorization. It also
// requires the commit to descend from the run's approved base commit.
func ValidatePublicationWorktree(worktree string, run Run) error {
	head, err := git(worktree, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("read worktree HEAD: %w: %s", err, head)
	}
	if strings.TrimSpace(string(head)) != run.CommitSHA {
		return fmt.Errorf("worktree HEAD does not match authorized commit")
	}
	branch, err := git(worktree, "rev-parse", "refs/heads/"+run.Branch)
	if err != nil {
		return fmt.Errorf("read worktree branch: %w: %s", err, branch)
	}
	if strings.TrimSpace(string(branch)) != run.CommitSHA {
		return fmt.Errorf("worktree branch does not match authorized commit")
	}
	return nil
}

// ValidateAuthorizedLineage requires the authorized commit to descend from
// the implementation base and, for v2 authorizations, the independently bound
// remote target SHA.
func ValidateAuthorizedLineage(worktree string, authorization SignedPublicationAuthorization) error {
	payload := authorization.Payload
	bases := []string{payload.ExpectedBaseSHA}
	if payload.FormatVersion == 2 {
		bases = []string{payload.ImplementationBaseSHA, payload.ExpectedTargetSHA}
	}
	for _, base := range bases {
		output, err := git(worktree, "merge-base", base, payload.CommitSHA)
		if err != nil {
			return fmt.Errorf("verify authorized lineage: %w: %s", err, output)
		}
		if strings.TrimSpace(string(output)) != base {
			return fmt.Errorf("authorized commit does not descend from bound SHA")
		}
	}
	return nil
}

// ValidateAuthorizedRemoteBase rejects publication when the authorized target
// ref has moved since its expected base SHA was signed.
func ValidateAuthorizedRemoteBase(worktree string, authorization SignedPublicationAuthorization) error {
	payload := authorization.Payload
	expectedTarget := payload.ExpectedBaseSHA
	if payload.FormatVersion == 2 {
		expectedTarget = payload.ExpectedTargetSHA
	}
	output, err := git(worktree, "ls-remote", "--exit-code", payload.Remote, "refs/heads/"+payload.PRTargetBranch)
	if err != nil {
		return fmt.Errorf("read authorized target ref: %w: %s", err, output)
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || fields[0] != expectedTarget || fields[1] != "refs/heads/"+payload.PRTargetBranch {
		return fmt.Errorf("authorized target ref does not match expected base SHA")
	}
	return nil
}

// RemoteBranchSHA reads the exact target SHA that an owner intends to bind.
// It uses Git's read-only remote query and performs no publication action.
func RemoteBranchSHA(worktree, remote, branch string) (string, error) {
	output, err := git(worktree, "ls-remote", "--exit-code", remote, "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("read remote target ref: %w: %s", err, output)
	}
	fields := strings.Fields(string(output))
	if len(fields) < 2 || len(fields[0]) < 7 || fields[1] != "refs/heads/"+branch {
		return "", fmt.Errorf("remote target ref is invalid")
	}
	return fields[0], nil
}

// PrepareSignedPush records the crash-safe state immediately before the
// already verified and consumed signed push operation.
func PrepareSignedPush(run *Run) error {
	if run.State != StateLocallyCommitted {
		return fmt.Errorf("implementation run is not ready for authorized push")
	}
	return run.Transition(StateReadyForRemoteApproval)
}

// PushSigned performs only the refspec bound by the signed authorization.
func PushSigned(run *Run, authorization SignedPublicationAuthorization, worktree string) error {
	if run.State != StateReadyForRemoteApproval {
		return fmt.Errorf("implementation run is not ready to push")
	}
	if output, err := git(worktree, "push", authorization.Payload.Remote, run.CommitSHA+":refs/heads/"+run.Branch); err != nil {
		return fmt.Errorf("push authorized branch: %w: %s", err, output)
	}
	return run.Transition(StatePushed)
}

// CreateSignedPullRequest invokes GitHub only with the authorized source and
// target refs, after the create-pr operation has been consumed.
func CreateSignedPullRequest(run *Run, authorization SignedPublicationAuthorization, worktree, title, body string) error {
	if run.State != StatePushed || title == "" || authorization.Payload.PRTargetBranch == "" {
		return fmt.Errorf("implementation run is not ready to create a pull request")
	}
	repository, err := GitHubRepository(authorization.Payload.RepositoryID)
	if err != nil {
		return err
	}
	command := exec.Command("gh", "pr", "create", "--repo", repository, "--head", run.Branch, "--base", authorization.Payload.PRTargetBranch, "--title", title, "--body", body)
	command.Dir = filepath.Clean(worktree)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create authorized pull request: %w: %s", err, output)
	}
	run.PullRequestURL = strings.TrimSpace(string(output))
	return run.Transition(StatePRCreated)
}

// GitHubRepository turns the configured repository identity into the exact
// owner/name argument required by GitHub CLI publication.
func GitHubRepository(repositoryID string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(repositoryID, "github.com/"), "/")
	if !strings.HasPrefix(repositoryID, "github.com/") || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("signed publication repository is not a GitHub repository identity")
	}
	return strings.Join(parts, "/"), nil
}

func Commit(run *Run, worktree, branch, message string) error {
	if run.State != StateReadyToCommit || !strings.HasPrefix(branch, "codex/") || message == "" {
		return fmt.Errorf("local commit is not authorized for this run")
	}
	if output, err := git(worktree, "switch", "-c", branch); err != nil {
		return fmt.Errorf("create run branch: %w: %s", err, output)
	}
	if _, err := git(worktree, "diff", "--cached", "--quiet"); err == nil {
		return fmt.Errorf("local commit requires staged changes")
	}
	if output, err := git(worktree, "commit", "-m", message); err != nil {
		return fmt.Errorf("create local commit: %w: %s", err, output)
	}
	sha, err := git(worktree, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	run.Branch, run.CommitSHA = branch, strings.TrimSpace(string(sha))
	return run.Transition(StateLocallyCommitted)
}

func RemoteURL(worktree, remote string) (string, error) {
	output, err := git(worktree, "remote", "get-url", remote)
	if err != nil {
		return "", fmt.Errorf("read remote URL: %w: %s", err, output)
	}
	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", fmt.Errorf("remote URL is empty")
	}
	return url, nil
}

func RemoteFingerprint(remoteURL string) string {
	sum := sha256.Sum256([]byte(remoteURL))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func git(dir string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = filepath.Clean(dir)
	return command.CombinedOutput()
}
