package filevalidator

import (
	"bytes"
	"context"
	"mime/multipart"
	"strings"
	"testing"
	"time"
)

// Additional tests for validator.go to increase coverage

func TestValidator_isAcceptedMIMEType(t *testing.T) {
	validator := NewBuilder().
		Accept("image/png", "image/jpeg", "application/*").
		Build()

	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"Exact match", "image/png", true},
		{"Wildcard match", "application/pdf", true},
		{"No match", "video/mp4", false},
		{"All match", "text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isAcceptedMIMEType(tt.mimeType)
			if result != tt.expected {
				t.Errorf("isAcceptedMIMEType(%s) = %v, want %v", tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestValidator_isAcceptedExtension(t *testing.T) {
	validator := NewBuilder().
		Extensions(".jpg", ".png", ".pdf").
		Build()

	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		{"Accepted extension", ".jpg", true},
		{"Accepted extension uppercase", ".JPG", true},
		{"Not accepted", ".exe", false},
		{"Empty extension", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isAcceptedExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("isAcceptedExtension(%s) = %v, want %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestValidator_expandedAcceptedTypes(t *testing.T) {
	validator := NewBuilder().
		Accept("image/*", "application/pdf").
		Build()

	expanded := validator.expandedAcceptedTypes()
	if len(expanded) < 2 {
		t.Errorf("expandedAcceptedTypes() returned %d types, expected at least 2", len(expanded))
	}

	// Should contain application/pdf
	found := false
	for _, mime := range expanded {
		if mime == "application/pdf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expandedAcceptedTypes() should contain application/pdf")
	}
}

func TestValidateWithContext_Cancellation(t *testing.T) {
	validator := NewBuilder().
		Accept("image/png").
		WithContentValidation().
		Build()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockFile := &multipart.FileHeader{
		Filename: "test.png",
		Size:     1000,
	}

	err := validator.ValidateWithContext(ctx, mockFile)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestValidateWithContext_CancellationDuringValidation(t *testing.T) {
	validator := NewBuilder().
		Accept("image/png").
		Build()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(5 * time.Millisecond)

	mockFile := &multipart.FileHeader{
		Filename: "test.png",
		Size:     1000,
	}

	err := validator.ValidateWithContext(ctx, mockFile)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestValidateReader_NonSeekable(t *testing.T) {
	validator := NewBuilder().
		Extensions(".txt").
		Build()

	// Use a non-seekable reader (strings.Reader is seekable, so we wrap it)
	content := "test content"
	reader := &nonSeekableReader{Reader: strings.NewReader(content)}

	err := validator.ValidateReader(reader, "test.txt", int64(len(content)))
	if err != nil {
		t.Errorf("ValidateReader() with non-seekable reader error = %v", err)
	}
}

func TestValidateReader_NonSeekableInvalidExtension(t *testing.T) {
	validator := NewBuilder().
		Extensions(".txt").
		Build()

	content := "test content"
	reader := &nonSeekableReader{Reader: strings.NewReader(content)}

	err := validator.ValidateReader(reader, "test.exe", int64(len(content)))
	if err == nil {
		t.Error("ValidateReader() should error for invalid extension with non-seekable reader")
	}
	if !IsErrorOfType(err, ErrorTypeExtension) {
		t.Errorf("Expected ErrorTypeExtension, got %v", GetErrorType(err))
	}
}

func TestValidateReader_SeekableWithMIME(t *testing.T) {
	validator := NewBuilder().
		Accept("text/plain").
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	err := validator.ValidateReader(reader, "test.txt", int64(len(content)))
	if err != nil {
		t.Errorf("ValidateReader() with seekable reader error = %v", err)
	}
}

func TestValidateReader_SeekableInvalidMIME(t *testing.T) {
	validator := NewBuilder().
		Accept("image/png").
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	err := validator.ValidateReader(reader, "test.txt", int64(len(content)))
	if err == nil {
		t.Error("ValidateReader() should error for invalid MIME type")
	}
	if !IsErrorOfType(err, ErrorTypeMIME) {
		t.Errorf("Expected ErrorTypeMIME, got %v", GetErrorType(err))
	}
}

func TestValidateReader_StrictMIME(t *testing.T) {
	validator := NewBuilder().
		Accept("text/plain").
		StrictMIME().
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	// Extension suggests text/plain, content is text/plain - should pass
	err := validator.ValidateReader(reader, "test.txt", int64(len(content)))
	if err != nil {
		t.Errorf("ValidateReader() with strict MIME error = %v", err)
	}
}

func TestValidateReader_StrictMIMEMismatch(t *testing.T) {
	validator := NewBuilder().
		Accept("image/png", "text/plain").
		StrictMIME().
		Build()

	// Text content with .png extension
	content := "test content"
	reader := bytes.NewReader([]byte(content))

	err := validator.ValidateReader(reader, "test.png", int64(len(content)))
	if err == nil {
		t.Error("ValidateReader() should error for MIME mismatch in strict mode")
	}
	if !IsErrorOfType(err, ErrorTypeMIME) {
		t.Errorf("Expected ErrorTypeMIME, got %v", GetErrorType(err))
	}
}

func TestValidateReader_WithContentValidation(t *testing.T) {
	validator := NewBuilder().
		Accept("text/plain").
		WithContentValidation().
		WithDefaultRegistry().
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	err := validator.ValidateReader(reader, "test.txt", int64(len(content)))
	if err != nil {
		t.Errorf("ValidateReader() with content validation error = %v", err)
	}
}

func TestValidateReader_NoSize(t *testing.T) {
	validator := NewBuilder().
		Extensions(".txt").
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	// Pass 0 as size
	err := validator.ValidateReader(reader, "test.txt", 0)
	if err != nil {
		t.Errorf("ValidateReader() with no size error = %v", err)
	}
}

func TestValidateBytes_Various(t *testing.T) {
	validator := NewBuilder().
		MaxSize(1 * KB).
		Extensions(".txt").
		Build()

	tests := []struct {
		name      string
		content   []byte
		filename  string
		shouldErr bool
	}{
		{
			name:      "Valid",
			content:   []byte("test"),
			filename:  "test.txt",
			shouldErr: false,
		},
		{
			name:      "Too large",
			content:   bytes.Repeat([]byte("a"), 2*1024),
			filename:  "test.txt",
			shouldErr: true,
		},
		{
			name:      "Invalid extension",
			content:   []byte("test"),
			filename:  "test.exe",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBytes(tt.content, tt.filename)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateBytes() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
		})
	}
}

func TestValidateFileName_RegexPattern(t *testing.T) {
	validator := NewBuilder().
		FileNamePatternString(`^[a-z0-9_]+\.[a-z]+$`).
		Build()

	tests := []struct {
		name      string
		filename  string
		shouldErr bool
	}{
		{"Valid pattern", "test_file.txt", false},
		{"Invalid pattern - uppercase", "TestFile.txt", true},
		{"Invalid pattern - space", "test file.txt", true},
		{"Invalid pattern - special char", "test-file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFile := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     1000,
			}

			err := validator.Validate(mockFile)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Validate() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
		})
	}
}

func TestValidateFileName_NoExtensionAllowed(t *testing.T) {
	validator := NewBuilder().
		AllowNoExtension().
		Build()

	mockFile := &multipart.FileHeader{
		Filename: "README",
		Size:     1000,
	}

	err := validator.Validate(mockFile)
	if err != nil {
		t.Errorf("Validate() should allow files without extension, got error: %v", err)
	}
}

func TestValidate_NoAcceptedTypes(t *testing.T) {
	// Validator with no accepted types should skip MIME validation
	validator := NewBuilder().
		MaxSize(1 * MB).
		Build()

	mockFile := &multipart.FileHeader{
		Filename: "test.anything",
		Size:     1000,
	}

	err := validator.Validate(mockFile)
	if err != nil {
		t.Errorf("Validate() with no accepted types should not error, got: %v", err)
	}
}

func TestValidateReader_NoAcceptedTypes(t *testing.T) {
	validator := NewBuilder().
		MaxSize(1 * MB).
		Build()

	content := "test content"
	reader := bytes.NewReader([]byte(content))

	err := validator.ValidateReader(reader, "test.anything", int64(len(content)))
	if err != nil {
		t.Errorf("ValidateReader() with no accepted types should not error, got: %v", err)
	}
}

// nonSeekableReader wraps a reader to make it non-seekable
type nonSeekableReader struct {
	Reader *strings.Reader
}

func (r *nonSeekableReader) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}
