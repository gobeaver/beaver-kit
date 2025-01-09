# FileKit

FileKit is a robust, secure, and easy-to-use filesystem abstraction package for Go that handles multiple cloud providers with a unified API.

## Features

- Small, composable interfaces following Go idioms
- Clear separation of concerns between different filesystem operations
- Strong security with built-in encryption and access control
- Streaming-first approach for efficient handling of large files
- Comprehensive error handling with rich context
- Extensive adapter support with a consistent API across all storage backends
- Configuration-based initialization with environment variable support
- Built-in file validation and constraints

## Installation

```bash
go get github.com/beaver-kit/filekit
```

## Quick Start

### Using Configuration

FileKit now supports configuration-based initialization using the `config` package:

```go
// Initialize from environment variables
if err := filekit.InitFromEnv(); err != nil {
    log.Fatal(err)
}

// Use the global instance
fs := filekit.FS()

// Upload a file
err := fs.Upload(context.Background(), "example.txt", content)
```

### Environment Variables

Configure FileKit using environment variables:

```bash
# Driver selection
BEAVER_FILEKIT_DRIVER=s3  # or "local"

# Local driver settings
BEAVER_FILEKIT_LOCAL_BASE_PATH=./storage

# S3 driver settings
BEAVER_FILEKIT_S3_REGION=us-east-1
BEAVER_FILEKIT_S3_BUCKET=my-bucket
BEAVER_FILEKIT_S3_PREFIX=uploads/
BEAVER_FILEKIT_S3_ACCESS_KEY_ID=your-key-id
BEAVER_FILEKIT_S3_SECRET_ACCESS_KEY=your-secret-key

# Default options
BEAVER_FILEKIT_DEFAULT_VISIBILITY=private
BEAVER_FILEKIT_DEFAULT_CACHE_CONTROL=max-age=3600
BEAVER_FILEKIT_DEFAULT_OVERWRITE=false

# File validation
BEAVER_FILEKIT_MAX_FILE_SIZE=10485760  # 10MB
BEAVER_FILEKIT_ALLOWED_MIME_TYPES=image/jpeg,image/png,application/pdf
BEAVER_FILEKIT_ALLOWED_EXTENSIONS=.jpg,.jpeg,.png,.pdf

# Encryption
BEAVER_FILEKIT_ENCRYPTION_ENABLED=false
BEAVER_FILEKIT_ENCRYPTION_ALGORITHM=AES-256-GCM
BEAVER_FILEKIT_ENCRYPTION_KEY=your-base64-encoded-32-byte-key
```

### Custom Configuration

Create a custom configuration programmatically:

```go
cfg := filekit.Config{
    Driver:        "s3",
    S3Region:      "us-west-2",
    S3Bucket:      "my-bucket",
    S3Prefix:      "uploads/",
    
    // Default options
    DefaultVisibility: "private",
    MaxFileSize:      5 * 1024 * 1024, // 5MB
    AllowedMimeTypes: "image/jpeg,image/png",
}

fs, err := filekit.New(cfg)
if err != nil {
    log.Fatal(err)
}
```

### Generating Encryption Keys

To use encryption, you need a base64-encoded 32-byte key:

```go
import (
    "crypto/rand"
    "encoding/base64"
)

// Generate a random 32-byte key
key := make([]byte, 32)
if _, err := rand.Read(key); err != nil {
    panic(err)
}

// Encode to base64
encodedKey := base64.StdEncoding.EncodeToString(key)
fmt.Println("BEAVER_FILEKIT_ENCRYPTION_KEY=" + encodedKey)
```

## Usage

### Local Filesystem

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gobeaver/beaver-kit/filekit"
	"github.com/gobeaver/beaver-kit/filekit/driver/local"
)

func main() {
	// Create a local filesystem adapter
	fs, err := local.New("/tmp/filekit")
	if err != nil {
		panic(err)
	}

	// Upload a file
	content := strings.NewReader("Hello, World!")
	err = fs.Upload(context.Background(), "example.txt", content, filekit.WithContentType("text/plain"))
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded successfully")

	// Download the file
	reader, err := fs.Download(context.Background(), "example.txt")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("File content:", string(data))

	// Delete the file
	err = fs.Delete(context.Background(), "example.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println("File deleted successfully")
}
```

### S3 Filesystem

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gobeaver/beaver-kit/filekit"
	"github.com/gobeaver/beaver-kit/filekit/driver/s3"
)

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Create S3 filesystem adapter
	fs := s3.New(s3Client, "my-bucket", s3.WithPrefix("my-prefix"))

	// Upload a file
	content := strings.NewReader("Hello, World!")
	err = fs.Upload(context.Background(), "example.txt", content, 
		filekit.WithContentType("text/plain"),
		filekit.WithVisibility(filekit.Private),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded successfully")

	// Download the file
	reader, err := fs.Download(context.Background(), "example.txt")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("File content:", string(data))

	// Delete the file
	err = fs.Delete(context.Background(), "example.txt")
	if err != nil {
		panic(err)
	}
	fmt.Println("File deleted successfully")
}
```

### Chunked Uploads

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/beaver-kit/filekit"
	"github.com/beaver-kit/filekit/driver/local"
)

