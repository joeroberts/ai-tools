package ticketplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const AuthorityContractFormatVersion = 1

var contractEvidenceFields = map[string]bool{
	"story.summary": true, "story.description": true, "story.acceptance_criteria": true,
	"slices[].id": true, "slices[].phase": true, "slices[].change_class": true,
	"slices[].review_budget": true, "slices[].allowed_paths": true, "slices[].dependencies": true,
	"slices[].adr": true, "slices[].summary": true, "slices[].scope": true,
	"slices[].non_goals": true, "slices[].acceptance_criteria": true, "slices[].validation_plan": true,
}

// AuthorityContract is the durable, deterministic authority boundary used by
// ticket-plan generation and validation. Sources are recorded once and roles
// bind to their stable IDs so evidence cannot silently substitute a source.
type AuthorityContract struct {
	FormatVersion  int                `json:"format_version"`
	Sources        []ContractSource   `json:"sources"`
	Roles          SourceRoleBindings `json:"role_bindings"`
	Story          ContractStory      `json:"story"`
	Slices         []ContractSlice    `json:"slices"`
	NarrativeRules []NarrativeRule    `json:"manager_narrative_rules"`
	Evidence       []ContractEvidence `json:"evidence"`
}

type ContractSource struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type SourceRoleBindings struct {
	PRD     string `json:"prd"`
	Spec    string `json:"spec"`
	Roadmap string `json:"roadmap"`
}

type ContractStory struct {
	Summary            string   `json:"summary"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type ContractSlice struct {
	ID            string             `json:"id"`
	Assignment    SliceAssignment    `json:"assignment"`
	SourceDerived SliceSourceDerived `json:"source_derived"`
}

type SliceAssignment struct {
	Phase        string       `json:"phase"`
	ChangeClass  string       `json:"change_class"`
	ReviewBudget ReviewBudget `json:"review_budget"`
	AllowedPaths []string     `json:"allowed_paths"`
	Dependencies []string     `json:"dependencies"`
	ADR          string       `json:"adr"`
}

type SliceSourceDerived struct {
	Summary            string   `json:"summary"`
	Scope              string   `json:"scope"`
	NonGoals           []string `json:"non_goals"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ValidationPlan     []string `json:"validation_plan"`
}

type NarrativeRule struct {
	Field        string `json:"field"`
	RequiredRole string `json:"required_role"`
	MinLength    int    `json:"min_length"`
}

type ContractEvidence struct {
	Field   string `json:"field"`
	Role    string `json:"role"`
	Section string `json:"section"`
	Excerpt string `json:"excerpt"`
}

