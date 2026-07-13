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
)

// RemoteAuthorization is a one-use approval for one specific remote action.
// It deliberately binds the action to the run's work item, branch, commit,
// and remote URL fingerprint rather than accepting a broad repository grant.
type RemoteAuthorization struct {
	FormatVersion     int       `json:"format_version"`
	WorkItemKey       string    `json:"work_item_key"`
	Remote            string    `json:"remote"`
	RemoteFingerprint string    `json:"remote_fingerprint"`
	Branch            string    `json:"branch"`
	CommitSHA         string    `json:"commit_sha"`
	Operation         string    `json:"operation"`
	Approver          string    `json:"approver"`
	ExpiresAt         time.Time `json:"expires_at"`
	Used              bool      `json:"used"`
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

func SaveAuthorization(path string, authorization RemoteAuthorization) error {
	if err := validateAuthorization(authorization); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return fmt.Errorf("refusing to overwrite authorization")
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeAuthorization(path, authorization)
}

func LoadAuthorization(path string) (RemoteAuthorization, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return RemoteAuthorization{}, err
	}
	var authorization RemoteAuthorization
	if err := json.Unmarshal(data, &authorization); err != nil {
		return RemoteAuthorization{}, err
	}
	return authorization, validateAuthorization(authorization)
}

// ConsumeAuthorization records use before the remote side effect. A failed
// remote command therefore requires a new human authorization rather than an
// implicit retry that could duplicate a remote action after a crash.
func ConsumeAuthorization(path string, authorization RemoteAuthorization) error {
	current, err := LoadAuthorization(path)
	if err != nil {
		return err
	}
	if current.Used {
		return fmt.Errorf("remote publication authorization has already been used")
	}
	if !sameAuthorization(current, authorization) {
		return fmt.Errorf("remote publication authorization changed before use")
	}
	current.Used = true
	return writeAuthorization(path, current)
}

func ValidatePublication(run Run, authorization RemoteAuthorization, operation, remoteURL string) error {
	if authorization.Used || authorization.WorkItemKey != run.WorkItemKey || authorization.Branch != run.Branch || authorization.CommitSHA != run.CommitSHA || authorization.Operation != operation || authorization.RemoteFingerprint != RemoteFingerprint(remoteURL) {
		return fmt.Errorf("remote publication authorization does not match this run")
	}
	if (operation == "push" && run.State != StateLocallyCommitted) || (operation == "create-pr" && run.State != StatePushed) {
		return fmt.Errorf("implementation run is not ready for authorized %s", operation)
	}
	return nil
}

// PreparePush moves the persisted lifecycle to the crash-safe pre-publication
// state after validating the exact authorization and remote destination.
func PreparePush(run *Run, authorization RemoteAuthorization, remoteURL string) error {
	if err := ValidatePublication(*run, authorization, "push", remoteURL); err != nil {
		return err
	}
	return run.Transition(StateReadyForRemoteApproval)
}

// Push performs the exact refspec already authorized and consumed by the CLI.
func Push(run *Run, authorization RemoteAuthorization, worktree string) error {
	if run.State != StateReadyForRemoteApproval || authorization.Operation != "push" {
		return fmt.Errorf("implementation run is not ready to push")
	}
	if output, err := git(worktree, "push", authorization.Remote, run.CommitSHA+":refs/heads/"+run.Branch); err != nil {
		return fmt.Errorf("push authorized branch: %w: %s", err, output)
	}
	return run.Transition(StatePushed)
}

// CreatePullRequest invokes the GitHub CLI only after a distinct create-pr
// authorization was consumed. Its output is the created pull request URL.
func CreatePullRequest(run *Run, authorization RemoteAuthorization, worktree, title, body string) error {
	if run.State != StatePushed || authorization.Operation != "create-pr" || title == "" {
		return fmt.Errorf("implementation run is not ready to create a pull request")
	}
	command := exec.Command("gh", "pr", "create", "--head", run.Branch, "--title", title, "--body", body)
	command.Dir = filepath.Clean(worktree)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create authorized pull request: %w: %s", err, output)
	}
	run.PullRequestURL = strings.TrimSpace(string(output))
	return run.Transition(StatePRCreated)
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

func validateAuthorization(authorization RemoteAuthorization) error {
	if authorization.FormatVersion != FormatVersion || authorization.WorkItemKey == "" || authorization.Remote == "" || authorization.RemoteFingerprint == "" || !strings.HasPrefix(authorization.Branch, "codex/") || len(authorization.CommitSHA) < 7 || authorization.Approver == "" || authorization.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("remote authorization is invalid or expired")
	}
	if authorization.Operation != "push" && authorization.Operation != "create-pr" {
		return fmt.Errorf("remote authorization operation is invalid")
	}
	return nil
}

func writeAuthorization(path string, authorization RemoteAuthorization) error {
	data, err := json.MarshalIndent(authorization, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), append(data, '\n'), 0o600)
}

func sameAuthorization(a, b RemoteAuthorization) bool {
	return a.FormatVersion == b.FormatVersion &&
		a.WorkItemKey == b.WorkItemKey &&
		a.Remote == b.Remote &&
		a.RemoteFingerprint == b.RemoteFingerprint &&
		a.Branch == b.Branch &&
		a.CommitSHA == b.CommitSHA &&
		a.Operation == b.Operation &&
		a.Approver == b.Approver &&
		a.ExpiresAt.Equal(b.ExpiresAt) &&
		a.Used == b.Used
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
