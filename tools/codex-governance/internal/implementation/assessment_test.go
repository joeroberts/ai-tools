package implementation

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"codex-governance/internal/jira"
	"codex-governance/internal/ollama"
	"codex-governance/internal/workitem"
)

func TestParseAssessment(t *testing.T) {
	assessment, err := parseAssessment([]byte(`{"findings":[{"id":"R-1","severity":"blocking","summary":"outside approved paths"}]}`))
	if err != nil || len(assessment.Findings) != 1 {
		t.Fatalf("parseAssessment() = %#v, %v", assessment, err)
	}
	if _, err := parseAssessment([]byte(`{"findings":[{"id":"","severity":"bad","summary":""}]}`)); err == nil {
		t.Fatal("invalid assessment was accepted")
	}
}

func TestParseAssessmentAcceptsDelimitedProtocol(t *testing.T) {
	assessment, err := parseAssessment([]byte("R-1|blocking|outside approved paths\nR-2|informational|looks good"))
	if err != nil || len(assessment.Findings) != 2 || assessment.Findings[1].Severity != "informational" {
		t.Fatalf("parseAssessment() = %#v, %v", assessment, err)
	}
	assessment, err = parseAssessment([]byte("ID|severity|summary\nR-1|minor|example"))
	if err != nil || len(assessment.Findings) != 1 || assessment.Findings[0].ID != "R-1" {
		t.Fatalf("parseAssessment(header) = %#v, %v", assessment, err)
	}
	assessment, err = parseAssessment([]byte("ID|severity|summary\nNONE"))
	if err != nil || len(assessment.Findings) != 0 {
		t.Fatalf("parseAssessment(header NONE) = %#v, %v", assessment, err)
	}
	assessment, err = parseAssessment([]byte("R-1|minor|example|"))
	if err != nil || len(assessment.Findings) != 1 || assessment.Findings[0].ID != "R-1" {
		t.Fatalf("parseAssessment(trailing delimiter) = %#v, %v", assessment, err)
	}
	assessment, err = parseAssessment([]byte("NONE"))
	if err != nil || len(assessment.Findings) != 0 {
		t.Fatalf("parseAssessment(NONE) = %#v, %v", assessment, err)
	}
	for _, response := range []string{"", "NONE\nR-1|minor|unexpected", "ID|severity|summary", "R-1|minor", "R-1|minor|summary|extra"} {
		if _, err := parseAssessment([]byte(response)); err == nil {
			t.Fatalf("invalid delimited response was accepted: %q", response)
		}
	}
}

func TestParseAssessmentForChunkRequiresChangedLineEvidence(t *testing.T) {
	diff := "diff --git a/internal/example.go b/internal/example.go\n--- a/internal/example.go\n+++ b/internal/example.go\n@@ -1,0 +1,2 @@\n+package example\n+var Enabled = true\n"
	assessment, err := parseAssessmentForChunk([]byte("R-1|blocking|internal/example.go:2|The value is always true|example"), diff)
	if err != nil || len(assessment.Findings) != 1 {
		t.Fatalf("parseAssessmentForChunk() = %#v, %v", assessment, err)
	}
	if _, err := parseAssessmentForChunk([]byte("R-1|blocking|internal/example.go:99|The value is always true|example"), diff); err == nil {
		t.Fatal("assessment with an unchanged location was accepted")
	}
	assessment, err = parseAssessmentForChunk([]byte("R-1|minor|tools/codex-governance/internal/example.go:2|The value is always true|example"), diff)
	if err != nil || assessment.Findings[0].Location != "internal/example.go:2" {
		t.Fatalf("prefixed location = %#v, %v", assessment, err)
	}
	prefixedDiff := strings.Replace(diff, "internal/example.go", "tools/codex-governance/internal/example.go", 2)
	assessment, err = parseAssessmentForChunk([]byte("R-1|minor|internal/example.go:2|The value is always true|example"), prefixedDiff)
	if err != nil || assessment.Findings[0].Location != "internal/example.go:2" {
		t.Fatalf("candidate-root location = %#v, %v", assessment, err)
	}
	if _, err := parseAssessmentForChunk([]byte("R-1|minor|internal/example.go:2|NONE|None"), diff); err == nil {
		t.Fatal("assessment with NONE fields was accepted")
	}
	assessment, err = parseAssessmentForChunk([]byte("R-1|informational|internal/example.go:2|example\nNONE"), diff)
	if err != nil || len(assessment.Findings) != 1 || assessment.Findings[0].Condition != "" {
		t.Fatalf("informational assessment = %#v, %v", assessment, err)
	}
}

func TestParseAssessmentAcceptsSingleJSONCodeFence(t *testing.T) {
	assessment, err := parseAssessment([]byte("```json\n{\"findings\":[]}\n```\n"))
	if err != nil || len(assessment.Findings) != 0 {
		t.Fatalf("parseAssessment() = %#v, %v", assessment, err)
	}
	if _, err := parseAssessment([]byte("before\n```json\n{\"findings\":[]}\n```")); err == nil {
		t.Fatal("assessment with surrounding prose was accepted")
	}
}