func (c AuthorityContract) Validate() error {
	if c.FormatVersion != AuthorityContractFormatVersion {
		return fmt.Errorf("unsupported authority contract format_version %d", c.FormatVersion)
	}
	if len(c.Sources) != 3 {
		return fmt.Errorf("authority contract requires exactly three sources")
	}
	ids, paths, digests := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, source := range c.Sources {
		if strings.TrimSpace(source.ID) != source.ID || source.ID == "" || ids[source.ID] || !validAllowedPath(source.Path) || paths[source.Path] || !digestPattern.MatchString(source.Digest) || digests[source.Digest] {
			return fmt.Errorf("authority contract source identities must have unique IDs, canonical paths, and content digests")
		}
		ids[source.ID], paths[source.Path], digests[source.Digest] = true, true, true
	}
	roles := []string{c.Roles.PRD, c.Roles.Spec, c.Roles.Roadmap}
	seenRoles := map[string]bool{}
	for _, id := range roles {
		if !ids[id] || seenRoles[id] {
			return fmt.Errorf("authority contract role bindings must reference distinct sources")
		}
		seenRoles[id] = true
	}
	if c.Story.Summary == "" || c.Story.Description == "" || len(c.Story.AcceptanceCriteria) == 0 || len(c.Slices) == 0 {
		return fmt.Errorf("authority contract story and declared slices are required")
	}
	declared := map[string]bool{}
	for _, slice := range c.Slices {
		a, s := slice.Assignment, slice.SourceDerived
		if slice.ID == "" || declared[slice.ID] || a.Phase == "" || !oneOf(a.ChangeClass, "trivial", "standard", "high-risk") || a.ReviewBudget.MaxChangedFiles < 1 || a.ReviewBudget.MaxChangedLines < 1 || len(a.ReviewBudget.Components) == 0 || len(a.AllowedPaths) == 0 || a.Dependencies == nil || !validContractADR(a.ADR) || s.Summary == "" || s.Scope == "" || len(s.NonGoals) == 0 || len(s.AcceptanceCriteria) == 0 || len(s.ValidationPlan) == 0 {
			return fmt.Errorf("declared slice %q is incomplete", slice.ID)
		}
		seenComponents := map[string]bool{}
		for _, component := range a.ReviewBudget.Components {
			if strings.TrimSpace(component) == "" || seenComponents[component] {
				return fmt.Errorf("declared slice %q review budget is invalid", slice.ID)
			}
			seenComponents[component] = true
		}
		seenPaths := map[string]bool{}
		for _, path := range a.AllowedPaths {
			if !validAllowedPath(path) || seenPaths[path] {
				return fmt.Errorf("declared slice %q has invalid allowed path %q", slice.ID, path)
			}
			seenPaths[path] = true
		}
		seenDependencies := map[string]bool{}
		for _, dependency := range a.Dependencies {
			if !declared[dependency] || seenDependencies[dependency] {
				return fmt.Errorf("declared slice %q dependency %q is missing or out of order", slice.ID, dependency)
			}
			seenDependencies[dependency] = true
		}
		declared[slice.ID] = true
	}
	allowedRules := map[string]bool{"story.summary": true, "story.description": true, "story.acceptance_criteria": true, "slices[].summary": true, "slices[].scope": true, "slices[].non_goals": true, "slices[].acceptance_criteria": true, "slices[].validation_plan": true, "slices[].adr": true}
	if c.NarrativeRules == nil {
		return fmt.Errorf("manager narrative rules must be declared, even when empty")
	}
	ruleFields := map[string]bool{}
	for _, rule := range c.NarrativeRules {
		if !allowedRules[rule.Field] || ruleFields[rule.Field] || !oneOf(rule.RequiredRole, "prd", "spec", "roadmap") || rule.MinLength < 1 {
			return fmt.Errorf("manager narrative rule is invalid")
		}
		ruleFields[rule.Field] = true
	}
	if len(c.Evidence) == 0 {
		return fmt.Errorf("authority contract evidence is required")
	}
	for _, evidence := range c.Evidence {
		if !contractEvidenceFields[evidence.Field] || !oneOf(evidence.Role, "prd", "spec", "roadmap") || strings.TrimSpace(evidence.Section) == "" || strings.TrimSpace(evidence.Excerpt) == "" {
			return fmt.Errorf("authority contract evidence is invalid")
		}
	}
	return nil
}

func validContractADR(value string) bool {
	if strings.HasPrefix(value, "No ADR needed: ") {
		return len(strings.TrimSpace(strings.TrimPrefix(value, "No ADR needed: "))) >= 10
	}
	return validAllowedPath(value) && strings.HasPrefix(value, "docs/decisions/") && filepath.Ext(value) == ".md"
}

func (c AuthorityContract) ValidateAgainst(repoRoot string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	canonical := map[string]bool{}
	for _, source := range c.Sources {
		file, err := ReadVerifiedSource(repoRoot, source.Path)
		if err != nil {
			return fmt.Errorf("verify authority contract source %q: %w", source.ID, err)
		}
		if file.RelativePath != source.Path || file.Digest != source.Digest || canonical[file.CanonicalPath] {
			return fmt.Errorf("authority contract source %q does not match its canonical identity", source.ID)
		}
		canonical[file.CanonicalPath] = true
	}
	return nil
}

func LoadAuthorityContract(path, repoRoot string) (AuthorityContract, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return AuthorityContract{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var contract AuthorityContract
	if err := decoder.Decode(&contract); err != nil {
		return AuthorityContract{}, fmt.Errorf("parse authority contract: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return AuthorityContract{}, fmt.Errorf("parse authority contract: multiple JSON values")
	}
	if err := contract.ValidateAgainst(repoRoot); err != nil {
		return AuthorityContract{}, err
	}
	return contract, nil
}

func SaveAuthorityContract(path, repoRoot string, contract AuthorityContract) (string, error) {
	if err := contract.ValidateAgainst(repoRoot); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("refusing to overwrite authority contract: %w", err)
	}
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Close()
	} else {
		_ = file.Close()
		_ = os.Remove(filepath.Clean(path))
	}
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func (c AuthorityContract) Digest() (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return digestBytes(append(data, '\n')), nil
}
