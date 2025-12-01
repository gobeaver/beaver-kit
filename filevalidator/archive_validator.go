package filevalidator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ArchiveValidator validates ZIP archive files for zip bombs and path traversal.
// Currently only supports ZIP format (including .jar, .war, .ear which are ZIP-based).
// For other formats (RAR, 7z, TAR), use dedicated libraries.
type ArchiveValidator struct {
	MaxCompressionRatio float64
	MaxFiles            int
	MaxUncompressedSize int64
	MaxNestedArchives   int
}

// DefaultArchiveValidator creates an archive validator with sensible defaults
func DefaultArchiveValidator() *ArchiveValidator {
	return &ArchiveValidator{
		MaxCompressionRatio: 100.0, // 100:1 compression ratio max
		MaxFiles:            1000,
		MaxUncompressedSize: 1 * GB,
		MaxNestedArchives:   3,
	}
}

// ValidateContent validates the content of an archive file.
// For efficient validation, pass a reader that implements io.ReaderAt (e.g., *os.File, *bytes.Reader).
// Non-seekable readers are only supported for small files (<1MB).
func (v *ArchiveValidator) ValidateContent(reader io.Reader, size int64) error {
	// Try to use ReaderAt for memory-efficient validation
	if readerAt, ok := reader.(io.ReaderAt); ok {
		return v.validateWithReaderAt(readerAt, size)
	}

	// Fallback for non-seekable readers - only allow small files
	if size > 1*MB {
		return NewValidationError(ErrorTypeContent,
			"large ZIP files require seekable reader (e.g., *os.File) for efficient validation")
	}

	// Read small file into memory
	data, err := io.ReadAll(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read archive content")
	}

	return v.validateWithReaderAt(bytes.NewReader(data), int64(len(data)))
}

// validateWithReaderAt validates ZIP without loading entire file into memory
func (v *ArchiveValidator) validateWithReaderAt(reader io.ReaderAt, size int64) error {
	// zip.NewReader reads only the central directory (at end of file)
	// It does NOT load the entire archive into memory
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return NewValidationError(ErrorTypeContent, fmt.Sprintf("cannot open archive: %v", err))
	}

	var totalUncompressedSize uint64
	fileCount := 0
	nestedArchives := 0

	// Check each file in the archive
	for _, file := range zipReader.File {
		fileCount++

		// Check file count limit
		if fileCount > v.MaxFiles {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("archive contains too many files: %d (max: %d)", fileCount, v.MaxFiles))
		}

		// Check for nested archives
		if v.isArchive(file.Name) {
			nestedArchives++
			if nestedArchives > v.MaxNestedArchives {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("too many nested archives: %d (max: %d)", nestedArchives, v.MaxNestedArchives))
			}
		}

		// Calculate compression ratio and total size
		if file.CompressedSize64 > 0 {
			ratio := float64(file.UncompressedSize64) / float64(file.CompressedSize64)
			if ratio > v.MaxCompressionRatio {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("suspicious compression ratio for %s: %.2f:1 (max: %.2f:1)",
						file.Name, ratio, v.MaxCompressionRatio))
			}
		}

		totalUncompressedSize += file.UncompressedSize64

		// Check if we've exceeded the total uncompressed size limit
		if v.MaxUncompressedSize > 0 && totalUncompressedSize > uint64(v.MaxUncompressedSize) { //nolint:gosec // MaxUncompressedSize is validated to be positive
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("archive would expand to %d bytes (max: %d bytes)",
					totalUncompressedSize, v.MaxUncompressedSize))
		}

		// Check directory traversal
		if v.isDangerousPath(file.Name) {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("dangerous path detected: %s", file.Name))
		}
	}

	// Additional check: total compression ratio
	if totalUncompressedSize > 0 && size > 0 {
		totalRatio := float64(totalUncompressedSize) / float64(size)
		if totalRatio > v.MaxCompressionRatio {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("archive has suspicious total compression ratio: %.2f:1", totalRatio))
		}
	}

	return nil
}

// SupportedMIMETypes returns the MIME types this validator can handle.
// Only ZIP-based formats are actually validated.
func (v *ArchiveValidator) SupportedMIMETypes() []string {
	return []string{
		"application/zip",
		"application/x-zip-compressed",
		"application/java-archive", // JAR (ZIP-based)
	}
}

// zipExtensions are the extensions we recognize as ZIP-based archives
var zipExtensions = []string{".zip", ".jar", ".war", ".ear"}

// isArchive checks if a filename indicates a ZIP-based archive
func (v *ArchiveValidator) isArchive(filename string) bool {
	for _, ext := range zipExtensions {
		if hasExtension(filename, ext) {
			return true
		}
	}
	return false
}

// isDangerousPath checks for directory traversal attempts
func (v *ArchiveValidator) isDangerousPath(path string) bool {
	dangerous := []string{
		"..",
		"../",
		"..\\",
		"/etc/",
		"/sys/",
		"/proc/",
		"/dev/",
		"C:\\Windows\\",
		"C:\\System32\\",
		"~",
	}

	for _, pattern := range dangerous {
		if containsPattern(path, pattern) {
			return true
		}
	}

	// Check for absolute paths
	if isAbsolutePath(path) {
		return true
	}

	return false
}

// hasExtension checks if a filename has a given extension (case-insensitive)
func hasExtension(filename, ext string) bool {
	if len(filename) < len(ext) {
		return false
	}
	suffix := strings.ToLower(filename[len(filename)-len(ext):])
	return suffix == strings.ToLower(ext)
}

// containsPattern checks if a path contains a dangerous pattern
func containsPattern(path, pattern string) bool {
	// Simple contains check - could be improved with proper path parsing
	return bytes.Contains([]byte(path), []byte(pattern))
}

// isAbsolutePath checks if a path is absolute
func isAbsolutePath(path string) bool {
	// Check for Unix-style absolute paths
	if len(path) > 0 && path[0] == '/' {
		return true
	}

	// Check for Windows-style absolute paths (C:\, D:\, etc.)
	if len(path) > 2 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}

	// Check for UNC paths (\\server\share)
	if len(path) > 1 && path[0] == '\\' && path[1] == '\\' {
		return true
	}

	return false
}
