package implementation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"codex-governance/internal/signature"
)

// AdoptionFormatVersion is deliberately independent from implementation-run
// versioning. Format-version-1 runs therefore retain their current meaning.
const AdoptionFormatVersion = 1

var (
	digestPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	repositoryPattern = regexp.MustCompile(`^github\.com/[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)
	signerPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{12,64}$`)
	commitPattern     = regexp.MustCompile(`^[a-f0-9]{40}([a-f0-9]{24})?$`)
	keyPattern        = regexp.MustCompile(`^[A-Z][A-Z0-9]+-[1-9][0-9]*$`)
	runPattern        = regexp.MustCompile(`^run-[a-f0-9]{16}$`)
	recordPattern     = regexp.MustCompile(`^adoption-[a-f0-9]{64}$`)
	eventPattern      = regexp.MustCompile(`^event-[a-f0-9]{64}$`)
	branchPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)
)

// AdoptionRecord is the unsigned payload of the ADR-selected, separately
// signed descendant-adoption record. Phase 1 only validates and encodes this
// value; it does not create, sign, persist, or consume a record.
type AdoptionRecord struct {
	FormatVersion         int                    `json:"format_version"`
	ID                    string                 `json:"id"`
	RepositoryID          string                 `json:"repository_id"`
	RepositoryDigest      string                 `json:"repository_digest"`
	WorkItemKey           string                 `json:"work_item_key"`
	PredecessorRunID      string                 `json:"predecessor_run_id"`
	PredecessorRunDigest  string                 `json:"predecessor_run_digest"`
	OriginalBaseSHA       string                 `json:"original_base_sha"`
	PredecessorCommitSHA  string                 `json:"predecessor_commit_sha"`
	CandidateBranch       string                 `json:"candidate_branch"`
	CandidateCommitSHA    string                 `json:"candidate_commit_sha"`
	AdoptedRange          string                 `json:"adopted_range"`
	CompleteDiffDigest    string                 `json:"complete_diff_digest"`
	WorkItemDigest        string                 `json:"work_item_digest"`
	SourceEnvelopeDigest  string                 `json:"source_envelope_digest"`
	TaskBundleDigest      string                 `json:"task_bundle_digest"`
	ConfigurationDigest   string                 `json:"configuration_digest"`
	GuidanceDigest        string                 `json:"guidance_digest"`
	ReviewEvidence        AdoptionReviewEvidence `json:"review_evidence"`
	DeterministicChecks   []CheckOutcome         `json:"deterministic_checks"`
	Reason                string                 `json:"reason"`
	AuthorizedRole        string                 `json:"authorized_role"`
	SignerIdentity        string                 `json:"signer_identity"`
	IssuedAt              time.Time              `json:"issued_at"`
	ExpiresAt             time.Time              `json:"expires_at"`
	PrecedingAuditEventID string                 `json:"preceding_audit_event_id"`
}

// AdoptionReviewEvidence binds distinct assessments to the adopted diff.
type AdoptionReviewEvidence struct {
	Reviewer       AssessmentBinding `json:"reviewer"`
	Verifier       AssessmentBinding `json:"verifier"`
	CombinedDigest string            `json:"combined_digest"`
}

// AssessmentBinding identifies one independent assessment artifact.
type AssessmentBinding struct {
	ExecutorID       string `json:"executor_id"`
	AssessmentDigest string `json:"assessment_digest"`
}

// CheckOutcome records one deterministic passing validation result.
type CheckOutcome struct {
	Name         string `json:"name"`
	Outcome      string `json:"outcome"`
	OutputDigest string `json:"output_digest"`
}

