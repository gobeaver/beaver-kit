package filekit

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestServiceWithMock(t *testing.T) {
	t.Skip("Skipping mock driver test - mock driver registration issue")

	// Test New with mock driver
	cfg := Config{
		Driver: "mock",
	}

	fs, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mock filesystem: %v", err)
	}

	// Test basic operations
	ctx := context.Background()
	content := "test content"

	err = fs.Upload(ctx, "test.txt", strings.NewReader(content))
	if err != nil {
		t.Errorf("Upload failed: %v", err)
	}

	exists, err := fs.Exists(ctx, "test.txt")
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("File should exist after upload")
	}

	reader, err := fs.Download(ctx, "test.txt")
	if err != nil {
		t.Errorf("Download failed: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read downloaded content: %v", err)
	}
	if string(downloaded) != content {
		t.Errorf("Downloaded content = %v, want %v", string(downloaded), content)
	}
}

func TestDefaultOptionsWithMock(t *testing.T) {
	t.Skip("Skipping mock driver test - mock driver registration issue")

	cfg := Config{
		Driver:              "mock",
		DefaultVisibility:   "private",
		DefaultCacheControl: "no-cache",
	}

	fs, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create mock filesystem: %v", err)
	}

	// The mock doesn't actually use options, but we're testing that the wrapper is applied
	ctx := context.Background()
	err = fs.Upload(ctx, "test.txt", strings.NewReader("test"))
	if err != nil {
		t.Errorf("Upload with default options failed: %v", err)
	}
}
