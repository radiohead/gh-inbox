package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Surface GitHub items needing your attention",
	Long:  "gh-inbox surfaces GitHub action items across PRs, Issues, and Discussions with smart CODEOWNERS filtering.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(prsCmd)
}
