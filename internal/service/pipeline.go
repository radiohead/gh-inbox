package service

import "github.com/radiohead/gh-inbox/internal/github"

// ClassifiedPR pairs a PullRequest with its precomputed Source classification.
// Source is empty when classification is skipped (e.g. PassthroughClassifier).
type ClassifiedPR struct {
	PR     github.PullRequest
	Source Source
}

// Fetcher fetches raw pull requests for an org.
type Fetcher interface {
	Fetch(org string) ([]github.PullRequest, error)
}

// Classifier assigns a Source classification to each PR.
type Classifier interface {
	ClassifyAll(prs []github.PullRequest) []ClassifiedPR
}

// PRFilter selects a subset of classified PRs.
type PRFilter interface {
	Apply(prs []ClassifiedPR) []ClassifiedPR
}

// Compile-time interface checks.
var (
	_ Fetcher    = FetchFunc(nil)
	_ Classifier = (*SourceClassifier)(nil)
	_ Classifier = PassthroughClassifier{}
	_ PRFilter   = (*ModeFilter)(nil)
)

// FetchFunc is a function adapter implementing Fetcher.
type FetchFunc func(org string) ([]github.PullRequest, error)

// Fetch implements Fetcher by calling f.
func (f FetchFunc) Fetch(org string) ([]github.PullRequest, error) {
	return f(org)
}

// SourceClassifier classifies each PR using the authenticated user's identity.
type SourceClassifier struct {
	Login string
	Teams *TeamService
}

// ClassifyAll classifies each PR by delegating to Classify.
func (c *SourceClassifier) ClassifyAll(prs []github.PullRequest) []ClassifiedPR {
	result := make([]ClassifiedPR, len(prs))
	for i, pr := range prs {
		result[i] = ClassifiedPR{
			PR:     pr,
			Source: Classify(pr, c.Login, c.Teams),
		}
	}
	return result
}

// PassthroughClassifier wraps PRs with an empty Source — used when
// classification is not required (e.g. ModeAll + JSON output).
type PassthroughClassifier struct{}

// ClassifyAll wraps each PR with an empty Source.
func (PassthroughClassifier) ClassifyAll(prs []github.PullRequest) []ClassifiedPR {
	result := make([]ClassifiedPR, len(prs))
	for i, pr := range prs {
		result[i] = ClassifiedPR{PR: pr}
	}
	return result
}

// ModeFilter keeps only PRs whose Source matches the configured mode.
// ModeAll passes all PRs through without inspecting Source.
type ModeFilter struct {
	Mode Mode
}

// Apply returns the subset of prs matching the configured mode.
func (f *ModeFilter) Apply(prs []ClassifiedPR) []ClassifiedPR {
	if f.Mode == ModeAll {
		return prs
	}
	target := modeToSource(f.Mode)
	result := make([]ClassifiedPR, 0, len(prs))
	for _, cp := range prs {
		if cp.Source == target {
			result = append(result, cp)
		}
	}
	return result
}

// Pipeline orchestrates fetch → classify → filter.
type Pipeline struct {
	fetcher    Fetcher
	classifier Classifier
	filter     PRFilter
}

// NewPipeline constructs a Pipeline from its three components.
func NewPipeline(fetcher Fetcher, classifier Classifier, filter PRFilter) *Pipeline {
	return &Pipeline{
		fetcher:    fetcher,
		classifier: classifier,
		filter:     filter,
	}
}

// Run fetches PRs for org, classifies them, and applies the filter.
func (p *Pipeline) Run(org string) ([]ClassifiedPR, error) {
	raw, err := p.fetcher.Fetch(org)
	if err != nil {
		return nil, err
	}
	classified := p.classifier.ClassifyAll(raw)
	return p.filter.Apply(classified), nil
}
