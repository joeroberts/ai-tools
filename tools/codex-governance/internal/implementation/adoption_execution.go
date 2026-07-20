package implementation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/jira"
	"codex-governance/internal/signature"
)

type AdoptionCheckEvidence struct {
	FormatVersion int            `json:"format_version"`
	Range         string         `json:"range"`
	Checks        []CheckOutcome `json:"checks"`
}

type AdoptionRequest struct {
	RunPath, BundlePath, CandidateWorktree                   string
	ReviewEvidencePath, CheckEvidencePath, AuditEvidencePath string
	RegistryPath, SignerPath, Reason                         string
	IssuedAt, ExpiresAt                                      time.Time
	Approve                                                  bool
}
type AdoptionResult struct {
	Record       AdoptionRecord      `json:"record"`
	RecordDigest string              `json:"record_digest"`
	RegistryPath string              `json:"registry_path"`
	Envelope     *signature.Envelope `json:"envelope,omitempty"`
}

func Adopt(r AdoptionRequest) (AdoptionResult, error) {
	if r.Approve && r.SignerPath == "" {
		return AdoptionResult{}, fmt.Errorf("approved adoption requires a technical-owner signer")
	}
	s, err := validateAdoption(r)
	if err != nil {
		return AdoptionResult{}, err
	}
	result := AdoptionResult{Record: s.record, RecordDigest: AdoptionRecordDigest(s.payload), RegistryPath: s.path}
	if !r.Approve {
		return result, nil
	}
	if err := ensureRegistry(s.root, true); err != nil {
		return AdoptionResult{}, err
	}
	if err := rejectRegistryConflict(s.root, s.record); err != nil {
		return AdoptionResult{}, err
	}
	trusted, private, err := signature.LoadLocalTechnicalOwnerSigner(r.SignerPath)
	if err != nil {
		return AdoptionResult{}, fmt.Errorf("load technical-owner signer: %w", err)
	}
	if trusted.KeyID != s.record.SignerIdentity || !configuredTechnicalOwner(s.cfg, trusted) {
		return AdoptionResult{}, fmt.Errorf("technical-owner signer does not exactly match configured trust")
	}
	envelope, err := signature.Sign(s.payload, trusted.KeyID, trusted.Role, private, s.record.IssuedAt, &s.record.ExpiresAt)
	if err != nil {
		return AdoptionResult{}, fmt.Errorf("sign adoption record: %w", err)
	}
	if _, err := ValidateSignedAdoptionRecord(envelope, s.registry, s.now, s.cfg.Signing.RepositoryID, s.bundle.WorkItem.Source.SubtaskKey, s.defaultBranch); err != nil {
		return AdoptionResult{}, fmt.Errorf("read back technical-owner signature: %w", err)
	}
	if err := persistAdoption(s.path, envelope); err != nil {
		return AdoptionResult{}, err
	}
	result.Envelope = &envelope
	return result, nil
}

type adoptionState struct {
	cfg                       config.Config
	bundle                    TaskBundle
	record                    AdoptionRecord
	payload                   []byte
	root, path, defaultBranch string
	registry                  signature.Registry
	now                       time.Time
}

