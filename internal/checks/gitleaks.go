package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/rishisinu/Cascade-CI/internal/report"
)

type gitleaksFinding struct {
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

func GitLeaks(repo string, base string) ([]report.Finding, error) {
	//Constructing args so we get correct final output
	args := []string{"git", "--no-banner", "--exit-code", "0",
		"--report-format", "json", "--report-path", "-"}
	//Basically if bad base
	if base != "" {
		if err := exec.Command("git", "-C", repo, "rev-parse", "--verify", base).Run(); err != nil {
			return nil, fmt.Errorf("base ref %q not found in %s (shallow clone?): %w", base, repo, err)
		}

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
	var completed []gitleaksFinding
	if err := json.Unmarshal(output, &completed); err != nil {
		return nil, fmt.Errorf("parsing gitleaks output: %w", err)
	}

	findings := make([]report.Finding, 0, len(completed))
	for _, g := range completed {
		findings = append(findings, report.Finding{
			ID:         g.Fingerprint,
			Tier:       0,
			SourceTool: report.ToolGitleaks,
			Severity:   report.SeverityHigh,
			Category:   g.RuleID,
			Location:   report.Location{File: g.File, Line: g.StartLine},
			RawMessage: g.Description,
		})
	}

	return findings, nil
}
