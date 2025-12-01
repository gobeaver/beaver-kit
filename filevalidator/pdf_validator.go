package filevalidator

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// PDFValidator validates PDF file structure (header/trailer).
// This is TYPE validation, not security scanning.
// For malware detection, integrate with ClamAV or similar.
type PDFValidator struct {
	MaxSize int64
}

// DefaultPDFValidator creates a PDF validator with sensible defaults
func DefaultPDFValidator() *PDFValidator {
	return &PDFValidator{
		MaxSize: 50 * MB,
	}
}

// ValidateContent validates that a file is a valid PDF by checking header and trailer.
// Only reads first 1KB and last 1KB - does NOT load entire file into memory.
func (v *PDFValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("PDF size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Try to use seeking for efficient validation
	if seeker, ok := reader.(io.ReadSeeker); ok {
		return v.validateWithSeeker(seeker, size)
	}

	// Fallback: read only what we need for small files, reject large non-seekable streams
	if size > 1*MB {
		return NewValidationError(ErrorTypeContent,
			"large PDF requires seekable reader for efficient validation")
	}

	return v.validateSmallFile(reader, size)
}

// validateWithSeeker efficiently validates PDF by reading only header and trailer
func (v *PDFValidator) validateWithSeeker(reader io.ReadSeeker, size int64) error {
	// Read header (first 1KB)
	headerSize := int64(1024)
	if size < headerSize {
		headerSize = size
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read PDF header")
	}

	if !v.hasValidPDFHeader(header) {
		return NewValidationError(ErrorTypeContent, "invalid PDF header")
	}

	// Read trailer (last 1KB)
	tailSize := int64(1024)
	if size < tailSize {
		tailSize = size
	}
	if _, err := reader.Seek(-tailSize, io.SeekEnd); err != nil {
		return NewValidationError(ErrorTypeContent, "failed to seek to PDF trailer")
	}

	trailer := make([]byte, tailSize)
	if _, err := io.ReadFull(reader, trailer); err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read PDF trailer")
	}

	if !v.hasValidPDFTrailer(trailer) {
		return NewValidationError(ErrorTypeContent, "invalid PDF trailer")
	}

	return nil
}

// validateSmallFile handles non-seekable readers for small files
func (v *PDFValidator) validateSmallFile(reader io.Reader, size int64) error {
	// Pre-allocate buffer based on known size for efficiency
	data := make([]byte, 0, size)
	buf := bytes.NewBuffer(data)
	if _, err := buf.ReadFrom(reader); err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read PDF content")
	}

	if !v.hasValidPDFHeader(buf.Bytes()) {
		return NewValidationError(ErrorTypeContent, "invalid PDF header")
	}

	if !v.hasValidPDFTrailer(buf.Bytes()) {
		return NewValidationError(ErrorTypeContent, "invalid PDF trailer")
	}

	return nil
}

// SupportedMIMETypes returns the MIME types this validator can handle
func (v *PDFValidator) SupportedMIMETypes() []string {
	return []string{
		"application/pdf",
		"application/x-pdf",
		"application/vnd.pdf",
	}
}

// hasValidPDFHeader checks if the data has a valid PDF header
func (v *PDFValidator) hasValidPDFHeader(data []byte) bool {
	// PDF files should start with %PDF-x.x
	if len(data) < 8 {
		return false
	}

	header := string(data[:8])
	return strings.HasPrefix(header, "%PDF-")
}

// hasValidPDFTrailer checks if the data has a valid PDF trailer
func (v *PDFValidator) hasValidPDFTrailer(data []byte) bool {
	// PDF files should end with %%EOF
	if len(data) < 5 {
		return false
	}

	return bytes.Contains(data, []byte("%%EOF"))
}
