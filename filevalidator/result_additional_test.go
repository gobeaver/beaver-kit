package filevalidator

import (
	"errors"
	"testing"
	"time"
)

func TestValidationResult_AllErrors(t *testing.T) {
	tests := []struct {
		name    string
		result  *ValidationResult
		wantErr bool
	}{
		{
			name: "Valid result",
			result: &ValidationResult{
				Valid: true,
			},
			wantErr: false,
		},
		{
			name: "Multiple errors",
			result: &ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{Type: ErrorTypeSize, Message: "too large"},
					{Type: ErrorTypeMIME, Message: "wrong type"},
				},
			},
			wantErr: true,
		},
		{
			name: "No errors",
			result: &ValidationResult{
				Valid:  false,
				Errors: []ValidationError{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.AllErrors()
			if (err != nil) != tt.wantErr {
				t.Errorf("AllErrors() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidationResult
		expected bool
	}{
		{
			name: "No warnings",
			result: &ValidationResult{
				Warnings: []string{},
			},
			expected: false,
		},
		{
			name: "Has warnings",
			result: &ValidationResult{
				Warnings: []string{"warning 1", "warning 2"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.HasWarnings()
			if result != tt.expected {
				t.Errorf("HasWarnings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResultBuilder_AddCheckWithDetails(t *testing.T) {
	builder := NewResultBuilder("test.txt", 1000)
	builder.AddCheckWithDetails("size", true, "size ok", "1000 bytes", 100*time.Microsecond)

	result := builder.Build()
	if len(result.Checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(result.Checks))
	}

	check := result.Checks[0]
	if check.Name != "size" {
		t.Errorf("Check name = %s, want size", check.Name)
	}
	if check.Details != "1000 bytes" {
		t.Errorf("Check details = %s, want 1000 bytes", check.Details)
	}
	if check.Took != 100*time.Microsecond {
		t.Errorf("Check took = %v, want 100µs", check.Took)
	}
}

func TestResultBuilder_AddCheckWithDetails_Failed(t *testing.T) {
	builder := NewResultBuilder("test.txt", 1000)
	builder.AddCheckWithDetails("size", false, "size too large", "2000 bytes", 100*time.Microsecond)

	result := builder.Build()
	if result.Valid {
		t.Error("Result should be invalid when check fails")
	}
}

func TestQuickResult_WithValidationError(t *testing.T) {
	err := NewValidationError(ErrorTypeSize, "file too large")
	result := QuickResult("test.txt", 1000, false, err)

	if result.Valid {
		t.Error("Result should be invalid")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Type != ErrorTypeSize {
		t.Errorf("Error type = %v, want %v", result.Errors[0].Type, ErrorTypeSize)
	}
}

func TestQuickResult_WithRegularError(t *testing.T) {
	err := errors.New("regular error")
	result := QuickResult("test.txt", 1000, false, err)

	if result.Valid {
		t.Error("Result should be invalid")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Type != ErrorTypeContent {
		t.Errorf("Error type = %v, want %v", result.Errors[0].Type, ErrorTypeContent)
	}
}

func TestQuickResult_Valid(t *testing.T) {
	result := QuickResult("test.txt", 1000, true, nil)

	if !result.Valid {
		t.Error("Result should be valid")
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

func TestResultBuilder_AddCheck_Failed(t *testing.T) {
	builder := NewResultBuilder("test.txt", 1000)
	builder.AddCheck("size", false, "size too large")

	result := builder.Build()
	if result.Valid {
		t.Error("Result should be invalid when check fails")
	}
}
