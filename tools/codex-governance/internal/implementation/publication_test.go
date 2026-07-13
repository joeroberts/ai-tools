package implementation

import (
	"path/filepath"
	"testing"
	"time"
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

func publicationAuthorization(run Run, operation, remoteURL string) RemoteAuthorization {
	return RemoteAuthorization{
		FormatVersion:     FormatVersion,
		WorkItemKey:       run.WorkItemKey,
		Remote:            "origin",
		RemoteFingerprint: RemoteFingerprint(remoteURL),
		Branch:            run.Branch,
		CommitSHA:         run.CommitSHA,
		Operation:         operation,
		Approver:          "operator@example.test",
		ExpiresAt:         time.Now().Add(time.Hour),
	}
}

func TestPreparePushRequiresExactRemoteAndTransitions(t *testing.T) {
	run := publicationRun()
	authorization := publicationAuthorization(run, "push", "https://example.test/acme/repo.git")
	if err := PreparePush(&run, authorization, "https://example.test/acme/other.git"); err == nil {
		t.Fatal("PreparePush accepted a different remote URL")
	}
	if err := PreparePush(&run, authorization, "https://example.test/acme/repo.git"); err != nil {
		t.Fatal(err)
	}
	if run.State != StateReadyForRemoteApproval {
		t.Fatalf("state = %s, want %s", run.State, StateReadyForRemoteApproval)
	}
}

func TestAuthorizationIsPersistedAsOneUse(t *testing.T) {
	run := publicationRun()
	path := filepath.Join(t.TempDir(), "authorization.json")
	authorization := publicationAuthorization(run, "push", "https://example.test/acme/repo.git")
	if err := SaveAuthorization(path, authorization); err != nil {
		t.Fatal(err)
	}
	if err := ConsumeAuthorization(path, authorization); err != nil {
		t.Fatal(err)
	}
	consumed, err := LoadAuthorization(path)
	if err != nil {
		t.Fatal(err)
	}
	if !consumed.Used {
		t.Fatal("authorization was not recorded as used")
	}
	if err := ValidatePublication(run, consumed, "push", "https://example.test/acme/repo.git"); err == nil {
		t.Fatal("used authorization remained valid")
	}
	if err := ConsumeAuthorization(path, authorization); err == nil {
		t.Fatal("authorization was consumed more than once")
	}
}

func TestCreatePullRequestRequiresSeparateAuthorization(t *testing.T) {
	run := publicationRun()
	run.State = StatePushed
	pushAuthorization := publicationAuthorization(run, "push", "https://example.test/acme/repo.git")
	if err := ValidatePublication(run, pushAuthorization, "create-pr", "https://example.test/acme/repo.git"); err == nil {
		t.Fatal("push authorization was accepted for pull request creation")
	}
	prAuthorization := publicationAuthorization(run, "create-pr", "https://example.test/acme/repo.git")
	if err := ValidatePublication(run, prAuthorization, "create-pr", "https://example.test/acme/repo.git"); err != nil {
		t.Fatal(err)
	}
}
