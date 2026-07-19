package implementation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"codex-governance/internal/signature"
)

func TestAdoptionRecordRoundTripIsDeterministic(t *testing.T) {
	record := validAdoptionRecord()
	first, err := MarshalAdoptionRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseAdoptionRecord(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalAdoptionRecord(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || AdoptionRecordDigest(first) != AdoptionRecordDigest(second) {
		t.Fatalf("adoption record encoding is not deterministic: %s != %s", first, second)
	}
}

func TestParseAdoptionRecordFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*AdoptionRecord)
		want   string
	}{
		{"missing predecessor", func(r *AdoptionRecord) { r.PredecessorRunID = "" }, "identity"},
		{"mismatched range", func(r *AdoptionRecord) { r.AdoptedRange = r.OriginalBaseSHA + ".." + r.PredecessorCommitSHA }, "range"},
		{"non-UTC issuance", func(r *AdoptionRecord) { r.IssuedAt = r.IssuedAt.In(time.FixedZone("offset", 3600)) }, "timestamps"},
		{"malformed digest", func(r *AdoptionRecord) { r.TaskBundleDigest = "sha256:upper" }, "digest"},
		{"unsupported version", func(r *AdoptionRecord) { r.FormatVersion = 2 }, "unsupported"},
		{"unpermitted role", func(r *AdoptionRecord) { r.AuthorizedRole = "repository-owner" }, "not permitted"},
		{"expired", func(r *AdoptionRecord) { r.ExpiresAt = r.IssuedAt }, "timestamps"},
		{"mutable alias", func(r *AdoptionRecord) { r.CandidateBranch = "HEAD" }, "mutable"},
		{"same executor", func(r *AdoptionRecord) { r.ReviewEvidence.Verifier.ExecutorID = r.ReviewEvidence.Reviewer.ExecutorID }, "distinct"},
		{"uncanonical checks", func(r *AdoptionRecord) {
			r.DeterministicChecks[0], r.DeterministicChecks[1] = r.DeterministicChecks[1], r.DeterministicChecks[0]
		}, "canonical"},
	} {
		t.Run(test.name, func(t *testing.T) {
			record := validAdoptionRecord()
			test.mutate(&record)
			data, err := jsonMarshalUnchecked(record)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseAdoptionRecord(data); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateAdoptionRecordForRejectsRepositoryAndWorkItemMismatch(t *testing.T) {
	record := validAdoptionRecord()
	for _, test := range []struct {
		repository string
		workItem   string
	}{
		{"github.com/other/governance", record.WorkItemKey},
		{record.RepositoryID, "REK-99"},
	} {
		if err := ValidateAdoptionRecordFor(record, test.repository, test.workItem, "main"); err == nil {
			t.Fatal("accepted mismatched repository or work item")
		}
	}
}

func TestValidateAdoptionRecordForRejectsConfiguredDefaultBranch(t *testing.T) {
	record := validAdoptionRecord()
	record.CandidateBranch = "release"
	if err := ValidateAdoptionRecordFor(record, record.RepositoryID, record.WorkItemKey, "release"); err == nil {
		t.Fatal("accepted the configured default branch")
	}
}

func TestValidateSignedAdoptionRecordBindsAuthorityAndEnvelope(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	record := validAdoptionRecord()
	trusted := signature.TrustedKey{KeyID: record.SignerIdentity, Role: "technical-owner", Algorithm: signature.Algorithm, PublicKey: base64.StdEncoding.EncodeToString(publicKey)}
	registry, err := signature.NewRegistry(signature.FormatVersion, []signature.TrustedKey{trusted})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := MarshalAdoptionRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := signature.Sign(payload, record.SignerIdentity, record.AuthorizedRole, privateKey, record.IssuedAt, &record.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateSignedAdoptionRecord(envelope, registry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err != nil {
		t.Fatal(err)
	}

	t.Run("expired", func(t *testing.T) {
		if _, err := ValidateSignedAdoptionRecord(envelope, registry, record.ExpiresAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted expired envelope")
		}
	})
	t.Run("not yet valid", func(t *testing.T) {
		if _, err := ValidateSignedAdoptionRecord(envelope, registry, record.IssuedAt.Add(-time.Second), record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted an envelope before issuance")
		}
	})
	t.Run("revoked", func(t *testing.T) {
		revoked, err := signature.NewRegistry(signature.FormatVersion, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateSignedAdoptionRecord(envelope, revoked, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted revoked signer")
		}
	})
	t.Run("unpermitted role", func(t *testing.T) {
		ownerKey := trusted
		ownerKey.Role = "repository-owner"
		ownerRegistry, err := signature.NewRegistry(signature.FormatVersion, []signature.TrustedKey{ownerKey})
		if err != nil {
			t.Fatal(err)
		}
		ownerEnvelope, err := signature.Sign(payload, record.SignerIdentity, "repository-owner", privateKey, record.IssuedAt, &record.ExpiresAt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateSignedAdoptionRecord(ownerEnvelope, ownerRegistry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted unpermitted role")
		}
	})
	t.Run("key binding mismatch", func(t *testing.T) {
		alternatePublicKey, alternatePrivateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		alternate := signature.TrustedKey{KeyID: "sha256:abcdef123456", Role: record.AuthorizedRole, Algorithm: signature.Algorithm, PublicKey: base64.StdEncoding.EncodeToString(alternatePublicKey)}
		alternateRegistry, err := signature.NewRegistry(signature.FormatVersion, []signature.TrustedKey{trusted, alternate})
		if err != nil {
			t.Fatal(err)
		}
		alternateEnvelope, err := signature.Sign(payload, alternate.KeyID, record.AuthorizedRole, alternatePrivateKey, record.IssuedAt, &record.ExpiresAt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateSignedAdoptionRecord(alternateEnvelope, alternateRegistry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted mismatched envelope key binding")
		}
	})
	t.Run("role binding mismatch", func(t *testing.T) {
		mismatched := envelope
		mismatched.SignerRole = "repository-owner"
		if _, err := ValidateSignedAdoptionRecord(mismatched, registry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted mismatched envelope role binding")
		}
	})
	t.Run("expiry binding mismatch", func(t *testing.T) {
		expiresAt := record.ExpiresAt.Add(time.Second)
		mismatched, err := signature.Sign(payload, record.SignerIdentity, record.AuthorizedRole, privateKey, record.IssuedAt, &expiresAt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateSignedAdoptionRecord(mismatched, registry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted mismatched envelope expiry binding")
		}
	})
	t.Run("missing expiry binding", func(t *testing.T) {
		mismatched := envelope
		mismatched.ExpiresAt = nil
		if _, err := ValidateSignedAdoptionRecord(mismatched, registry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted envelope without expiry binding")
		}
	})
	t.Run("timestamp binding mismatch", func(t *testing.T) {
		issuedAt := record.IssuedAt.Add(time.Second)
		mismatched, err := signature.Sign(payload, record.SignerIdentity, record.AuthorizedRole, privateKey, issuedAt, &record.ExpiresAt)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ValidateSignedAdoptionRecord(mismatched, registry, record.IssuedAt, record.RepositoryID, record.WorkItemKey, "main"); err == nil {
			t.Fatal("accepted mismatched envelope timestamp")
		}
	})
}

func TestParseAdoptionRecordRejectsUnknownAndTrailingFields(t *testing.T) {
	data, err := MarshalAdoptionRecord(validAdoptionRecord())
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{string(data[:len(data)-1]) + `,"extra":true}`, string(data) + `{}`} {
		if _, err := ParseAdoptionRecord([]byte(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}

func TestParseAdoptionRecordRejectsDuplicateMembersAtEveryObjectDepth(t *testing.T) {
	data, err := MarshalAdoptionRecord(validAdoptionRecord())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		needle  string
		replace string
	}{
		{"root", `"id":"adoption-`, `"id":"ignored","id":"adoption-`},
		{"nested object", `"reviewer":{"assessment_digest":`, `"reviewer":{"assessment_digest":"ignored","assessment_digest":`},
		{"array object", `"name":"make build"`, `"name":"ignored","name":"make build"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := strings.Replace(string(data), test.needle, test.replace, 1)
			if _, err := ParseAdoptionRecord([]byte(input)); err == nil || !strings.Contains(err.Error(), "duplicate JSON member") {
				t.Fatalf("error = %v, want duplicate-member rejection", err)
			}
		})
	}
}

func validAdoptionRecord() AdoptionRecord {
	digest := func(c string) string { return "sha256:" + strings.Repeat(c, 64) }
	sha := func(c string) string { return strings.Repeat(c, 40) }
	issued := time.Date(2026, time.July, 19, 0, 0, 0, 0, time.UTC)
	return AdoptionRecord{FormatVersion: 1, ID: "adoption-" + strings.Repeat("a", 64), RepositoryID: "github.com/acme/governance", RepositoryDigest: digest("b"), WorkItemKey: "REK-42", PredecessorRunID: "run-" + strings.Repeat("c", 16), PredecessorRunDigest: digest("d"), OriginalBaseSHA: sha("e"), PredecessorCommitSHA: sha("f"), CandidateBranch: "remediation/rek-42", CandidateCommitSHA: sha("1"), AdoptedRange: sha("f") + ".." + sha("1"), CompleteDiffDigest: digest("2"), WorkItemDigest: digest("3"), SourceEnvelopeDigest: digest("4"), TaskBundleDigest: digest("5"), ConfigurationDigest: digest("6"), GuidanceDigest: digest("7"), ReviewEvidence: AdoptionReviewEvidence{Reviewer: AssessmentBinding{ExecutorID: "reviewer-1", AssessmentDigest: digest("8")}, Verifier: AssessmentBinding{ExecutorID: "verifier-1", AssessmentDigest: digest("9")}, CombinedDigest: digest("a")}, DeterministicChecks: []CheckOutcome{{Name: "make build", Outcome: "passed", OutputDigest: digest("b")}, {Name: "make test", Outcome: "passed", OutputDigest: digest("c")}}, Reason: "Apply reviewed remediation", AuthorizedRole: "technical-owner", SignerIdentity: "sha256:0123456789ab", IssuedAt: issued, ExpiresAt: issued.Add(time.Hour), PrecedingAuditEventID: "event-" + strings.Repeat("e", 64)}
}

func jsonMarshalUnchecked(value any) ([]byte, error) { return json.Marshal(value) }
