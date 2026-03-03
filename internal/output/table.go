package output

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/cli/go-gh/v2/pkg/term"

	"github.com/radiohead/gh-inbox/internal/service"
)

// WriteTable writes prs as a human-readable table to w.
// In TTY mode, columns are rendered with colors and aligned to terminal width.
// In non-TTY mode, output is tab-separated (suitable for scripting).
// An empty list produces a single informational message instead of an empty table.
func WriteTable(w io.Writer, prs []service.ClassifiedPR) error {
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
		pi, pj := service.Priority(prs[i]), service.Priority(prs[j])
		if pi != pj {
			return pi < pj
		}
		return prs[i].PR.CreatedAt.Before(prs[j].PR.CreatedAt)
	})

	tp := tableprinter.New(w, isTTY, width)
	tp.AddHeader([]string{"REPO", "PR", "TITLE", "URL", "TYPE", "SOURCE", "AGE"})

	for _, cp := range prs {
		pr := cp.PR
		repo := pr.Repository.Owner + "/" + pr.Repository.Name
		prNum := fmt.Sprintf("#%d", pr.Number)
		rt := reviewTypeLabel(cp.ReviewType)
		src := authorSourceLabel(cp.AuthorSource)
		age := humanAge(pr.CreatedAt)

		tp.AddField(repo)
		tp.AddField(prNum)
		tp.AddField(pr.Title)
		tp.AddField(pr.URL)
		tp.AddField(rt, tableprinter.WithColor(reviewTypeColor(rt)))
		tp.AddField(src, tableprinter.WithColor(authorSourceColor(src)))
		tp.AddField(age)
		tp.EndRow()
	}

	return tp.Render()
}

// reviewTypeLabel converts a ReviewType to its display string.
// An empty ReviewType (from PassthroughClassifier) renders as "-".
func reviewTypeLabel(s service.ReviewType) string {
	if s == "" {
		return "-"
	}
	return string(s)
}

// reviewTypeColor returns an ANSI color function for the given review type label.
// Colors are only applied when the tableprinter is in TTY mode.
func reviewTypeColor(rt string) func(string) string {
	switch rt {
	case "direct":
		return func(s string) string { return "\033[32m" + s + "\033[0m" } // green
	case "team":
		return func(s string) string { return "\033[33m" + s + "\033[0m" } // yellow
	default: // codeowner or "-"
		return func(s string) string { return "\033[36m" + s + "\033[0m" } // cyan
	}
}

// authorSourceLabel converts an AuthorSource to its display string.
// An empty AuthorSource renders as "-".
func authorSourceLabel(s service.AuthorSource) string {
	if s == "" {
		return "-"
	}
	return string(s)
}

// authorSourceColor returns an ANSI color function for the given author source label.
func authorSourceColor(src string) func(string) string {
	switch src {
	case "TEAM":
		return func(s string) string { return "\033[32m" + s + "\033[0m" } // green
	case "GROUP":
		return func(s string) string { return "\033[33m" + s + "\033[0m" } // yellow
	case "ORG":
		return func(s string) string { return "\033[36m" + s + "\033[0m" } // cyan
	default: // OTHER or "-"
		return func(s string) string { return s } // default
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
