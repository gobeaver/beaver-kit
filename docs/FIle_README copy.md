# FileKit

A comprehensive, production-ready filesystem abstraction library for Go with support for multiple storage backends, encryption, validation, and virtual path mounting.

[![Go Reference](https://pkg.go.dev/badge/github.com/beaver-kit/filekit.svg)](https://pkg.go.dev/github.com/beaver-kit/filekit)
[![Go Report Card](https://goreportcard.com/badge/github.com/beaver-kit/filekit)](https://goreportcard.com/report/github.com/beaver-kit/filekit)

## Features

- **7 Storage Backends** - Local, S3, GCS, Azure Blob, SFTP, Memory, ZIP
- **Unified API** - Same interface across all storage backends
- **Mount Manager** - Virtual path namespacing with cross-mount operations
- **Stackable Decorators** - ReadOnly, Caching, Encryption, Validation (composable)
- **File Selection** - Composable selectors (Glob, Extension, Size, ModTime, ContentType, Metadata)
- **Built-in Encryption** - AES-256-GCM encryption layer
- **Metadata Caching** - Pluggable cache for FileExists/Stat/ListContents operations
- **File Validation** - Integration with filevalidator for MIME type, size, and extension validation
- **Progress Tracking** - Upload progress callbacks for large files
- **Chunked Uploads** - Multipart upload support for S3
- **Pre-signed URLs** - Generate temporary access URLs for cloud storage
- **Pure Go** - No CGO dependencies, easy cross-compilation
- **Thread-Safe** - All operations are safe for concurrent use

## Installation

```bash
go get github.com/beaver-kit/filekit
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/beaver-kit/filekit/driver/local"
)

func main() {
    // Create a local filesystem
    fs, _ := local.New("./storage")

    ctx := context.Background()

    // Write a file
    fs.Write(ctx, "hello.txt", strings.NewReader("Hello, World!"))

    // Read the file
    reader, _ := fs.Read(ctx, "hello.txt")
    defer reader.Close()

    // Or read entire file into memory (convenience method)
    data, _ := fs.ReadAll(ctx, "hello.txt")
    fmt.Println(string(data))

    // Check if file exists
    exists, _ := fs.FileExists(ctx, "hello.txt")

    // Check if directory exists
    dirExists, _ := fs.DirExists(ctx, "uploads")

    // Get file metadata
    info, _ := fs.Stat(ctx, "hello.txt")

    // List files (non-recursive)
    files, _ := fs.ListContents(ctx, "/", false)

    // List files (recursive)
    allFiles, _ := fs.ListContents(ctx, "/", true)

    // Delete the file
    fs.Delete(ctx, "hello.txt")

    _ = exists
    _ = dirExists
    _ = info
    _ = files
    _ = allFiles
}
```

---

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [Optional Capability Interfaces](#optional-capability-interfaces)
- [Driver Implementation Matrix](#driver-implementation-matrix)
- [Storage Drivers](#storage-drivers)
- [Mount Manager](#mount-manager)
- [Middleware & Wrappers](#middleware--wrappers)
- [Configuration](#configuration)
- [Write Options](#write-options)
- [Error Handling](#error-handling)
- [Feature Comparison](#feature-comparison)

---

## Core Interfaces

FileKit uses interface segregation for compile-time safety. Use `FileReader` for read-only access and `FileSystem` for full access.

### FileReader Interface (Read-Only)

```go
type FileReader interface {
    // Read returns a stream for reading file content.
    Read(ctx context.Context, path string) (io.ReadCloser, error)

    // ReadAll reads entire file into memory. Use for small files only.
    ReadAll(ctx context.Context, path string) ([]byte, error)

    // FileExists checks if a file exists at path.
    FileExists(ctx context.Context, path string) (bool, error)

    // DirExists checks if a directory exists at path.
    DirExists(ctx context.Context, path string) (bool, error)

    // Stat returns file/directory metadata.
    Stat(ctx context.Context, path string) (*FileInfo, error)

    // ListContents lists directory contents.
    // If recursive is true, includes all descendants.
    ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error)
}
```

### FileWriter Interface (Write Operations)

```go
type FileWriter interface {
    // Write writes content from reader to path.
    Write(ctx context.Context, path string, r io.Reader, opts ...Option) error

    // Delete removes a file.
    Delete(ctx context.Context, path string) error

    // CreateDir creates a directory (and parents if needed).
    CreateDir(ctx context.Context, path string) error

    // DeleteDir removes a directory and all contents.
    DeleteDir(ctx context.Context, path string) error
}
```

### FileSystem Interface (Full Access)

```go
type FileSystem interface {
    FileReader
    FileWriter
}
```

### FileInfo Struct

```go
type FileInfo struct {
    Name        string            // Base name of the file
    Path        string            // Full path to the file
    Size        int64             // Size in bytes
    ModTime     time.Time         // Last modification time
    IsDir       bool              // True if directory
    ContentType string            // MIME type
    Metadata    map[string]string // Custom metadata
}
```

### ChunkedUploader Interface

For filesystems that support multipart uploads (like S3):

```go
type ChunkedUploader interface {
    InitiateUpload(ctx context.Context, path string) (string, error)
    UploadPart(ctx context.Context, uploadID string, partNumber int, data []byte) error
    CompleteUpload(ctx context.Context, uploadID string) error
    AbortUpload(ctx context.Context, uploadID string) error
}
```

---

## Optional Capability Interfaces

FileKit uses optional interfaces to expose native capabilities of each storage backend. Check for interface support at runtime using type assertions.

### Interface Definitions

| Interface | Description | Methods |
|-----------|-------------|---------|
| `CanCopy` | Native file copy within same backend | `Copy(ctx, src, dst) error` |
| `CanMove` | Native file move/rename within same backend | `Move(ctx, src, dst) error` |
| `CanSignURL` | Generate pre-signed URLs for direct access | `SignedURL(ctx, path, expires)`, `SignedUploadURL(ctx, path, expires)` |
| `CanChecksum` | Calculate file checksums/hashes | `Checksum(ctx, path, algorithm)`, `Checksums(ctx, path, algorithms)` |
| `CanWatch` | File change detection (ChangeToken pattern) | `Watch(ctx, pattern) (ChangeToken, error)` |

### Interface Details

```go
// CanCopy - Native file copy (more efficient than read+write)
type CanCopy interface {
    Copy(ctx context.Context, src, dst string) error
}

// CanMove - Native file move/rename
type CanMove interface {
    Move(ctx context.Context, src, dst string) error
}

// CanSignURL - Pre-signed URL generation
type CanSignURL interface {
    SignedURL(ctx context.Context, path string, expires time.Duration) (string, error)
    SignedUploadURL(ctx context.Context, path string, expires time.Duration) (string, error)
}

// CanChecksum - File integrity verification (single and multi-hash)
type CanChecksum interface {
    Checksum(ctx context.Context, path string, algorithm ChecksumAlgorithm) (string, error)
    Checksums(ctx context.Context, path string, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error)
}

// ChangeToken - File change notification token (Microsoft IChangeToken pattern)
type ChangeToken interface {
    // HasChanged returns true if a change has occurred (single-use)
    HasChanged() bool
    // ActiveChangeCallbacks indicates if callbacks are proactively raised
    ActiveChangeCallbacks() bool
    // RegisterChangeCallback registers a callback for change notification
    RegisterChangeCallback(callback func()) (unregister func())
}

// CanWatch - File change detection using ChangeToken pattern
type CanWatch interface {
    // Watch creates a change token for the specified filter pattern
    // Supports glob patterns: "**/*.txt", "config/*", "*.json"
    Watch(ctx context.Context, pattern string) (ChangeToken, error)
}
```

### Checksum Algorithms

```go
const (
    ChecksumMD5    ChecksumAlgorithm = "md5"     // Fast, not cryptographically secure
    ChecksumSHA1   ChecksumAlgorithm = "sha1"    // Legacy, 160-bit
    ChecksumSHA256 ChecksumAlgorithm = "sha256"  // Recommended, 256-bit
    ChecksumSHA512 ChecksumAlgorithm = "sha512"  // Most secure, 512-bit
    ChecksumCRC32  ChecksumAlgorithm = "crc32"   // Fastest, integrity only
    ChecksumXXHash ChecksumAlgorithm = "xxhash"  // Extremely fast, non-cryptographic
)
```

### Checksum Usage

```go
// Calculate SHA256 checksum
if cs, ok := fs.(filekit.CanChecksum); ok {
    hash, err := cs.Checksum(ctx, "document.pdf", filekit.ChecksumSHA256)
    fmt.Printf("SHA256: %s\n", hash)

    // Calculate multiple checksums in single pass (efficient for large files)
    hashes, err := cs.Checksums(ctx, "large-file.zip", []filekit.ChecksumAlgorithm{
        filekit.ChecksumMD5,
        filekit.ChecksumSHA256,
        filekit.ChecksumXXHash,
    })
    fmt.Printf("MD5: %s\n", hashes[filekit.ChecksumMD5])
    fmt.Printf("SHA256: %s\n", hashes[filekit.ChecksumSHA256])
    fmt.Printf("XXHash: %s\n", hashes[filekit.ChecksumXXHash])

    // Verify file integrity
    expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    actual, _ := cs.Checksum(ctx, "file.txt", filekit.ChecksumSHA256)
    if actual != expected {
        log.Fatal("File integrity check failed!")
    }
}
```

### ChangeToken Types

```go
// CallbackChangeToken - For native events (local, memory drivers)
// Supports active callbacks that are invoked immediately on change

// PollingChangeToken - For polling-based detection (cloud drivers)
// Polls at configured intervals to detect changes

// CompositeChangeToken - Combines multiple tokens
// HasChanged returns true if ANY underlying token has changed

// CancelledChangeToken - Already "changed" state
// Useful for signaling that watching is not supported

// NeverChangeToken - Never changes
// Useful for static content (e.g., ZIP archives)
```

### Usage Example

```go
// Check if driver supports native copy
if copier, ok := fs.(filekit.CanCopy); ok {
    // Use native copy (more efficient)
    err := copier.Copy(ctx, "source.txt", "destination.txt")
} else {
    // Fall back to read + write
    reader, _ := fs.Read(ctx, "source.txt")
    fs.Write(ctx, "destination.txt", reader)
}

// Generate pre-signed URL for direct download
if signer, ok := fs.(filekit.CanSignURL); ok {
    url, err := signer.SignedURL(ctx, "file.pdf", 15*time.Minute)
    // Share URL with client for direct download from storage
}

// Watch for file changes using ChangeToken pattern
if watcher, ok := fs.(filekit.CanWatch); ok {
    token, err := watcher.Watch(ctx, "**/*.json")
    if err != nil {
        log.Fatal(err)
    }

    // Option 1: Poll for changes
    if token.HasChanged() {
        log.Println("Files changed!")
    }

    // Option 2: Register callback (if supported)
    if token.ActiveChangeCallbacks() {
        unregister := token.RegisterChangeCallback(func() {
            log.Println("Change detected!")
            reloadConfig()
        })
        defer unregister()
    }
}

// Continuous watching with OnChange helper
cancel := filekit.OnChange(
    func() (filekit.ChangeToken, error) {
        return fs.(filekit.CanWatch).Watch(ctx, "config.json")
    },
    func() {
        log.Println("Config changed, reloading...")
        reloadConfig()
    },
)
defer cancel()
```

---

## File Selection & Filtering

FileKit provides a file selection API inspired by Apache Commons VFS (proven stable 20+ years). The same selectors work across all drivers.

### Usage

```go
// List with selector (like VFS findFiles)
files, err := filekit.ListWithSelector(ctx, fs, "/images", filekit.Glob("*.jpg"), true)

// Non-recursive
files, err := filekit.ListWithSelector(ctx, fs, "/", filekit.All(), false)
```

### Built-in Selectors

| Selector | VFS Equivalent | Description |
|----------|----------------|-------------|
| `All()` | AllFileSelector | Matches all files |
| `Glob(pattern)` | WildcardFileSelector | Glob patterns: `*`, `?`, `[a-z]` |
| `Depth(max, base)` | FileDepthSelector | Limit traversal depth |
| `And(selectors...)` | - | All must match |
| `Or(selectors...)` | - | Any must match |
| `Not(selector)` | - | Invert match |
| `FuncSelector(fn)` | - | Custom function (escape hatch) |

### Composing Selectors

```go
// JPG files under 10MB
selector := filekit.And(
    filekit.Glob("*.jpg"),
    filekit.FuncSelector(func(f *filekit.FileInfo) bool {
        return f.Size < 10*1024*1024
    }),
)
files, _ := filekit.ListWithSelector(ctx, fs, "/uploads", selector, true)

// Multiple patterns
selector := filekit.Or(
    filekit.Glob("*.txt"),
    filekit.Glob("*.json"),
)

// Exclude patterns
selector := filekit.And(
    filekit.Glob("*"),
    filekit.Not(filekit.Glob("*.tmp")),
)
```

### Custom Selectors

`FuncSelector` is the escape hatch for any logic not covered by built-ins:

```go
// Any custom filtering
selector := filekit.FuncSelector(func(f *filekit.FileInfo) bool {
    return f.Size > 1024 &&
           strings.Contains(f.Name, "report") &&
           f.ModTime.After(lastWeek)
})

// With traversal control
selector := filekit.FuncSelectorFull(
    func(f *filekit.FileInfo) bool { return f.Size > 0 },
    func(f *filekit.FileInfo) bool { return !strings.HasPrefix(f.Name, ".") },
)
```

### Why This API?

1. **Proven** - VFS pattern stable for 20+ years
2. **Minimal** - 4 built-in selectors + FuncSelector escape hatch
3. **Cross-driver** - Same code on local, S3, GCS, Azure, SFTP
4. **Composable** - And/Or/Not build any query
5. **Future-proof** - Add selectors later without breaking changes

---

## Driver Implementation Matrix

| Driver | FileSystem | CanCopy | CanMove | CanSignURL | CanChecksum | CanWatch | ChunkedUploader |
|--------|------------|---------|---------|------------|-------------|----------|-----------------|
| `local` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ Native | ❌ |
| `s3` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ Polling | ✅ |
| `gcs` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ Polling | ❌ |
| `azure` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ Polling | ❌ |
| `sftp` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ Polling | ❌ |
| `memory` | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ Native | ❌ |
| `zip` | ✅ | ✅ | ✅ | ❌ | ✅ | ⚠️ Never | ❌ |

**Watcher Types:**
- **Native**: Uses fsnotify for real-time file system events (local) or internal hooks (memory)
- **Polling**: Periodically checks for file changes (default: 30 second interval for cloud drivers)
- **Never**: Returns NeverChangeToken (for static content like ZIP archives)

---

## Storage Drivers

### Local Filesystem

```go
import "github.com/beaver-kit/filekit/driver/local"

// Create local filesystem rooted at a directory
fs, err := local.New("/var/uploads")
if err != nil {
    log.Fatal(err)
}

// All paths are relative to the root
fs.Write(ctx, "images/photo.jpg", reader)
fs.Read(ctx, "images/photo.jpg")
```

### Amazon S3

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    s3driver "github.com/beaver-kit/filekit/driver/s3"
)

// Load AWS config
cfg, _ := config.LoadDefaultConfig(ctx)
client := s3.NewFromConfig(cfg)

// Create S3 filesystem
fs := s3driver.New(client, "my-bucket",
    s3driver.WithPrefix("uploads/"),
)

// Write with options
fs.Write(ctx, "document.pdf", reader,
    filekit.WithContentType("application/pdf"),
    filekit.WithVisibility(filekit.Public),
    filekit.WithMetadata(map[string]string{"author": "john"}),
)

// Generate pre-signed download URL (valid for 1 hour)
url, _ := fs.SignedURL(ctx, "document.pdf", time.Hour)
```

S3-compatible services (MinIO, DigitalOcean Spaces, etc.):

```go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithEndpointResolver(aws.EndpointResolverFunc(
        func(service, region string) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL:           "https://nyc3.digitaloceanspaces.com",
                SigningRegion: "nyc3",
            }, nil
        })),
)
```

### Google Cloud Storage

```go
import (
    "cloud.google.com/go/storage"
    "github.com/beaver-kit/filekit/driver/gcs"
)

// Create GCS client (uses GOOGLE_APPLICATION_CREDENTIALS)
client, _ := storage.NewClient(ctx)

// Create GCS filesystem
fs := gcs.New(client, "my-bucket",
    gcs.WithPrefix("uploads/"),
)

// Generate signed URL
url, _ := fs.SignedURL(ctx, "file.pdf", 30*time.Minute)
```

### Azure Blob Storage

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
    "github.com/beaver-kit/filekit/driver/azure"
)

// Create Azure client
cred, _ := azblob.NewSharedKeyCredential(accountName, accountKey)
client, _ := azblob.NewClientWithSharedKeyCredential(
    fmt.Sprintf("https://%s.blob.core.windows.net", accountName),
    cred, nil,
)

// Create Azure filesystem
fs := azure.New(client, "my-container", accountName, accountKey,
    azure.WithPrefix("uploads/"),
)
```

### SFTP

```go
import "github.com/beaver-kit/filekit/driver/sftp"

// Create SFTP filesystem
fs, err := sftp.New(sftp.Config{
    Host:       "sftp.example.com",
    Port:       22,
    Username:   "user",
    Password:   "password",  // or use PrivateKey
    BasePath:   "/uploads",
})
if err != nil {
    log.Fatal(err)
}
defer fs.Close()

fs.Write(ctx, "report.csv", reader)
```

With private key authentication:

```go
fs, _ := sftp.New(sftp.Config{
    Host:       "sftp.example.com",
    Port:       22,
    Username:   "user",
    PrivateKey: "/path/to/id_rsa",
    BasePath:   "/uploads",
})
```

### Memory (In-Memory)

Perfect for testing and caching:

```go
import "github.com/beaver-kit/filekit/driver/memory"

// Create in-memory filesystem
fs := memory.New()

// Or with size limit
fs := memory.New(memory.Config{
    MaxSize: 100 * 1024 * 1024, // 100MB max
})

// Use like any other filesystem
fs.Write(ctx, "test.txt", strings.NewReader("hello"))
```

### ZIP Archive

Read and write ZIP files as a filesystem:

```go
import "github.com/beaver-kit/filekit/driver/zip"

// Read existing ZIP
fs, _ := zip.Open("/path/to/archive.zip")
defer fs.Close()

files, _ := fs.ListContents(ctx, "/", false)
reader, _ := fs.Read(ctx, "document.pdf")

// Create new ZIP
fs, _ := zip.Create("/path/to/new.zip")
fs.Write(ctx, "file.txt", strings.NewReader("hello"))
fs.CreateDir(ctx, "images")
fs.Write(ctx, "images/photo.jpg", imageReader)
fs.Close() // Finalizes the ZIP

// Read-write mode (modify existing ZIP)
fs, _ := zip.OpenOrCreate("/path/to/archive.zip")
fs.Write(ctx, "new-file.txt", reader)
fs.Delete(ctx, "old-file.txt")
fs.Close() // Rewrites ZIP with changes
```

---

## Mount Manager

The Mount Manager provides virtual path namespacing, allowing you to combine multiple storage backends under a unified path structure.

```go
import (
    "github.com/beaver-kit/filekit"
    "github.com/beaver-kit/filekit/driver/local"
    "github.com/beaver-kit/filekit/driver/memory"
)

// Create mount manager
mounts := filekit.NewMountManager()

// Mount different backends under virtual paths
localFS, _ := local.New("/var/uploads")
mounts.Mount("/local", localFS)
mounts.Mount("/cloud", s3Driver)
mounts.Mount("/cache", memory.New())

// Nested mounts are supported (longest-prefix matching)
mounts.Mount("/cloud/archive", glacierDriver)

// Transparent access - routes to correct backend
mounts.Write(ctx, "/local/file.txt", reader)
mounts.Read(ctx, "/cloud/image.png")
mounts.Read(ctx, "/cloud/archive/old-data.tar") // Goes to glacier

// Cross-mount operations (automatically handles read+write)
mounts.Copy(ctx, "/local/file.txt", "/cloud/backup/file.txt")
mounts.Move(ctx, "/cache/temp.txt", "/local/permanent.txt")

// List root shows all mount points
files, _ := mounts.ListContents(ctx, "/", false)
// Returns: [{Name: "local", IsDir: true}, {Name: "cloud", IsDir: true}, {Name: "cache", IsDir: true}]

// Manage mounts
mounts.Unmount("/cache")
allMounts := mounts.Mounts() // map[string]FileSystem
```

### Mount Manager Features

- **Virtual path namespacing** - Organize multiple backends under one path tree
- **Nested mount support** - Longest-prefix matching for mount resolution
- **Cross-mount copy/move** - Automatically reads from source, writes to destination
- **Native operations when possible** - Uses native Copy/Move if same backend supports it
- **Thread-safe** - All operations protected with RWMutex
- **Full FileSystem interface** - Can be used anywhere a FileSystem is expected

---

## Middleware & Wrappers

FileKit provides several decorators that wrap any `FileSystem` to add orthogonal concerns. These can be stacked: `Encryption → Validation → Caching → ReadOnly`.

### ReadOnly Filesystem

Prevent all write operations on a filesystem:

```go
import "github.com/beaver-kit/filekit"

// Create read-only wrapper
readOnly := filekit.NewReadOnlyFileSystem(fs)

// Read operations work normally
reader, _ := readOnly.Read(ctx, "file.txt")
exists, _ := readOnly.FileExists(ctx, "file.txt")
files, _ := readOnly.ListContents(ctx, "/", false)

// Write operations return ErrReadOnly
err := readOnly.Write(ctx, "file.txt", reader)
// Error: "write file.txt: filesystem is read-only"

// Check if error is due to read-only mode
if filekit.IsReadOnlyError(err) {
    log.Println("Cannot write to read-only filesystem")
}
```

With options for partial write permissions:

```go
// Allow directory creation but block other writes
readOnly := filekit.NewReadOnlyFileSystem(fs,
    filekit.WithAllowCreateDir(true),
)
readOnly.CreateDir(ctx, "logs")  // Allowed
readOnly.Write(ctx, "file.txt", reader)  // Blocked

// Allow deletion (use with caution)
readOnly := filekit.NewReadOnlyFileSystem(fs,
    filekit.WithAllowDelete(true),
)

// Custom handler for write attempts (logging, metrics)
readOnly := filekit.NewReadOnlyFileSystem(fs,
    filekit.WithWriteAttemptHandler(func(op, path string) error {
        log.Printf("Write attempt blocked: %s %s", op, path)
        metrics.IncrementCounter("readonly.blocked")
        return filekit.ErrReadOnly  // Return nil to allow
    }),
)

// Custom error wrapper
readOnly := filekit.NewReadOnlyFileSystem(fs,
    filekit.WithErrorWrapper(func(op, path string, err error) error {
        return fmt.Errorf("access denied: %s on %s: %w", op, path, err)
    }),
)
```

### Caching Filesystem

Cache metadata operations (FileExists, Stat, ListContents) for improved performance:

```go
import "github.com/beaver-kit/filekit"

// Create with default in-memory cache (5 minute TTL)
cached := filekit.NewCachingFileSystem(fs)

// First call hits the backend
exists, _ := cached.FileExists(ctx, "file.txt")  // Backend call
exists, _ := cached.FileExists(ctx, "file.txt")  // Cache hit!

// Writes automatically invalidate cache
cached.Write(ctx, "file.txt", reader)
exists, _ := cached.FileExists(ctx, "file.txt")  // Fresh backend call
```

With custom options:

```go
cached := filekit.NewCachingFileSystem(fs,
    filekit.WithCacheTTL(10 * time.Minute),
    filekit.WithCacheExists(true),     // Cache FileExists() results
    filekit.WithCacheFileInfo(true),   // Cache Stat() results
    filekit.WithCacheList(true),       // Cache ListContents() results
    filekit.WithInvalidateOnWrite(true), // Clear cache on write ops
)

// Path-based filtering
cached := filekit.NewCachingFileSystem(fs,
    filekit.WithCachePathFilter(func(path string) bool {
        // Only cache files in /static/ directory
        return strings.HasPrefix(path, "/static/")
    }),
)

// Cache hit/miss callbacks for monitoring
cached := filekit.NewCachingFileSystem(fs,
    filekit.WithCacheHitCallback(func(op, path string) {
        metrics.IncrementCounter("cache.hit", op, path)
    }),
    filekit.WithCacheMissCallback(func(op, path string) {
        metrics.IncrementCounter("cache.miss", op, path)
    }),
)
```

Using a custom cache backend (Redis, Memcached, etc.):

```go
// Implement the Cache interface
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
}

// Use custom backend
redisCache := myapp.NewRedisCache(redisClient)
cached := filekit.NewCachingFileSystem(fs,
    filekit.WithCache(redisCache),
)

// Pre-warm cache for hot files
err := filekit.WarmCache(ctx, cached, []string{
    "config.json",
    "templates/base.html",
    "static/logo.png",
})
```

Cache statistics (with MemoryCache):

```go
cache := filekit.NewMemoryCache()
cached := filekit.NewCachingFileSystem(fs, filekit.WithCache(cache))

// Perform operations...

// Get cache statistics
stats := cache.Stats()
fmt.Printf("Hits: %d, Misses: %d, Hit Rate: %.2f%%\n",
    stats.Hits, stats.Misses, stats.HitRate*100)
fmt.Printf("Size: %d entries, Evictions: %d\n",
    stats.Size, stats.Evictions)
```

Integration with Watcher for automatic invalidation:

```go
// The CachingFileSystem automatically invalidates cache
// when ChangeToken indicates a change
cached := filekit.NewCachingFileSystem(fs)

// If underlying fs supports CanWatch, changes are detected
if watcher, ok := fs.(filekit.CanWatch); ok {
    token, _ := watcher.Watch(ctx, "**/*")
    // CachingFileSystem will check token and invalidate as needed
}
```

### Encrypted Filesystem

Transparently encrypt/decrypt files using AES-256-GCM:

```go
import "github.com/beaver-kit/filekit"

// Generate a 32-byte encryption key
key := make([]byte, 32)
rand.Read(key)

// Wrap any filesystem with encryption
encryptedFS := filekit.NewEncryptedFS(fs, key)

// Write automatically encrypts
encryptedFS.Write(ctx, "secret.txt", strings.NewReader("sensitive data"))

// Read automatically decrypts
reader, _ := encryptedFS.Read(ctx, "secret.txt")
// reader contains decrypted content

// Reading with the original fs shows encrypted data
raw, _ := fs.Read(ctx, "secret.txt")
// raw contains encrypted binary data
```

### Validated Filesystem

Automatically validate files before write using filevalidator:

```go
import (
    "github.com/beaver-kit/filekit"
    "github.com/beaver-kit/filevalidator"
)

// Create validator with constraints
validator := filevalidator.New(filevalidator.Constraints{
    MaxFileSize:   10 * 1024 * 1024, // 10MB
    AcceptedTypes: []string{"image/*", "application/pdf"},
    AllowedExts:   []string{".jpg", ".jpeg", ".png", ".pdf"},
})

// Wrap filesystem with validation
validatedFS := filekit.NewValidatedFileSystem(fs, validator)

// Write fails if file doesn't meet constraints
err := validatedFS.Write(ctx, "malware.exe", reader)
// Error: file type not allowed

// Or use per-write validation
fs.Write(ctx, "photo.jpg", reader,
    filekit.WithValidator(validator),
)
```

---

## Configuration

### Environment Variables

Configure FileKit using environment variables with the `FILEKIT_` prefix:

```bash
# Driver selection
FILEKIT_DRIVER=s3  # local, s3, gcs, azure, sftp, memory

# Local driver
FILEKIT_LOCAL_BASE_PATH=./storage

# S3 driver
FILEKIT_S3_REGION=us-east-1
FILEKIT_S3_BUCKET=my-bucket
FILEKIT_S3_PREFIX=uploads/
FILEKIT_S3_ENDPOINT=                    # Optional: custom endpoint for S3-compatible services
FILEKIT_S3_ACCESS_KEY_ID=your-key
FILEKIT_S3_SECRET_ACCESS_KEY=your-secret
FILEKIT_S3_FORCE_PATH_STYLE=false       # Set true for MinIO

# GCS driver
FILEKIT_GCS_BUCKET=my-bucket
FILEKIT_GCS_PREFIX=uploads/
FILEKIT_GCS_PROJECT_ID=my-project
FILEKIT_GCS_CREDENTIALS_FILE=/path/to/credentials.json

# Azure driver
FILEKIT_AZURE_ACCOUNT_NAME=myaccount
FILEKIT_AZURE_ACCOUNT_KEY=...
FILEKIT_AZURE_CONTAINER_NAME=mycontainer
FILEKIT_AZURE_PREFIX=uploads/
FILEKIT_AZURE_ENDPOINT=                 # Optional: custom endpoint

# SFTP driver
FILEKIT_SFTP_HOST=sftp.example.com
FILEKIT_SFTP_PORT=22
FILEKIT_SFTP_USERNAME=user
FILEKIT_SFTP_PASSWORD=secret
FILEKIT_SFTP_PRIVATE_KEY=/path/to/id_rsa
FILEKIT_SFTP_BASE_PATH=/uploads

# Default upload options
FILEKIT_DEFAULT_VISIBILITY=private      # private, public
FILEKIT_DEFAULT_CACHE_CONTROL=max-age=3600
FILEKIT_DEFAULT_OVERWRITE=false
FILEKIT_DEFAULT_PRESERVE_FILENAME=false

# File validation defaults
FILEKIT_MAX_FILE_SIZE=10485760          # 10MB
FILEKIT_ALLOWED_MIME_TYPES=image/jpeg,image/png,application/pdf
FILEKIT_BLOCKED_MIME_TYPES=application/x-executable
FILEKIT_ALLOWED_EXTENSIONS=.jpg,.png,.pdf
FILEKIT_BLOCKED_EXTENSIONS=.exe,.bat

# Encryption
FILEKIT_ENCRYPTION_ENABLED=false
FILEKIT_ENCRYPTION_ALGORITHM=AES-256-GCM
FILEKIT_ENCRYPTION_KEY=base64-encoded-32-byte-key
```

### Programmatic Configuration

```go
cfg := filekit.Config{
    Driver:   "s3",
    S3Region: "us-west-2",
    S3Bucket: "my-bucket",
    S3Prefix: "uploads/",

    // Default options
    DefaultVisibility: "private",
    MaxFileSize:       5 * 1024 * 1024, // 5MB
    AllowedMimeTypes:  "image/jpeg,image/png",
}

fs, err := filekit.New(cfg)
```

### Config Struct

```go
type Config struct {
    Driver string `env:"FILEKIT_DRIVER,default:local"`

    // Local driver
    LocalBasePath string `env:"FILEKIT_LOCAL_BASE_PATH,default:./storage"`

    // S3 driver
    S3Region          string `env:"FILEKIT_S3_REGION,default:us-east-1"`
    S3Bucket          string `env:"FILEKIT_S3_BUCKET"`
    S3Prefix          string `env:"FILEKIT_S3_PREFIX"`
    S3Endpoint        string `env:"FILEKIT_S3_ENDPOINT"`
    S3AccessKeyID     string `env:"FILEKIT_S3_ACCESS_KEY_ID"`
    S3SecretAccessKey string `env:"FILEKIT_S3_SECRET_ACCESS_KEY"`
    S3ForcePathStyle  bool   `env:"FILEKIT_S3_FORCE_PATH_STYLE,default:false"`

    // GCS driver
    GCSBucket          string `env:"FILEKIT_GCS_BUCKET"`
    GCSPrefix          string `env:"FILEKIT_GCS_PREFIX"`
    GCSCredentialsFile string `env:"FILEKIT_GCS_CREDENTIALS_FILE"`
    GCSProjectID       string `env:"FILEKIT_GCS_PROJECT_ID"`

    // Azure driver
    AzureAccountName   string `env:"FILEKIT_AZURE_ACCOUNT_NAME"`
    AzureAccountKey    string `env:"FILEKIT_AZURE_ACCOUNT_KEY"`
    AzureContainerName string `env:"FILEKIT_AZURE_CONTAINER_NAME"`
    AzurePrefix        string `env:"FILEKIT_AZURE_PREFIX"`
    AzureEndpoint      string `env:"FILEKIT_AZURE_ENDPOINT"`

    // SFTP driver
    SFTPHost       string `env:"FILEKIT_SFTP_HOST"`
    SFTPPort       int    `env:"FILEKIT_SFTP_PORT,default:22"`
    SFTPUsername   string `env:"FILEKIT_SFTP_USERNAME"`
    SFTPPassword   string `env:"FILEKIT_SFTP_PASSWORD"`
    SFTPPrivateKey string `env:"FILEKIT_SFTP_PRIVATE_KEY"`
    SFTPBasePath   string `env:"FILEKIT_SFTP_BASE_PATH"`

    // Default options
    DefaultVisibility       string `env:"FILEKIT_DEFAULT_VISIBILITY,default:private"`
    DefaultCacheControl     string `env:"FILEKIT_DEFAULT_CACHE_CONTROL"`
    DefaultOverwrite        bool   `env:"FILEKIT_DEFAULT_OVERWRITE,default:false"`
    DefaultPreserveFilename bool   `env:"FILEKIT_DEFAULT_PRESERVE_FILENAME,default:false"`

    // Validation
    MaxFileSize       int64  `env:"FILEKIT_MAX_FILE_SIZE,default:10485760"`
    AllowedMimeTypes  string `env:"FILEKIT_ALLOWED_MIME_TYPES"`
    BlockedMimeTypes  string `env:"FILEKIT_BLOCKED_MIME_TYPES"`
    AllowedExtensions string `env:"FILEKIT_ALLOWED_EXTENSIONS"`
    BlockedExtensions string `env:"FILEKIT_BLOCKED_EXTENSIONS"`

    // Encryption
    EncryptionEnabled   bool   `env:"FILEKIT_ENCRYPTION_ENABLED,default:false"`
    EncryptionAlgorithm string `env:"FILEKIT_ENCRYPTION_ALGORITHM,default:AES-256-GCM"`
    EncryptionKey       string `env:"FILEKIT_ENCRYPTION_KEY"`
}
```

---

## Write Options

### Available Options

```go
// Content type
filekit.WithContentType("application/pdf")

// Custom metadata
filekit.WithMetadata(map[string]string{
    "author": "john",
    "version": "1.0",
})

// Visibility (affects ACL on cloud storage)
filekit.WithVisibility(filekit.Public)   // public-read
filekit.WithVisibility(filekit.Private)  // private

// Cache control header
filekit.WithCacheControl("max-age=86400")

// Overwrite existing files
filekit.WithOverwrite(true)

// Encryption settings
filekit.WithEncryption("AES-256-GCM", encryptionKey)
filekit.WithEncryptionKeyID("AES-256-GCM", key, "key-v2")

// Expiration time
filekit.WithExpires(time.Now().Add(24 * time.Hour))

// Content disposition
filekit.WithContentDisposition("attachment; filename=\"report.pdf\"")

// Custom ACL (cloud specific)
filekit.WithACL("bucket-owner-full-control")

// Additional headers
filekit.WithHeaders(map[string]string{
    "X-Custom-Header": "value",
})

// File validator
filekit.WithValidator(myValidator)
```

### Write with Progress

```go
file, _ := os.Open("large-file.zip")
defer file.Close()

info, _ := file.Stat()

err := filekit.WriteWithProgress(ctx, fs, "large-file.zip", file, info.Size(), &filekit.WriteOptions{
    ContentType: "application/zip",
    ChunkSize:   5 * 1024 * 1024, // 5MB chunks
    Progress: func(transferred, total int64) {
        percent := float64(transferred) / float64(total) * 100
        fmt.Printf("\rWriting: %.2f%%", percent)
    },
})
```

---

## Error Handling

### Error Types

```go
var (
    ErrNotExist      = errors.New("file does not exist")
    ErrExist         = errors.New("file already exists")
    ErrPermission    = errors.New("permission denied")
    ErrClosed        = errors.New("file already closed")
    ErrNotDir        = errors.New("not a directory")
    ErrIsDir         = errors.New("is a directory")
    ErrNotEmpty      = errors.New("directory not empty")
    ErrInvalidName   = errors.New("invalid name")
    ErrInvalidOffset = errors.New("invalid offset")
    ErrInvalidWhence = errors.New("invalid whence")
    ErrNotSupported  = errors.New("operation not supported")
    ErrNotAllowed    = errors.New("operation not allowed")
    ErrInvalidSize   = errors.New("invalid file size")
    ErrNoSpace       = errors.New("no space left on device")
)
```

### PathError

```go
type PathError struct {
    Op   string // Operation that failed
    Path string // Path involved
    Err  error  // Underlying error
}
```

### Error Checking

```go
_, err := fs.Read(ctx, "nonexistent.txt")
if err != nil {
    var pathErr *filekit.PathError
    if errors.As(err, &pathErr) {
        fmt.Printf("Operation: %s\n", pathErr.Op)
        fmt.Printf("Path: %s\n", pathErr.Path)
        fmt.Printf("Error: %v\n", pathErr.Err)
    }

    // Use helper functions
    if filekit.IsNotExist(err) {
        fmt.Println("File does not exist")
    } else if filekit.IsPermission(err) {
        fmt.Println("Permission denied")
    } else if filekit.IsExist(err) {
        fmt.Println("File already exists")
    }
}
```

---

## Package Structure

```
filekit/
├── fs.go              # Core interfaces (FileReader, FileWriter, FileSystem, CanCopy, CanMove, etc.)
├── config.go          # Configuration struct and loader
├── service.go         # Global instance management
├── mount.go           # MountManager for virtual paths
├── errors.go          # Error types and helpers
├── options.go         # Write options (WithContentType, etc.)
├── readonly.go        # ReadOnlyFileSystem decorator
├── cache.go           # CachingFileSystem decorator & Cache interface
├── selector.go        # FileSelector interface & built-in selectors
├── encryption.go      # EncryptedFS wrapper
├── validated_fs.go    # ValidatedFileSystem wrapper
├── checksum.go        # Checksum utilities
├── changetoken.go     # ChangeToken implementation
└── driver/
    ├── local/         # Local filesystem
    ├── s3/            # Amazon S3 & S3-compatible
    ├── gcs/           # Google Cloud Storage
    ├── azure/         # Azure Blob Storage
    ├── sftp/          # SFTP/SSH
    ├── memory/        # In-memory (testing)
    └── zip/           # ZIP archive
```

---

## Feature Comparison

| Feature | FileKit | Afero | Flysystem (PHP) |
|---------|---------|-------|-----------------|
| Local filesystem | ✅ | ✅ | ✅ |
| Amazon S3 | ✅ | ⚠️ Third-party | ✅ |
| Google Cloud Storage | ✅ | ❌ | ✅ |
| Azure Blob Storage | ✅ | ❌ | ✅ |
| SFTP | ✅ | ❌ | ✅ |
| In-memory | ✅ | ✅ | ✅ |
| ZIP Archive | ✅ | ❌ | ✅ |
| **Mount Manager** | ✅ | ⚠️ | ✅ |
| **Built-in encryption** | ✅ | ❌ | ❌ |
| **File validation** | ✅ | ❌ | ❌ |
| **Checksums** | ✅ | ❌ | ✅ |
| **File watching** | ✅ | ❌ | ❌ |
| **Pre-signed URLs** | ✅ | ❌ | ✅ |
| **Progress tracking** | ✅ | ❌ | ❌ |
| **Chunked uploads** | ✅ | ❌ | ✅ |
| Pure Go (no CGO) | ✅ | ✅ | N/A |

---

## Roadmap

- [x] Core FileSystem interface (FileReader, FileWriter, FileSystem)
- [x] 7 storage drivers
- [x] Mount Manager
- [x] Encryption layer
- [x] File validation
- [x] Optional capability interfaces (CanCopy, CanMove, CanSignURL, CanChecksum, CanWatch)
- [x] Checksums (MD5, SHA1, SHA256, SHA512, CRC32, XXHash)
- [x] File watching (ChangeToken pattern - native for local/memory, polling for cloud)
- [x] ReadOnly decorator with configurable options
- [x] Caching layer with pluggable backends (Memory, Redis, etc.)
- [x] FileSelector interface (VFS-inspired: Glob, Depth, And/Or/Not, FuncSelector escape hatch)
- [ ] Retry/resilience middleware

### Composite/Fallback Filesystem (Under Consideration)

We are evaluating the best approach for combining multiple filesystems. There are several possible patterns:

**1. CompositeFileSystem (Microsoft IFileProvider approach)**
- Combines multiple providers, "first match wins" for reads
- If provider A doesn't have the file, try provider B, then C
- Merges directory listings across all providers
- Simple mental model, works well for read-heavy scenarios
- Microsoft's IFileProvider is read-only by design, so this is their only pattern

**2. Fallback/Mirror Drivers**
- **Fallback**: Primary fails → try secondary (error-based, not existence-based)
- **Mirror**: Write to multiple backends simultaneously for redundancy

**Our Decision: CompositeFileSystem (first-match) only**

We're leaning toward Microsoft's approach for these reasons:

1. **First-match IS fallback for reads** - Checking existence then returning is inherently fallback behavior
2. **Mirror writes are rare in practice** - Most mirroring use cases are better handled by:
   - Infrastructure-level replication (S3 cross-region, Azure geo-redundancy)
   - Async backup jobs (scheduled, not real-time)
   - CachingFileSystem for hot/cold storage patterns
3. **Simplicity** - One pattern, one mental model, fewer edge cases
4. **Write semantics are complex** - Mirror writes raise questions: What if one succeeds and another fails? Rollback? Partial state?

If you have a use case that requires mirror writes, please open an issue to discuss.

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
