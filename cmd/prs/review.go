package prs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/output"
	"github.com/radiohead/gh-inbox/internal/service"
)

type reviewOptions struct {
	org        string
	filterMode string
}

var reviewOpts reviewOptions

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show PRs awaiting my review",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, err := parseFilterMode(reviewOpts.filterMode)
		if err != nil {
			return err
		}

		client, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("creating GitHub client: %w", err)
		}

		prs, err := client.FetchReviewRequestedPRs(reviewOpts.org)
		if err != nil {
			return fmt.Errorf("fetching PRs: %w", err)
		}

		login, err := client.FetchCurrentUser()
		if err != nil {
			return fmt.Errorf("fetching current user: %w", err)
		}

		svc := service.NewTeamService(client)

		if mode != service.ModeAll {
			prs = service.Filter(prs, login, svc, mode)
		}

		switch outputFormat {
		case "json":
			return output.WriteJSON(cmd.OutOrStdout(), prs)
		case "table", "":
			return output.WriteTable(cmd.OutOrStdout(), prs, login, svc)
		default:
			return fmt.Errorf("unknown output format %q: must be table or json", outputFormat)
		}
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewOpts.org, "org", "", "GitHub organization to filter by (default: all orgs)")
	reviewCmd.Flags().StringVar(&reviewOpts.filterMode, "filter", "all", "Filter mode: all|direct|codeowner|team")
}

// parseFilterMode converts a filter flag string to a service.Mode.
func parseFilterMode(s string) (service.Mode, error) {
	switch s {
	case "all", "":
		return service.ModeAll, nil
	case "direct":
		return service.ModeDirect, nil
	case "codeowner":
		return service.ModeCodeowner, nil
	case "team":
		return service.ModeTeam, nil
	default:
		return 0, fmt.Errorf("unknown filter mode %q: must be all, direct, codeowner, or team", s)
	}
}
