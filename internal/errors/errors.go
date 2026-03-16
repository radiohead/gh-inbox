package errors

// Severity indicates how a caller should handle a classified error.
type Severity int

const (
	// SeverityCritical means the error is fatal — stop and return it to the caller.
	SeverityCritical Severity = iota
	// SeverityWarning means partial data is available — log a summary and continue.
	SeverityWarning
	// SeveritySilent means the error can be ignored silently.
	SeveritySilent
)

// ClassifiedError wraps an error with a severity level and a human-readable
// summary. It implements the error interface and supports unwrapping.
type ClassifiedError struct {
	severity Severity
	summary  string
	err      error
}

// NewClassifiedError creates a ClassifiedError with the given severity, summary,
// and underlying error.
func NewClassifiedError(severity Severity, summary string, err error) *ClassifiedError {
	return &ClassifiedError{severity: severity, summary: summary, err: err}
}

// Error implements the error interface and returns the human-readable summary.
func (e *ClassifiedError) Error() string { return e.summary }

// Unwrap returns the original error, enabling errors.Is / errors.As chains.
func (e *ClassifiedError) Unwrap() error { return e.err }

// Severity returns the classified severity level.
func (e *ClassifiedError) Severity() Severity { return e.severity }

// Summary returns the human-readable description of the error.
func (e *ClassifiedError) Summary() string { return e.summary }

// Classifier inspects an error and optionally returns a classification.
// Returns (true, result) if the classifier matched, (false, nil) otherwise.
type Classifier func(error) (bool, *ClassifiedError)

// Classify runs err through a chain of classifiers. The first matching
// classifier wins. If no classifier matches, err is wrapped as Critical with
// its original error message.
func Classify(err error, classifiers ...Classifier) *ClassifiedError {
	for _, classify := range classifiers {
		if matched, result := classify(err); matched {
			return result
		}
	}
	return NewClassifiedError(SeverityCritical, err.Error(), err)
}
