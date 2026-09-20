package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/rishisinu/Cascade-CI/internal/report"
)

type Finding struct {
	Description string   `json:"Description"`
	StartLine   int      `json:"StartLine"`
	EndLine     int      `json:"EndLine"`
	StartColumn int      `json:"StartColumn"`
	EndColumn   int      `json:"EndColumn"`
	Line        string   `json:"Line"`
	Match       string   `json:"Match"`
	Secret      string   `json:"Secret"`
	File        string   `json:"File"`
	SymlinkFile string   `json:"SymlinkFile"`
	Commit      string   `json:"Commit"`
	Entropy     float32  `json:"Entropy"`
	Author      string   `json:"Author"`
	Email       string   `json:"Email"`
	Date        string   `json:"Date"`
	Message     string   `json:"Message"`
	Tags        []string `json:"Tags"`
	RuleID      string   `json:"RuleID"`
	Fingerprint string   `json:"Fingerprint"`
	Link        string   `json:"Link,omitempty"`
}

func useGitLeaks(repo string, base string) ([]report.Finding, error) {
	//Constructing args so we get correct final output
	args := []string{"git", "--no-banner", "--exit-code", "0",
		"--report-format", "json", "--report-path", "-"}
	if base != "" {
		args = append(args, "--log-opts", base+"..HEAD")
	}
	args = append(args, repo)
	cmd := exec.Command("gitleaks", args...) // THe ... essentially expands the arr so its like you pass in those args individually

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gitleaks unfort failed: %w: %s", err, stderr.String())
	}
	var completed []Finding
	if err := json.Unmarshal(output, &completed); err != nil {
		return nil, fmt.Errorf("parsing gitleaks output: %w", err)
	}

	findings := make([]report.Finding, 0, len(completed))
	for _, g := range completed {
		findings = append(findings, report.Finding{
			ID:         g.Fingerprint,
			Tier:       "0",
			SourceTool: report.ToolGitleaks,
			Severity:   report.SeverityHigh,
			Category:   g.RuleID,
		})
	}

	return findings, nil
}