func main() {
	// Create a local filesystem adapter
	fs, err := local.New("/tmp/filekit")
	if err != nil {
		panic(err)
	}

	// Open a large file
	file, err := os.Open("large-file.zip")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}

	// Upload with progress reporting
	err = filekit.Upload(context.Background(), fs, "large-file.zip", file, fileInfo.Size(), &filekit.UploadOptions{
		ContentType: "application/zip",
		ChunkSize:   5 * 1024 * 1024, // 5MB chunks
		Progress: func(transferred, total int64) {
			fmt.Printf("Progress: %d of %d bytes (%.2f%%)\n", transferred, total, float64(transferred)/float64(total)*100)
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("File uploaded successfully")
}
```

### Encrypted Filesystem

```go
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"github.com/beaver-kit/filekit"
	"github.com/beaver-kit/filekit/driver/local"
)

func main() {
	// Create a local filesystem adapter
	fs, err := local.New("/tmp/filekit")
	if err != nil {
		panic(err)
	}

	// Generate a random encryption key (32 bytes for AES-256)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	// Create an encrypted filesystem
	encryptedFS := filekit.NewEncryptedFS(fs, key)

	// Upload a file with encryption
	content := strings.NewReader("This is a secret message")
	err = encryptedFS.Upload(context.Background(), "secret.txt", content)
	if err != nil {
		panic(err)
	}
	fmt.Println("Encrypted file uploaded successfully")

	// Download and decrypt the file
	reader, err := encryptedFS.Download(context.Background(), "secret.txt")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("Decrypted content:", string(data))

	// Try to download with the original filesystem (will be encrypted)
	reader, err = fs.Download(context.Background(), "secret.txt")
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	encryptedData, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("Raw encrypted content length:", len(encryptedData))
}
```

### File Validation

FileKit integrates with the `filevalidator` package for comprehensive file validation:

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/beaver-kit/filekit"
	"github.com/beaver-kit/filekit/driver/local"
	"github.com/beaver-kit/filevalidator"
)

func main() {
	// Create a local filesystem adapter
	fs, err := local.New("/tmp/filekit")
	if err != nil {
		panic(err)
	}

	// Create a file validator with constraints
	constraints := filevalidator.Constraints{
		MaxFileSize:   10 * 1024 * 1024, // 10MB
		AcceptedTypes: []string{"image/*", "application/pdf"},
		AllowedExts:   []string{".jpg", ".jpeg", ".png", ".pdf"},
	}
	validator := filevalidator.New(constraints)

	// Create a validated filesystem
	validatedFS := filekit.NewValidatedFileSystem(fs, validator)

	// Upload a file with automatic validation
	content := strings.NewReader("PDF content here...")
	err = validatedFS.Upload(context.Background(), "document.pdf", content, 
		filekit.WithContentType("application/pdf"))
	if err != nil {
		fmt.Println("Validation error:", err)
		return
	}
	fmt.Println("File uploaded successfully")
}
```

## Adapters

FileKit includes the following adapters:

- Local filesystem
- S3 compatible storage
- Encrypted filesystem wrapper

## Error Handling

FileKit provides detailed error handling with context:

```go
err := fs.Upload(context.Background(), "example.txt", content)
if err != nil {
	var pathErr *filekit.PathError
	if errors.As(err, &pathErr) {
		fmt.Printf("Operation: %s, Path: %s, Error: %v\n", pathErr.Op, pathErr.Path, pathErr.Err)
	}
	
	if filekit.IsNotExist(err) {
		fmt.Println("File does not exist")
	} else if filekit.IsPermission(err) {
		fmt.Println("Permission denied")
	}
}
```

## Package Structure

```
filekit/
├── fs.go           (main interfaces)
├── config.go       (configuration struct and loader)
├── service.go      (global instance management)
├── upload.go       (upload specific implementations)
├── stream.go       (streaming specific implementations)
├── errors.go       (error handling)
├── validated_fs.go (validation wrapper)
├── encryption.go   (encryption layer)
├── driver/
│   ├── local/
│   │   └── local.go
│   ├── s3/
│   │   └── s3.go
│   └── gcs/
│       └── gcs.go (future implementation)
└── options.go      (shared options and configurations)
```

## TODO

- FTP & SFTP adapter
- Google Cloud Storage adapter
- Memory adapter for testing
- Additional encryption methods

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.