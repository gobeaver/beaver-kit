package filevalidator

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ValidationResult contains detailed information about a validation attempt
type ValidationResult struct {
	// Valid indicates whether the file passed all validations
	Valid bool

	// Filename is the name of the validated file
	Filename string

	// Size is the file size in bytes
	Size int64

	// DetectedMIME is the MIME type detected from file content
	DetectedMIME string

	// DeclaredMIME is the MIME type from the file extension (if available)
	DeclaredMIME string

	// Errors contains all validation errors encountered
	Errors []ValidationError

	// Warnings contains non-blocking issues (e.g., MIME mismatch when not strict)
	Warnings []string

	// Duration is how long validation took
	Duration time.Duration

	// Checks contains details about each validation check performed
	Checks []CheckResult
}

// CheckResult represents the result of a single validation check
type CheckResult struct {
	Name    string        // e.g., "size", "mime", "content", "filename"
	Passed  bool          // whether this check passed
	Message string        // human-readable result
	Details string        // additional details (optional)
	Took    time.Duration // how long this check took
}

// Error returns a combined error message if validation failed, nil if valid
func (r *ValidationResult) Error() error {
	if r.Valid {
		return nil
	}
	if len(r.Errors) == 0 {
		return nil
	}
	return &r.Errors[0]
}

// AllErrors returns all errors as a single combined error
func (r *ValidationResult) AllErrors() error {
	if r.Valid || len(r.Errors) == 0 {
		return nil
	}

	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = e.Message
	}
	return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
}

// Summary returns a human-readable summary of the validation
func (r *ValidationResult) Summary() string {
	if r.Valid {
		return fmt.Sprintf("✓ %s (%s, %s) validated in %v",
			r.Filename,
			r.DetectedMIME,
			FormatSizeReadable(r.Size),
			r.Duration.Round(time.Microsecond),
		)
	}

	return fmt.Sprintf("✗ %s failed: %s",
		r.Filename,
		r.Errors[0].Message,
	)
}

// HasWarnings returns true if there are any warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// FailedChecks returns only the checks that failed
func (r *ValidationResult) FailedChecks() []CheckResult {
	var failed []CheckResult
	for _, check := range r.Checks {
		if !check.Passed {
			failed = append(failed, check)
		}
	}
	return failed
}

// PassedChecks returns only the checks that passed
func (r *ValidationResult) PassedChecks() []CheckResult {
	var passed []CheckResult
	for _, check := range r.Checks {
		if check.Passed {
			passed = append(passed, check)
		}
	}
	return passed
}

// ResultBuilder helps construct ValidationResult
type ResultBuilder struct {
	result    ValidationResult
	startTime time.Time
}

// NewResultBuilder creates a new result builder
func NewResultBuilder(filename string, size int64) *ResultBuilder {
	return &ResultBuilder{
		result: ValidationResult{
			Valid:    true, // Assume valid until proven otherwise
			Filename: filename,
			Size:     size,
			Checks:   make([]CheckResult, 0),
		},
		startTime: time.Now(),
	}
}

// SetDetectedMIME sets the detected MIME type
func (b *ResultBuilder) SetDetectedMIME(mime string) *ResultBuilder {
	b.result.DetectedMIME = mime
	return b
}

// SetDeclaredMIME sets the declared MIME type (from extension)
func (b *ResultBuilder) SetDeclaredMIME(mime string) *ResultBuilder {
	b.result.DeclaredMIME = mime
	return b
}

// AddCheck adds a check result
func (b *ResultBuilder) AddCheck(name string, passed bool, message string) *ResultBuilder {
	b.result.Checks = append(b.result.Checks, CheckResult{
		Name:    name,
		Passed:  passed,
		Message: message,
	})
	if !passed {
		b.result.Valid = false
	}
	return b
}

// AddCheckWithDetails adds a check result with additional details
func (b *ResultBuilder) AddCheckWithDetails(name string, passed bool, message, details string, took time.Duration) *ResultBuilder {
	b.result.Checks = append(b.result.Checks, CheckResult{
		Name:    name,
		Passed:  passed,
		Message: message,
		Details: details,
		Took:    took,
	})
	if !passed {
		b.result.Valid = false
	}
	return b
}

// AddError adds an error and marks result as invalid
func (b *ResultBuilder) AddError(errType ValidationErrorType, message string) *ResultBuilder {
	b.result.Valid = false
	b.result.Errors = append(b.result.Errors, ValidationError{
		Type:    errType,
		Message: message,
	})
	return b
}

// AddWarning adds a warning (non-blocking)
func (b *ResultBuilder) AddWarning(message string) *ResultBuilder {
	b.result.Warnings = append(b.result.Warnings, message)
	return b
}

// Build finalizes and returns the ValidationResult
func (b *ResultBuilder) Build() *ValidationResult {
	b.result.Duration = time.Since(b.startTime)
	return &b.result
}

// QuickResult creates a simple pass/fail result without detailed checks
func QuickResult(filename string, size int64, valid bool, err error) *ValidationResult {
	result := &ValidationResult{
		Valid:    valid,
		Filename: filename,
		Size:     size,
	}
	if err != nil {
		var vErr *ValidationError
		if errors.As(err, &vErr) {
			result.Errors = append(result.Errors, *vErr)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Type:    ErrorTypeContent,
				Message: err.Error(),
			})
		}
	}
	return result
}
