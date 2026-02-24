package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/output"
)

type prsOptions struct {
	org               string
	review            bool
	authored          bool
	includeCodeowners bool
	codeownersSolo    bool
}

var prsOpts prsOptions

var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "Show PRs needing your attention",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validatePRsFlags(prsOpts); err != nil {
			return err
		}

		client, err := github.NewClient()
		if err != nil {
			return fmt.Errorf("creating GitHub client: %w", err)
		}

		prs, err := client.FetchReviewRequestedPRs(prsOpts.org)
		if err != nil {
			return fmt.Errorf("fetching PRs: %w", err)
		}

		return output.WriteJSON(cmd.OutOrStdout(), prs)
	},
}

// validatePRsFlags checks for invalid flag combinations.
func validatePRsFlags(opts prsOptions) error {
	if opts.org == "" {
		return fmt.Errorf("--org is required")
	}
	if opts.includeCodeowners && !opts.review {
		return fmt.Errorf("--include-codeowners requires --review")
	}
	if opts.codeownersSolo && !opts.review {
		return fmt.Errorf("--codeowners-solo requires --review")
	}
	if opts.includeCodeowners && opts.codeownersSolo {
		return fmt.Errorf("--include-codeowners and --codeowners-solo are mutually exclusive")
	}
	return nil
}

func init() {
	prsCmd.Flags().StringVar(&prsOpts.org, "org", "", "GitHub organization to query (required)")
	prsCmd.Flags().BoolVar(&prsOpts.review, "review", false, "Show PRs awaiting my review")
	prsCmd.Flags().BoolVar(&prsOpts.authored, "authored", false, "Show my PRs needing attention")
	prsCmd.Flags().BoolVar(&prsOpts.includeCodeowners, "include-codeowners", false, "Include PRs where I'm only a CODEOWNERS reviewer (requires --review)")
	prsCmd.Flags().BoolVar(&prsOpts.codeownersSolo, "codeowners-solo", false, "Show only CODEOWNERS PRs where I'm the sole reviewer (requires --review)")
}
