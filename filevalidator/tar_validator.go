package filevalidator

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
)

// TarValidator validates TAR and TAR.GZ archives.
// Uses Go stdlib only - no external dependencies.
type TarValidator struct {
	MaxSize             int64
	MaxFiles            int
	MaxUncompressedSize int64
	MaxCompressionRatio float64 // For gzipped tars
	AllowSymlinks       bool
	AllowHardlinks      bool
}

// DefaultTarValidator creates a tar validator with sensible defaults
func DefaultTarValidator() *TarValidator {
	return &TarValidator{
		MaxSize:             1 * GB,
		MaxFiles:            10000,
		MaxUncompressedSize: 10 * GB,
		MaxCompressionRatio: 100.0,
		AllowSymlinks:       false, // Symlinks can escape extraction directory
		AllowHardlinks:      false, // Hardlinks can overwrite files
	}
}

// ValidateContent validates TAR archives
func (v *TarValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	return v.validateTar(reader, size, false)
}

// ValidateGzipContent validates GZIP-compressed TAR archives
func (v *TarValidator) ValidateGzipContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, fmt.Sprintf("invalid gzip: %v", err))
	}
	defer gzReader.Close()

	return v.validateTar(gzReader, size, true)
}

func (v *TarValidator) validateTar(reader io.Reader, compressedSize int64, isGzipped bool) error {
	tarReader := tar.NewReader(reader)

	var (
		fileCount         int
		totalUncompressed int64
	)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return NewValidationError(ErrorTypeContent, fmt.Sprintf("invalid tar: %v", err))
		}

		fileCount++
		if fileCount > v.MaxFiles {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("too many files: %d (max: %d)", fileCount, v.MaxFiles))
		}

		// Check for path traversal
		if v.isDangerousPath(header.Name) {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("dangerous path: %s", header.Name))
		}

		// Check symlinks
		if header.Typeflag == tar.TypeSymlink && !v.AllowSymlinks {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("symlinks not allowed: %s", header.Name))
		}

		// Check hardlinks
		if header.Typeflag == tar.TypeLink && !v.AllowHardlinks {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("hardlinks not allowed: %s", header.Name))
		}

		// Track uncompressed size
		if header.Size > 0 {
			totalUncompressed += header.Size

			if totalUncompressed > v.MaxUncompressedSize {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("uncompressed size exceeds limit: %d", v.MaxUncompressedSize))
			}

			// Check compression ratio for gzipped tars
			if isGzipped && compressedSize > 0 {
				ratio := float64(totalUncompressed) / float64(compressedSize)
				if ratio > v.MaxCompressionRatio {
					return NewValidationError(ErrorTypeContent,
						fmt.Sprintf("suspicious compression ratio: %.2f:1", ratio))
				}
			}
		}

		// Skip content - we only validate headers
		// Note: Decompression bomb protection is handled by MaxUncompressedSize and MaxCompressionRatio checks above
		if _, err := io.Copy(io.Discard, tarReader); err != nil { //nolint:gosec // decompression bomb mitigated by size/ratio checks
			return NewValidationError(ErrorTypeContent, fmt.Sprintf("failed to read entry: %v", err))
		}
	}

	return nil
}

func (v *TarValidator) isDangerousPath(path string) bool {
	// Check for path traversal
	if strings.Contains(path, "..") {
		return true
	}
	// Check for absolute paths
	if strings.HasPrefix(path, "/") {
		return true
	}
	return false
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *TarValidator) SupportedMIMETypes() []string {
	return []string{
		"application/x-tar",
		"application/tar",
	}
}

// GzipValidator validates standalone GZIP files (not tar.gz)
type GzipValidator struct {
	MaxSize             int64
	MaxUncompressedSize int64
	MaxCompressionRatio float64
}

// DefaultGzipValidator creates a gzip validator with sensible defaults
func DefaultGzipValidator() *GzipValidator {
	return &GzipValidator{
		MaxSize:             1 * GB,
		MaxUncompressedSize: 10 * GB,
		MaxCompressionRatio: 100.0,
	}
}

// ValidateContent validates GZIP files by checking header and decompression ratio
func (v *GzipValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, fmt.Sprintf("invalid gzip: %v", err))
	}
	defer gzReader.Close()

	// Read and count uncompressed bytes (discarding content)
	var uncompressedSize int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := gzReader.Read(buf)
		uncompressedSize += int64(n)

		// Check size limit
		if uncompressedSize > v.MaxUncompressedSize {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("uncompressed size exceeds limit: %d", v.MaxUncompressedSize))
		}

		// Check compression ratio
		if size > 0 {
			ratio := float64(uncompressedSize) / float64(size)
			if ratio > v.MaxCompressionRatio {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("suspicious compression ratio: %.2f:1", ratio))
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return NewValidationError(ErrorTypeContent, fmt.Sprintf("gzip read error: %v", err))
		}
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *GzipValidator) SupportedMIMETypes() []string {
	return []string{
		"application/gzip",
		"application/x-gzip",
	}
}

// TarGzValidator validates .tar.gz files
type TarGzValidator struct {
	*TarValidator
}

// DefaultTarGzValidator creates a tar.gz validator
func DefaultTarGzValidator() *TarGzValidator {
	return &TarGzValidator{
		TarValidator: DefaultTarValidator(),
	}
}

// ValidateContent validates tar.gz files
func (v *TarGzValidator) ValidateContent(reader io.Reader, size int64) error {
	return v.TarValidator.ValidateGzipContent(reader, size)
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *TarGzValidator) SupportedMIMETypes() []string {
	return []string{
		"application/x-gtar",
		"application/x-tar+gzip",
		"application/x-compressed-tar",
	}
}
