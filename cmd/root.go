/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

// tier0Options holds the values parsed from the `tier0 run` flags.
type tier0Options struct {
	repoPath   string
	baseRef    string
	binaryPath string
	outputPath string
}

var tier0Opts tier0Options

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cascade",
	Short: "Security checks for pull requests, run in tiers from fast to deep",
	Long: `Cascade runs security checks against a pull request and reports the findings
as JSON. The exit code lets it gate CI: 0 when nothing is found, 1 when there
are findings, and 2 when a check couldn't run (missing tool, bad ref, bad flags).

Tier 0 (implemented): fast checks that run on every PR
  gitleaks   secrets committed in the PR's commits
  semgrep    static analysis on the files the PR changed, using Cascade's
             built-in rulesets: Go, Python, JavaScript/TypeScript, Bash,
             Dockerfile, GitHub Actions workflows, and agentic/LLM code
             (Rust files use the Semgrep registry's p/rust, which needs network)
  elf        exploit-mitigation checks on the Linux binary the PR builds
             (--binary): PIE, non-executable stack, RELRO, stack canaries,
             RPATH. Non-ELF files (Windows/macOS builds) are skipped.

Planned: a "doctor" command that checks the required tools are installed.

Requires git, gitleaks and semgrep on PATH.

Example:
  cascade tier0 run --repo . --base origin/main`,
	Version: "1.0-dev",
	Run: func(cmd *cobra.Command, args []string) {
		// Running bare `cascade` prints the overview above instead of doing nothing
		cmd.Help()
	},
}

// Exit codes, so CI can tell "you have a vulnerability" apart from "the scan broke".
const (
	exitClean    = 0
	exitFindings = 1
	exitError    = 2
)

// codeError is an error that also sets the process exit code.
type codeError struct {
	code int
	err  error
}

func (e *codeError) Error() string { return e.err.Error() }
func (e *codeError) Unwrap() error { return e.err }

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		var ce *codeError
		if errors.As(err, &ce) {
			os.Exit(ce.code)
		}
		os.Exit(exitError) // anything else is a usage problem, e.g. an unknown flag
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
	tier0RunCmd.Flags().StringVar(&tier0Opts.binaryPath, "binary", "", "path to the Linux (ELF) binary built from this PR; enables the ELF hardening checks")
	tier0RunCmd.Flags().StringVar(&tier0Opts.outputPath, "output", "", "path to the output file to write the JSON report to")

}
