// Package filevalidator provides comprehensive file validation functionality.
package filevalidator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// Validator provides the main interface for validating files
type Validator interface {
	// Validate validates a file against the validator's constraints
	Validate(file *multipart.FileHeader) error

	// ValidateWithContext validates a file with context for potential cancellation
	ValidateWithContext(ctx context.Context, file *multipart.FileHeader) error

	// ValidateReader validates a file from an io.Reader with a filename
	ValidateReader(reader io.Reader, filename string, size int64) error

	// ValidateBytes validates a file from a byte slice with a filename
	ValidateBytes(content []byte, filename string) error

	// GetConstraints returns the current validation constraints
	GetConstraints() Constraints
}

// FileValidator implements the Validator interface
type FileValidator struct {
	constraints Constraints
}

// New creates a new file validator with the given constraints
func New(constraints Constraints) *FileValidator {
	return &FileValidator{
		constraints: constraints,
	}
}

// NewDefault creates a new file validator with sensible default constraints
func NewDefault() *FileValidator {
	return &FileValidator{
		constraints: DefaultConstraints(),
	}
}

// Validate validates a file against the validator's constraints
func (v *FileValidator) Validate(file *multipart.FileHeader) error {
	return v.ValidateWithContext(context.Background(), file)
}

// ValidateWithContext validates a file with context for potential cancellation
func (v *FileValidator) ValidateWithContext(ctx context.Context, file *multipart.FileHeader) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue validation
	}

	// Validate filename first
	if err := v.validateFileName(file.Filename); err != nil {
		return err
	}

	// Get the file size from FileHeader
	fileSize := file.Size

	// Check if file size is within the allowed range
	if v.constraints.MaxFileSize > 0 && fileSize > v.constraints.MaxFileSize {
		return NewValidationError(ErrorTypeSize, fmt.Sprintf("file size too big: %d bytes (max: %d bytes)", fileSize, v.constraints.MaxFileSize))
	}

	if v.constraints.MinFileSize > 0 && fileSize < v.constraints.MinFileSize {
		return NewValidationError(ErrorTypeSize, fmt.Sprintf("file size too small: %d bytes (min: %d bytes)", fileSize, v.constraints.MinFileSize))
	}

	// Skip MIME validation if no accepted types are specified
	if len(v.constraints.AcceptedTypes) == 0 {
		return nil
	}

	// Open the file to detect its MIME type
	f, err := file.Open()
	if err != nil {
		return NewValidationError(ErrorTypeMIME, "failed to open file for MIME type detection")
	}
	defer f.Close()

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue validation
	}

	// Detect MIME type using enhanced magic bytes detection
	mimeType, err := DetectMIME(f)
	if err != nil {
		return err
	}

	// Validate MIME type against accepted types
	if !v.isAcceptedMIMEType(mimeType) {
		return NewValidationError(
			ErrorTypeMIME,
			fmt.Sprintf("file type %s is not accepted; allowed types: %v", mimeType, v.expandedAcceptedTypes()),
		)
	}

	// Strict MIME type validation: ensure extension matches detected MIME type
	if v.constraints.StrictMIMETypeValidation {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != "" {
			expectedMIME := MIMETypeForExtension(ext)
			if expectedMIME != "" && expectedMIME != mimeType {
				return NewValidationError(
					ErrorTypeMIME,
					fmt.Sprintf("MIME type mismatch: extension %s suggests %s but detected %s", ext, expectedMIME, mimeType),
				)
			}
		}
	}

	// Perform content validation if enabled
	if v.constraints.ContentValidationEnabled && v.constraints.ContentValidatorRegistry != nil {
		// Reset file reader for content validation
		if _, err := f.Seek(0, 0); err != nil {
			return NewValidationError(ErrorTypeContent, "failed to reset file reader for content validation")
		}

		if err := v.constraints.ContentValidatorRegistry.ValidateContent(mimeType, f, fileSize); err != nil {
			// If content validation is required, return the error
			if v.constraints.RequireContentValidation {
				return err
			}
			// Otherwise, content validation failures are warnings (logged but not blocking)
		}
	}

	return nil
}

