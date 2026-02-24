package prs

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authoredCmd = &cobra.Command{
	Use:   "authored",
	Short: "Show my PRs needing attention",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented")
	},
}