// ParseAdoptionRecord rejects unknown fields, trailing JSON, and every
// structurally incomplete or syntactically ambiguous authority binding.
func ParseAdoptionRecord(data []byte) (AdoptionRecord, error) {
	if err := rejectDuplicateJSONMembers(data); err != nil {
		return AdoptionRecord{}, fmt.Errorf("parse adoption record: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record AdoptionRecord
	if err := decoder.Decode(&record); err != nil {
		return AdoptionRecord{}, fmt.Errorf("parse adoption record: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return AdoptionRecord{}, fmt.Errorf("parse adoption record: input contains multiple JSON values")
		}
		return AdoptionRecord{}, fmt.Errorf("parse adoption record: %w", err)
	}
	if err := ValidateAdoptionRecord(record); err != nil {
		return AdoptionRecord{}, err
	}
	return record, nil
}

// MarshalAdoptionRecord never repairs invalid data before encoding it.
func MarshalAdoptionRecord(record AdoptionRecord) ([]byte, error) {
	if err := ValidateAdoptionRecord(record); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return signature.Canonicalize(encoded)
}

// ValidateAdoptionRecord validates the context-independent payload contract.
func ValidateAdoptionRecord(record AdoptionRecord) error {
	if record.FormatVersion != AdoptionFormatVersion {
		return fmt.Errorf("unsupported adoption-record version %d", record.FormatVersion)
	}
	if !recordPattern.MatchString(record.ID) || !repositoryPattern.MatchString(record.RepositoryID) || !keyPattern.MatchString(record.WorkItemKey) || !runPattern.MatchString(record.PredecessorRunID) || !eventPattern.MatchString(record.PrecedingAuditEventID) {
		return fmt.Errorf("adoption record contains an invalid immutable identity")
	}
	for _, binding := range []string{record.RepositoryDigest, record.PredecessorRunDigest, record.CompleteDiffDigest, record.WorkItemDigest, record.SourceEnvelopeDigest, record.TaskBundleDigest, record.ConfigurationDigest, record.GuidanceDigest, record.ReviewEvidence.CombinedDigest, record.ReviewEvidence.Reviewer.AssessmentDigest, record.ReviewEvidence.Verifier.AssessmentDigest} {
		if !digestPattern.MatchString(binding) {
			return fmt.Errorf("adoption record contains a malformed digest")
		}
	}
	if !commitPattern.MatchString(record.OriginalBaseSHA) || !commitPattern.MatchString(record.PredecessorCommitSHA) || !commitPattern.MatchString(record.CandidateCommitSHA) {
		return fmt.Errorf("adoption record contains an invalid commit identity")
	}
	if record.PredecessorCommitSHA == record.CandidateCommitSHA || record.AdoptedRange != record.PredecessorCommitSHA+".."+record.CandidateCommitSHA {
		return fmt.Errorf("adoption record range does not bind predecessor and candidate commits")
	}
	if !branchPattern.MatchString(record.CandidateBranch) || record.CandidateBranch == "HEAD" || strings.HasPrefix(record.CandidateBranch, "refs/") || strings.EqualFold(record.CandidateBranch, "main") || strings.EqualFold(record.CandidateBranch, "master") {
		return fmt.Errorf("adoption record contains a mutable candidate alias")
	}
	if record.ReviewEvidence.Reviewer.ExecutorID == "" || record.ReviewEvidence.Verifier.ExecutorID == "" || record.ReviewEvidence.Reviewer.ExecutorID == record.ReviewEvidence.Verifier.ExecutorID {
		return fmt.Errorf("adoption record requires distinct reviewer and verifier identities")
	}
	if record.AuthorizedRole != "technical-owner" || !signerPattern.MatchString(record.SignerIdentity) {
		return fmt.Errorf("adoption record signer is not permitted")
	}
	if record.IssuedAt.IsZero() || record.ExpiresAt.IsZero() || record.IssuedAt.Location() != time.UTC || record.ExpiresAt.Location() != time.UTC || !record.ExpiresAt.After(record.IssuedAt) {
		return fmt.Errorf("adoption record has invalid issuance or expiry timestamps")
	}
	if strings.TrimSpace(record.Reason) == "" || record.Reason != strings.TrimSpace(record.Reason) || len(record.Reason) > 1024 {
		return fmt.Errorf("adoption record reason is invalid")
	}
	if len(record.DeterministicChecks) == 0 {
		return fmt.Errorf("adoption record is missing deterministic check outcomes")
	}
	names := make([]string, len(record.DeterministicChecks))
	for index, check := range record.DeterministicChecks {
		if strings.TrimSpace(check.Name) == "" || check.Name != strings.TrimSpace(check.Name) || check.Outcome != "passed" || !digestPattern.MatchString(check.OutputDigest) {
			return fmt.Errorf("adoption record check outcome is invalid")
		}
		names[index] = check.Name
	}
	if !sort.StringsAreSorted(names) || hasDuplicate(names) {
		return fmt.Errorf("adoption record check outcomes are not canonical")
	}
	return nil
}

// ValidateAdoptionRecordFor binds a structurally valid payload to the exact
// repository, Jira work item, and normalized configured default branch expected
// by its consumer.
func ValidateAdoptionRecordFor(record AdoptionRecord, repositoryID, workItemKey, defaultBranch string) error {
	if err := ValidateAdoptionRecord(record); err != nil {
		return err
	}
	if record.RepositoryID != repositoryID || record.WorkItemKey != workItemKey {
		return fmt.Errorf("adoption record repository or work item does not match")
	}
	if !branchPattern.MatchString(defaultBranch) || strings.HasPrefix(defaultBranch, "refs/") {
		return fmt.Errorf("adoption record consumer has an invalid configured default branch")
	}
	if record.CandidateBranch == defaultBranch {
		return fmt.Errorf("adoption record candidate branch matches the configured default branch")
	}
	return nil
}

// ValidateSignedAdoptionRecord verifies the common signed envelope and binds
// its signer and timestamps exactly to the validated payload. A key omitted
// from the registry is treated as revoked by the shared signature contract.
func ValidateSignedAdoptionRecord(envelope signature.Envelope, registry signature.Registry, now time.Time, repositoryID, workItemKey, defaultBranch string) (AdoptionRecord, error) {
	if err := registry.Verify(envelope, []string{"technical-owner"}, now); err != nil {
		return AdoptionRecord{}, fmt.Errorf("verify signed adoption record: %w", err)
	}
	record, err := ParseAdoptionRecord(envelope.Payload)
	if err != nil {
		return AdoptionRecord{}, err
	}
	if err := ValidateAdoptionRecordFor(record, repositoryID, workItemKey, defaultBranch); err != nil {
		return AdoptionRecord{}, err
	}
	if now.Before(record.IssuedAt) {
		return AdoptionRecord{}, fmt.Errorf("adoption record is not yet valid")
	}
	if envelope.KeyID != record.SignerIdentity || envelope.SignerRole != record.AuthorizedRole || envelope.ExpiresAt == nil || !envelope.IssuedAt.Equal(record.IssuedAt) || !envelope.ExpiresAt.Equal(record.ExpiresAt) {
		return AdoptionRecord{}, fmt.Errorf("adoption record does not bind its signed envelope")
	}
	return record, nil
}

// rejectDuplicateJSONMembers walks the raw token stream before decoding into
// structs because encoding/json otherwise silently keeps the final value of a
// duplicate member. Every object, including nested evidence and check objects,
// must have unique names before it can carry authority.
func rejectDuplicateJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return fmt.Errorf("adoption record must be a JSON object")
	}
	if err := walkJSONObject(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("input contains multiple JSON values")
		}
		return err
	}
	return nil
}

func walkJSONObject(decoder *json.Decoder) error {
	members := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := token.(string)
		if !ok {
			return fmt.Errorf("object member name is invalid")
		}
		if _, exists := members[name]; exists {
			return fmt.Errorf("duplicate JSON member %q", name)
		}
		members[name] = struct{}{}
		if err := walkJSONValue(decoder); err != nil {
			return err
		}
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return fmt.Errorf("object is not terminated")
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		return walkJSONObject(decoder)
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing, ok := token.(json.Delim); !ok || closing != ']' {
			return fmt.Errorf("array is not terminated")
		}
		return nil
	default:
		return fmt.Errorf("invalid JSON delimiter %q", delimiter)
	}
}

func hasDuplicate(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

// AdoptionRecordDigest returns the stable digest of an already-valid encoded
// payload. It is useful to a later signing and persistence phase only.
func AdoptionRecordDigest(encoded []byte) string {
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
