package ticketplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	digestPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	headingPattern    = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t]*#*[ \t]*$`)
	traceTokenPattern = regexp.MustCompile(`[A-Za-z0-9]{3,}`)
	traceStopWords    = map[string]bool{
		"and": true, "are": true, "for": true, "from": true, "has": true, "have": true,
		"into": true, "its": true, "not": true, "that": true, "the": true, "this": true,
		"with": true, "source": true, "evidence": true, "section": true, "document": true,
		"details": true, "information": true, "requirement": true, "requirements": true,
		"ticket": true, "tickets": true, "plan": true, "plans": true,
	}
)

type Plan struct {
	FormatVersion int       `json:"format_version"`
	Status        string    `json:"status"`
	Sources       Sources   `json:"sources"`
	Story         Story     `json:"story"`
	Subtasks      []Subtask `json:"subtasks"`
}

type Sources struct {
	PRD     Source `json:"prd"`
	Spec    Source `json:"spec"`
	Roadmap Source `json:"roadmap"`
}

type Source struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type Story struct {
	Summary            string   `json:"summary"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Traceability       TraceMap `json:"traceability"`
}

type ReviewBudget struct {
	MaxChangedFiles int      `json:"max_changed_files"`
	MaxChangedLines int      `json:"max_changed_lines"`
	Components      []string `json:"components"`
}

type Subtask struct {
	ID                 string       `json:"id"`
	Summary            string       `json:"summary"`
	Phase              string       `json:"phase"`
	ChangeClass        string       `json:"change_class"`
	ReviewBudget       ReviewBudget `json:"review_budget"`
	Scope              string       `json:"scope"`
	NonGoals           []string     `json:"non_goals"`
	AcceptanceCriteria []string     `json:"acceptance_criteria"`
	ValidationPlan     []string     `json:"validation_plan"`
	AllowedPaths       []string     `json:"allowed_paths"`
	ADR                string       `json:"adr"`
	Dependencies       []string     `json:"dependencies"`
	Traceability       TraceMap     `json:"traceability"`
}

type Reference struct {
	Source    string `json:"source"`
	Section   string `json:"section"`
	Excerpt   string `json:"excerpt"`
	Authority string `json:"authority,omitempty"`
}

type TraceMap map[string][]Reference

// VerifiedFile is a file whose opened descriptor was verified to resolve inside
// the repository root. Data and Digest always describe that same descriptor.
type VerifiedFile struct {
	Data          []byte
	Digest        string
	CanonicalPath string
	RelativePath  string
}

func Load(path string) (Plan, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Plan{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("parse ticket plan: %w", err)
	}
	if decoder.More() {
		return Plan{}, fmt.Errorf("parse ticket plan: multiple JSON values")
	}
	return plan, nil
}

func FileDigest(path string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

func (p Plan) Validate() []string {
	var issues []string
	if p.FormatVersion != 1 {
		issues = append(issues, "format_version must be 1")
	}
	if !oneOf(p.Status, "draft", "needs-remediation", "ready-for-approval", "approved", "escalated") {
		issues = append(issues, "status is invalid")
	}
	for name, source := range p.sourceMap() {
		if source.Path == "" || !digestPattern.MatchString(source.Digest) {
			issues = append(issues, name+" source must include path and sha256 digest")
		}
	}
	if p.Story.Summary == "" || p.Story.Description == "" || len(p.Story.AcceptanceCriteria) == 0 || !validTrace(p.Story.Traceability, "summary", "description", "acceptance_criteria") {
		issues = append(issues, "story is incomplete")
	}
	if len(p.Subtasks) == 0 {
		issues = append(issues, "at least one subtask is required")
	}
	seen, graph := map[string]bool{}, map[string][]string{}
	for _, subtask := range p.Subtasks {
		if subtask.ID == "" || seen[subtask.ID] {
			issues = append(issues, "subtask IDs must be unique and nonempty")
		}
		seen[subtask.ID] = true
		if !validSubtask(subtask) {
			issues = append(issues, "subtask "+subtask.ID+" is incomplete")
		}
		for _, allowedPath := range subtask.AllowedPaths {
			if !validAllowedPath(allowedPath) {
				issues = append(issues, "subtask "+subtask.ID+" has invalid allowed path "+allowedPath)
			}
		}
		graph[subtask.ID] = subtask.Dependencies
	}
	for id, dependencies := range graph {
		for _, dependency := range dependencies {
			if dependency == id || !seen[dependency] {
				issues = append(issues, "subtask "+id+" has invalid dependency "+dependency)
			}
		}
	}
	if hasCycle(graph) {
		issues = append(issues, "subtask dependencies contain a cycle")
	}
	return issues
}

func (p Plan) ValidateAgainst(repoRoot string) []string {
	issues := p.Validate()
	sections := map[string]map[string]string{}
	for name, source := range p.sourceMap() {
		file, err := ReadVerifiedSource(repoRoot, source.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				issues = append(issues, name+" source is unavailable")
			} else {
				issues = append(issues, name+" source path is outside repository root")
			}
			continue
		}
		if file.Digest != source.Digest {
			issues = append(issues, name+" source digest does not match current file")
		}
		sections[name] = markdownSections(file.Data)
	}
	issues = append(issues, validateTraceability("story", p.Story.Traceability, sections, storyFieldValues(p.Story), "summary", "description", "acceptance_criteria")...)
	for _, subtask := range p.Subtasks {
		issues = append(issues, validateTraceability("subtask "+subtask.ID, subtask.Traceability, sections, subtaskFieldValues(subtask), subtaskTraceFields...)...)
		if !validADR(repoRoot, subtask.ADR) {
			issues = append(issues, "subtask "+subtask.ID+" ADR reference or rationale is invalid")
		}
	}
	return issues
}

var subtaskTraceFields = []string{"summary", "phase", "change_class", "review_budget", "scope", "non_goals", "acceptance_criteria", "validation_plan", "allowed_paths", "adr", "dependencies"}

func (p Plan) sourceMap() map[string]Source {
	return map[string]Source{"prd": p.Sources.PRD, "spec": p.Sources.Spec, "roadmap": p.Sources.Roadmap}
}

func validSubtask(subtask Subtask) bool {
	return subtask.Summary != "" && subtask.Phase != "" && oneOf(subtask.ChangeClass, "trivial", "standard", "high-risk") &&
		subtask.ReviewBudget.MaxChangedFiles > 0 && subtask.ReviewBudget.MaxChangedLines > 0 && len(subtask.ReviewBudget.Components) > 0 &&
		subtask.Scope != "" && len(subtask.NonGoals) > 0 && len(subtask.AcceptanceCriteria) > 0 && len(subtask.ValidationPlan) > 0 &&
		len(subtask.AllowedPaths) > 0 && subtask.ADR != "" && validTrace(subtask.Traceability, subtaskTraceFields...)
}

func validAllowedPath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "*?[") {
		return false
	}
	for _, segment := range strings.Split(filepath.ToSlash(value), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	cleaned := filepath.Clean(filepath.FromSlash(value))
	if cleaned != filepath.FromSlash(value) || cleaned == "." || cleaned == ".." {
		return false
	}
	return !strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
}

func validADR(repoRoot, value string) bool {
	const noADR = "No ADR needed: "
	if strings.HasPrefix(value, noADR) {
		rationale := strings.TrimSpace(strings.TrimPrefix(value, noADR))
		return len(rationale) >= 10 && !oneOf(strings.ToLower(rationale), "unknown", "none", "n/a", "not applicable")
	}
	path := filepath.FromSlash(value)
	cleaned := filepath.Clean(path)
	prefix := filepath.Join("docs", "decisions") + string(filepath.Separator)
	if filepath.IsAbs(path) || cleaned != path || !strings.HasPrefix(cleaned, prefix) || filepath.Ext(cleaned) != ".md" {
		return false
	}
	decisionRoot, err := filepath.EvalSymlinks(filepath.Join(repoRoot, "docs", "decisions"))
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(repoRoot, cleaned))
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(decisionRoot, resolved)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func validTrace(trace TraceMap, fields ...string) bool {
	for _, field := range fields {
		refs := trace[field]
		if len(refs) == 0 {
			return false
		}
		for _, ref := range refs {
			if ref.Authority != "" && (ref.Authority != "assignment" || !isAssignmentTraceField(field)) {
				return false
			}
			if !oneOf(ref.Source, "prd", "spec", "roadmap") || strings.TrimSpace(ref.Section) == "" || len(strings.TrimSpace(ref.Excerpt)) < 10 || !hasSubstantiveTokens(ref.Excerpt) {
				return false
			}
		}
	}
	return true
}

func validateTraceability(subject string, trace TraceMap, sections map[string]map[string]string, fields map[string][]string, required ...string) []string {
	var issues []string
	for _, field := range required {
		if field == "dependencies" && len(fields[field]) == 0 {
			continue
		}
		for _, value := range fields[field] {
			matched := false
			for _, ref := range trace[field] {
				if ref.Authority == "assignment" && isAssignmentTraceField(field) {
					matched = true
					break
				}
				section, ok := sections[ref.Source][strings.TrimSpace(ref.Section)]
				if !ok {
					continue
				}
				// Trace excerpts are manager-generated presentation data. Validate
				// against the canonical verified source section instead, so harmless
				// excerpt formatting cannot make the plan non-deterministic.
				if traceSupportsField(field, section, value) {
					matched = true
					break
				}
			}
			if !matched {
				issues = append(issues, subject+" "+field+" traceability lacks matching source evidence")
				break
			}
		}
		if len(fields[field]) == 0 {
			issues = append(issues, subject+" "+field+" traceability lacks matching source evidence")
		}
	}
	return issues
}

func traceSupportsField(field, excerpt, value string) bool {
	if field == "dependencies" && value == "" {
		return true
	}
	if field == "allowed_paths" {
		return containsNormalizedPath(excerpt, value)
	}
	if field == "phase" {
		return containsNormalizedPhrase(excerpt, value)
	}
	if isListTraceField(field) {
		if field == "validation_plan" {
			overlap := substantiveTokenOverlapCount(excerpt, value)
			return overlap >= 2
		}
		if field == "review_budget" && isPositiveInteger(value) {
			return containsWholeNumber(excerpt, value)
		}
		return containsNormalizedPhrase(excerpt, value)
	}
	if field == "change_class" {
		return hasSubstantiveTokenOverlap(excerpt, value)
	}
	valueTokens := substantiveTokens(value)
	overlap := substantiveTokenOverlapCount(excerpt, value)
	return overlap >= 2 || len(valueTokens) == 1 && overlap == 1
}

func isListTraceField(field string) bool {
	return oneOf(field, "acceptance_criteria", "review_budget", "non_goals", "validation_plan", "dependencies")
}

func isAssignmentTraceField(field string) bool {
	return oneOf(field, "phase", "change_class", "review_budget", "allowed_paths", "dependencies", "adr")
}

func containsNormalizedPhrase(excerpt, value string) bool {
	normalize := func(input string) []string {
		return tracePhraseTokenPattern.FindAllString(strings.ToLower(input), -1)
	}
	needle := normalize(value)
	if len(needle) == 0 {
		return false
	}
	haystack := normalize(excerpt)
	for start := 0; start+len(needle) <= len(haystack); start++ {
		matched := true
		for index := range needle {
			if haystack[start+index] != needle[index] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

var tracePhraseTokenPattern = regexp.MustCompile(`[A-Za-z0-9]+`)

func isPositiveInteger(value string) bool {
	parsed, err := strconv.Atoi(value)
	return err == nil && parsed > 0
}

func containsWholeNumber(excerpt, value string) bool {
	for _, token := range tracePhraseTokenPattern.FindAllString(excerpt, -1) {
		if token == value {
			return true
		}
	}
	return false
}

func containsNormalizedPath(excerpt, value string) bool {
	path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if !validAllowedPath(path) {
		return false
	}
	text := filepath.ToSlash(excerpt)
	for start := 0; ; {
		index := strings.Index(text[start:], path)
		if index < 0 {
			return false
		}
		index += start
		end := index + len(path)
		if pathBoundary(text, index-1, false) && (end < len(text) && text[end] == '/' || pathBoundary(text, end, true)) {
			return true
		}
		start = end
	}
}

func pathBoundary(value string, index int, afterPath bool) bool {
	if index < 0 || index >= len(value) {
		return true
	}
	if value[index] == '.' {
		adjacent := index - 1
		if afterPath {
			adjacent = index + 1
		}
		return adjacent < 0 || adjacent >= len(value) || !asciiAlphaNumeric(value[adjacent])
	}
	return !((value[index] >= 'a' && value[index] <= 'z') ||
		(value[index] >= 'A' && value[index] <= 'Z') ||
		(value[index] >= '0' && value[index] <= '9') ||
		value[index] == '_' || value[index] == '-' || value[index] == '/')
}

func asciiAlphaNumeric(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

func hasSubstantiveTokenOverlap(left, right string) bool {
	return substantiveTokenOverlapCount(left, right) > 0
}

func substantiveTokenOverlapCount(left, right string) int {
	rightTokens := substantiveTokens(right)
	count := 0
	for token := range substantiveTokens(left) {
		if rightTokens[token] {
			count++
		}
	}
	return count
}

func hasSubstantiveTokens(value string) bool {
	return len(substantiveTokens(value)) != 0
}

func substantiveTokens(value string) map[string]bool {
	tokens := map[string]bool{}
	for _, token := range traceTokenPattern.FindAllString(strings.ToLower(value), -1) {
		if !traceStopWords[token] {
			tokens[token] = true
		}
	}
	return tokens
}

func markdownSections(data []byte) map[string]string {
	result := map[string]string{}
	matches := headingPattern.FindAllSubmatchIndex(data, -1)
	for index, match := range matches {
		end := len(data)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		result[strings.TrimSpace(string(data[match[2]:match[3]]))] = string(data[match[0]:end])
	}
	return result
}

func storyFieldValues(story Story) map[string][]string {
	return map[string][]string{
		"summary":             {story.Summary},
		"description":         {story.Description},
		"acceptance_criteria": story.AcceptanceCriteria,
	}
}

func subtaskFieldValues(subtask Subtask) map[string][]string {
	return map[string][]string{
		"summary":             {subtask.Summary},
		"phase":               {subtask.Phase},
		"change_class":        {subtask.ChangeClass},
		"review_budget":       reviewBudgetTraceValues(subtask.ReviewBudget),
		"scope":               {subtask.Scope},
		"non_goals":           subtask.NonGoals,
		"acceptance_criteria": subtask.AcceptanceCriteria,
		"validation_plan":     subtask.ValidationPlan,
		"allowed_paths":       subtask.AllowedPaths,
		"adr":                 {subtask.ADR},
		"dependencies":        subtask.Dependencies,
	}
}

func reviewBudgetTraceValues(budget ReviewBudget) []string {
	values := []string{strconv.Itoa(budget.MaxChangedFiles), strconv.Itoa(budget.MaxChangedLines)}
	return append(values, budget.Components...)
}

func resolvePath(repoRoot, path string) (string, error) {
	file, err := ReadVerifiedSource(repoRoot, path)
	if err != nil {
		return "", err
	}
	return file.CanonicalPath, nil
}

// ReadVerifiedSource reads a clean repository-relative source path through one
// descriptor. The post-open identity check detects symlink replacement without
// reopening the path used for digesting or parsing.
func ReadVerifiedSource(repoRoot, path string) (VerifiedFile, error) {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(filepath.FromSlash(path)) != filepath.FromSlash(path) {
		return VerifiedFile{}, fmt.Errorf("source path must be clean and relative")
	}
	cleanPath := filepath.Clean(filepath.FromSlash(path))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return VerifiedFile{}, fmt.Errorf("source path escapes repository root")
	}
	return readVerifiedFile(repoRoot, cleanPath)
}

// ReadVerifiedFile verifies an arbitrary source path against repoRoot. It is
// used when agentplan receives approved paths as absolute filesystem paths.
func ReadVerifiedFile(repoRoot, path string) (VerifiedFile, error) {
	if path == "" {
		return VerifiedFile{}, fmt.Errorf("source path is empty")
	}
	return readVerifiedFile(repoRoot, filepath.Clean(path))
}

func readVerifiedFile(repoRoot, path string) (VerifiedFile, error) {
	canonicalRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return VerifiedFile{}, fmt.Errorf("resolve repository root: %w", err)
	}
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(canonicalRoot, candidate)
	}
	file, err := os.Open(candidate)
	if err != nil {
		if os.IsNotExist(err) && missingPathEscapesRoot(canonicalRoot, candidate) {
			return VerifiedFile{}, fmt.Errorf("source path escapes repository root")
		}
		return VerifiedFile{}, fmt.Errorf("open source: %w", err)
	}
	defer file.Close()
	canonicalPath, relative, err := verifyOpenFile(canonicalRoot, candidate, file)
	if err != nil {
		return VerifiedFile{}, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return VerifiedFile{}, fmt.Errorf("read source: %w", err)
	}
	return VerifiedFile{Data: data, Digest: digestBytes(data), CanonicalPath: canonicalPath, RelativePath: filepath.ToSlash(relative)}, nil
}

// missingPathEscapesRoot is used only after opening a path failed. It preserves
// the distinction between a missing in-root source and a missing descendant of
// an external symlink without introducing a validate-then-open sequence.
func missingPathEscapesRoot(canonicalRoot, candidate string) bool {
	for ancestor := candidate; ; ancestor = filepath.Dir(ancestor) {
		resolved, err := filepath.EvalSymlinks(ancestor)
		if err == nil {
			relative, relErr := filepath.Rel(canonicalRoot, resolved)
			return relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))
		}
		if !os.IsNotExist(err) {
			return false
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return false
		}
	}
}

func verifyOpenFile(canonicalRoot, candidate string, file *os.File) (string, string, error) {
	opened, err := file.Stat()
	if err != nil {
		return "", "", fmt.Errorf("stat opened source: %w", err)
	}
	if !opened.Mode().IsRegular() {
		return "", "", fmt.Errorf("source is not a regular file")
	}
	canonicalPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", fmt.Errorf("resolve opened source: %w", err)
	}
	current, err := os.Stat(canonicalPath)
	if err != nil {
		return "", "", fmt.Errorf("stat canonical source: %w", err)
	}
	if !os.SameFile(opened, current) {
		return "", "", fmt.Errorf("source changed while opening")
	}
	relative, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("source path escapes repository root")
	}
	return canonicalPath, relative, nil
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func hasCycle(graph map[string][]string) bool {
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if done[id] {
			return false
		}
		visiting[id] = true
		for _, dependency := range graph[id] {
			if visit(dependency) {
				return true
			}
		}
		visiting[id] = false
		done[id] = true
		return false
	}
	for id := range graph {
		if visit(id) {
			return true
		}
	}
	return false
}
