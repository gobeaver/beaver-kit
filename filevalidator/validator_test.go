package filevalidator

import (
	"bytes"
	"context"
	"mime/multipart"
	"regexp"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	constraints := Constraints{
		MaxFileSize: 10 * MB,
		MinFileSize: 1 * KB,
	}

	validator := New(constraints)
	if validator == nil {
		t.Error("New() returned nil")
	}

	gotConstraints := validator.GetConstraints()
	if gotConstraints.MaxFileSize != constraints.MaxFileSize {
		t.Errorf("Expected MaxFileSize %d, got %d", constraints.MaxFileSize, gotConstraints.MaxFileSize)
	}
	if gotConstraints.MinFileSize != constraints.MinFileSize {
		t.Errorf("Expected MinFileSize %d, got %d", constraints.MinFileSize, gotConstraints.MinFileSize)
	}
}

func TestNewDefault(t *testing.T) {
	validator := NewDefault()
	if validator == nil {
		t.Error("NewDefault() returned nil")
	}

	constraints := validator.GetConstraints()
	if constraints.MaxFileSize != 10*MB {
		t.Errorf("Expected default MaxFileSize %d, got %d", 10*MB, constraints.MaxFileSize)
	}
}

func TestFluentBuilder(t *testing.T) {
	regex := regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)
	validator := NewBuilder().
		MaxSize(20 * MB).
		MinSize(2 * KB).
		Accept("image/jpeg", "image/png").
		Extensions(".jpg", ".png").
		BlockExtensions(".exe", ".php").
		MaxNameLength(100).
		FileNamePattern(regex).
		DangerousChars("../", ";").
		RequireExtension().
		StrictMIME().
		Build()

	constraints := validator.GetConstraints()

	if constraints.MaxFileSize != 20*MB {
		t.Errorf("Expected MaxFileSize %d, got %d", 20*MB, constraints.MaxFileSize)
	}
	if constraints.MinFileSize != 2*KB {
		t.Errorf("Expected MinFileSize %d, got %d", 2*KB, constraints.MinFileSize)
	}
	if len(constraints.AcceptedTypes) != 2 {
		t.Errorf("Expected AcceptedTypes length 2, got %d", len(constraints.AcceptedTypes))
	}
	if len(constraints.AllowedExts) != 2 {
		t.Errorf("Expected AllowedExts length 2, got %d", len(constraints.AllowedExts))
	}
	if constraints.MaxNameLength != 100 {
		t.Errorf("Expected MaxNameLength 100, got %d", constraints.MaxNameLength)
	}
	if constraints.FileNameRegex != regex {
		t.Error("Expected FileNameRegex to match")
	}
	if !constraints.RequireExtension {
		t.Error("Expected RequireExtension to be true")
	}
	if !constraints.StrictMIMETypeValidation {
		t.Error("Expected StrictMIMETypeValidation to be true")
	}
}

