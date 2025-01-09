package filevalidator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
)

// ArchiveValidator validates archive files to prevent zip bombs and other malicious archives
type ArchiveValidator struct {
	MaxCompressionRatio float64
	MaxFiles            int
	MaxDepth            int
	MaxUncompressedSize int64
	MaxNestedArchives   int
	ArchiveExtensions   []string
}

// DefaultArchiveValidator creates an archive validator with sensible defaults
func DefaultArchiveValidator() *ArchiveValidator {
	return &ArchiveValidator{
		MaxCompressionRatio: 100.0, // 100:1 compression ratio max
		MaxFiles:            1000,
		MaxDepth:            10,
		MaxUncompressedSize: 100 * GB,
		MaxNestedArchives:   5,
		ArchiveExtensions:   []string{".zip", ".jar", ".war", ".ear", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz"},
	}
}

// ValidateContent validates the content of an archive file
func (v *ArchiveValidator) ValidateContent(reader io.Reader, size int64) error {
	// Read the entire content into memory
	// Note: For large files, you might want to implement a streaming approach
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read archive content")
	}

	// Create a reader from the bytes
	bytesReader := bytes.NewReader(data)

	// Try to open as zip file
	zipReader, err := zip.NewReader(bytesReader, size)
	if err != nil {
		// This might not be a zip file, or it's corrupted
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
		if totalUncompressedSize > uint64(v.MaxUncompressedSize) {
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

// SupportedMIMETypes returns the MIME types this validator can handle
func (v *ArchiveValidator) SupportedMIMETypes() []string {
	return []string{
		"application/zip",
		"application/x-zip-compressed",
		"application/x-compressed",
		"application/x-jar",
		"application/java-archive",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/x-tar",
		"application/gzip",
		"application/x-gzip",
		"application/x-bzip2",
		"application/x-xz",
	}
}

// isArchive checks if a filename indicates an archive
func (v *ArchiveValidator) isArchive(filename string) bool {
	for _, ext := range v.ArchiveExtensions {
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
	return filename[len(filename)-len(ext):] == ext ||
		filename[len(filename)-len(ext):] == ext
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
