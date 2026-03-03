package service

import "github.com/radiohead/gh-inbox/internal/github"

// ClassifiedPR pairs a PullRequest with its precomputed ReviewType classification
// and AuthorSource. ReviewType is empty when classification is skipped
// (e.g. PassthroughClassifier).
type ClassifiedPR struct {
	PR           github.PullRequest
	ReviewType   ReviewType
	AuthorSource AuthorSource
}

// Fetcher fetches raw pull requests for an org.
type Fetcher interface {
	Fetch(org string) ([]github.PullRequest, error)
}

// Classifier assigns a ReviewType classification to each PR.
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
	_ PRFilter   = (*CriteriaFilter)(nil)
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

// ClassifyAll classifies each PR by delegating to Classify and ClassifyAuthorSource.
func (c *SourceClassifier) ClassifyAll(prs []github.PullRequest) []ClassifiedPR {
	result := make([]ClassifiedPR, len(prs))
	for i, pr := range prs {
		result[i] = ClassifiedPR{
			PR:           pr,
			ReviewType:   Classify(pr, c.Login, c.Teams),
			AuthorSource: ClassifyAuthorSource(pr, c.Login, c.Teams),
		}
	}
	return result
}

// PassthroughClassifier wraps PRs with an empty ReviewType — used when
// classification is not required.
type PassthroughClassifier struct{}

// ClassifyAll wraps each PR with an empty ReviewType.
func (PassthroughClassifier) ClassifyAll(prs []github.PullRequest) []ClassifiedPR {
	result := make([]ClassifiedPR, len(prs))
	for i, pr := range prs {
		result[i] = ClassifiedPR{PR: pr}
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
