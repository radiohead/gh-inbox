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

		var classifier service.Classifier
		if needsUserContext(mode, outputFormat) {
			login, err := client.FetchCurrentUser()
			if err != nil {
				return fmt.Errorf("fetching current user: %w", err)
			}
			svc := service.NewTeamService(client)
			classifier = &service.SourceClassifier{Login: login, Teams: svc}
		} else {
			classifier = service.PassthroughClassifier{}
		}

		pipeline := service.NewPipeline(
			service.FetchFunc(client.FetchReviewRequestedPRs),
			classifier,
			&service.ModeFilter{Mode: mode},
		)

		results, err := pipeline.Run(reviewOpts.org)
		if err != nil {
			return fmt.Errorf("fetching PRs: %w", err)
		}

		switch outputFormat {
		case "json":
			type jsonPR struct {
				github.PullRequest
				ReviewType string `json:"reviewType,omitempty"`
				Source     string `json:"source,omitempty"`
			}
			out := make([]jsonPR, len(results))
			for i, cp := range results {
				out[i] = jsonPR{
					PullRequest: cp.PR,
					ReviewType:  string(cp.ReviewType),
					Source:      string(cp.AuthorSource),
				}
			}
			return output.WriteJSON(cmd.OutOrStdout(), out)
		case "table", "":
			return output.WriteTable(cmd.OutOrStdout(), results)
		default:
			return fmt.Errorf("unknown output format %q: must be table or json", outputFormat)
		}
	},
}

// needsUserContext reports whether user/team lookups are required.
// They are needed for classification (review type + author source labeling).
func needsUserContext(mode service.Mode, output string) bool {
	return true
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
