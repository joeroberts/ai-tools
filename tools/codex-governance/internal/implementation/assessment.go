package implementation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"codex-governance/internal/ollama"
)

// maxAssessmentPromptBytes keeps each local-model review request well below
// the measured 32K runtime context while retaining the complete diff across
// deterministic chunks.
const maxAssessmentPromptBytes = 24 * 1024

const maxAssessmentAttempts = 3

type AssessmentRequest struct {
	Role       string
	Model      string
	Policy     ollama.Policy
	Bundle     TaskBundle
	Worktree   string
	OutputPath string
}

// assessmentContext contains the approved review contract without duplicating
// the full Jira export and signed envelope already verified at preflight.
type assessmentContext struct {
	WorkItem                    string                      `json:"work_item"`
	Phase                       string                      `json:"phase"`
	ChangeClass                 string                      `json:"change_class"`
	AllowedPaths                []string                    `json:"allowed_paths"`
	NonGoals                    []string                    `json:"non_goals"`
	TechnicalAcceptanceCriteria []string                    `json:"technical_acceptance_criteria"`
	ValidationPlan              []string                    `json:"validation_plan"`
	ADR                         string                      `json:"adr"`
	Guidance                    string                      `json:"guidance"`
	SourceEvidence              sourceEvidenceForAssessment `json:"source_evidence"`
}

type sourceEvidenceForAssessment struct {
	EnvelopeDigest string `json:"envelope_digest"`
	IssuerKeyID    string `json:"issuer_key_id"`
	CapturedAt     string `json:"captured_at"`
}

// GenerateAssessment invokes only the governed local gateway. The model gets
// a bounded task bundle and diff and cannot receive remote credentials.
func GenerateAssessment(request AssessmentRequest) (Assessment, error) {
	if request.Role != "reviewer" && request.Role != "verifier" || request.Model == "" || request.OutputPath == "" {
		return Assessment{}, fmt.Errorf("assessment request is incomplete")
	}
	diff, err := workingDiff(request.Worktree)
	if err != nil {
		return Assessment{}, err
	}
	return generateAssessmentForDiff(request, diff, ollama.Run)
}

func generateAssessmentForDiff(request AssessmentRequest, diff string, run func(*http.Client, ollama.Policy, ollama.Request) (string, error)) (Assessment, error) {
	modelID, err := localAssessmentModelID(request.Policy, request.Model)
	if err != nil {
		return Assessment{}, err
	}
	prompts, err := assessmentPrompts(request.Role, request.Bundle, diff)
	if err != nil {
		return Assessment{}, err
	}
	startedAt := time.Now().UTC()
	var combined Assessment
	rawOutputs := make([]string, 0, len(prompts))
	think := false
	for index, prompt := range prompts {
		var assessment Assessment
		var output string
		accepted := false
		candidatePrompt := prompt
		for attempt := 1; attempt <= maxAssessmentAttempts; attempt++ {
			output, err = run(ollama.Client(request.Policy), request.Policy, ollama.Request{Model: request.Model, Role: request.Role, TaskType: "implementation-review", Input: []byte(candidatePrompt), Think: &think})
			if err != nil {
				return Assessment{}, fmt.Errorf("assess diff chunk %d of %d attempt %d: %w", index+1, len(prompts), attempt, err)
			}
			assessment, err = parseAssessmentForChunk([]byte(output), diffChunk(prompt))
			if err == nil {
				accepted = true
				rawOutputs = append(rawOutputs, output)
				break
			}
			rawPath, saveErr := saveRawAssessmentAttempt(request.OutputPath, attempt, []byte(output))
			if saveErr != nil {
				return Assessment{}, fmt.Errorf("parse diff chunk %d of %d attempt %d: %w (save raw response: %v)", index+1, len(prompts), attempt, err, saveErr)
			}
			if attempt == maxAssessmentAttempts {
				return Assessment{}, fmt.Errorf("parse diff chunk %d of %d exhausted %d attempts: %w (raw response saved to %s)", index+1, len(prompts), maxAssessmentAttempts, err, rawPath)
			}
			candidatePrompt = prompt + "\n\nCORRECTION REQUIRED: Your previous response was invalid: " + err.Error() + ". Return only the required line protocol or NONE."
		}
		if !accepted {
			return Assessment{}, fmt.Errorf("assessment retry state is invalid")
		}
		for findingIndex := range assessment.Findings {
			assessment.Findings[findingIndex].ID = fmt.Sprintf("C%d-%s", index+1, assessment.Findings[findingIndex].ID)
		}
		combined.Findings = append(combined.Findings, assessment.Findings...)
	}
	if err := SaveAssessment(request.OutputPath, combined); err != nil {
		return Assessment{}, err
	}
	rawPath := request.OutputPath + ".raw.valid"
	rawData := []byte(strings.Join(rawOutputs, "\n--- assessment chunk ---\n"))
	if err := writeAssessmentArtifact(rawPath, rawData); err != nil {
		return Assessment{}, err
	}
	findings, err := os.ReadFile(filepath.Clean(request.OutputPath))
	if err != nil {
		return Assessment{}, err
	}
	promptData := []byte(strings.Join(prompts, "\n--- assessment prompt ---\n"))
	envelope := AssessmentEnvelope{
		FormatVersion: 1, Provider: "local", Role: request.Role, ModelName: request.Model, ModelID: modelID,
		PolicyDigest: request.Policy.Fingerprint, DiffDigest: digestBytes([]byte(diff)), PromptDigest: digestBytes(promptData),
		RawOutputPath: rawPath, RawOutputDigest: digestBytes(rawData), FindingsPath: request.OutputPath, FindingsDigest: digestBytes(findings),
		StartedAt: startedAt, CompletedAt: time.Now().UTC(),
	}
	if err := SaveAssessmentEnvelope(request.OutputPath+".envelope.json", envelope); err != nil {
		return Assessment{}, err
	}
	return combined, nil
}

