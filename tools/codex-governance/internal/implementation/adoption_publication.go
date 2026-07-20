package implementation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-governance/internal/config"
	"codex-governance/internal/signature"
)

// PublicationSuccessorView is the immutable candidate identity established by
// a separately signed technical-owner adoption record. It is not publication
// authority; repository-owner authorization remains a distinct requirement.
type PublicationSuccessorView struct {
	RecordID        string
	CandidateBranch string
	CandidateCommit string
	PredecessorRun  string
}

// ResolvePublicationSuccessor reads one immutable registry entry and proves it
// still describes this exact predecessor run and checked-out candidate.
func ResolvePublicationSuccessor(runPath string, run Run, registryPath, recordID, worktree string, cfg config.Config, now time.Time) (PublicationSuccessorView, error) {
	if !recordPattern.MatchString(recordID) || run.ID == "" || run.CommitSHA == "" || run.BaseSHA == "" {
		return PublicationSuccessorView{}, fmt.Errorf("successor resolution inputs are incomplete")
	}
	root, path, err := adoptionRegistryPath(registryPath, worktree, recordID)
	if err != nil {
		return PublicationSuccessorView{}, err
	}
	if err := ensureRegistry(root, false); err != nil {
		return PublicationSuccessorView{}, err
	}
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return PublicationSuccessorView{}, fmt.Errorf("adoption registry entry is missing or unsafe")
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return PublicationSuccessorView{}, err
	}
	var envelope signature.Envelope
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil || decoder.More() {
		return PublicationSuccessorView{}, fmt.Errorf("adoption registry entry is malformed")
	}
	registry, err := cfg.TrustedKeyRegistry()
	if err != nil {
		return PublicationSuccessorView{}, err
	}
	repositoryID, err := cfg.PublicationRepositoryID()
	if err != nil {
		return PublicationSuccessorView{}, err
	}
	defaultBranch, err := configuredDefaultBranch(worktree)
	if err != nil {
		return PublicationSuccessorView{}, err
	}
	record, err := ValidateSignedAdoptionRecord(envelope, registry, now, repositoryID, run.WorkItemKey, defaultBranch)
	if err != nil {
		return PublicationSuccessorView{}, fmt.Errorf("verify signed successor record: %w", err)
	}
	if record.ID != recordID || record.ID != adoptionID(record) || record.PredecessorRunID != run.ID || record.PredecessorCommitSHA != run.CommitSHA || record.OriginalBaseSHA != run.BaseSHA {
		return PublicationSuccessorView{}, fmt.Errorf("successor record does not bind this predecessor run")
	}
	runData, err := os.ReadFile(filepath.Clean(runPath))
	if err != nil || record.PredecessorRunDigest != digest(runData) {
		return PublicationSuccessorView{}, fmt.Errorf("successor record does not bind predecessor run bytes")
	}
	head, err := resolveCommit(worktree, "HEAD")
	if err != nil || head != record.CandidateCommitSHA {
		return PublicationSuccessorView{}, fmt.Errorf("checked-out candidate does not match successor record")
	}
	branch, err := resolveCommit(worktree, "refs/heads/"+record.CandidateBranch)
	if err != nil || branch != record.CandidateCommitSHA {
		return PublicationSuccessorView{}, fmt.Errorf("candidate branch does not match successor record")
	}
	diff, err := DiffBytes(worktree, "--no-renames", record.OriginalBaseSHA+".."+record.CandidateCommitSHA)
	if err != nil || record.CompleteDiffDigest != digestBytes(diff) {
		return PublicationSuccessorView{}, fmt.Errorf("successor record complete diff no longer matches candidate")
	}
	if err := rejectSuccessorAmbiguity(root, recordID, record); err != nil {
		return PublicationSuccessorView{}, err
	}
	return PublicationSuccessorView{RecordID: record.ID, CandidateBranch: record.CandidateBranch, CandidateCommit: record.CandidateCommitSHA, PredecessorRun: record.PredecessorRunID}, nil
}

func rejectSuccessorAmbiguity(root, selected string, wanted AdoptionRecord) error {
	entries, err := os.ReadDir(root)
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
		var envelope signature.Envelope
		if json.Unmarshal(data, &envelope) != nil {
			return fmt.Errorf("adoption registry contains malformed record")
		}
		record, err := ParseAdoptionRecord(envelope.Payload)
		if err != nil {
			return fmt.Errorf("adoption registry contains invalid record")
		}
		if record.PredecessorRunID == wanted.PredecessorRunID && record.RepositoryID == wanted.RepositoryID && record.WorkItemKey == wanted.WorkItemKey && strings.TrimSuffix(entry.Name(), ".json") != selected {
			return fmt.Errorf("predecessor has an ambiguous or replayed successor record")
		}
	}
	return nil
}
