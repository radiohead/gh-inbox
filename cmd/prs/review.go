package prs

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

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
	filterStatus string
	refresh      bool
}

var reviewOpts reviewOptions

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show PRs awaiting my review",
	RunE: func(cmd *cobra.Command, args []string) error {
		criteria, err := resolveFilter(reviewOpts.filter, reviewOpts.filterType, reviewOpts.filterSource, reviewOpts.filterStatus)
		if err != nil {
			return err
		}

		diskCache, cacheErr := cache.NewDiskCacher("", 0)
		if cacheErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not initialize disk cache: %v\n", cacheErr)
		}
		prCache, prCacheErr := cache.NewDiskCacher("", 5*time.Minute)
		if prCacheErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not initialize PR disk cache: %v\n", prCacheErr)
		}

		var opts []github.ClientOption
		if diskCache != nil {
			opts = append(opts, github.WithCache(diskCache))
		}
		if prCache != nil {
			opts = append(opts, github.WithPRCache(prCache))
		}
		if reviewOpts.refresh {
			opts = append(opts, github.WithRefresh())
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

		var prs []github.PullRequest
		g := new(errgroup.Group)
		g.Go(func() error {
			var fetchErr error
			prs, fetchErr = client.FetchReviewRequestedPRs(reviewOpts.org)
			return fetchErr
		})
		g.Go(func() error {
			_ = svc.PreloadTeams() // fail-closed: error is stored in TeamService
			return nil
		})
		if err := g.Wait(); err != nil {
			return fmt.Errorf("fetching PRs: %w", err)
		}

		classified := classifier.ClassifyAll(prs)
		results := (&service.CriteriaFilter{Criteria: criteria}).Apply(classified)

		switch outputFormat {
		case "json":
			type jsonPR struct {
				github.PullRequest
				ReviewType   string `json:"reviewType,omitempty"`
				Source       string `json:"source,omitempty"`
				ReviewStatus string `json:"reviewStatus,omitempty"`
			}
			out := make([]jsonPR, len(results))
			for i, cp := range results {
				out[i] = jsonPR{
					PullRequest:  cp.PR,
					ReviewType:   string(cp.ReviewType),
					Source:       string(cp.AuthorSource),
					ReviewStatus: string(cp.ReviewStatus),
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
	reviewCmd.Flags().StringVar(&reviewOpts.filterStatus, "filter-status", "", "Filter by review status: open|in_review|approved|all")
	reviewCmd.Flags().BoolVar(&reviewOpts.refresh, "refresh", false, "Bypass PR cache and fetch fresh data from GitHub")
	reviewCmd.MarkFlagsMutuallyExclusive("filter", "filter-type")
	reviewCmd.MarkFlagsMutuallyExclusive("filter", "filter-source")
}

// resolveFilter builds a FilterCriteria from the filter flags.
// Granular flags (filter-type, filter-source) take precedence and are ANDed together.
// If neither granular flag is set, filter is treated as a preset name.
// --filter-status is orthogonal: it overrides the preset's ReviewStatuses default when provided.
// Default when no flags: ReviewStatuses={open: true} (FR-002).
func resolveFilter(filter, filterType, filterSource, filterStatus string) (service.FilterCriteria, error) {
	var criteria service.FilterCriteria
	var err error

	if filterType != "" || filterSource != "" {
		criteria, err = resolveGranular(filterType, filterSource)
		if err != nil {
			return service.FilterCriteria{}, err
		}
		// Granular mode: apply default open status unless overridden.
		criteria.ReviewStatuses = service.ReviewStatusSet{service.ReviewStatusOpen: true}
	} else if filter != "" {
		p := service.Preset(filter)
		switch p {
		case service.PresetAll, service.PresetFocus, service.PresetNearby, service.PresetOrg:
			criteria = service.PresetCriteria(p)
		default:
			return service.FilterCriteria{}, fmt.Errorf("unknown filter preset %q: must be all, focus, nearby, or org", filter)
		}
	} else {
		// No flags: default to open status only.
		criteria = service.FilterCriteria{
			ReviewStatuses: service.ReviewStatusSet{service.ReviewStatusOpen: true},
		}
	}

	// --filter-status overrides the preset/default ReviewStatuses.
	if filterStatus != "" {
		rs, err := resolveReviewStatus(filterStatus)
		if err != nil {
			return service.FilterCriteria{}, err
		}
		criteria.ReviewStatuses = rs
	}

	return criteria, nil
}

// resolveReviewStatus parses a --filter-status flag value into a ReviewStatusSet.
// "all" returns nil (no filtering). Other valid values: open, in_review, approved.
func resolveReviewStatus(filterStatus string) (service.ReviewStatusSet, error) {
	switch filterStatus {
	case "all":
		return nil, nil
	case string(service.ReviewStatusOpen):
		return service.ReviewStatusSet{service.ReviewStatusOpen: true}, nil
	case string(service.ReviewStatusInReview):
		return service.ReviewStatusSet{service.ReviewStatusInReview: true}, nil
	case string(service.ReviewStatusApproved):
		return service.ReviewStatusSet{service.ReviewStatusApproved: true}, nil
	default:
		return nil, fmt.Errorf("unknown review status %q: must be open, in_review, approved, or all", filterStatus)
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
