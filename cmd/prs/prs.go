package prs

import "github.com/spf13/cobra"

// Cmd is the parent "prs" command. Subcommands are registered in init().
var Cmd = &cobra.Command{
	Use:   "prs",
	Short: "Show PRs needing your attention",
}

func init() {
	Cmd.AddCommand(reviewCmd)
	Cmd.AddCommand(authoredCmd)
}