func TestPredefinedConstraints(t *testing.T) {
	testCases := []struct {
		name          string
		constraints   Constraints
		acceptedTypes []string
		acceptedExts  []string
		rejectedExts  []string
	}{
		{
			name:          "ImageOnlyConstraints",
			constraints:   ImageOnlyConstraints(),
			acceptedTypes: []string{"image/jpeg", "image/png"},
			acceptedExts:  []string{".jpg", ".png", ".gif"},
			rejectedExts:  []string{".pdf", ".doc", ".mp3"},
		},
		{
			name:          "DocumentOnlyConstraints",
			constraints:   DocumentOnlyConstraints(),
			acceptedTypes: []string{"application/pdf", "application/msword"},
			acceptedExts:  []string{".pdf", ".doc", ".docx"},
			rejectedExts:  []string{".jpg", ".mp3", ".mp4"},
		},
		{
			name:          "MediaOnlyConstraints",
			constraints:   MediaOnlyConstraints(),
			acceptedTypes: []string{"audio/mpeg", "video/mp4"},
			acceptedExts:  []string{".mp3", ".mp4", ".wav"},
			rejectedExts:  []string{".jpg", ".pdf", ".doc"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create validator without MIME validation for extension testing
			extOnlyConstraints := tc.constraints
			extOnlyConstraints.AcceptedTypes = nil // Disable MIME validation for extension tests
			extValidator := New(extOnlyConstraints)

			// Test extension validation
			for _, ext := range tc.acceptedExts {
				mockFile := createMockFile("test"+ext, "application/octet-stream", 1000)
				err := extValidator.Validate(mockFile)
				if err != nil {
					t.Errorf("Expected extension %s to be accepted, got error: %v", ext, err)
				}
			}

			// Test rejected extensions
			for _, ext := range tc.rejectedExts {
				mockFile := createMockFile("test"+ext, "application/octet-stream", 1000)
				err := extValidator.Validate(mockFile)
				if err == nil || !IsErrorOfType(err, ErrorTypeExtension) {
					t.Errorf("Expected extension %s to be rejected, got error: %v", ext, err)
				}
			}

			// Test MIME type configuration is correct
			validator := New(tc.constraints)
			constraints := validator.GetConstraints()

			// Verify accepted types are properly expanded
			expandedTypes := ExpandAcceptedTypes(constraints.AcceptedTypes)
			for _, mimeType := range tc.acceptedTypes {
				found := false
				for _, accepted := range expandedTypes {
					if accepted == mimeType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected MIME type %s to be in accepted types", mimeType)
				}
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	validator := New(DefaultConstraints())

	testCases := []struct {
		name      string
		filename  string
		shouldErr bool
		errorType ValidationErrorType
	}{
		{
			name:      "Valid filename",
			filename:  "test.jpg",
			shouldErr: false,
		},
		{
			name:      "Empty filename",
			filename:  "",
			shouldErr: true,
			errorType: ErrorTypeFileName,
		},
		{
			name:      "Filename too long",
			filename:  strings.Repeat("a", 300) + ".jpg",
			shouldErr: true,
			errorType: ErrorTypeFileName,
		},
		{
			name:      "Dangerous character",
			filename:  "../test.jpg",
			shouldErr: true,
			errorType: ErrorTypeFileName,
		},
		{
			name:      "Dangerous character 2",
			filename:  "test;.jpg",
			shouldErr: true,
			errorType: ErrorTypeFileName,
		},
		{
			name:      "No extension",
			filename:  "test",
			shouldErr: true,
			errorType: ErrorTypeExtension,
		},
		{
			name:      "Blocked extension",
			filename:  "test.exe",
			shouldErr: true,
			errorType: ErrorTypeExtension,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFile := &multipart.FileHeader{
				Filename: tc.filename,
				Size:     1000,
			}

			err := validator.Validate(mockFile)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error for filename %s, got nil", tc.filename)
					return
				}
				if !IsErrorOfType(err, tc.errorType) {
					t.Errorf("Expected error type %s, got %s", tc.errorType, GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for filename %s, got %v", tc.filename, err)
				}
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	constraints := Constraints{
		MaxFileSize: 10 * KB,
		MinFileSize: 1 * KB,
	}
	validator := New(constraints)

	testCases := []struct {
		name      string
		size      int64
		shouldErr bool
		errorType ValidationErrorType
	}{
		{
			name:      "Valid size",
			size:      5 * KB,
			shouldErr: false,
		},
		{
			name:      "Too small",
			size:      500,
			shouldErr: true,
			errorType: ErrorTypeSize,
		},
		{
			name:      "Too large",
			size:      15 * KB,
			shouldErr: true,
			errorType: ErrorTypeSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFile := &multipart.FileHeader{
				Filename: "test.jpg",
				Size:     tc.size,
			}

			err := validator.Validate(mockFile)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error for size %d, got nil", tc.size)
					return
				}
				if !IsErrorOfType(err, tc.errorType) {
					t.Errorf("Expected error type %s, got %s", tc.errorType, GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for size %d, got %v", tc.size, err)
				}
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	validator := NewDefault()

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create a mock file
	mockFile := &multipart.FileHeader{
		Filename: "test.jpg",
		Size:     1000,
	}

	// Cancel the context before validation
	cancel()

	// Try to validate with the cancelled context
	err := validator.ValidateWithContext(ctx, mockFile)

	// Check if the error is from context cancellation
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestValidateReader(t *testing.T) {
	validator := New(DefaultConstraints())

	testCases := []struct {
		name      string
		content   string
		filename  string
		size      int64
		shouldErr bool
		errorType ValidationErrorType
	}{
		{
			name:      "Valid file",
			content:   "test content",
			filename:  "test.txt",
			size:      12,
			shouldErr: false,
		},
		{
			name:      "Invalid extension",
			content:   "test content",
			filename:  "test.exe",
			size:      12,
			shouldErr: true,
			errorType: ErrorTypeExtension,
		},
		{
			name:      "Too large",
			content:   strings.Repeat("a", 15*1024*1024),
			filename:  "test.txt",
			size:      15 * MB,
			shouldErr: true,
			errorType: ErrorTypeSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.content)
			err := validator.ValidateReader(reader, tc.filename, tc.size)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
					return
				}
				if !IsErrorOfType(err, tc.errorType) {
					t.Errorf("Expected error type %s, got %s", tc.errorType, GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tc.name, err)
				}
			}
		})
	}
}

func TestValidateBytes(t *testing.T) {
	validator := New(DefaultConstraints())

	testCases := []struct {
		name      string
		content   []byte
		filename  string
		shouldErr bool
		errorType ValidationErrorType
	}{
		{
			name:      "Valid file",
			content:   []byte("test content"),
			filename:  "test.txt",
			shouldErr: false,
		},
		{
			name:      "Invalid extension",
			content:   []byte("test content"),
			filename:  "test.exe",
			shouldErr: true,
			errorType: ErrorTypeExtension,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateBytes(tc.content, tc.filename)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
					return
				}
				if !IsErrorOfType(err, tc.errorType) {
					t.Errorf("Expected error type %s, got %s", tc.errorType, GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got %v", tc.name, err)
				}
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("FormatSizeReadable", func(t *testing.T) {
		testCases := []struct {
			size     int64
			expected string
		}{
			{500, "500 B"},
			{1500, "1.5 KB"},
			{1572864, "1.5 MB"},    // 1.5 * 1024 * 1024
			{1610612736, "1.5 GB"}, // 1.5 * 1024 * 1024 * 1024
		}

		for _, tc := range testCases {
			result := FormatSizeReadable(tc.size)
			if result != tc.expected {
				t.Errorf("FormatSizeReadable(%d) = %s, expected %s", tc.size, result, tc.expected)
			}
		}
	})

	t.Run("HasSupportedImageExtension", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected bool
		}{
			{"test.jpg", true},
			{"test.pdf", false},
			{"test.txt", false},
			{"test.png", true},
			{"test.exe", false},
		}

		for _, tc := range testCases {
			result := HasSupportedImageExtension(tc.filename)
			if result != tc.expected {
				t.Errorf("HasSupportedImageExtension(%s) = %v, expected %v", tc.filename, result, tc.expected)
			}
		}
	})

	t.Run("HasSupportedDocumentExtension", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected bool
		}{
			{"test.pdf", true},
			{"test.jpg", false},
			{"test.txt", true},
			{"test.docx", true},
			{"test.exe", false},
		}

		for _, tc := range testCases {
			result := HasSupportedDocumentExtension(tc.filename)
			if result != tc.expected {
				t.Errorf("HasSupportedDocumentExtension(%s) = %v, expected %v", tc.filename, result, tc.expected)
			}
		}
	})

	t.Run("DetectContentType", func(t *testing.T) {
		// PNG header magic bytes
		pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		contentType := DetectContentType(pngBytes)
		if contentType != "image/png" {
			t.Errorf("DetectContentType(pngBytes) = %s, expected image/png", contentType)
		}
	})
}

func TestExpandAcceptedTypes(t *testing.T) {
	testCases := []struct {
		name          string
		acceptedTypes []string
		expectedCount int
	}{
		{
			name:          "Basic types",
			acceptedTypes: []string{"image/jpeg", "application/pdf"},
			expectedCount: 2,
		},
		{
			name:          "With media group",
			acceptedTypes: []string{"image/*", "application/pdf"},
			expectedCount: 9, // 8 image types + 1 pdf
		},
		{
			name:          "All images",
			acceptedTypes: []string{string(AllowAllImages)},
			expectedCount: 9, // All image types
		},
		{
			name:          "Multiple groups",
			acceptedTypes: []string{string(AllowAllImages), string(AllowAllDocuments)},
			expectedCount: 20, // All image + document types
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := ExpandAcceptedTypes(tc.acceptedTypes)
			if len(expanded) < tc.expectedCount {
				t.Errorf("ExpandAcceptedTypes(%v) returned %d types, expected at least %d",
					tc.acceptedTypes, len(expanded), tc.expectedCount)
			}
		})
	}
}

// Helper function to create a mock multipart.FileHeader for testing
func createMockFile(filename string, contentType string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Header:   map[string][]string{"Content-Type": {contentType}},
		Size:     size,
	}
}

// Mock implementation of io.ReadCloser for testing
type mockFile struct {
	*bytes.Reader
	closed bool
}

func (m *mockFile) Close() error {
	m.closed = true
	return nil
}

func newMockFile(content []byte) *mockFile {
	return &mockFile{
		Reader: bytes.NewReader(content),
		closed: false,
	}
}

// getExtensionForMimeType returns an appropriate file extension for a given MIME type
func getExtensionForMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	case "application/msword":
		return ".doc"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ".docx"
	case "text/plain":
		return ".txt"
	case "audio/mpeg":
		return ".mp3"
	case "video/mp4":
		return ".mp4"
	default:
		return ".bin"
	}
}