func saveRawAssessmentAttempt(outputPath string, attempt int, response []byte) (string, error) {
	if attempt == 1 {
		return SaveRawAssessment(outputPath, response)
	}
	path := fmt.Sprintf("%s.raw.%d", outputPath, attempt)
	if err := writeAssessmentArtifact(path, response); err != nil {
		return "", err
	}
	return path, nil
}

func localAssessmentModelID(policy ollama.Policy, model string) (string, error) {
	for _, configured := range policy.Models {
		if configured.Name == model && configured.ID != "" {
			return configured.ID, nil
		}
	}
	return "", fmt.Errorf("assessment model is not allowlisted with an immutable ID")
}

func writeAssessmentArtifact(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Clean(path)); err == nil {
		return fmt.Errorf("refusing to overwrite assessment artifact")
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(filepath.Clean(path), data, 0o600)
}

func workingDiff(worktree string) (string, error) {
	command := exec.Command("git", "diff", "HEAD")
	command.Dir = filepath.Clean(worktree)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("read implementation diff: %w", err)
	}
	if len(output) > 256*1024 {
		return "", fmt.Errorf("implementation diff exceeds assessment limit")
	}
	return string(output), nil
}

func assessmentPrompt(role string, bundle TaskBundle, diff string) (string, error) {
	return assessmentPromptForChunk(role, bundle, diff, 1, 1)
}

func assessmentPrompts(role string, bundle TaskBundle, diff string) ([]string, error) {
	base, err := assessmentPromptForChunk(role, bundle, "", 1, 1)
	if err != nil {
		return nil, err
	}
	if len(base) >= maxAssessmentPromptBytes {
		return nil, fmt.Errorf("assessment context exceeds chunk limit")
	}
	chunks, err := splitAssessmentDiff(diff, maxAssessmentPromptBytes-len(base))
	if err != nil {
		return nil, err
	}
	prompts := make([]string, 0, len(chunks))
	for index, chunk := range chunks {
		prompt, err := assessmentPromptForChunk(role, bundle, chunk, index+1, len(chunks))
		if err != nil {
			return nil, err
		}
		if len(prompt) > maxAssessmentPromptBytes {
			return nil, fmt.Errorf("assessment prompt chunk %d exceeds chunk limit", index+1)
		}
		prompts = append(prompts, prompt)
	}
	return prompts, nil
}

func assessmentPromptForChunk(role string, bundle TaskBundle, diff string, chunk, total int) (string, error) {
	context := assessmentContext{
		WorkItem:                    bundle.WorkItem.Source.SubtaskKey,
		Phase:                       bundle.WorkItem.Scope.Phase,
		ChangeClass:                 bundle.WorkItem.Scope.ChangeClass,
		AllowedPaths:                bundle.AllowedPaths,
		NonGoals:                    bundle.WorkItem.Scope.NonGoals,
		TechnicalAcceptanceCriteria: bundle.WorkItem.Scope.TechnicalAcceptanceCriteria,
		ValidationPlan:              bundle.Commands,
		ADR:                         bundle.ADR,
		Guidance:                    bundle.Guidance,
		SourceEvidence: sourceEvidenceForAssessment{
			EnvelopeDigest: bundle.SourceEvidence.EnvelopeDigest,
			IssuerKeyID:    bundle.SourceEvidence.IssuerKeyID,
			CapturedAt:     bundle.SourceEvidence.CapturedAt,
		},
	}
	data, err := json.Marshal(context)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("You are an independent %s. Assess this implementation diff chunk %d of %d only against the approved review context. This is not the complete diff: review every supplied line, but do not report missing implementation or an out-of-scope change solely because it is absent from this chunk. Files, package-local symbols, and imports absent from this chunk may exist elsewhere in the repository; their absence is never evidence of a compile failure, typo, or defect. Report a finding only when the supplied lines directly demonstrate a concrete defect. If a claim depends on omitted code, return NONE for that claim. Do not edit files or contact external systems.\n\nReturn only an assessment record in this exact line protocol:\nID|severity|path:line|observable failure condition|summary\nThe separator is the literal vertical-bar character |. Every finding must cite a changed added line in this diff chunk and state the concrete failure condition directly evidenced by that line. Copy path exactly from a DIFF CHUNK header, relative to the candidate root; never prefix it with `tools/codex-governance/` or another repository path. Return exactly NONE when there are no findings. NONE is valid only as the complete response, never as a field value. Emit one finding per line and do not emit a header row. Do not use Markdown, JSON, headings, surrounding prose, vertical bars in a field, or line breaks in a field. Use backticks or single quotes rather than double quotes in text. Severity must be blocking, important, minor, or informational.\n\nREVIEW CONTEXT:\n%s\n\nDIFF CHUNK:\n%s", role, chunk, total, data, diff), nil
}

