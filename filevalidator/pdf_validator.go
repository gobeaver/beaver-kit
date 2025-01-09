package filevalidator

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

// PDFValidator validates PDF files for malicious content
type PDFValidator struct {
	AllowJavaScript    bool
	AllowEmbeddedFiles bool
	AllowForms         bool
	AllowActions       bool
	MaxSize            int64
	ValidateStructure  bool
}

// DefaultPDFValidator creates a PDF validator with secure defaults
func DefaultPDFValidator() *PDFValidator {
	return &PDFValidator{
		AllowJavaScript:    false,
		AllowEmbeddedFiles: false,
		AllowForms:         true,
		AllowActions:       false,
		MaxSize:            50 * MB,
		ValidateStructure:  true,
	}
}

// ValidateContent validates the content of a PDF file
func (v *PDFValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("PDF size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Read the entire content
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read PDF content")
	}

	// Validate PDF header
	if !v.hasValidPDFHeader(data) {
		return NewValidationError(ErrorTypeContent, "invalid PDF header")
	}

	// Validate PDF trailer
	if !v.hasValidPDFTrailer(data) {
		return NewValidationError(ErrorTypeContent, "invalid PDF trailer")
	}

	// Check for JavaScript
	if !v.AllowJavaScript && v.containsJavaScript(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains JavaScript which is not allowed")
	}

	// Check for embedded files
	if !v.AllowEmbeddedFiles && v.containsEmbeddedFiles(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains embedded files which are not allowed")
	}

	// Check for forms
	if !v.AllowForms && v.containsForms(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains forms which are not allowed")
	}

	// Check for actions
	if !v.AllowActions && v.containsActions(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains actions which are not allowed")
	}

	// Check for launch actions (always dangerous)
	if v.containsLaunchActions(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains launch actions which are always blocked")
	}

	// Check for suspicious patterns
	if v.containsSuspiciousPatterns(data) {
		return NewValidationError(ErrorTypeContent, "PDF contains suspicious patterns")
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

	// Look for %%EOF in the last 1024 bytes
	tailSize := min(len(data), 1024)
	tail := data[len(data)-tailSize:]

	return bytes.Contains(tail, []byte("%%EOF"))
}

// containsJavaScript checks for JavaScript in the PDF
func (v *PDFValidator) containsJavaScript(data []byte) bool {
	patterns := [][]byte{
		[]byte("/JavaScript"),
		[]byte("/JS"),
		[]byte("app.alert"),
		[]byte("app.launchURL"),
		[]byte("this.exportDataObject"),
		[]byte("util.printf"),
	}

	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	return false
}

// containsEmbeddedFiles checks for embedded files in the PDF
func (v *PDFValidator) containsEmbeddedFiles(data []byte) bool {
	patterns := [][]byte{
		[]byte("/EmbeddedFiles"),
		[]byte("/Filespec"),
		[]byte("/EmbeddedFile"),
	}

	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	return false
}

// containsForms checks for forms in the PDF
func (v *PDFValidator) containsForms(data []byte) bool {
	patterns := [][]byte{
		[]byte("/AcroForm"),
		[]byte("/XFA"),
		[]byte("/Field"),
	}

	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	return false
}

// containsActions checks for actions in the PDF
func (v *PDFValidator) containsActions(data []byte) bool {
	patterns := [][]byte{
		[]byte("/Action"),
		[]byte("/OpenAction"),
		[]byte("/AA"), // Additional Actions
		[]byte("/Named"),
		[]byte("/SubmitForm"),
		[]byte("/ImportData"),
		[]byte("/ResetForm"),
		[]byte("/Hide"),
	}

	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	return false
}

// containsLaunchActions checks for launch actions (always dangerous)
func (v *PDFValidator) containsLaunchActions(data []byte) bool {
	patterns := [][]byte{
		[]byte("/Launch"),
		[]byte("/GoToR"), // GoTo remote
		[]byte("/URI"),
		[]byte("/GoToE"), // GoTo embedded
	}

	for _, pattern := range patterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	return false
}

// containsSuspiciousPatterns checks for various suspicious patterns
func (v *PDFValidator) containsSuspiciousPatterns(data []byte) bool {
	// Check for suspicious URLs
	urlPattern := regexp.MustCompile(`https?://[a-zA-Z0-9\-\.]+\.(tk|ml|ga|cf|pw|cc|su|bid|download|stream)`)
	if urlPattern.Match(data) {
		return true
	}

	// Check for suspicious executables
	execPatterns := [][]byte{
		[]byte(".exe"),
		[]byte(".bat"),
		[]byte(".cmd"),
		[]byte(".com"),
		[]byte(".scr"),
		[]byte(".vbs"),
		[]byte(".ps1"),
		[]byte("cmd.exe"),
		[]byte("powershell.exe"),
		[]byte("wscript.exe"),
		[]byte("cscript.exe"),
	}

	for _, pattern := range execPatterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}

	// Check for obfuscated content
	obfuscationPatterns := [][]byte{
		[]byte("#"), // Hex strings
		[]byte("/ASCIIHexDecode"),
		[]byte("/ASCII85Decode"),
		[]byte("/FlateDecode"),
		[]byte("/LZWDecode"),
		[]byte("/RunLengthDecode"),
	}

	obfuscationCount := 0
	for _, pattern := range obfuscationPatterns {
		if bytes.Contains(data, pattern) {
			obfuscationCount++
		}
	}

	// Too much obfuscation is suspicious
	if obfuscationCount > 3 {
		return true
	}

	return false
}