func validateAdoption(r AdoptionRequest) (adoptionState, error) {
	if r.RunPath == "" || r.BundlePath == "" || r.CandidateWorktree == "" || r.ReviewEvidencePath == "" || r.CheckEvidencePath == "" || r.AuditEvidencePath == "" || r.RegistryPath == "" || strings.TrimSpace(r.Reason) == "" {
		return adoptionState{}, fmt.Errorf("adoption requires run, bundle, candidate, review, checks, audit evidence, registry, and reason")
	}
	now := time.Now().UTC()
	run, err := LoadRun(r.RunPath)
	if err != nil {
		return adoptionState{}, fmt.Errorf("load predecessor run: %w", err)
	}
	if run.CommitSHA == "" || !commitPattern.MatchString(run.CommitSHA) {
		return adoptionState{}, fmt.Errorf("predecessor run lacks an immutable committed SHA")
	}
	bundle, err := LoadTaskBundle(r.BundlePath)
	if err != nil {
		return adoptionState{}, err
	}
	if bundle.WorkItem.Source.SubtaskKey != run.WorkItemKey {
		return adoptionState{}, fmt.Errorf("predecessor run and task bundle work items differ")
	}
	bundleData, err := os.ReadFile(filepath.Clean(r.BundlePath))
	if err != nil {
		return adoptionState{}, err
	}
	if digest(bundleData) != run.TaskBundleDigest {
		return adoptionState{}, fmt.Errorf("task bundle digest does not match predecessor run")
	}
	cfg, err := config.Load(filepath.Join(r.CandidateWorktree, "governance.yml"))
	if err != nil {
		return adoptionState{}, fmt.Errorf("load current governance config: %w", err)
	}
	if cfg.Signing.RepositoryID == "" {
		return adoptionState{}, fmt.Errorf("adoption requires signing.repository_id")
	}
	registry, err := cfg.TrustedKeyRegistry()
	if err != nil {
		return adoptionState{}, err
	}
	maxAge, err := cfg.OfflineExportMaxAge()
	if err != nil {
		return adoptionState{}, err
	}
	// Legacy v1 bundles have no configuration digest. Bind current configuration
	// independently in the successor record rather than changing their meaning.
	if err := VerifyDispatchReadiness(run, bundle, r.BundlePath, cfg, now); err != nil {
		return adoptionState{}, fmt.Errorf("revalidate fresh source and bundle: %w", err)
	}
	if _, err := jira.VerifySignedOfflineExport(bundle.SourceEnvelope, registry, maxAge, now); err != nil {
		return adoptionState{}, fmt.Errorf("verify fresh source: %w", err)
	}
	guidance, err := readGuidance(r.CandidateWorktree)
	if err != nil {
		return adoptionState{}, err
	}
	if guidance != bundle.Guidance {
		return adoptionState{}, fmt.Errorf("repository guidance changed since task bundle")
	}
	if err := ensureCleanWorktree(r.CandidateWorktree); err != nil {
		return adoptionState{}, err
	}
	branch, err := currentBranch(r.CandidateWorktree)
	if err != nil {
		return adoptionState{}, err
	}
	def, err := configuredDefaultBranch(r.CandidateWorktree)
	if err != nil {
		return adoptionState{}, err
	}
	if branch == def || branch == "HEAD" || strings.HasPrefix(branch, "refs/") {
		return adoptionState{}, fmt.Errorf("candidate branch is mutable or configured default branch")
	}
	candidate, err := resolveCommit(r.CandidateWorktree, "HEAD")
	if err != nil {
		return adoptionState{}, err
	}
	if err := validateLineage(r.CandidateWorktree, bundle.WorkItem.GitRange.BaseSHA, run.CommitSHA, candidate); err != nil {
		return adoptionState{}, err
	}
	diff, err := DiffBytes(r.CandidateWorktree, bundle.WorkItem.GitRange.BaseSHA+".."+candidate)
	if err != nil {
		return adoptionState{}, err
	}
	if err := validateScope(r.CandidateWorktree, bundle, bundle.WorkItem.GitRange.BaseSHA, candidate); err != nil {
		return adoptionState{}, err
	}
	checks, err := loadChecks(r.CheckEvidencePath, run.CommitSHA+".."+candidate, bundle.Commands)
	if err != nil {
		return adoptionState{}, err
	}
	if err := ValidateReviewEvidence(r.ReviewEvidencePath, diff); err != nil {
		return adoptionState{}, fmt.Errorf("validate complete-range review evidence: %w", err)
	}
	review, err := loadReviewBinding(r.ReviewEvidencePath)
	if err != nil {
		return adoptionState{}, err
	}
	runData, err := os.ReadFile(filepath.Clean(r.RunPath))
	if err != nil {
		return adoptionState{}, err
	}
	auditEventID, err := loadAuditEvidence(r.AuditEvidencePath, run)
	if err != nil {
		return adoptionState{}, err
	}
	issued, expires := r.IssuedAt.UTC(), r.ExpiresAt.UTC()
	if issued.IsZero() || expires.IsZero() || !expires.After(issued) || issued.After(now.Add(5*time.Minute)) || !expires.After(now) {
		return adoptionState{}, fmt.Errorf("adoption issuance or expiry is invalid")
	}
	configData, err := os.ReadFile(filepath.Join(r.CandidateWorktree, "governance.yml"))
	if err != nil {
		return adoptionState{}, err
	}
	record := AdoptionRecord{FormatVersion: 1, RepositoryID: cfg.Signing.RepositoryID, RepositoryDigest: digest([]byte(cfg.Signing.RepositoryID)), WorkItemKey: run.WorkItemKey, PredecessorRunID: run.ID, PredecessorRunDigest: digest(runData), OriginalBaseSHA: bundle.WorkItem.GitRange.BaseSHA, PredecessorCommitSHA: run.CommitSHA, CandidateBranch: branch, CandidateCommitSHA: candidate, AdoptedRange: run.CommitSHA + ".." + candidate, CompleteDiffDigest: digestBytes(diff), WorkItemDigest: workItemDigest(bundle.WorkItem), SourceEnvelopeDigest: bundle.SourceEvidence.EnvelopeDigest, TaskBundleDigest: run.TaskBundleDigest, ConfigurationDigest: digest(configData), GuidanceDigest: digest([]byte(guidance)), ReviewEvidence: review, DeterministicChecks: checks, Reason: strings.TrimSpace(r.Reason), AuthorizedRole: "technical-owner", SignerIdentity: configuredTechnicalOwnerID(cfg), IssuedAt: issued, ExpiresAt: expires, PrecedingAuditEventID: auditEventID}
	if record.SignerIdentity == "" {
		return adoptionState{}, fmt.Errorf("configured technical-owner trust is unavailable")
	}
	record.ID = adoptionID(record)
	payload, err := MarshalAdoptionRecord(record)
	if err != nil {
		return adoptionState{}, err
	}
	root, path, err := adoptionRegistryPath(r.RegistryPath, r.CandidateWorktree, record.ID)
	if err != nil {
		return adoptionState{}, err
	}
	if err := ensureRegistry(root, false); err != nil {
		return adoptionState{}, err
	}
	if err := rejectRegistryConflict(root, record); err != nil {
		return adoptionState{}, err
	}
	return adoptionState{cfg: cfg, bundle: bundle, record: record, payload: payload, root: root, path: path, defaultBranch: def, registry: registry, now: now}, nil
}
func ensureCleanWorktree(dir string) error {
	out, err := git(dir, "status", "--porcelain=v1")
	if err != nil {
		return fmt.Errorf("read candidate worktree state: %w", err)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		return fmt.Errorf("candidate worktree is dirty")
	}
	return nil
}
func currentBranch(dir string) (string, error) {
	out, err := git(dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("candidate must be checked out on a local branch")
	}
	return strings.TrimSpace(string(out)), nil
}
func configuredDefaultBranch(dir string) (string, error) {
	out, err := git(dir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("derive configured default branch: %w", err)
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"), nil
}
func resolveCommit(dir, ref string) (string, error) {
	out, err := git(dir, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("derive immutable candidate commit: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
func validateLineage(dir, base, pre, candidate string) error {
	for _, a := range []string{base, pre} {
		if out, err := git(dir, "merge-base", "--is-ancestor", a, candidate); err != nil {
			return fmt.Errorf("candidate does not descend from required SHA %s: %s", a, out)
		}
	}
	if out, err := git(dir, "rev-list", "--merges", base+".."+candidate); err != nil || len(bytes.TrimSpace(out)) != 0 {
		return fmt.Errorf("candidate history contains a merge or cannot be inspected")
	}
	if out, err := git(dir, "rev-list", "--reverse", pre+".."+candidate); err != nil || len(bytes.TrimSpace(out)) == 0 {
		return fmt.Errorf("candidate does not contain a linear remediation descendant")
	}
	return nil
}
func validateScope(dir string, b TaskBundle, base, candidate string) error {
	out, err := git(dir, "diff", "--numstat", base+".."+candidate)
	if err != nil {
		return fmt.Errorf("read complete diff budget: %w", err)
	}
	files := map[string]bool{}
	components := map[string]bool{}
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return fmt.Errorf("complete diff has malformed numstat")
		}
		if !allowedPath(f[2], b.AllowedPaths) || classifiedPath(f[2], b) {
			return fmt.Errorf("complete diff contains prohibited path %q", f[2])
		}
		files[f[2]] = true
		components[firstComponent(f[2])] = true
		for _, n := range f[:2] {
			if n == "-" {
				return fmt.Errorf("complete diff contains binary change")
			}
			var v int
			if _, err := fmt.Sscanf(n, "%d", &v); err != nil {
				return fmt.Errorf("complete diff line count is invalid")
			}
			lines += v
		}
	}
	budget := b.WorkItem.Scope.ReviewBudget
	if len(files) > budget.MaxChangedFiles || lines > budget.MaxChangedLines || len(components) > budget.MaxComponents {
		return fmt.Errorf("complete diff exceeds approved scope budget")
	}
	commits, err := git(dir, "rev-list", "--reverse", base+".."+candidate)
	if err != nil {
		return err
	}
	for _, commit := range strings.Fields(string(commits)) {
		numstat, err := git(dir, "diff-tree", "--no-commit-id", "--numstat", "-r", commit)
		if err != nil {
			return err
		}
		commitFiles, commitLines, commitComponents := map[string]bool{}, 0, map[string]bool{}
		for _, line := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
			if line == "" {
				continue
			}
			f := strings.Split(line, "\t")
			if len(f) != 3 || f[0] == "-" || f[1] == "-" {
				return fmt.Errorf("intermediate commit has malformed or binary diff")
			}
			var add, del int
			if _, err := fmt.Sscanf(f[0], "%d", &add); err != nil {
				return err
			}
			if _, err := fmt.Sscanf(f[1], "%d", &del); err != nil {
				return err
			}
			commitFiles[f[2]], commitComponents[firstComponent(f[2])], commitLines = true, true, commitLines+add+del
		}
		if len(commitFiles) > budget.MaxChangedFiles || commitLines > budget.MaxChangedLines || len(commitComponents) > budget.MaxComponents {
			return fmt.Errorf("intermediate commit exceeds approved scope budget")
		}
		changed, err := git(dir, "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
		if err != nil {
			return err
		}
		for _, p := range strings.Fields(string(changed)) {
			if !allowedPath(p, b.AllowedPaths) || classifiedPath(p, b) {
				return fmt.Errorf("intermediate commit contains prohibited path %q", p)
			}
		}
	}
	return nil
}
func allowedPath(p string, allowed []string) bool {
	for _, prefix := range allowed {
		prefix = strings.TrimSuffix(prefix, "/")
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}
func classifiedPath(p string, b TaskBundle) bool {
	for _, x := range append(b.WorkItem.Scope.FileClassification.GeneratedPaths, b.WorkItem.Scope.FileClassification.LockfilePaths...) {
		if p == x || strings.HasPrefix(p, strings.TrimSuffix(x, "/")+"/") {
			return true
		}
	}
	return false
}
func firstComponent(p string) string {
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}
func loadChecks(path, rng string, required []string) ([]CheckOutcome, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var e AdoptionCheckEvidence
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil || e.FormatVersion != 1 || e.Range != rng {
		return nil, fmt.Errorf("deterministic check evidence is incomplete or bound to another range")
	}
	names := make([]string, len(e.Checks))
	for i := range e.Checks {
		names[i] = e.Checks[i].Name
	}
	sort.Strings(names)
	wanted := append([]string(nil), required...)
	sort.Strings(wanted)
	if strings.Join(names, "\x00") != strings.Join(wanted, "\x00") {
		return nil, fmt.Errorf("deterministic checks do not match the approved validation plan")
	}
	for _, c := range e.Checks {
		if c.Outcome != "passed" || !digestPattern.MatchString(c.OutputDigest) {
			return nil, fmt.Errorf("deterministic check evidence is invalid")
		}
	}
	sort.Slice(e.Checks, func(i, j int) bool { return e.Checks[i].Name < e.Checks[j].Name })
	return e.Checks, nil
}
func loadReviewBinding(path string) (AdoptionReviewEvidence, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return AdoptionReviewEvidence{}, err
	}
	var e ReviewEvidence
	if err := json.Unmarshal(data, &e); err != nil {
		return AdoptionReviewEvidence{}, err
	}
	return AdoptionReviewEvidence{Reviewer: AssessmentBinding{ExecutorID: e.Reviewer.ExecutorID, AssessmentDigest: e.Reviewer.AssessmentDigest}, Verifier: AssessmentBinding{ExecutorID: e.Verifier.ExecutorID, AssessmentDigest: e.Verifier.AssessmentDigest}, CombinedDigest: digest(data)}, nil
}
func loadAuditEvidence(path string, run Run) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	var e AuditRecord
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil || e.FormatVersion != 1 || e.RunID != run.ID || e.WorkItemKey != run.WorkItemKey || e.Adapter != run.Adapter || e.State != run.State || e.BaseSHA != run.BaseSHA || e.Branch != run.Branch || e.Attempts != run.Attempts || e.ReviewCycles != run.ReviewCycles {
		return "", fmt.Errorf("predecessor audit evidence is incomplete or does not bind the run")
	}
	canonical, err := signature.Canonicalize(data)
	if err != nil {
		return "", fmt.Errorf("canonicalize predecessor audit evidence: %w", err)
	}
	return "event-" + strings.TrimPrefix(signature.Digest(canonical), "sha256:"), nil
}
func workItemDigest(v interface{}) string {
	data, _ := json.Marshal(v)
	canonical, _ := signature.Canonicalize(data)
	return signature.Digest(canonical)
}
func configuredTechnicalOwnerID(c config.Config) string {
	for _, k := range c.Signing.TrustedKeys {
		if k.Role == "technical-owner" {
			return k.KeyID
		}
	}
	return ""
}
func configuredTechnicalOwner(c config.Config, key signature.TrustedKey) bool {
	for _, k := range c.Signing.TrustedKeys {
		if k.KeyID == key.KeyID && k.Role == key.Role && k.Algorithm == key.Algorithm && k.PublicKey == key.PublicKey {
			return true
		}
	}
	return false
}

func adoptionID(r AdoptionRecord) string {
	value := strings.Join([]string{r.RepositoryID, r.WorkItemKey, r.PredecessorRunID, r.OriginalBaseSHA, r.PredecessorCommitSHA, r.CandidateCommitSHA}, "\x00")
	return "adoption-" + strings.TrimPrefix(signature.Digest([]byte(value)), "sha256:")
}
func adoptionRegistryPath(root, repo, id string) (string, string, error) {
	if !filepath.IsAbs(root) {
		return "", "", fmt.Errorf("adoption registry must be an absolute path outside the repository")
	}
	rel, err := filepath.Rel(filepath.Clean(repo), filepath.Clean(root))
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", "", fmt.Errorf("adoption registry must be outside the repository")
	}
	return filepath.Clean(root), filepath.Join(filepath.Clean(root), id+".json"), nil
}
func ensureRegistry(root string, create bool) error {
	root = filepath.Clean(root)
	parts := []string{}
	for p := root; p != filepath.Dir(p); p = filepath.Dir(p) {
		parts = append(parts, p)
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	for _, p := range parts {
		info, err := os.Lstat(p)
		if os.IsNotExist(err) {
			if !create {
				return nil
			}
			if err := os.Mkdir(p, 0o700); err != nil && !os.IsExist(err) {
				return fmt.Errorf("create adoption registry: %w", err)
			}
			info, err = os.Lstat(p)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("adoption registry path contains a symlink or non-directory")
		}
		if p == root && (info.Mode().Perm()&0o077 != 0 || registryOwner(info) != uint32(os.Getuid())) {
			return fmt.Errorf("adoption registry must be owner-only")
		}
	}
	return nil
}
func registryOwner(info os.FileInfo) uint32 {
	if s, ok := info.Sys().(*syscall.Stat_t); ok {
		return s.Uid
	}
	return ^uint32(0)
}
func rejectRegistryConflict(root string, r AdoptionRecord) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("adoption registry contains an unsafe entry")
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return err
		}
		var env signature.Envelope
		if json.Unmarshal(data, &env) != nil {
			return fmt.Errorf("adoption registry contains malformed record")
		}
		old, err := ParseAdoptionRecord(env.Payload)
		if err != nil {
			return fmt.Errorf("adoption registry contains invalid record")
		}
		if old.PredecessorRunID == r.PredecessorRunID && old.RepositoryID == r.RepositoryID && old.WorkItemKey == r.WorkItemKey {
			if old.CandidateCommitSHA == r.CandidateCommitSHA {
				return fmt.Errorf("adoption record already exists or has been replayed")
			}
			return fmt.Errorf("predecessor already has a conflicting successor")
		}
	}
	return nil
}
func persistAdoption(path string, e signature.Envelope) error {
	dir := filepath.Dir(path)
	if err := ensureRegistry(dir, false); err != nil {
		return err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	temp, err := os.OpenFile(filepath.Join(dir, ".adoption-"+strings.TrimPrefix(e.PayloadDigest, "sha256:")+".tmp"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create adoption record atomically: %w", err)
	}
	tempPath := temp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("adoption record already exists or has been replayed")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Link(tempPath, path); err != nil {
		return fmt.Errorf("persist adoption record without overwrite: %w", err)
	}
	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("adoption record persisted but temporary cleanup failed: %w", err)
	}
	ok = true
	return nil
}
