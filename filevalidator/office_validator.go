package filevalidator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
)

// OfficeValidator validates Microsoft Office Open XML formats (DOCX, XLSX, PPTX).
// These are ZIP-based formats with specific required files inside.
type OfficeValidator struct {
	// Shared settings (inherits zip bomb protection)
	MaxSize             int64
	MaxFiles            int
	MaxUncompressedSize int64
	MaxCompressionRatio float64

	// Office-specific settings
	AllowMacros bool // Allow macro-enabled formats (.docm, .xlsm, .pptm)
}

// DefaultOfficeValidator creates an office validator with sensible defaults
func DefaultOfficeValidator() *OfficeValidator {
	return &OfficeValidator{
		MaxSize:             100 * MB,
		MaxFiles:            10000,
		MaxUncompressedSize: 1 * GB,
		MaxCompressionRatio: 100.0,
		AllowMacros:         false, // Macros disabled by default for security
	}
}

// ValidateContent validates Office documents by checking ZIP structure and required files
func (v *OfficeValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Try to use ReaderAt for memory-efficient validation
	if readerAt, ok := reader.(io.ReaderAt); ok {
		return v.validateWithReaderAt(readerAt, size)
	}

	// Fallback for small files
	if size > 1*MB {
		return NewValidationError(ErrorTypeContent,
			"large Office files require seekable reader for efficient validation")
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read file content")
	}

	return v.validateWithReaderAt(bytes.NewReader(data), int64(len(data)))
}

func (v *OfficeValidator) validateWithReaderAt(reader io.ReaderAt, size int64) error {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return NewValidationError(ErrorTypeContent, fmt.Sprintf("invalid ZIP structure: %v", err))
	}

	var (
		totalUncompressed uint64
		fileCount         int
		hasContentTypes   bool
		hasRels           bool
		hasMacros         bool
		docType           string
	)

	for _, file := range zipReader.File {
		fileCount++

		if fileCount > v.MaxFiles {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("too many files in archive: %d (max: %d)", fileCount, v.MaxFiles))
		}

		// Check compression ratio per file
		if file.CompressedSize64 > 0 {
			ratio := float64(file.UncompressedSize64) / float64(file.CompressedSize64)
			if ratio > v.MaxCompressionRatio {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("suspicious compression ratio: %.2f:1", ratio))
			}
		}

		totalUncompressed += file.UncompressedSize64
		if v.MaxUncompressedSize > 0 && totalUncompressed > uint64(v.MaxUncompressedSize) { //nolint:gosec // MaxUncompressedSize is validated to be positive
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("uncompressed size exceeds limit: %d", v.MaxUncompressedSize))
		}

		// Check for required Office files
		switch file.Name {
		case "[Content_Types].xml":
			hasContentTypes = true
		case "_rels/.rels":
			hasRels = true
		}

		// Detect document type
		if docType == "" {
			docType = v.detectDocType(file.Name)
		}

		// Check for macros (VBA)
		if v.isMacroFile(file.Name) {
			hasMacros = true
		}
	}

	// Validate required files exist
	if !hasContentTypes {
		return NewValidationError(ErrorTypeContent, "missing [Content_Types].xml - not a valid Office document")
	}
	if !hasRels {
		return NewValidationError(ErrorTypeContent, "missing _rels/.rels - not a valid Office document")
	}

	// Check macros policy
	if hasMacros && !v.AllowMacros {
		return NewValidationError(ErrorTypeContent, "macro-enabled documents are not allowed")
	}

	return nil
}

// detectDocType identifies the Office document type from internal paths
func (v *OfficeValidator) detectDocType(path string) string {
	switch {
	case len(path) > 5 && path[:5] == "word/":
		return "docx"
	case len(path) > 3 && path[:3] == "xl/":
		return "xlsx"
	case len(path) > 4 && path[:4] == "ppt/":
		return "pptx"
	}
	return ""
}

// isMacroFile checks if a file path indicates VBA macros
func (v *OfficeValidator) isMacroFile(path string) bool {
	macroIndicators := []string{
		"vbaProject.bin",
		"vbaData.xml",
		"word/vbaProject.bin",
		"xl/vbaProject.bin",
		"ppt/vbaProject.bin",
	}
	for _, indicator := range macroIndicators {
		if path == indicator {
			return true
		}
	}
	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *OfficeValidator) SupportedMIMETypes() []string {
	types := []string{
		// Word
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document", // .docx
		// Excel
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", // .xlsx
		// PowerPoint
		"application/vnd.openxmlformats-officedocument.presentationml.presentation", // .pptx
	}

	if v.AllowMacros {
		types = append(types,
			"application/vnd.ms-word.document.macroEnabled.12",           // .docm
			"application/vnd.ms-excel.sheet.macroEnabled.12",             // .xlsm
			"application/vnd.ms-powerpoint.presentation.macroEnabled.12", // .pptm
		)
	}

	return types
}
