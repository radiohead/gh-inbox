package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/radiohead/gh-inbox/internal/github"
	"github.com/radiohead/gh-inbox/internal/output"
)

type prsOptions struct {
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

		// T1: hardcoded dummy data — real fetch implemented in T2.
		prs := []github.PullRequest{
			{
				Number:     1234,
				Title:      "Add CODEOWNERS support",
				URL:        "https://github.com/example/repo/pull/1234",
				Repository: github.Repository{NameWithOwner: "example/repo"},
			},
			{
				Number:     5678,
				Title:      "Fix nil pointer in filter",
				URL:        "https://github.com/example/repo/pull/5678",
				Repository: github.Repository{NameWithOwner: "example/repo"},
			},
		}

		return output.WriteJSON(cmd.OutOrStdout(), prs)
	},
}

// validatePRsFlags checks for invalid flag combinations.
func validatePRsFlags(opts prsOptions) error {
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
	prsCmd.Flags().BoolVar(&prsOpts.review, "review", false, "Show PRs awaiting my review")
	prsCmd.Flags().BoolVar(&prsOpts.authored, "authored", false, "Show my PRs needing attention")
	prsCmd.Flags().BoolVar(&prsOpts.includeCodeowners, "include-codeowners", false, "Include PRs where I'm only a CODEOWNERS reviewer (requires --review)")
	prsCmd.Flags().BoolVar(&prsOpts.codeownersSolo, "codeowners-solo", false, "Show only CODEOWNERS PRs where I'm the sole reviewer (requires --review)")
}
