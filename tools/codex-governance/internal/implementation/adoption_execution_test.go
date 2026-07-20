package implementation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/signature"
	"codex-governance/internal/workitem"
)

func TestAdoptPreviewApproveAndReplaySafety(t *testing.T) {
	root, _, predecessor := testRepository(t)
	export, exportPublic := signedFixtureExport(t, "export-issuer")
	ownerDir := filepath.Join(t.TempDir(), "owner")
	if err := os.Mkdir(ownerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	signerPath := filepath.Join(ownerDir, "technical-owner.json")
	owner, err := signature.CreateLocalTechnicalOwnerSigner(signerPath)
	if err != nil {
		t.Fatal(err)
	}
	writeAdoptionConfig(t, root, exportPublic, owner)
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# Guidance\n")
	writeFile(t, filepath.Join(root, "docs", "decisions", "ADR-0001.md"), "# ADR\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "governance")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	writeFile(t, filepath.Join(root, "internal", "value.go"), "package internal\nconst Value = 2\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "predecessor")
	predecessor = strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	runGit(t, root, "remote", "add", "origin", root)
	runGit(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master")
	item, err := workitem.Load(filepath.Join("..", "..", "testdata", "work-items", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	item.GitRange.BaseSHA, item.GitRange.HeadSHA = base, predecessor
	itemPath := filepath.Join(root, "work-item.json")
	writeFile(t, filepath.Join(root, ".git", "info", "exclude"), "work-item.json\n")
	writeJSON(t, itemPath, item)
	runtime := filepath.Join(t.TempDir(), "runtime")
	preflight, err := Preflight(PreflightRequest{WorkItemPath: itemPath, OfflineExportPath: export, RepoRoot: root, RuntimeRoot: runtime, Adapter: "fake", BundlePath: filepath.Join(runtime, "bundle.json"), RunPath: filepath.Join(runtime, "run.json")})
	if err != nil {
		t.Fatal(err)
	}
	run := preflight.Run
	run.CommitSHA = predecessor
	if err := SaveRun(filepath.Join(runtime, "run.json"), run); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "checkout", "-b", "remediation/CG-2")
	writeFile(t, filepath.Join(root, "internal", "value.go"), "package internal\nconst Value = 3\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "remediation")
	candidate := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	diff, err := DiffBytes(root, base+".."+candidate)
	if err != nil {
		t.Fatal(err)
	}
	review := writePassingReviewEvidence(t, diff)
	checks := filepath.Join(t.TempDir(), "checks.json")
	writeJSON(t, checks, AdoptionCheckEvidence{FormatVersion: 1, Range: predecessor + ".." + candidate, Checks: []CheckOutcome{{Name: "go test ./...", Outcome: "passed", OutputDigest: digest([]byte("ok"))}}})
	audit := filepath.Join(t.TempDir(), "audit.json")
	if err := ExportAudit(audit, run); err != nil {
		t.Fatal(err)
	}
	registryParent, err := os.MkdirTemp("/private/tmp", "adoption-registry-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(registryParent) })
	registry := filepath.Join(registryParent, "registry", "records")
	req := AdoptionRequest{RunPath: filepath.Join(runtime, "run.json"), BundlePath: preflight.BundlePath, CandidateWorktree: root, ReviewEvidencePath: review, CheckEvidencePath: checks, AuditEvidencePath: audit, RegistryPath: registry, SignerPath: signerPath, Reason: "approved remediation", IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour)}
	preview, err := Adopt(req)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatalf("preview created registry: %v", err)
	}
	req.Reason, req.IssuedAt, req.ExpiresAt = "changed reason", req.IssuedAt.Add(time.Minute), req.ExpiresAt.Add(time.Minute)
	second, err := Adopt(req)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Record.ID != second.Record.ID {
		t.Fatal("mutable metadata changed stable adoption identity")
	}
	req.Reason, req.IssuedAt, req.ExpiresAt, req.Approve = "approved remediation", time.Now().UTC(), time.Now().UTC().Add(time.Hour), true
	approved, err := Adopt(req)
	if err != nil {
		t.Fatalf("approved adoption: %v", err)
	}
	if approved.Envelope == nil {
		t.Fatal("approved adoption did not return envelope")
	}
	if _, err := os.Stat(approved.RegistryPath); err != nil {
		t.Fatal(err)
	}
	if _, err := Adopt(req); err == nil || !strings.Contains(err.Error(), "replayed") {
		t.Fatalf("retry error = %v", err)
	}
}

func TestAdoptionRegistryAndAuditEvidenceFailClosed(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")
	if err := os.Symlink(root, registry); err != nil {
		t.Fatal(err)
	}
	if err := ensureRegistry(registry, false); err == nil {
		t.Fatal("accepted symlink registry")
	}
	if _, err := loadAuditEvidence(filepath.Join(root, "missing"), Run{ID: "run-aaaaaaaaaaaaaaaa"}); err == nil {
		t.Fatal("accepted absent audit evidence")
	}
	record := validAdoptionRecord()
	changed := record
	changed.Reason = "another reason"
	changed.IssuedAt = changed.IssuedAt.Add(time.Hour)
	if adoptionID(record) != adoptionID(changed) {
		t.Fatal("stable identity included mutable metadata")
	}
	changed.RepositoryID, changed.WorkItemKey = "github.com/other/repository", "ALT-9"
	if adoptionID(record) == adoptionID(changed) {
		t.Fatal("cross-repository work item reused adoption identity")
	}
}

func writeAdoptionConfig(t *testing.T, root, exportPublic string, owner signature.TrustedKey) {
	t.Helper()
	writeFile(t, filepath.Join(root, "governance.yml"), fmt.Sprintf("format_version: 1\nprofile: generic\njira:\n  issue_key_pattern: '^[A-Z]+-[0-9]+$'\n  required_sections: [Scope]\nreview_budget:\n  max_changed_files: 5\n  max_changed_lines: 50\n  max_components: 1\nci:\n  provider: github-actions\n  mode: warn\nupstream: {}\nimplementation:\n  allowed_adapters: [fake]\n  local_code_edit_enabled: false\nsigning:\n  format_version: 1\n  repository_id: github.com/acme/governance\n  offline_export_max_age: 8760h\n  trusted_keys:\n    - key_id: fixture-issuer\n      role: export-issuer\n      algorithm: ed25519\n      public_key: %s\n    - key_id: %s\n      role: technical-owner\n      algorithm: ed25519\n      public_key: %s\n", exportPublic, owner.KeyID, owner.PublicKey))
}

func writePassingReviewEvidence(t *testing.T, diff []byte) string {
	t.Helper()
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "reviewer.json"), filepath.Join(dir, "verifier.json")}
	for _, path := range paths {
		if err := SaveAssessment(path, Assessment{}); err != nil {
			t.Fatal(err)
		}
	}
	data := func(path, executor string) AssessmentRecord {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return AssessmentRecord{ExecutorID: executor, AssessmentPath: path, AssessmentDigest: digestBytes(b)}
	}
	evidence := ReviewEvidence{FormatVersion: 1, DiffDigest: digestBytes(diff), Reviewer: data(paths[0], "reviewer"), Verifier: data(paths[1], "verifier")}
	path := filepath.Join(dir, "evidence.json")
	writeJSON(t, path, evidence)
	return path
}
