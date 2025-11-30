package filevalidator

import (
	"strings"
	"testing"
	"time"
)

func TestValidationResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidationResult
		contains string
	}{
		{
			name: "valid result",
			result: &ValidationResult{
				Valid:        true,
				Filename:     "test.png",
				Size:         1024,
				DetectedMIME: "image/png",
				Duration:     100 * time.Microsecond,
			},
			contains: "✓",
		},
		{
			name: "invalid result",
			result: &ValidationResult{
				Valid:    false,
				Filename: "test.exe",
				Errors: []ValidationError{
					{Type: ErrorTypeExtension, Message: "blocked extension"},
				},
			},
			contains: "✗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.Summary()
			if !strings.Contains(summary, tt.contains) {
				t.Errorf("Summary() = %q, want to contain %q", summary, tt.contains)
			}
		})
	}
}

func TestValidationResult_Error(t *testing.T) {
	t.Run("valid result returns nil", func(t *testing.T) {
		r := &ValidationResult{Valid: true}
		if r.Error() != nil {
			t.Error("expected nil error for valid result")
		}
	})

	t.Run("invalid result returns error", func(t *testing.T) {
		r := &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Type: ErrorTypeSize, Message: "too large"},
			},
		}
		err := r.Error()
		if err == nil {
			t.Error("expected error for invalid result")
		}
	})
}

func TestResultBuilder(t *testing.T) {
	builder := NewResultBuilder("test.pdf", 1024)

	result := builder.
		SetDetectedMIME("application/pdf").
		SetDeclaredMIME("application/pdf").
		AddCheck("filename", true, "filename is valid").
		AddCheck("size", true, "size within limits").
		AddWarning("file is large").
		Build()

	if !result.Valid {
		t.Error("expected valid result")
	}
	if result.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want %q", result.Filename, "test.pdf")
	}
	if result.Size != 1024 {
		t.Errorf("Size = %d, want %d", result.Size, 1024)
	}
	if result.DetectedMIME != "application/pdf" {
		t.Errorf("DetectedMIME = %q, want %q", result.DetectedMIME, "application/pdf")
	}
	if len(result.Checks) != 2 {
		t.Errorf("len(Checks) = %d, want %d", len(result.Checks), 2)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("len(Warnings) = %d, want %d", len(result.Warnings), 1)
	}
	if result.Duration == 0 {
		t.Error("Duration should be set")
	}
}

func TestResultBuilder_WithError(t *testing.T) {
	builder := NewResultBuilder("test.exe", 1024)

	result := builder.
		AddCheck("filename", true, "filename is valid").
		AddError(ErrorTypeExtension, "blocked extension").
		Build()

	if result.Valid {
		t.Error("expected invalid result")
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want %d", len(result.Errors), 1)
	}
}

func TestValidationResult_FailedChecks(t *testing.T) {
	result := &ValidationResult{
		Checks: []CheckResult{
			{Name: "size", Passed: true},
			{Name: "mime", Passed: false},
			{Name: "content", Passed: true},
			{Name: "filename", Passed: false},
		},
	}

	failed := result.FailedChecks()
	if len(failed) != 2 {
		t.Errorf("len(FailedChecks()) = %d, want %d", len(failed), 2)
	}

	passed := result.PassedChecks()
	if len(passed) != 2 {
		t.Errorf("len(PassedChecks()) = %d, want %d", len(passed), 2)
	}
}

func TestQuickResult(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		r := QuickResult("test.png", 1024, true, nil)
		if !r.Valid {
			t.Error("expected valid result")
		}
	})

	t.Run("invalid with error", func(t *testing.T) {
		err := NewValidationError(ErrorTypeSize, "too large")
		r := QuickResult("test.png", 999999999, false, err)
		if r.Valid {
			t.Error("expected invalid result")
		}
		if len(r.Errors) != 1 {
			t.Errorf("len(Errors) = %d, want %d", len(r.Errors), 1)
		}
	})
}
