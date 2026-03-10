package prs

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/radiohead/gh-inbox/internal/cache"
	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/output"
	"github.com/radiohead/gh-inbox/internal/service"
)

type reviewOptions struct {
	org          string
	filter       string
	filterType   string
	filterSource string
}

var reviewOpts reviewOptions

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show PRs awaiting my review",
	RunE: func(cmd *cobra.Command, args []string) error {
		criteria, err := resolveFilter(reviewOpts.filter, reviewOpts.filterType, reviewOpts.filterSource)
		if err != nil {
			return err
		}

		diskCache, cacheErr := cache.NewDiskCacher("", 0)
		if cacheErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not initialize disk cache: %v\n", cacheErr)
		}
		var opts []github.ClientOption
		if diskCache != nil {
			opts = append(opts, github.WithCache(diskCache))
		}

		client, err := github.NewClient(opts...)
		if err != nil {
			return fmt.Errorf("creating GitHub client: %w", err)
		}

		login, err := client.FetchCurrentUser()
		if err != nil {
			return fmt.Errorf("fetching current user: %w", err)
		}
		svc := service.NewTeamService(client)
		classifier := &service.SourceClassifier{Login: login, Teams: svc}

		pipeline := service.NewPipeline(
			service.FetchFunc(client.FetchReviewRequestedPRs),
			classifier,
			&service.CriteriaFilter{Criteria: criteria},
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

func init() {
	reviewCmd.Flags().StringVar(&reviewOpts.org, "org", "", "GitHub organization to filter by (default: all orgs)")
	reviewCmd.Flags().StringVar(&reviewOpts.filter, "filter", "", "Filter preset: all|focus|nearby|org (default: all)")
	reviewCmd.Flags().StringVar(&reviewOpts.filterType, "filter-type", "", "Filter by review type: direct|codeowner|team")
	reviewCmd.Flags().StringVar(&reviewOpts.filterSource, "filter-source", "", "Filter by author source: TEAM|GROUP|ORG|OTHER")
	reviewCmd.MarkFlagsMutuallyExclusive("filter", "filter-type")
	reviewCmd.MarkFlagsMutuallyExclusive("filter", "filter-source")
}

// resolveFilter builds a FilterCriteria from the three filter flags.
// Granular flags (filter-type, filter-source) take precedence and are ANDed together.
// If neither granular flag is set, filter is treated as a preset name (default: "all").
func resolveFilter(filter, filterType, filterSource string) (service.FilterCriteria, error) {
	if filterType != "" || filterSource != "" {
		return resolveGranular(filterType, filterSource)
	}
	if filter == "" {
		filter = "all"
	}
	p := service.Preset(filter)
	switch p {
	case service.PresetAll, service.PresetFocus, service.PresetNearby, service.PresetOrg:
		return service.PresetCriteria(p), nil
	default:
		return service.FilterCriteria{}, fmt.Errorf("unknown filter preset %q: must be all, focus, nearby, or org", filter)
	}
}

// resolveGranular builds a FilterCriteria from raw type/source flag strings.
func resolveGranular(filterType, filterSource string) (service.FilterCriteria, error) {
	var criteria service.FilterCriteria

	if filterType != "" {
		rt := service.ReviewType(filterType)
		switch rt {
		case service.ReviewTypeDirect, service.ReviewTypeCodeowner, service.ReviewTypeTeam:
			criteria.ReviewTypes = service.ReviewTypeSet{rt: true}
		default:
			return service.FilterCriteria{}, fmt.Errorf("unknown review type %q: must be direct, codeowner, or team", filterType)
		}
	}

	if filterSource != "" {
		as := service.AuthorSource(strings.ToUpper(filterSource))
		switch as {
		case service.AuthorSourceTeam, service.AuthorSourceGroup, service.AuthorSourceOrg, service.AuthorSourceOther:
			criteria.AuthorSources = service.AuthorSourceSet{as: true}
		default:
			return service.FilterCriteria{}, fmt.Errorf("unknown author source %q: must be TEAM, GROUP, ORG, or OTHER", filterSource)
		}
	}

	return criteria, nil
}
