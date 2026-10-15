package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rishisinu/Cascade-CI/internal/tier0"
	"github.com/spf13/cobra"
)

var tier0Cmd = &cobra.Command{
	Use:   "tier0",
	Short: "Fast, always-on checks for every PR (gitleaks, semgrep, ELF hardening)",
	Long: `Tier 0 is the fast tier: cheap, deterministic checks that run on every PR.

Checks:
  gitleaks   scans the PR's commits (base..HEAD) for committed secrets
  semgrep    scans the files the PR changed with Cascade's built-in rules,
             reporting only findings the PR introduced
  elf        checks the binary passed with --binary for missing exploit
             mitigations (only runs when --binary is set)

Planned: "tier0 doctor" to verify that gitleaks and semgrep are installed.`,
}

var tier0RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the gitleaks secret scan, semgrep static analysis and ELF checks",
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

ELF checks (only with --binary; Windows/macOS builds are skipped):
  elf-not-pie            code loads at a fixed address, defeating ASLR
                         (Go: add -buildmode=pie to your workflow's go build)
  elf-executable-stack   stack memory is executable
  elf-no-relro           GOT is writable (partial RELRO is reported as low)
  elf-no-stack-canary    C/C++ built without -fstack-protector
  elf-relative-rpath     shared libraries can load from the working directory
  RELRO and canary checks don't apply to Go; canaries don't apply to Rust.

The report is always printed to stdout, and also written to --output if set.
If a check fails to run, the report is still written with status "error" and
whatever the earlier checks found.

Exit codes:
  0  nothing found
  1  findings
  2  a check couldn't run (missing tool, bad ref, unreadable binary)`,
	Example: `  cascade tier0 run
  cascade tier0 run --repo ../my-service --base origin/main
  cascade tier0 run --binary ./bin/server
  cascade tier0 run --output cascade-report.json
  cascade tier0 run --base ""   # scan the whole repository`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Flags parsed fine by this point, so a failed scan shouldn't dump the usage text
		cmd.SilenceUsage = true

		run, runErr := tier0.Run(tier0Opts.repoPath, tier0Opts.baseRef, tier0Opts.binaryPath)

		// Write the report before acting on runErr, so a run that errored partway still
		// leaves a report (status "error") with the findings from the checks that finished.
		//Essentially takes the return struct and turns it into pretty json, first "" is prefix for each line and the " " is indent for every level
		out, err := json.MarshalIndent(run, "", "  ")
		if err != nil {
			return &codeError{exitError, err}
		}
		fmt.Println(string(out))
		if tier0Opts.outputPath != "" {
			if err := os.WriteFile(tier0Opts.outputPath, append(out, '\n'), 0o644); err != nil {
				return &codeError{exitError, fmt.Errorf("writing report: %w", err)}
			}
		}

		if runErr != nil {
			return &codeError{exitError, runErr}
		}
		//If anythying is found ...
		if len(run.Findings) > 0 {
			return &codeError{exitFindings, fmt.Errorf("tier0 run found %d finding(s)", len(run.Findings))}
		}
		return nil
	},
}