func splitAssessmentDiff(diff string, maxBytes int) ([]string, error) {
	if maxBytes < 1 {
		return nil, fmt.Errorf("assessment diff chunk limit is invalid")
	}
	sections := diffSections(diff)
	var chunks []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
	}
	for _, section := range sections {
		if len(section) <= maxBytes {
			if current.Len() > 0 && current.Len()+len(section) > maxBytes {
				flush()
			}
			current.WriteString(section)
			continue
		}
		flush()
		for _, part := range splitAssessmentSection(section, maxBytes) {
			chunks = append(chunks, part)
		}
	}
	flush()
	if len(chunks) == 0 {
		return []string{""}, nil
	}
	return chunks, nil
}

func diffSections(diff string) []string {
	starts := []int{0}
	for offset := 1; ; {
		index := strings.Index(diff[offset:], "\ndiff --git ")
		if index < 0 {
			break
		}
		start := offset + index + 1
		starts = append(starts, start)
		offset = start + len("diff --git ")
	}
	sections := make([]string, 0, len(starts))
	for index, start := range starts {
		end := len(diff)
		if index+1 < len(starts) {
			end = starts[index+1]
		}
		sections = append(sections, diff[start:end])
	}
	return sections
}

func splitAssessmentSection(section string, maxBytes int) []string {
	lines := strings.SplitAfter(section, "\n")
	parts := make([]string, 0, len(lines))
	var current strings.Builder
	for _, line := range lines {
		if len(line) > maxBytes {
			for len(line) > maxBytes {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
				parts = append(parts, line[:maxBytes])
				line = line[maxBytes:]
			}
		}
		if current.Len() > 0 && current.Len()+len(line) > maxBytes {
			parts = append(parts, current.String())
			current.Reset()
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func parseAssessment(data []byte) (Assessment, error) {
	value := strings.TrimSpace(string(data))
	if value == "NONE" {
		return Assessment{}, nil
	}
	if !strings.HasPrefix(value, "{") && !strings.HasPrefix(value, "```") {
		return parseDelimitedAssessment(value)
	}
	return parseJSONAssessment(value)
}

func parseAssessmentForChunk(data []byte, diff string) (Assessment, error) {
	assessment, err := parseAssessment(data)
	if err != nil {
		return Assessment{}, err
	}
	locations := changedLocations(diff)
	for index := range assessment.Findings {
		finding := &assessment.Findings[index]
		finding.Location = normalizeAssessmentLocation(finding.Location)
		if finding.Location == "" || strings.EqualFold(finding.Summary, "none") || !locations[finding.Location] {
			return Assessment{}, fmt.Errorf("assessment finding %q lacks changed-line evidence", finding.ID)
		}
		if (finding.Severity == "blocking" || finding.Severity == "important") && (finding.Condition == "" || strings.EqualFold(finding.Condition, "none")) {
			return Assessment{}, fmt.Errorf("assessment finding %q lacks an observable failure condition", finding.ID)
		}
	}
	return assessment, nil
}

func normalizeAssessmentLocation(location string) string {
	return strings.TrimPrefix(location, "tools/codex-governance/")
}

func parseDelimitedAssessment(value string) (Assessment, error) {
	if value == "" {
		return Assessment{}, fmt.Errorf("assessment response is empty")
	}
	lines := strings.Split(value, "\n")
	if lines[0] == "ID|severity|path:line|observable failure condition|summary" || lines[0] == "ID|severity|summary" {
		lines = lines[1:]
	}
	if len(lines) == 0 {
		return Assessment{}, fmt.Errorf("assessment response contains only a header")
	}
	if len(lines) == 1 && lines[0] == "NONE" {
		return Assessment{}, nil
	}
	var assessment Assessment
	for index, line := range lines {
		if line == "NONE" {
			if index != len(lines)-1 || len(assessment.Findings) == 0 {
				return Assessment{}, fmt.Errorf("assessment line %d is invalid", index+1)
			}
			for _, finding := range assessment.Findings {
				if finding.Severity != "informational" {
					return Assessment{}, fmt.Errorf("assessment line %d is invalid", index+1)
				}
			}
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) == 4 && fields[3] == "" {
			fields = fields[:3]
		}
		if len(fields) == 6 && fields[5] == "" {
			fields = fields[:5]
		}
		if len(fields) == 5 && fields[0] != "" && fields[2] != "" && fields[3] != "" && fields[4] != "" && validSeverity(fields[1]) {
			assessment.Findings = append(assessment.Findings, Finding{ID: fields[0], Severity: fields[1], Location: fields[2], Condition: fields[3], Summary: fields[4]})
			continue
		}
		if len(fields) == 4 && fields[0] != "" && fields[1] == "informational" && fields[2] != "" && fields[3] != "" {
			assessment.Findings = append(assessment.Findings, Finding{ID: fields[0], Severity: fields[1], Location: fields[2], Summary: fields[3]})
			continue
		}
		if len(fields) != 3 || fields[0] == "" || fields[2] == "" || !validSeverity(fields[1]) {
			return Assessment{}, fmt.Errorf("assessment line %d is invalid", index+1)
		}
		assessment.Findings = append(assessment.Findings, Finding{ID: fields[0], Severity: fields[1], Summary: fields[2]})
	}
	return assessment, nil
}

func diffChunk(prompt string) string {
	parts := strings.SplitN(prompt, "DIFF CHUNK:\n", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func changedLocations(diff string) map[string]bool {
	locations := map[string]bool{}
	path, line := "", 0
	for _, value := range strings.Split(diff, "\n") {
		if strings.HasPrefix(value, "+++ b/") {
			path = normalizeAssessmentLocation(strings.TrimPrefix(value, "+++ b/"))
			continue
		}
		if strings.HasPrefix(value, "@@ ") {
			parts := strings.Fields(value)
			if len(parts) > 2 {
				start := strings.TrimPrefix(parts[2], "+")
				start = strings.SplitN(start, ",", 2)[0]
				line, _ = strconv.Atoi(start)
			}
			continue
		}
		if path == "" || line == 0 || strings.HasPrefix(value, "+++") || strings.HasPrefix(value, "---") {
			continue
		}
		if strings.HasPrefix(value, "+") {
			locations[fmt.Sprintf("%s:%d", path, line)] = true
			line++
		} else if !strings.HasPrefix(value, "-") {
			line++
		}
	}
	return locations
}

func parseJSONAssessment(value string) (Assessment, error) {
	data := []byte(repairAssessmentJSON(unfenceJSON(value)))
	var assessment Assessment
	if err := json.Unmarshal(data, &assessment); err != nil {
		return Assessment{}, fmt.Errorf("parse assessment: %w", err)
	}
	for _, finding := range assessment.Findings {
		if finding.ID == "" || finding.Summary == "" || !validSeverity(finding.Severity) {
			return Assessment{}, fmt.Errorf("assessment contains an invalid finding")
		}
	}
	return assessment, nil
}

// repairAssessmentJSON accepts one known local-model serialization defect: a
// valid top-level assessment object followed only by an extra array close.
// It deliberately rejects prose and every other trailing token.
func repairAssessmentJSON(value string) string {
	value = strings.TrimSpace(value)
	if json.Valid([]byte(value)) || !strings.HasPrefix(value, "{") || (!strings.HasSuffix(value, "]") && !strings.HasSuffix(value, "]}")) {
		return value
	}
	end, ok := jsonObjectEnd(value)
	trailing := strings.TrimSpace(value[end:])
	if !ok || (trailing != "]" && trailing != "]}") {
		return value
	}
	candidate := value[:end]
	if json.Valid([]byte(candidate)) {
		return candidate
	}
	return value
}

func jsonObjectEnd(value string) (int, bool) {
	depth := 0
	inString, escaped := false, false
	for index := 0; index < len(value); index++ {
		character := value[index]
		if inString {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}
		switch character {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index + 1, true
			}
		}
	}
	return 0, false
}

// unfenceJSON accepts the single Markdown code fence some local models add
// despite the JSON-only response contract. It does not accept surrounding
// prose or multiple fenced blocks.
func unfenceJSON(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"```json", "```JSON", "```"} {
		if !strings.HasPrefix(value, prefix) {
			continue
		}
		body := strings.TrimPrefix(value, prefix)
		if !strings.HasSuffix(body, "```") {
			return value
		}
		return strings.TrimSpace(strings.TrimSuffix(body, "```"))
	}
	return value
}
