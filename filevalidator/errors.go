package filevalidator

import (
	"errors"
	"fmt"
)

// ValidationErrorType represents different types of validation errors
type ValidationErrorType string

const (
	ErrorTypeSize      ValidationErrorType = "size"
	ErrorTypeMIME      ValidationErrorType = "mime"
	ErrorTypeFileName  ValidationErrorType = "filename"
	ErrorTypeExtension ValidationErrorType = "extension"
	ErrorTypeContent   ValidationErrorType = "content"
)

// ValidationError represents a custom error for file validation
type ValidationError struct {
	Type    ValidationErrorType
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s validation error: %s", e.Type, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(errType ValidationErrorType, message string) *ValidationError {
	return &ValidationError{
		Type:    errType,
		Message: message,
	}
}

// IsValidationError checks if an error is a ValidationError
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsErrorOfType checks if an error is a ValidationError of the specified type
func IsErrorOfType(err error, errType ValidationErrorType) bool {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Type == errType
	}
	return false
}

// GetErrorType returns the type of a ValidationError, or empty string if not a ValidationError
func GetErrorType(err error) ValidationErrorType {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Type
	}
	return ""
}

// GetErrorMessage returns the message of a ValidationError, or empty string if not a ValidationError
func GetErrorMessage(err error) string {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Message
	}
	return ""
}