func TestParseAssessmentAcceptsOnlySingleDanglingArrayClose(t *testing.T) {
	assessment, err := parseAssessment([]byte("```json\n{\"findings\":[]}\n]\n```"))
	if err != nil || len(assessment.Findings) != 0 {
		t.Fatalf("parseAssessment() = %#v, %v", assessment, err)
	}
	assessment, err = parseAssessment([]byte("```json\n{\"findings\":[]}\n]}\n```"))
	if err != nil || len(assessment.Findings) != 0 {
		t.Fatalf("parseAssessment() extra close = %#v, %v", assessment, err)
	}
	if _, err := parseAssessment([]byte(`{"findings":[]} trailing`)); err == nil {
		t.Fatal("assessment with prose was accepted")
	}
}

func TestAssessmentPromptExcludesDuplicatedSignedExport(t *testing.T) {
	bundle := TaskBundle{
		WorkItem: workitem.Item{Source: workitem.Source{SubtaskKey: "REK-3"}, Scope: workitem.Scope{
			Phase: "Jira planning", ChangeClass: "high-risk", AllowedPaths: []string{"internal/implementation"},
			NonGoals: []string{"no Jira writes"}, TechnicalAcceptanceCriteria: []string{"require evidence"}, ValidationPlan: []string{"make test"},
		}},
		AllowedPaths: []string{"internal/implementation"}, Commands: []string{"make test"}, ADR: "no ADR", Guidance: "review guidance",
		SourceEvidence: jira.OfflineExportEvidence{EnvelopeDigest: "sha256:export", IssuerKeyID: "issuer", CapturedAt: "2026-07-14T00:00:00Z"},
		TicketBaseline: jira.OfflineExport{Story: jira.Issue{Description: "duplicated ticket body"}},
	}
	prompt, err := assessmentPrompt("reviewer", bundle, "diff --git a/file")
	if err != nil || !strings.Contains(prompt, `"work_item":"REK-3"`) || !strings.Contains(prompt, `"allowed_paths":["internal/implementation"]`) {
		t.Fatalf("assessmentPrompt() = %q, %v", prompt, err)
	}
	if !strings.Contains(prompt, "package-local symbols, and imports absent from this chunk may exist elsewhere") || !strings.Contains(prompt, "Report a finding only when the supplied lines directly demonstrate a concrete defect") {
		t.Fatalf("assessment prompt lacks chunk-evidence guardrail: %q", prompt)
	}
	if strings.Contains(prompt, "duplicated ticket body") || strings.Contains(prompt, "source_envelope") || strings.Contains(prompt, "ticket_baseline") {
		t.Fatalf("assessment prompt retained duplicated source material: %q", prompt)
	}
}

func TestAssessmentPromptsKeepDiffChunksWithinLimit(t *testing.T) {
	bundle := assessmentTestBundle()
	diff := testDiffSection("first", 14000) + testDiffSection("second", 14000)
	prompts, err := assessmentPrompts("reviewer", bundle, diff)
	if err != nil || len(prompts) != 2 {
		t.Fatalf("assessmentPrompts() chunks = %d, error = %v", len(prompts), err)
	}
	var joined strings.Builder
	for index, prompt := range prompts {
		if len(prompt) > maxAssessmentPromptBytes || !strings.Contains(prompt, "diff chunk "+strconv.Itoa(index+1)+" of 2") {
			t.Fatalf("prompt %d is invalid", index+1)
		}
		parts := strings.SplitN(prompt, "DIFF CHUNK:\n", 2)
		if len(parts) != 2 {
			t.Fatalf("prompt %d does not contain a diff chunk", index+1)
		}
		joined.WriteString(parts[1])
	}
	if joined.String() != diff {
		t.Fatal("diff chunks did not preserve the complete diff")
	}
}

func TestSplitAssessmentDiffSplitsOversizedFileWithoutLoss(t *testing.T) {
	diff := testDiffSection("large", maxAssessmentPromptBytes*2)
	chunks, err := splitAssessmentDiff(diff, maxAssessmentPromptBytes)
	if err != nil || len(chunks) < 2 {
		t.Fatalf("splitAssessmentDiff() chunks = %d, error = %v", len(chunks), err)
	}
	for _, chunk := range chunks {
		if len(chunk) > maxAssessmentPromptBytes {
			t.Fatalf("chunk exceeds limit: %d", len(chunk))
		}
	}
	if strings.Join(chunks, "") != diff {
		t.Fatal("oversized diff was not preserved")
	}
}

