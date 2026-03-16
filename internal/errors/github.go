package errors

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// GitHubClassifiers is the default classifier chain for GitHub API errors.
// Callers pass this to Classify to get GitHub-aware error classification.
var GitHubClassifiers = []Classifier{
	ClassifySAMLGraphQL,
	ClassifySAMLHTTP,
}

// ClassifySAMLGraphQL matches a *api.GraphQLError where every error item
// contains "SAML enforcement" in its message. This indicates the request was
// partially blocked by org SAML enforcement and partial data may still be
// available.
func ClassifySAMLGraphQL(err error) (bool, *ClassifiedError) {
	var gqlErr *api.GraphQLError
	if !stderrors.As(err, &gqlErr) {
		return false, nil
	}
	if len(gqlErr.Errors) == 0 {
		return false, nil
	}
	for _, item := range gqlErr.Errors {
		if !strings.Contains(item.Message, "SAML enforcement") {
			return false, nil
		}
	}
	summary := fmt.Sprintf(
		"%d result(s) skipped (org SAML enforcement — grant token access at https://github.com/settings/tokens)",
		len(gqlErr.Errors),
	)
	return true, NewClassifiedError(SeverityWarning, summary, err)
}

// ClassifySAMLHTTP matches a *api.HTTPError with status 403 where the message
// contains "SAML".
func ClassifySAMLHTTP(err error) (bool, *ClassifiedError) {
	var httpErr *api.HTTPError
	if !stderrors.As(err, &httpErr) {
		return false, nil
	}
	if httpErr.StatusCode == 403 && strings.Contains(httpErr.Message, "SAML") {
		return true, NewClassifiedError(
			SeverityWarning,
			"request blocked by org SAML enforcement",
			err,
		)
	}
	return false, nil
}