// getMockContentForMimeType returns mock content that will be detected as the specified MIME type
func getMockContentForMimeType(mimeType string) []byte {
	switch mimeType {
	case "image/jpeg":
		// JPEG magic bytes
		return []byte{0xFF, 0xD8, 0xFF}
	case "image/png":
		// PNG magic bytes
		return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	case "image/gif":
		// GIF magic bytes
		return []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}
	case "application/pdf":
		// PDF magic bytes
		return []byte{0x25, 0x50, 0x44, 0x46}
	case "application/msword":
		// DOC magic bytes (OLE header) - filling to 512 bytes for proper detection
		header := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
		content := make([]byte, 512)
		copy(content, header)
		return content
	case "text/plain":
		// Plain text - needs enough content for detection
		content := make([]byte, 512)
		copy(content, []byte("This is plain text content"))
		return content
	case "audio/mpeg":
		// MP3 magic bytes (ID3) - filling to 512 bytes
		header := []byte{0x49, 0x44, 0x33}
		content := make([]byte, 512)
		copy(content, header)
		return content
	case "video/mp4":
		// MP4 magic bytes - ftyp box
		header := []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x69, 0x73, 0x6F, 0x6D}
		content := make([]byte, 512)
		copy(content, header)
		return content
	default:
		// Default binary content
		return make([]byte, 512)
	}
}

