package filekit

import (
	"context"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestValidationIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		config      Config
		filename    string
		content     string
		contentType string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid file within size limit",
			config: Config{
				Driver:        "local",
				LocalBasePath: tmpDir,
				MaxFileSize:   1024, // 1KB
			},
			filename:    "small.txt",
			content:     "Hello, world!",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name: "file exceeds size limit",
			config: Config{
				Driver:        "local",
				LocalBasePath: tmpDir,
				MaxFileSize:   10, // 10 bytes
			},
			filename:    "large.txt",
			content:     "This content is larger than 10 bytes",
			contentType: "text/plain",
			wantErr:     true,
			errContains: "size",
		},
		{
			name: "allowed mime type",
			config: Config{
				Driver:           "local",
				LocalBasePath:    tmpDir,
				AllowedMimeTypes: "text/plain,application/json",
			},
			filename:    "allowed.txt",
			content:     "Plain text content",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name: "disallowed mime type",
			config: Config{
				Driver:           "local",
				LocalBasePath:    tmpDir,
				AllowedMimeTypes: "image/jpeg,image/png",
			},
			filename:    "disallowed.txt",
			content:     "Text content",
			contentType: "text/plain",
			wantErr:     true,
			errContains: "type",
		},
		{
			name: "blocked mime type",
			config: Config{
				Driver:           "local",
				LocalBasePath:    tmpDir,
				BlockedMimeTypes: "application/x-executable",
			},
			filename:    "blocked.exe",
			content:     "Binary content",
			contentType: "application/x-executable",
			wantErr:     true,
			errContains: "extension", // .exe is blocked by default
		},
		{
			name: "allowed extension",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				AllowedExtensions: ".txt,.json",
			},
			filename:    "allowed.txt",
			content:     "Text content",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name: "disallowed extension",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				AllowedExtensions: ".jpg,.png",
			},
			filename:    "disallowed.txt",
			content:     "Text content",
			contentType: "text/plain",
			wantErr:     true,
			errContains: "extension",
		},
		{
			name: "blocked extension",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				BlockedExtensions: ".exe,.bat",
			},
			filename:    "malicious.exe",
			content:     "Binary content",
			contentType: "application/x-executable",
			wantErr:     true,
			errContains: "extension",
		},
		{
			name: "multiple constraints - all pass",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				MaxFileSize:       1024,
				AllowedMimeTypes:  "text/plain,application/json",
				AllowedExtensions: ".txt,.json",
			},
			filename:    "valid.txt",
			content:     "Small text content",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name: "multiple constraints - one fails",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				MaxFileSize:       1024,
				AllowedMimeTypes:  "text/plain",
				AllowedExtensions: ".json", // This will fail for .txt file
			},
			filename:    "invalid.txt",
			content:     "Text content",
			contentType: "text/plain",
			wantErr:     true,
			errContains: "extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := New(tt.config)
			if err != nil {
				t.Fatalf("Failed to create filesystem: %v", err)
			}

			ctx := context.Background()
			content := strings.NewReader(tt.content)

			var opts []Option
			if tt.contentType != "" {
				opts = append(opts, WithContentType(tt.contentType))
			}

			err = fs.Write(ctx, tt.filename, content, opts...)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
				t.Errorf("Write() error = %v, want error containing %v", err, tt.errContains)
			}

			// If write succeeded, verify the file exists
			if err == nil {
				exists, err := fs.FileExists(ctx, tt.filename)
				if err != nil {
					t.Errorf("FileExists() error = %v", err)
				}
				if !exists {
					t.Error("File should exist after successful write")
				}

				// Clean up
				_ = fs.Delete(ctx, tt.filename)
			}
		})
	}
}

func TestValidationWithEncryption(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate encryption key
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	cfg := Config{
		Driver:            "local",
		LocalBasePath:     tmpDir,
		MaxFileSize:       1024,
		AllowedMimeTypes:  "text/plain,application/octet-stream",
		AllowedExtensions: ".txt",
		EncryptionEnabled: true,
		EncryptionKey:     base64.StdEncoding.EncodeToString(encKey),
	}

	fs, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	ctx := context.Background()

	// Test that validation happens before encryption

	// 1. Try to write a file that violates validation
	largeContent := strings.Repeat("a", 2000) // Exceeds 1024 byte limit
	err = fs.Write(ctx, "large.txt", strings.NewReader(largeContent), WithContentType("text/plain"))
	if err == nil {
		t.Error("Expected validation error for large file")
	}

	// 2. Write a valid file
	validContent := "This is valid content"
	err = fs.Write(ctx, "valid.txt", strings.NewReader(validContent), WithContentType("text/plain"))
	if err != nil {
		t.Errorf("Write of valid file failed: %v", err)
	}

	// 3. Verify the content is encrypted and then properly decrypted
	reader, err := fs.Read(ctx, "valid.txt")
	if err != nil {
		t.Errorf("Read failed: %v", err)
		return // Exit early if read fails
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read content: %v", err)
	}

	if string(data) != validContent {
		t.Errorf("Decrypted content = %v, want %v", string(data), validContent)
	}
}
