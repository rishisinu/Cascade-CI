package cmd

import (
	"fmt"

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
		fmt.Println("tier0 called")
		return nil
	},
}
