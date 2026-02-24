package prs

import "github.com/spf13/cobra"

// outputFormat is shared by all prs subcommands via PersistentFlags.
var outputFormat string

// Cmd is the parent "prs" command. Subcommands are registered in init().
var Cmd = &cobra.Command{
	Use:   "prs",
	Short: "Show PRs needing your attention",
}

func init() {
	Cmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format: table|json")
	Cmd.AddCommand(reviewCmd)
	Cmd.AddCommand(authoredCmd)
}
