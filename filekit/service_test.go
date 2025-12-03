package filekit

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty driver",
			config:  Config{},
			wantErr: true,
			errMsg:  "driver is required",
		},
		{
			name:    "invalid driver",
			config:  Config{Driver: "invalid"},
			wantErr: true,
			errMsg:  "unknown driver: invalid",
		},
		{
			name:    "local driver without base path",
			config:  Config{Driver: "local"},
			wantErr: true,
			errMsg:  "local base path is required for local driver",
		},
		{
			name:    "local driver with base path",
			config:  Config{Driver: "local", LocalBasePath: "/tmp"},
			wantErr: false,
		},
		{
			name:    "s3 driver without bucket",
			config:  Config{Driver: "s3"},
			wantErr: true,
			errMsg:  "S3 bucket is required for S3 driver",
		},
		{
			name:    "s3 driver with bucket",
			config:  Config{Driver: "s3", S3Bucket: "test-bucket"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateConfig() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestNew(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "local config with validation",
			config: Config{
				Driver:           "local",
				LocalBasePath:    tmpDir,
				MaxFileSize:      1024,
				AllowedMimeTypes: "text/plain,application/json",
			},
			wantErr: false,
		},
		{
			name: "local config with defaults",
			config: Config{
				Driver:              "local",
				LocalBasePath:       tmpDir,
				DefaultVisibility:   "private",
				DefaultCacheControl: "no-cache",
			},
			wantErr: false,
		},
		{
			name: "local config with encryption",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				EncryptionEnabled: true,
				EncryptionKey:     base64.StdEncoding.EncodeToString(make([]byte, 32)),
			},
			wantErr: false,
		},
		{
			name: "invalid encryption key",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				EncryptionEnabled: true,
				EncryptionKey:     "invalid-base64",
			},
			wantErr: true,
			errMsg:  "invalid encryption key",
		},
		{
			name: "wrong encryption key length",
			config: Config{
				Driver:            "local",
				LocalBasePath:     tmpDir,
				EncryptionEnabled: true,
				EncryptionKey:     base64.StdEncoding.EncodeToString(make([]byte, 16)), // 16 bytes instead of 32
			},
			wantErr: true,
			errMsg:  "encryption key must be 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("New() error = %v, want error containing %v", err, tt.errMsg)
			}
			if err == nil && fs == nil {
				t.Error("New() returned nil filesystem without error")
			}
		})
	}
}

func TestShouldApplyDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "no defaults",
			config: Config{},
			want:   false,
		},
		{
			name:   "with visibility",
			config: Config{DefaultVisibility: "public"},
			want:   true,
		},
		{
			name:   "with cache control",
			config: Config{DefaultCacheControl: "max-age=3600"},
			want:   true,
		},
		{
			name:   "with overwrite",
			config: Config{DefaultOverwrite: true},
			want:   true,
		},
		{
			name:   "with preserve filename",
			config: Config{DefaultPreserveFilename: true},
			want:   true,
		},
		{
			name: "multiple defaults",
			config: Config{
				DefaultVisibility:   "private",
				DefaultCacheControl: "no-cache",
				DefaultOverwrite:    true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldApplyDefaults(tt.config); got != tt.want {
				t.Errorf("shouldApplyDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDefaultOptions(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   int // number of options
	}{
		{
			name:   "no defaults",
			config: Config{},
			want:   0,
		},
		{
			name:   "with visibility",
			config: Config{DefaultVisibility: "public"},
			want:   1,
		},
		{
			name:   "with cache control",
			config: Config{DefaultCacheControl: "max-age=3600"},
			want:   1,
		},
		{
			name:   "with overwrite",
			config: Config{DefaultOverwrite: true},
			want:   1,
		},
		{
			name:   "with preserve filename",
			config: Config{DefaultPreserveFilename: true},
			want:   1,
		},
		{
			name: "all defaults",
			config: Config{
				DefaultVisibility:       "private",
				DefaultCacheControl:     "no-cache",
				DefaultOverwrite:        true,
				DefaultPreserveFilename: true,
			},
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := createDefaultOptions(tt.config)
			if len(opts) != tt.want {
				t.Errorf("createDefaultOptions() returned %d options, want %d", len(opts), tt.want)
			}
		})
	}
}

func TestGlobalInstance(t *testing.T) {
	// Reset global state
	Reset()

	// Test InitFromEnv
	os.Setenv("BEAVER_FILEKIT_DRIVER", "local")
	os.Setenv("BEAVER_FILEKIT_LOCAL_BASE_PATH", t.TempDir())
	defer os.Unsetenv("BEAVER_FILEKIT_DRIVER")
	defer os.Unsetenv("BEAVER_FILEKIT_LOCAL_BASE_PATH")

	err := InitFromEnv()
	if err != nil {
		t.Fatalf("InitFromEnv() error = %v", err)
	}

	// Test FS() returns the same instance
	fs1 := FS()
	_ = FS() // Get second instance but don't use it - we can't compare wrapped instances
	if fs1 == nil {
		t.Fatal("FS() returned nil")
	}
	// Note: We can't directly compare pointers due to wrapping, but we can test functionality

	// Test Reset
	Reset()
	fs3 := FS()
	if fs3 == nil {
		t.Fatal("FS() returned nil after Reset")
	}
}

func TestDefaultOptionsFS(t *testing.T) {
	// Create a mock filesystem
	tmpDir := t.TempDir()
	baseFS, err := New(Config{
		Driver:        "local",
		LocalBasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create base filesystem: %v", err)
	}

	// Create defaultOptionsFS
	defaultFS := &defaultOptionsFS{
		fs: baseFS,
		options: []Option{
			WithContentType("text/plain"),
			WithVisibility(Public),
		},
	}

	// Test that default options are applied
	ctx := context.Background()
	content := strings.NewReader("test content")
	err = defaultFS.Write(ctx, "test.txt", content)
	if err != nil {
		t.Errorf("Write with default options failed: %v", err)
	}

	// Verify file exists
	exists, err := defaultFS.FileExists(ctx, "test.txt")
	if err != nil {
		t.Errorf("FileExists() error = %v", err)
	}
	if !exists {
		t.Error("File should exist after write")
	}

	// Test other methods are passed through
	info, err := defaultFS.Stat(ctx, "test.txt")
	if err != nil {
		t.Errorf("Stat() error = %v", err)
	}
	if info == nil {
		t.Error("Stat() returned nil")
	}

	// Test delete
	err = defaultFS.Delete(ctx, "test.txt")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tmpDir := t.TempDir()

	// Test full configuration with validation and encryption
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	cfg := Config{
		Driver:            "local",
		LocalBasePath:     tmpDir,
		MaxFileSize:       1024 * 1024, // 1MB
		AllowedMimeTypes:  "text/plain,application/json,application/octet-stream",
		AllowedExtensions: ".txt,.json",
		DefaultVisibility: "private",
		EncryptionEnabled: true,
		EncryptionKey:     base64.StdEncoding.EncodeToString(encKey),
	}

	fs, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	ctx := context.Background()

	// Test write
	content := strings.NewReader("Hello, encrypted world!")
	err = fs.Write(ctx, "encrypted.txt", content, WithContentType("text/plain"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Test read
	reader, err := fs.Read(ctx, "encrypted.txt")
	if err != nil {
		t.Errorf("Read failed: %v", err)
		return // Exit early if read fails
	}
	defer reader.Close()

	// Verify content is decrypted correctly
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read content: %v", err)
	}
	if string(data) != "Hello, encrypted world!" {
		t.Errorf("Content = %v, want %v", string(data), "Hello, encrypted world!")
	}

	// Verify file is actually encrypted on disk
	rawPath := filepath.Join(tmpDir, "encrypted.txt")
	rawContent, err := os.ReadFile(rawPath)
	if err != nil {
		t.Errorf("Failed to read raw file: %v", err)
	}
	if string(rawContent) == "Hello, encrypted world!" {
		t.Error("File should be encrypted on disk")
	}
}
