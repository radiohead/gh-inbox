package output

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/cli/go-gh/v2/pkg/term"

	"github.com/radiohead/gh-inbox/internal/github"
)

// WriteTable writes prs as a human-readable table to w.
// In TTY mode, columns are rendered with colors and aligned to terminal width.
// In non-TTY mode, output is tab-separated (suitable for scripting).
// An empty list produces a single informational message instead of an empty table.
func WriteTable(w io.Writer, prs []github.PullRequest) error {
	if len(prs) == 0 {
		_, err := fmt.Fprintln(w, "No pull requests found.")
		return err
	}

	t := term.FromEnv()
	isTTY := t.IsTerminalOutput()
	width, _, err := t.Size()
	if err != nil || width <= 0 {
		width = 120
	}

	sort.SliceStable(prs, func(i, j int) bool {
		return prs[i].CreatedAt.Before(prs[j].CreatedAt)
	})

	tp := tableprinter.New(w, isTTY, width)
	tp.AddHeader([]string{"REPO", "PR", "TITLE", "URL", "SOURCE", "AGE"})

	for _, pr := range prs {
		repo := pr.Repository.Owner + "/" + pr.Repository.Name
		prNum := fmt.Sprintf("#%d", pr.Number)
		source := sourceOf(pr)
		age := humanAge(pr.CreatedAt)

		tp.AddField(repo)
		tp.AddField(prNum)
		tp.AddField(pr.Title)
		tp.AddField(pr.URL)
		tp.AddField(source, tableprinter.WithColor(sourceColor(source)))
		tp.AddField(age)
		tp.EndRow()
	}

	return tp.Render()
}

// sourceOf derives a SOURCE label from the review requests on pr.
// Returns "direct" if any non-CODEOWNERS user request exists,
// "team" if any non-CODEOWNERS team request exists, or "codeowner" otherwise.
func sourceOf(pr github.PullRequest) string {
	hasTeam := false
	for _, rr := range pr.ReviewRequests.Nodes {
		if !rr.AsCodeOwner {
			switch rr.RequestedReviewer.Type {
			case "User":
				return "direct"
			case "Team":
				hasTeam = true
			}
		}
	}
	if hasTeam {
		return "team"
	}
	return "codeowner"
}

// sourceColor returns an ANSI color function for the given source label.
// Colors are only applied when the tableprinter is in TTY mode.
func sourceColor(source string) func(string) string {
	switch source {
	case "direct":
		return func(s string) string { return "\033[32m" + s + "\033[0m" } // green
	case "team":
		return func(s string) string { return "\033[33m" + s + "\033[0m" } // yellow
	default: // codeowner
		return func(s string) string { return "\033[36m" + s + "\033[0m" } // cyan
	}
}

// humanAge formats the duration since t as a human-readable relative age.
// Examples: "5m", "3h", "2d", "1w", "3mo", "1y".
func humanAge(t time.Time) string {
	d := time.Since(t)

	minutes := int(d.Minutes())
	hours := int(d.Hours())
	days := int(d.Hours() / 24)
	weeks := days / 7
	months := int(d.Hours() / (24 * 30))
	years := int(d.Hours() / (24 * 365))

	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", minutes)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", hours)
	case days < 7:
		return fmt.Sprintf("%dd", days)
	case weeks < 5:
		return fmt.Sprintf("%dw", weeks)
	case months < 12:
		return fmt.Sprintf("%dmo", months)
	default:
		return fmt.Sprintf("%dy", years)
	}
}