func TestGenerateAssessmentForDiffAggregatesChunkFindings(t *testing.T) {
	bundle := assessmentTestBundle()
	diff := testDiffSection("first", 14000) + testDiffSection("second", 14000)
	output := filepath.Join(t.TempDir(), "assessment.json")
	request := AssessmentRequest{Role: "reviewer", Model: "local-model", Policy: assessmentTestPolicy(), Bundle: bundle, OutputPath: output}
	calls := 0
	assessment, err := generateAssessmentForDiff(request, diff, func(_ *http.Client, _ ollama.Policy, received ollama.Request) (string, error) {
		calls++
		if len(received.Input) > maxAssessmentPromptBytes {
			t.Fatalf("received prompt exceeds limit: %d", len(received.Input))
		}
		if received.Think == nil || *received.Think {
			t.Fatalf("assessment thinking preference = %#v", received.Think)
		}
		if received.Format != "" {
			t.Fatalf("assessment format = %q", received.Format)
		}
		if calls == 1 {
			return "R-1|minor|first:1|changed line is a test fixture|example", nil
		}
		return "R-1|minor|second:1|changed line is a test fixture|example", nil
	})
	if err != nil || calls != 2 || len(assessment.Findings) != 2 || assessment.Findings[0].ID != "C1-R-1" || assessment.Findings[1].ID != "C2-R-1" {
		t.Fatalf("generateAssessmentForDiff() = %#v, calls = %d, error = %v", assessment, calls, err)
	}
	if _, err := LoadAssessment(output); err != nil {
		t.Fatalf("saved aggregate assessment: %v", err)
	}
	if _, err := LoadAssessmentEnvelope(output + ".envelope.json"); err != nil {
		t.Fatalf("saved provenance envelope: %v", err)
	}
}

func TestGenerateAssessmentForDiffSavesMalformedResponse(t *testing.T) {
	output := filepath.Join(t.TempDir(), "assessment.json")
	request := AssessmentRequest{Role: "reviewer", Model: "local-model", Policy: assessmentTestPolicy(), Bundle: assessmentTestBundle(), OutputPath: output}
	_, err := generateAssessmentForDiff(request, "diff --git a/file b/file\n", func(_ *http.Client, _ ollama.Policy, _ ollama.Request) (string, error) {
		return "invalid assessment response", nil
	})
	if err == nil || !strings.Contains(err.Error(), "raw response saved to") {
		t.Fatalf("generateAssessmentForDiff() error = %v", err)
	}
	rawPath := output + ".raw"
	data, readErr := os.ReadFile(rawPath)
	if readErr != nil || string(data) != "invalid assessment response" {
		t.Fatalf("raw response = %q, read error = %v", data, readErr)
	}
	info, statErr := os.Stat(rawPath)
	if statErr != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("raw response mode = %v, stat error = %v", info.Mode(), statErr)
	}
}

func TestGenerateAssessmentForDiffRetriesMalformedResponse(t *testing.T) {
	output := filepath.Join(t.TempDir(), "assessment.json")
	request := AssessmentRequest{Role: "reviewer", Model: "local-model", Policy: assessmentTestPolicy(), Bundle: assessmentTestBundle(), OutputPath: output}
	calls := 0
	assessment, err := generateAssessmentForDiff(request, "diff --git a/file b/file\n", func(_ *http.Client, _ ollama.Policy, _ ollama.Request) (string, error) {
		calls++
		if calls == 1 {
			return "invalid assessment response", nil
		}
		return "NONE", nil
	})
	if err != nil || calls != 2 || len(assessment.Findings) != 0 {
		t.Fatalf("generateAssessmentForDiff retry = %#v, calls = %d, error = %v", assessment, calls, err)
	}
	if _, err := os.Stat(output + ".raw"); err != nil {
		t.Fatalf("first malformed attempt was not preserved: %v", err)
	}
	if _, err := LoadAssessmentEnvelope(output + ".envelope.json"); err != nil {
		t.Fatalf("retry did not produce provenance envelope: %v", err)
	}
}

func assessmentTestBundle() TaskBundle {
	return TaskBundle{WorkItem: workitem.Item{Source: workitem.Source{SubtaskKey: "REK-3"}, Scope: workitem.Scope{Phase: "Jira planning", ChangeClass: "high-risk", AllowedPaths: []string{"internal/implementation"}, NonGoals: []string{"no Jira writes"}, TechnicalAcceptanceCriteria: []string{"require evidence"}, ValidationPlan: []string{"make test"}}}, AllowedPaths: []string{"internal/implementation"}, Commands: []string{"make test"}, ADR: "no ADR", Guidance: "review guidance", SourceEvidence: jira.OfflineExportEvidence{EnvelopeDigest: "sha256:export", IssuerKeyID: "issuer", CapturedAt: "2026-07-14T00:00:00Z"}}
}

func assessmentTestPolicy() ollama.Policy {
	return ollama.Policy{Endpoint: "http://127.0.0.1:11434", RequestTimeoutSeconds: 60, Fingerprint: "sha256:test-policy", Models: []ollama.Model{{Name: "local-model", ID: "sha256:local-model"}}}
}

func testDiffSection(name string, bodyBytes int) string {
	return "diff --git a/" + name + " b/" + name + "\nindex 0000000..1111111 100644\n--- a/" + name + "\n+++ b/" + name + "\n@@ -0,0 +1 @@\n+" + strings.Repeat("x", bodyBytes) + "\n"
}