// createMockFileWithContent creates a mock file with actual content bytes for MIME detection
func createMockFileWithContent(filename string, mimeType string, size int64) *multipart.FileHeader {
	// Create content that will be detected as the correct MIME type
	var content []byte
	switch mimeType {
	case "image/jpeg":
		// JPEG magic bytes
		content = []byte{0xFF, 0xD8, 0xFF}
	case "image/png":
		// PNG magic bytes
		content = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	case "image/gif":
		// GIF magic bytes
		content = []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}
	case "application/pdf":
		// PDF magic bytes
		content = []byte{0x25, 0x50, 0x44, 0x46}
	default:
		// Default binary content
		content = make([]byte, 512)
	}

	// Create a temporary buffer with the content
	buffer := &bytes.Buffer{}
	buffer.Write(content)

	// Fill to requested size
	remaining := int(size) - len(content)
	if remaining > 0 {
		buffer.Write(make([]byte, remaining))
	}

	// Create the FileHeader
	header := &multipart.FileHeader{
		Filename: filename,
		Header:   map[string][]string{"Content-Type": {mimeType}},
		Size:     size,
	}

	// Mock the Open method to return a reader with our content
	header.Header["__test_content"] = []string{string(buffer.Bytes())}

	return header
}

// MockableFileHeader is a mock FileHeader that can be used for testing
type MockableFileHeader struct {
	*multipart.FileHeader
	content []byte
}

func (m *MockableFileHeader) Open() (multipart.File, error) {
	reader := bytes.NewReader(m.content)
	return &mockFile{Reader: reader}, nil
}

// createMockFileHeaderWithContent creates a proper mock FileHeader for testing
func createMockFileHeaderWithContent(filename, mimeType string, content []byte) *multipart.FileHeader {
	header := &multipart.FileHeader{
		Filename: filename,
		Header:   map[string][]string{"Content-Type": {mimeType}},
		Size:     int64(len(content)),
	}

	// We'll use reflection to mock the Open method
	return header
}
