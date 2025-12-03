package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gobeaver/beaver-kit/filekit"
)

func main() {
	// Example 1: Initialize from environment variables
	if err := filekit.InitFromEnv(); err != nil {
		log.Fatal("Failed to initialize filekit:", err)
	}

	// Use the global instance
	fs := filekit.FS()
	fmt.Printf("Global filesystem initialized: %v\n", fs != nil)

	// Example 2: Create a custom configuration
	cfg := filekit.Config{
		Driver:        "local",
		LocalBasePath: "./uploads",

		// Default options
		DefaultVisibility:   "private",
		DefaultCacheControl: "max-age=3600",

		// File validation
		MaxFileSize:       5 * 1024 * 1024, // 5MB
		AllowedMimeTypes:  "image/jpeg,image/png,application/pdf",
		AllowedExtensions: ".jpg,.jpeg,.png,.pdf",
	}

	// Create a new instance with custom config
	customFS, err := filekit.New(cfg)
	if err != nil {
		log.Fatal("Failed to create filekit instance:", err)
	}

	// Example 3: S3 configuration with encryption
	// Generate a 32-byte key and encode it to base64:
	// key := make([]byte, 32)
	// rand.Read(key)
	// encodedKey := base64.StdEncoding.EncodeToString(key)
	s3Config := filekit.Config{
		Driver:            "s3",
		S3Region:          "us-west-2",
		S3Bucket:          "my-bucket",
		S3Prefix:          "uploads/",
		S3AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		S3SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),

		// Enable encryption with base64-encoded key
		EncryptionEnabled:   true,
		EncryptionAlgorithm: "AES-256-GCM",
		EncryptionKey:       os.Getenv("ENCRYPTION_KEY"), // Must be base64-encoded 32-byte key
	}

	s3FS, err := filekit.New(s3Config)
	if err != nil {
		log.Fatal("Failed to create S3 filekit instance:", err)
	}

	// Example usage
	ctx := context.Background()

	// Write a file
	content := strings.NewReader("Hello, World!")
	if err := customFS.Write(ctx, "hello.txt", content); err != nil {
		log.Printf("Upload failed: %v", err)
	}

	// Check if file exists
	exists, err := customFS.FileExists(ctx, "hello.txt")
	if err != nil {
		log.Printf("Exists check failed: %v", err)
	}
	fmt.Printf("File exists: %v\n", exists)

	// Get file info
	info, err := customFS.Stat(ctx, "hello.txt")
	if err != nil {
		log.Printf("FileInfo failed: %v", err)
	} else {
		fmt.Printf("File info: %+v\n", info)
	}

	// List files
	files, err := customFS.ListContents(ctx, "", false)
	if err != nil {
		log.Printf("List failed: %v", err)
	} else {
		fmt.Printf("Files: %+v\n", files)
	}

	// Example with S3 (commented out unless you have S3 configured)
	_ = s3FS // Avoid unused variable warning
	/*
		if err := s3FS.Write(ctx, "test.txt", strings.NewReader("S3 test")); err != nil {
			log.Printf("S3 upload failed: %v", err)
		}
	*/
}
