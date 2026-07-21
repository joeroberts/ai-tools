package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLedgerTracksOpenAgents(t *testing.T) {
	root := t.TempDir()
	event := Event{WorkItem: "CG-1", AgentID: "agent-1", Role: "reviewer", State: "started"}
	if err := Record(root, event); err != nil {
		t.Fatal(err)
	}
	open, err := OpenAgents(root, "CG-1")
	if err != nil || len(open) != 1 {
		t.Fatalf("OpenAgents() = %#v, %v", open, err)
	}
	if err := Record(root, Event{WorkItem: "CG-1", AgentID: "agent-1", Role: "reviewer", State: "completed", ResultRef: "review.md"}); err != nil {
		t.Fatal(err)
	}
	if err := Record(root, Event{WorkItem: "CG-1", AgentID: "agent-1", Role: "reviewer", State: "closed"}); err != nil {
		t.Fatal(err)
	}
	open, err = OpenAgents(root, "CG-1")
	if err != nil || len(open) != 0 {
		t.Fatalf("OpenAgents() after close = %#v, %v", open, err)
	}
}

func TestLifecycleEventsArePrivacySafeAndOwnerOnly(t *testing.T) {
	root := t.TempDir()
	event := LifecycleEvent{RunID: "run-1", WorkItem: "REK-66", Phase: "implementation", State: "running"}
	if err := RecordLifecycle(root, event); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(root, "lifecycle-events.jsonl"))
	if err != nil || strings.Contains(string(stored), "result_ref") || strings.Contains(string(stored), "/private/") {
		t.Fatalf("stored lifecycle event is not privacy-safe: %q, %v", stored, err)
	}
	info, err := os.Stat(filepath.Join(root, "lifecycle-events.jsonl"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("ledger mode = %v, %v", info.Mode().Perm(), err)
	}
	events, err := LoadLifecycle(root, "run-1")
	if err != nil || len(events) != 1 || events[0].State != "running" {
		t.Fatalf("LoadLifecycle() = %#v, %v", events, err)
	}
}

func TestLifecycleEventsAcceptPlanningAndOperationPhases(t *testing.T) {
	root := t.TempDir()
	for _, phase := range []string{"planning", "operation"} {
		if err := RecordLifecycle(root, LifecycleEvent{RunID: phase + "-1", WorkItem: "REK-67", Phase: phase, State: "dispatched"}); err != nil {
			t.Fatalf("RecordLifecycle(%s) = %v", phase, err)
		}
	}
}

func TestCacheRedactsSecrets(t *testing.T) {
	root := t.TempDir()
	key := CacheKey("input")
	if err := StoreCache(root, key, "token=abc123"); err != nil {
		t.Fatal(err)
	}
	entry, ok, err := LoadCache(root, key)
	if err != nil || !ok || entry.Summary == "token=abc123" {
		t.Fatalf("LoadCache() = %#v, %v, %v", entry, ok, err)
	}
}
