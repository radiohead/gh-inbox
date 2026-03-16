package errors_test

import (
	stderrors "errors"
	"testing"

	gherrors "github.com/radiohead/gh-inbox/internal/errors"
)

func TestClassifiedError_ImplementsError(t *testing.T) {
	ce := gherrors.NewClassifiedError(gherrors.SeverityWarning, "something happened", stderrors.New("orig"))
	var _ error = ce // compile-time check
	if ce.Error() != "something happened" {
		t.Errorf("Error() = %q, want %q", ce.Error(), "something happened")
	}
}

func TestClassifiedError_Unwrap(t *testing.T) {
	orig := stderrors.New("original error")
	ce := gherrors.NewClassifiedError(gherrors.SeverityCritical, "wrapped", orig)

	if !stderrors.Is(ce, orig) {
		t.Error("errors.Is(ce, orig) = false, want true")
	}

	unwrapped := stderrors.Unwrap(ce)
	if unwrapped != orig {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, orig)
	}
}

func TestClassifiedError_SeverityAndSummary(t *testing.T) {
	tests := []struct {
		name     string
		severity gherrors.Severity
		summary  string
	}{
		{"critical", gherrors.SeverityCritical, "fatal error"},
		{"warning", gherrors.SeverityWarning, "partial data"},
		{"silent", gherrors.SeveritySilent, "ignored"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := gherrors.NewClassifiedError(tt.severity, tt.summary, stderrors.New("orig"))
			if ce.Severity() != tt.severity {
				t.Errorf("Severity() = %v, want %v", ce.Severity(), tt.severity)
			}
			if ce.Summary() != tt.summary {
				t.Errorf("Summary() = %q, want %q", ce.Summary(), tt.summary)
			}
		})
	}
}

func TestClassify_NoMatchDefaultsCritical(t *testing.T) {
	orig := stderrors.New("some unknown error")
	result := gherrors.Classify(orig)

	if result.Severity() != gherrors.SeverityCritical {
		t.Errorf("Severity() = %v, want Critical", result.Severity())
	}
	if result.Summary() != orig.Error() {
		t.Errorf("Summary() = %q, want %q", result.Summary(), orig.Error())
	}
	if !stderrors.Is(result, orig) {
		t.Error("errors.Is(result, orig) = false, want true")
	}
}

func TestClassify_FirstMatchWins(t *testing.T) {
	orig := stderrors.New("test error")

	first := func(err error) (bool, *gherrors.ClassifiedError) {
		return true, gherrors.NewClassifiedError(gherrors.SeverityWarning, "first matched", err)
	}
	second := func(err error) (bool, *gherrors.ClassifiedError) {
		return true, gherrors.NewClassifiedError(gherrors.SeveritySilent, "second matched", err)
	}

	result := gherrors.Classify(orig, first, second)
	if result.Summary() != "first matched" {
		t.Errorf("Summary() = %q, want %q", result.Summary(), "first matched")
	}
}

func TestClassify_SkipsNonMatchingClassifier(t *testing.T) {
	orig := stderrors.New("test error")

	noMatch := func(err error) (bool, *gherrors.ClassifiedError) {
		return false, nil
	}
	match := func(err error) (bool, *gherrors.ClassifiedError) {
		return true, gherrors.NewClassifiedError(gherrors.SeverityWarning, "matched", err)
	}

	result := gherrors.Classify(orig, noMatch, match)
	if result.Summary() != "matched" {
		t.Errorf("Summary() = %q, want %q", result.Summary(), "matched")
	}
}
