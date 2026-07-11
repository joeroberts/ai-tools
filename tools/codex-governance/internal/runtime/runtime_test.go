package runtime

import "testing"

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
