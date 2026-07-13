package implementation

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"codex-governance/internal/ollama"
)

type AssessmentRequest struct {
	Role       string
	Model      string
	Policy     ollama.Policy
	Bundle     TaskBundle
	Worktree   string
	OutputPath string
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
	prompt, err := assessmentPrompt(request.Role, request.Bundle, diff)
	if err != nil {
		return Assessment{}, err
	}
	output, err := ollama.Run(ollama.Client(request.Policy), request.Policy, ollama.Request{Model: request.Model, Role: request.Role, TaskType: "implementation-review", Input: []byte(prompt)})
	if err != nil {
		return Assessment{}, err
	}
	assessment, err := parseAssessment([]byte(output))
	if err != nil {
		return Assessment{}, err
	}
	if err := SaveAssessment(request.OutputPath, assessment); err != nil {
		return Assessment{}, err
	}
	return assessment, nil
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
	data, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("You are an independent %s. Assess this implementation diff only against the approved task bundle. Do not edit files or contact external systems. Return only JSON matching {\"findings\":[{\"id\":string,\"severity\":\"blocking|important|minor|informational\",\"summary\":string}]}.\n\nTASK BUNDLE:\n%s\n\nDIFF:\n%s", role, data, diff), nil
}

func parseAssessment(data []byte) (Assessment, error) {
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
