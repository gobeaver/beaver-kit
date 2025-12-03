package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gobeaver/beaver-kit/filekit"
	"github.com/gobeaver/beaver-kit/filekit/driver/local"
	"github.com/gobeaver/beaver-kit/filevalidator"
)

func main() {
	// Create a temporary directory for our examples
	tempDir := createTempDir()
	fmt.Printf("Using temporary directory: %s\n", tempDir)
	defer os.RemoveAll(tempDir)

	// Create a local filesystem adapter
	fs, err := local.New(tempDir)
	if err != nil {
		panic(err)
	}

	// Example 1: Basic file upload and download
	basicExample(fs)

	// Example 2: Directory operations
	directoryExample(fs)

	// Example 3: File metadata
	metadataExample(fs)

	// Example 4: File validation
	validationExample(fs)

	fmt.Println("All examples completed successfully!")
}

func basicExample(fs filekit.FileSystem) {
	fmt.Println("\n--- Basic Example ---")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Write a file
	content := strings.NewReader("Hello, World! This is a test file.")
	err := fs.Write(ctx, "test.txt", content, filekit.WithContentType("text/plain"))
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded successfully")

	// Check if the file exists
	exists, err := fs.FileExists(ctx, "test.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("File exists: %v\n", exists)

	// Read the file
	reader, err := fs.Read(ctx, "test.txt")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	// Read the content
	data, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("File content: %s\n", string(data))

	// Delete the file
	err = fs.Delete(ctx, "test.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println("File deleted successfully")

	// Verify it's gone
	exists, err = fs.FileExists(ctx, "test.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("File exists after deletion: %v\n", exists)
}

func directoryExample(fs filekit.FileSystem) {
	fmt.Println("\n--- Directory Example ---")

	ctx := context.Background()

	// Create a directory
	err := fs.CreateDir(ctx, "images")
	if err != nil {
		panic(err)
	}
	fmt.Println("Directory created successfully")

	// Write a file to the directory
	content := strings.NewReader("Image data would go here")
	err = fs.Write(ctx, "images/photo.jpg", content, filekit.WithContentType("image/jpeg"))
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded to directory successfully")

	// List directory contents
	files, err := fs.ListContents(ctx, "images", false)
	if err != nil {
		panic(err)
	}
	fmt.Println("Directory contents:")
	for _, file := range files {
		fmt.Printf("- %s (Size: %d bytes, Type: %s)\n", file.Name, file.Size, file.ContentType)
	}

	// Delete the directory
	err = fs.DeleteDir(ctx, "images")
	if err != nil {
		panic(err)
	}
	fmt.Println("Directory deleted successfully")
}

func metadataExample(fs filekit.FileSystem) {
	fmt.Println("\n--- Metadata Example ---")

	ctx := context.Background()

	// Write a file with metadata
	content := strings.NewReader("File with metadata")
	metadata := map[string]string{
		"owner":      "John Doe",
		"department": "Engineering",
		"project":    "File System Demo",
	}
	err := fs.Write(ctx, "metadata.txt", content,
		filekit.WithContentType("text/plain"),
		filekit.WithMetadata(metadata),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded with metadata successfully")

	// Get file info
	fileInfo, err := fs.Stat(ctx, "metadata.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("File: %s\n", fileInfo.Name)
	fmt.Printf("Size: %d bytes\n", fileInfo.Size)
	fmt.Printf("Content Type: %s\n", fileInfo.ContentType)
	fmt.Printf("Last Modified: %s\n", fileInfo.ModTime)
	if fileInfo.Metadata != nil {
		fmt.Println("Metadata:")
		for k, v := range fileInfo.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// Clean up
	err = fs.Delete(ctx, "metadata.txt")
	if err != nil {
		panic(err)
	}
}

func validationExample(fs filekit.FileSystem) {
	fmt.Println("\n--- Validation Example ---")

	ctx := context.Background()

	// Create a file validator with constraints
	constraints := filevalidator.Constraints{
		MaxFileSize:   1024 * 1024, // 1MB
		AcceptedTypes: []string{"text/plain", "application/json"},
		AllowedExts:   []string{".txt", ".json"},
	}
	validator := filevalidator.New(constraints)

	// Create a validated filesystem
	validatedFS := filekit.NewValidatedFileSystem(fs, validator)

	// Create a valid file
	validContent := strings.NewReader("This is a valid text file")
	err := validatedFS.Write(ctx, "valid.txt", validContent, filekit.WithContentType("text/plain"))
	if err != nil {
		fmt.Printf("Upload error: %v\n", err)
	} else {
		fmt.Println("Valid file uploaded successfully")
	}

	// Try to create an invalid file (wrong extension)
	invalidContent := strings.NewReader("This file has the wrong extension")
	err = validatedFS.Write(ctx, "invalid.png", invalidContent, filekit.WithContentType("text/plain"))
	if err != nil {
		fmt.Printf("Validation error as expected: %v\n", err)
	} else {
		fmt.Println("Invalid file unexpectedly passed validation")
		// Clean up if it was created
		_ = fs.Delete(ctx, "invalid.png")
	}

	// Try to upload a file that's too large
	largeContent := strings.NewReader(strings.Repeat("x", 2*1024*1024)) // 2MB
	err = validatedFS.Write(ctx, "large.txt", largeContent, filekit.WithContentType("text/plain"))
	if err != nil {
		fmt.Printf("Size validation error as expected: %v\n", err)
	} else {
		fmt.Println("Large file unexpectedly passed validation")
		// Clean up if it was created
		_ = fs.Delete(ctx, "large.txt")
	}

	// Clean up
	_ = fs.Delete(ctx, "valid.txt")
}

func createTempDir() string {
	tempDir, err := os.MkdirTemp("", "filekit-example")
	if err != nil {
		panic(err)
	}
	return tempDir
}
