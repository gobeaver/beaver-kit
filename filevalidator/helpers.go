package filevalidator

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FormatSizeReadable converts a size in bytes to a human-readable string
func FormatSizeReadable(size int64) string {
	if size < KB {
		return fmt.Sprintf("%d B", size)
	}
	if size < MB {
		value := float64(size) / float64(KB)
		// Round to 1 decimal place properly
		rounded := math.Round(value*10) / 10
		if rounded == float64(int(rounded)) {
			return fmt.Sprintf("%.0f KB", rounded)
		}
		return fmt.Sprintf("%.1f KB", rounded)
	}
	if size < GB {
		value := float64(size) / float64(MB)
		// Round to 1 decimal place properly
		rounded := math.Round(value*10) / 10
		if rounded == float64(int(rounded)) {
			return fmt.Sprintf("%.0f MB", rounded)
		}
		return fmt.Sprintf("%.1f MB", rounded)
	}
	value := float64(size) / float64(GB)
	// Round to 1 decimal place properly
	rounded := math.Round(value*10) / 10
	if rounded == float64(int(rounded)) {
		return fmt.Sprintf("%.0f GB", rounded)
	}
	return fmt.Sprintf("%.1f GB", rounded)
}

// ValidateLocalFile validates a local file path against the validator's constraints
func ValidateLocalFile(validator Validator, filePath string) error {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewValidationError(ErrorTypeFileName, "file does not exist")
		}
		return err
	}

	// Check if it's a regular file (not a directory)
	if fileInfo.IsDir() {
		return NewValidationError(ErrorTypeFileName, "path is a directory, not a file")
	}

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a mock multipart.FileHeader with the file info
	header := &multipart.FileHeader{
		Filename: filepath.Base(filePath),
		Size:     fileInfo.Size(),
	}

	// Validate using the validator
	return validator.Validate(header)
}

// CreateFileFromBytes creates a multipart.FileHeader from a byte slice for validation testing
func CreateFileFromBytes(filename string, content []byte) *multipart.FileHeader {
	header := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(content)),
	}
	return header
}

// CreateFileFromReader creates a multipart.FileHeader from an io.Reader for validation testing
func CreateFileFromReader(filename string, reader io.Reader) (*multipart.FileHeader, error) {
	// Read all content to determine size
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	header := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(content)),
	}
	return header, nil
}

// HasSupportedImageExtension checks if a filename has a supported image extension
func HasSupportedImageExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif"}

	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}

	return false
}

// HasSupportedDocumentExtension checks if a filename has a supported document extension
func HasSupportedDocumentExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".pdf", ".doc", ".docx", ".txt", ".rtf", ".xls", ".xlsx", ".ppt", ".pptx", ".csv"}

	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}

	return false
}

// DetectContentType detects the content type of a file from its bytes
func DetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// DetectContentTypeFromFile detects the content type of a file from its path
func DetectContentTypeFromFile(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	// Detect the content type
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

// IsImage checks if a content type belongs to the image category
func IsImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// IsDocument checks if a content type is a common document format
func IsDocument(contentType string) bool {
	documentTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/csv",
	}

	for _, docType := range documentTypes {
		if contentType == docType {
			return true
		}
	}

	return false
}

// StreamValidate validates a stream of bytes as they're read
// This is useful for validating large files without reading them entirely into memory
func StreamValidate(reader io.Reader, filename string, validator Validator, bufferSize int64) error {
	if bufferSize <= 0 {
		bufferSize = 4096 // Default buffer size: 4KB
	}

	var totalSize int64 = 0
	buffer := make([]byte, bufferSize)

	// Create a buffered reader with a copy of the header for MIME detection
	var headerBuffer bytes.Buffer
	teeReader := io.TeeReader(io.LimitReader(reader, 512), &headerBuffer)

	// Read the header to detect MIME type
	headerBytes := make([]byte, 512)
	n, err := teeReader.Read(headerBytes)
	if err != nil && err != io.EOF {
		return err
	}

	// Detect content type from header
	contentType := http.DetectContentType(headerBytes[:n])
	totalSize += int64(n)

	// Get constraints
	constraints := validator.GetConstraints()

	// Check MIME type if needed
	if len(constraints.AcceptedTypes) > 0 {
		// Truncate the MIME type if there are extra parameters (like charset)
		if idx := strings.Index(contentType, ";"); idx > 0 {
			contentType = contentType[:idx]
		}

		// Check against accepted types
		accepted := false
		for _, acceptedType := range ExpandAcceptedTypes(constraints.AcceptedTypes) {
			if acceptedType == contentType || acceptedType == "*/*" {
				accepted = true
				break
			}

			// Handle wildcards like "image/*"
			if strings.HasSuffix(acceptedType, "/*") {
				prefix := strings.TrimSuffix(acceptedType, "/*")
				if strings.HasPrefix(contentType, prefix+"/") {
					accepted = true
					break
				}
			}
		}

		if !accepted {
			return NewValidationError(
				ErrorTypeMIME,
				"file type "+contentType+" is not accepted",
			)
		}
	}

	// Create a new reader with the header bytes we've already read
	combinedReader := io.MultiReader(bytes.NewReader(headerBytes[:n]), reader)

	// Continue reading the rest of the stream to validate size
	for {
		n, err := combinedReader.Read(buffer)
		totalSize += int64(n)

		// Check max size constraint
		if constraints.MaxFileSize > 0 && totalSize > constraints.MaxFileSize {
			return NewValidationError(
				ErrorTypeSize,
				"file size exceeds maximum allowed size",
			)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	// Check min size constraint
	if constraints.MinFileSize > 0 && totalSize < constraints.MinFileSize {
		return NewValidationError(
			ErrorTypeSize,
			"file size is less than minimum required size",
		)
	}

	return nil
}
