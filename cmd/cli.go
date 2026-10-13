package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/rishisinu/Cascade-CI/internal/tier0"
	"github.com/spf13/cobra"
)

var tier0Cmd = &cobra.Command{
	Use:   "tier0",
	Short: "Fast, always-on checks for every PR (gitleaks + semgrep)",
	Long: `Tier 0 is the fast tier: cheap, deterministic checks that run on every PR.

Checks:
  gitleaks   scans the PR's commits (base..HEAD) for committed secrets
  semgrep    scans the files the PR changed with Cascade's built-in rules,
             reporting only findings the PR introduced

Planned: ELF binary hygiene checks, and "tier0 doctor" to verify that
gitleaks and semgrep are installed.`,
}

var tier0RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the gitleaks secret scan and semgrep static analysis",
	Long: `Runs every tier 0 check against the repository at --repo and prints one JSON
report with all findings.

How the PR is scoped:
  --base is the git ref the PR is compared against (a branch, tag or commit SHA).
  gitleaks scans the commits in base..HEAD. semgrep picks rulesets from the file
  types changed in base...HEAD, and reports only findings that are new since base.
  With --base "", everything in the repository is scanned.

semgrep rulesets by file type:
  .go                          go, agentic
  .py                          python, agentic
  .js .jsx .mjs .cjs .ts .tsx  javascript, agentic
  .sh                          bash
  Dockerfile, *.dockerfile     dockerfile
  .github/workflows/*.yml      github-actions
  .rs                          Semgrep registry p/rust (needs network)

Exit code is 0 when nothing is found and 1 when there are findings or a check fails.`,
	Example: `  cascade tier0 run
  cascade tier0 run --repo ../my-service --base origin/main
  cascade tier0 run --base ""   # scan the whole repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		run, err := tier0.Run(tier0Opts.repoPath, tier0Opts.baseRef, tier0Opts.binaryPath)
		if err != nil {
			return err
		}
		//Essentially takes the return struct and turns it into pretty json, first "" is prefix for each line and the " " is indent for every level
		out, err := json.MarshalIndent(run, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		//If anythying is found ...
		if len(run.Findings) > 0 {
			return fmt.Errorf("tier0 failed with %d finding's'", len(run.Findings))
		}
		return nil
	},
}
