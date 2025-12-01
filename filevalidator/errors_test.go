package filevalidator

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError(ErrorTypeSize, "file too large")
	expected := "size validation error: file too large"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ValidationError",
			err:      NewValidationError(ErrorTypeSize, "test"),
			expected: true,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsErrorOfType(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		errorType ValidationErrorType
		expected  bool
	}{
		{
			name:      "Matching type",
			err:       NewValidationError(ErrorTypeSize, "test"),
			errorType: ErrorTypeSize,
			expected:  true,
		},
		{
			name:      "Non-matching type",
			err:       NewValidationError(ErrorTypeSize, "test"),
			errorType: ErrorTypeMIME,
			expected:  false,
		},
		{
			name:      "Regular error",
			err:       errors.New("regular error"),
			errorType: ErrorTypeSize,
			expected:  false,
		},
		{
			name:      "Nil error",
			err:       nil,
			errorType: ErrorTypeSize,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrorOfType(tt.err, tt.errorType)
			if result != tt.expected {
				t.Errorf("IsErrorOfType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ValidationErrorType
	}{
		{
			name:     "Size error",
			err:      NewValidationError(ErrorTypeSize, "test"),
			expected: ErrorTypeSize,
		},
		{
			name:     "MIME error",
			err:      NewValidationError(ErrorTypeMIME, "test"),
			expected: ErrorTypeMIME,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: "",
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorType(tt.err)
			if result != tt.expected {
				t.Errorf("GetErrorType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ValidationError",
			err:      NewValidationError(ErrorTypeSize, "file too large"),
			expected: "file too large",
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: "",
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorMessage(tt.err)
			if result != tt.expected {
				t.Errorf("GetErrorMessage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError(ErrorTypeContent, "invalid content")
	if err.Type != ErrorTypeContent {
		t.Errorf("Type = %v, want %v", err.Type, ErrorTypeContent)
	}
	if err.Message != "invalid content" {
		t.Errorf("Message = %v, want %v", err.Message, "invalid content")
	}
}
