package prs

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radiohead/gh-inbox/internal/filter"
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

		if mode != filter.ModeAll {
			login, err := client.FetchCurrentUser()
			if err != nil {
				return fmt.Errorf("fetching current user: %w", err)
			}
			svc := service.NewTeamService(client)
			isMyTeam := func(org, slug string) bool {
				return svc.IsTeamMember(org, slug, login)
			}
			prs = filter.Filter(prs, login, isMyTeam, mode)
		}

		switch outputFormat {
		case "json":
			return output.WriteJSON(cmd.OutOrStdout(), prs)
		case "table", "":
			return output.WriteTable(cmd.OutOrStdout(), prs)
		default:
			return fmt.Errorf("unknown output format %q: must be table or json", outputFormat)
		}
	},
}

func init() {
	reviewCmd.Flags().StringVar(&reviewOpts.org, "org", "", "GitHub organization to filter by (default: all orgs)")
	reviewCmd.Flags().StringVar(&reviewOpts.filterMode, "filter", "all", "Filter mode: all|direct|codeowner")
}

// parseFilterMode converts a filter flag string to a filter.Mode.
func parseFilterMode(s string) (filter.Mode, error) {
	switch s {
	case "all", "":
		return filter.ModeAll, nil
	case "direct":
		return filter.ModeDirect, nil
	case "codeowner":
		return filter.ModeCodeowner, nil
	default:
		return 0, fmt.Errorf("unknown filter mode %q: must be all, direct, or codeowner", s)
	}
}
