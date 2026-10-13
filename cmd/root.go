/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// tier0Options holds the values parsed from the `tier0 run` flags.
type tier0Options struct {
	repoPath   string
	baseRef    string
	binaryPath string
}

var tier0Opts tier0Options

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cascade",
	Short: "Security checks for pull requests, run in tiers from fast to deep",
	Long: `Cascade runs security checks against a pull request and reports the findings
as JSON. A run fails (exit code 1) if anything is found, so it can gate CI.

Tier 0 (implemented): fast checks that run on every PR
  gitleaks   secrets committed in the PR's commits
  semgrep    static analysis on the files the PR changed, using Cascade's
             built-in rulesets: Go, Python, JavaScript/TypeScript, Bash,
             Dockerfile, GitHub Actions workflows, and agentic/LLM code
             (Rust files use the Semgrep registry's p/rust, which needs network)

Planned: ELF binary hygiene checks (--binary), and a "doctor" command that
checks the required tools are installed.

Requires git, gitleaks and semgrep on PATH.

Example:
  cascade tier0 run --repo . --base origin/main`,
	Version: "1.0-dev",
	Run: func(cmd *cobra.Command, args []string) {
		// Running bare `cascade` prints the overview above instead of doing nothing
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cascade-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(tier0Cmd)

	tier0Cmd.AddCommand(tier0RunCmd)

	tier0RunCmd.Flags().StringVar(&tier0Opts.repoPath, "repo", ".", "path to the repository to scan")
	tier0RunCmd.Flags().StringVar(&tier0Opts.baseRef, "base", "origin/main", "git ref to diff against, scoping the scan to the PR")
	tier0RunCmd.Flags().StringVar(&tier0Opts.binaryPath, "binary", "", "path to the ELF binary built from this PR (accepted, but ELF checks are not implemented yet)")
}