// ValidateReader validates a file from an io.Reader with a filename
func (v *FileValidator) ValidateReader(reader io.Reader, filename string, size int64) error {
	// Validate filename first
	if err := v.validateFileName(filename); err != nil {
		return err
	}

	// Check file size if provided
	if size > 0 {
		if v.constraints.MaxFileSize > 0 && size > v.constraints.MaxFileSize {
			return NewValidationError(ErrorTypeSize, fmt.Sprintf("file size too big: %d bytes (max: %d bytes)", size, v.constraints.MaxFileSize))
		}

		if v.constraints.MinFileSize > 0 && size < v.constraints.MinFileSize {
			return NewValidationError(ErrorTypeSize, fmt.Sprintf("file size too small: %d bytes (min: %d bytes)", size, v.constraints.MinFileSize))
		}
	}

	// Skip MIME validation if no accepted types are specified
	if len(v.constraints.AcceptedTypes) == 0 {
		return nil
	}

	// Detect MIME type from reader
	// We need to peek at the beginning of the file without consuming the reader
	if seekable, ok := reader.(io.Seeker); ok {
		// Try to detect MIME type if the reader is also a seeker
		oldPos, err := seekable.Seek(0, io.SeekCurrent)
		if err != nil {
			return NewValidationError(ErrorTypeMIME, "failed to get current position in reader")
		}

		mimeType, err := DetectMIME(reader)
		if err != nil {
			return err
		}

		// Reset the reader position
		_, err = seekable.Seek(oldPos, io.SeekStart)
		if err != nil {
			return NewValidationError(ErrorTypeMIME, "failed to reset reader position after MIME detection")
		}

		// Validate MIME type against accepted types
		if !v.isAcceptedMIMEType(mimeType) {
			return NewValidationError(
				ErrorTypeMIME,
				fmt.Sprintf("file type %s is not accepted; allowed types: %v", mimeType, v.expandedAcceptedTypes()),
			)
		}

		// Strict MIME type validation: ensure extension matches detected MIME type
		if v.constraints.StrictMIMETypeValidation {
			ext := strings.ToLower(filepath.Ext(filename))
			if ext != "" {
				expectedMIME := MIMETypeForExtension(ext)
				if expectedMIME != "" && expectedMIME != mimeType {
					return NewValidationError(
						ErrorTypeMIME,
						fmt.Sprintf("MIME type mismatch: extension %s suggests %s but detected %s", ext, expectedMIME, mimeType),
					)
				}
			}
		}

		// Perform content validation if enabled
		if v.constraints.ContentValidationEnabled && v.constraints.ContentValidatorRegistry != nil {
			// Reset reader position for content validation
			_, err = seekable.Seek(0, io.SeekStart)
			if err != nil {
				return NewValidationError(ErrorTypeContent, "failed to reset reader position for content validation")
			}

			if err := v.constraints.ContentValidatorRegistry.ValidateContent(mimeType, reader, size); err != nil {
				// If content validation is required, return the error
				if v.constraints.RequireContentValidation {
					return err
				}
				// Otherwise, content validation failures are warnings (logged but not blocking)
			}
		}
	} else {
		// If we can't seek, we have to detect based on the file extension
		ext := strings.ToLower(filepath.Ext(filename))
		if !v.isAcceptedExtension(ext) {
			return NewValidationError(
				ErrorTypeExtension,
				fmt.Sprintf("file extension %s is not allowed", ext),
			)
		}
	}

	return nil
}

// ValidateBytes validates a file from a byte slice with a filename
func (v *FileValidator) ValidateBytes(content []byte, filename string) error {
	// Create a reader from the byte slice
	reader := bytes.NewReader(content)

	// Validate using the reader
	return v.ValidateReader(reader, filename, int64(len(content)))
}

// GetConstraints returns the current validation constraints
func (v *FileValidator) GetConstraints() Constraints {
	return v.constraints
}

// validateFileName validates a filename against the validator's constraints
func (v *FileValidator) validateFileName(filename string) error {
	if len(filename) == 0 {
		return NewValidationError(ErrorTypeFileName, "empty filename")
	}

	// Check filename length
	if v.constraints.MaxNameLength > 0 && len(filename) > v.constraints.MaxNameLength {
		return NewValidationError(
			ErrorTypeFileName,
			fmt.Sprintf("filename exceeds maximum length of %d characters", v.constraints.MaxNameLength),
		)
	}

	// Check for potentially dangerous characters
	for _, char := range v.constraints.DangerousChars {
		if strings.Contains(filename, char) {
			return NewValidationError(
				ErrorTypeFileName,
				fmt.Sprintf("filename contains invalid character: %s", char),
			)
		}
	}

	// Optional regex validation
	if v.constraints.FileNameRegex != nil {
		if !v.constraints.FileNameRegex.MatchString(filename) {
			return NewValidationError(
				ErrorTypeFileName,
				"filename doesn't match the required pattern",
			)
		}
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) == 0 && v.constraints.RequireExtension {
		return NewValidationError(ErrorTypeExtension, "file must have an extension")
	}

	// Check blocked extensions first
	for _, blockedExt := range v.constraints.BlockedExts {
		if strings.EqualFold(ext, blockedExt) {
			return NewValidationError(
				ErrorTypeExtension,
				fmt.Sprintf("file extension %s is blocked", ext),
			)
		}
	}

	// If allowed extensions are specified, check against them
	if len(v.constraints.AllowedExts) > 0 {
		if !v.isAcceptedExtension(ext) {
			return NewValidationError(
				ErrorTypeExtension,
				fmt.Sprintf("file extension %s is not allowed", ext),
			)
		}
	}

	return nil
}

// isAcceptedMIMEType checks if a MIME type is accepted by the validator
func (v *FileValidator) isAcceptedMIMEType(mimeType string) bool {
	expandedTypes := v.expandedAcceptedTypes()
	for _, acceptedType := range expandedTypes {
		if acceptedType == mimeType || acceptedType == "*/*" {
			return true
		}

		// Handle wildcards like "image/*"
		if strings.HasSuffix(acceptedType, "/*") {
			prefix := strings.TrimSuffix(acceptedType, "/*")
			if strings.HasPrefix(mimeType, prefix+"/") {
				return true
			}
		}
	}
	return false
}

// isAcceptedExtension checks if a file extension is accepted by the validator
func (v *FileValidator) isAcceptedExtension(ext string) bool {
	// If no allowed extensions specified, all are allowed (unless blocked)
	if len(v.constraints.AllowedExts) == 0 {
		return true
	}

	for _, allowedExt := range v.constraints.AllowedExts {
		if strings.EqualFold(ext, allowedExt) {
			return true
		}
	}
	return false
}

// expandedAcceptedTypes expands all MediaTypeGroup wildcards in AcceptedTypes
func (v *FileValidator) expandedAcceptedTypes() []string {
	return ExpandAcceptedTypes(v.constraints.AcceptedTypes)
}
