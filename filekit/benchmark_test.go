package filekit

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func BenchmarkFilesystem(b *testing.B) {
	tmpDir := b.TempDir()
	content := strings.Repeat("Hello, World! ", 100) // ~1.4KB of content

	configs := map[string]Config{
		"basic": {
			Driver:        "local",
			LocalBasePath: tmpDir,
		},
		"with_validation": {
			Driver:           "local",
			LocalBasePath:    tmpDir,
			MaxFileSize:      10 * 1024 * 1024, // 10MB
			AllowedMimeTypes: "text/plain",
		},
		"with_encryption": {
			Driver:            "local",
			LocalBasePath:     tmpDir,
			EncryptionEnabled: true,
			EncryptionKey:     base64.StdEncoding.EncodeToString(make([]byte, 32)),
		},
		"with_defaults": {
			Driver:              "local",
			LocalBasePath:       tmpDir,
			DefaultVisibility:   "private",
			DefaultCacheControl: "no-cache",
		},
		"with_all": {
			Driver:              "local",
			LocalBasePath:       tmpDir,
			MaxFileSize:         10 * 1024 * 1024,
			AllowedMimeTypes:    "text/plain",
			EncryptionEnabled:   true,
			EncryptionKey:       base64.StdEncoding.EncodeToString(make([]byte, 32)),
			DefaultVisibility:   "private",
			DefaultCacheControl: "no-cache",
		},
	}

	for name, cfg := range configs {
		b.Run(name, func(b *testing.B) {
			fs, err := New(cfg)
			if err != nil {
				b.Fatalf("Failed to create filesystem: %v", err)
			}

			ctx := context.Background()

			b.Run("upload", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					reader := strings.NewReader(content)
					err := fs.Upload(ctx, "bench.txt", reader, WithContentType("text/plain"))
					if err != nil {
						b.Fatalf("Upload failed: %v", err)
					}
					_ = fs.Delete(ctx, "bench.txt")
				}
			})

			// Setup file for download benchmark
			_ = fs.Upload(ctx, "bench.txt", strings.NewReader(content), WithContentType("text/plain"))

			b.Run("download", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					reader, err := fs.Download(ctx, "bench.txt")
					if err != nil {
						b.Fatalf("Download failed: %v", err)
					}
					reader.Close()
				}
			})

			b.Run("exists", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := fs.Exists(ctx, "bench.txt")
					if err != nil {
						b.Fatalf("Exists failed: %v", err)
					}
				}
			})

			b.Run("fileinfo", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := fs.FileInfo(ctx, "bench.txt")
					if err != nil {
						b.Fatalf("FileInfo failed: %v", err)
					}
				}
			})

			// Cleanup
			_ = fs.Delete(ctx, "bench.txt")
		})
	}
}

func BenchmarkConfigCreation(b *testing.B) {
	// Benchmark config loading from environment
	os.Setenv("BEAVER_FILEKIT_DRIVER", "s3")
	os.Setenv("BEAVER_FILEKIT_S3_BUCKET", "test-bucket")
	os.Setenv("BEAVER_FILEKIT_S3_REGION", "us-west-2")
	os.Setenv("BEAVER_FILEKIT_MAX_FILE_SIZE", "10485760")
	os.Setenv("BEAVER_FILEKIT_ALLOWED_MIME_TYPES", "image/jpeg,image/png,text/plain")
	defer func() {
		os.Unsetenv("BEAVER_FILEKIT_DRIVER")
		os.Unsetenv("BEAVER_FILEKIT_S3_BUCKET")
		os.Unsetenv("BEAVER_FILEKIT_S3_REGION")
		os.Unsetenv("BEAVER_FILEKIT_MAX_FILE_SIZE")
		os.Unsetenv("BEAVER_FILEKIT_ALLOWED_MIME_TYPES")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetConfig()
		if err != nil {
			b.Fatalf("GetConfig failed: %v", err)
		}
	}
}

func BenchmarkFilesystemCreation(b *testing.B) {
	tmpDir := b.TempDir()

	configs := map[string]Config{
		"minimal": {
			Driver:        "local",
			LocalBasePath: tmpDir,
		},
		"with_validation": {
			Driver:            "local",
			LocalBasePath:     tmpDir,
			MaxFileSize:       10 * 1024 * 1024,
			AllowedMimeTypes:  "text/plain,application/json",
			AllowedExtensions: ".txt,.json",
		},
		"full_featured": {
			Driver:              "local",
			LocalBasePath:       tmpDir,
			MaxFileSize:         10 * 1024 * 1024,
			AllowedMimeTypes:    "text/plain,application/json",
			AllowedExtensions:   ".txt,.json",
			BlockedMimeTypes:    "application/x-executable",
			BlockedExtensions:   ".exe,.bat",
			EncryptionEnabled:   true,
			EncryptionKey:       base64.StdEncoding.EncodeToString(make([]byte, 32)),
			DefaultVisibility:   "private",
			DefaultCacheControl: "no-cache",
			DefaultOverwrite:    true,
		},
	}

	for name, cfg := range configs {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := New(cfg)
				if err != nil {
					b.Fatalf("New failed: %v", err)
				}
			}
		})
	}
}
