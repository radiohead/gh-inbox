package prs

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authoredCmd = &cobra.Command{
	Use:   "authored",
	Short: "Show my PRs needing attention",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("gh inbox prs authored is not yet implemented, see https://github.com/radiohead/gh-inbox for updates")
	},
}
