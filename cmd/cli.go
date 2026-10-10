package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/rishisinu/Cascade-CI/internal/tier0"
	"github.com/spf13/cobra"
)

var tier0Cmd = &cobra.Command{
	Use: "tier0",
	Long: `Fast, always-on checks for every PR.

			Available Tier0 Commands:

				run: executes the tier0 check,
				doctor: Checks that gitleaks and Docker are installed and prints their versions!`,
}

var tier0RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the secret pattern and ELF hygine checks",
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
