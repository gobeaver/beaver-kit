package filevalidator

import (
	"io"
)

// ContentValidator is an interface for validating file contents beyond MIME type
type ContentValidator interface {
	// ValidateContent validates the content of a file
	ValidateContent(reader io.Reader, size int64) error
	// SupportedMIMETypes returns the MIME types this validator can handle
	SupportedMIMETypes() []string
}

// ContentValidatorRegistry manages content validators for different file types
type ContentValidatorRegistry struct {
	validators map[string]ContentValidator
}

// NewContentValidatorRegistry creates a new content validator registry
func NewContentValidatorRegistry() *ContentValidatorRegistry {
	return &ContentValidatorRegistry{
		validators: make(map[string]ContentValidator),
	}
}

// Register registers a content validator for specific MIME types
func (r *ContentValidatorRegistry) Register(mimeType string, validator ContentValidator) {
	r.validators[mimeType] = validator
}

// GetValidator returns the validator for a given MIME type
func (r *ContentValidatorRegistry) GetValidator(mimeType string) ContentValidator {
	return r.validators[mimeType]
}

// ValidateContent validates content using the appropriate validator
func (r *ContentValidatorRegistry) ValidateContent(mimeType string, reader io.Reader, size int64) error {
	validator := r.GetValidator(mimeType)
	if validator == nil {
		// No validator for this MIME type, which is okay
		return nil
	}
	return validator.ValidateContent(reader, size)
}
