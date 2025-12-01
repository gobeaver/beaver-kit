package filevalidator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSizeReadable_AllCases(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{2048, "2 KB"},
		{1048576, "1 MB"},
		{1572864, "1.5 MB"},
		{2097152, "2 MB"},
		{1073741824, "1 GB"},
		{1610612736, "1.5 GB"},
		{2147483648, "2 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatSizeReadable(tt.size)
			if result != tt.expected {
				t.Errorf("FormatSizeReadable(%d) = %s, want %s", tt.size, result, tt.expected)
			}
		})
	}
}

func TestValidateLocalFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := NewBuilder().
		MaxSize(1 * MB).
		Extensions(".txt").
		Build()

	t.Run("Valid file", func(t *testing.T) {
		err := ValidateLocalFile(validator, testFile)
		if err != nil {
			t.Errorf("ValidateLocalFile() error = %v, want nil", err)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		err := ValidateLocalFile(validator, filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("ValidateLocalFile() should error for non-existent file")
		}
		if !IsErrorOfType(err, ErrorTypeFileName) {
			t.Errorf("Expected ErrorTypeFileName, got %v", GetErrorType(err))
		}
	})

	t.Run("Directory instead of file", func(t *testing.T) {
		err := ValidateLocalFile(validator, tmpDir)
		if err == nil {
			t.Error("ValidateLocalFile() should error for directory")
		}
		if !IsErrorOfType(err, ErrorTypeFileName) {
			t.Errorf("Expected ErrorTypeFileName, got %v", GetErrorType(err))
		}
	})

	t.Run("Invalid extension", func(t *testing.T) {
		invalidFile := filepath.Join(tmpDir, "test.exe")
		if err := os.WriteFile(invalidFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err := ValidateLocalFile(validator, invalidFile)
		if err == nil {
			t.Error("ValidateLocalFile() should error for invalid extension")
		}
	})
}

func TestCreateFileFromBytes(t *testing.T) {
	content := []byte("test content")
	filename := "test.txt"

	header := CreateFileFromBytes(filename, content)

	if header.Filename != filename {
		t.Errorf("Filename = %s, want %s", header.Filename, filename)
	}
	if header.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", header.Size, len(content))
	}
}

func TestCreateFileFromReader(t *testing.T) {
	content := "test content"
	reader := strings.NewReader(content)
	filename := "test.txt"

	header, err := CreateFileFromReader(filename, reader)
	if err != nil {
		t.Fatalf("CreateFileFromReader() error = %v", err)
	}

	if header.Filename != filename {
		t.Errorf("Filename = %s, want %s", header.Filename, filename)
	}
	if header.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", header.Size, len(content))
	}
}

func TestDetectContentTypeFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("PNG file", func(t *testing.T) {
		pngFile := filepath.Join(tmpDir, "test.png")
		pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		if err := os.WriteFile(pngFile, pngBytes, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		contentType, err := DetectContentTypeFromFile(pngFile)
		if err != nil {
			t.Errorf("DetectContentTypeFromFile() error = %v", err)
		}
		if contentType != "image/png" {
			t.Errorf("contentType = %s, want image/png", contentType)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := DetectContentTypeFromFile(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("DetectContentTypeFromFile() should error for non-existent file")
		}
	})
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"application/pdf", false},
		{"text/plain", false},
		{"video/mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := IsImage(tt.contentType)
			if result != tt.expected {
				t.Errorf("IsImage(%s) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestIsDocument(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/pdf", true},
		{"application/msword", true},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", true},
		{"text/plain", true},
		{"text/csv", true},
		{"image/png", false},
		{"video/mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := IsDocument(tt.contentType)
			if result != tt.expected {
				t.Errorf("IsDocument(%s) = %v, want %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestStreamValidate(t *testing.T) {
	validator := NewBuilder().
		MaxSize(1 * KB).
		MinSize(10).
		Accept("text/plain").
		Build()

	t.Run("Valid stream", func(t *testing.T) {
		content := []byte("test content for streaming validation")
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.txt", validator, 512)
		if err != nil {
			t.Errorf("StreamValidate() error = %v, want nil", err)
		}
	})

	t.Run("Stream too large", func(t *testing.T) {
		content := bytes.Repeat([]byte("a"), 2*1024) // 2KB
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.txt", validator, 512)
		if err == nil {
			t.Error("StreamValidate() should error for large stream")
		}
		if !IsErrorOfType(err, ErrorTypeSize) {
			t.Errorf("Expected ErrorTypeSize, got %v", GetErrorType(err))
		}
	})

	t.Run("Stream too small", func(t *testing.T) {
		content := []byte("tiny")
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.txt", validator, 512)
		if err == nil {
			t.Error("StreamValidate() should error for small stream")
		}
		if !IsErrorOfType(err, ErrorTypeSize) {
			t.Errorf("Expected ErrorTypeSize, got %v", GetErrorType(err))
		}
	})

	t.Run("Invalid MIME type", func(t *testing.T) {
		// PNG header
		content := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		content = append(content, bytes.Repeat([]byte{0}, 100)...)
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.png", validator, 512)
		if err == nil {
			t.Error("StreamValidate() should error for invalid MIME type")
		}
		if !IsErrorOfType(err, ErrorTypeMIME) {
			t.Errorf("Expected ErrorTypeMIME, got %v", GetErrorType(err))
		}
	})

	t.Run("Default buffer size", func(t *testing.T) {
		content := []byte("test content")
		reader := bytes.NewReader(content)

		// Pass 0 or negative buffer size to test default
		err := StreamValidate(reader, "test.txt", validator, 0)
		if err != nil {
			t.Errorf("StreamValidate() with default buffer error = %v", err)
		}
	})

	t.Run("Wildcard MIME type", func(t *testing.T) {
		wildcardValidator := NewBuilder().
			MaxSize(1 * KB).
			Accept("image/*").
			Build()

		// PNG header
		content := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		content = append(content, bytes.Repeat([]byte{0}, 100)...)
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.png", wildcardValidator, 512)
		if err != nil {
			t.Errorf("StreamValidate() with wildcard error = %v", err)
		}
	})

	t.Run("Accept all", func(t *testing.T) {
		allValidator := NewBuilder().
			MaxSize(1 * KB).
			Accept("*/*").
			Build()

		content := []byte("any content")
		reader := bytes.NewReader(content)

		err := StreamValidate(reader, "test.txt", allValidator, 512)
		if err != nil {
			t.Errorf("StreamValidate() with */* error = %v", err)
		}
	})
}
