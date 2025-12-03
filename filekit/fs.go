package filekit

import (
	"context"
	"io"
	"time"
)

// FileInfo represents file/directory metadata
type FileInfo struct {
	Name        string
	Path        string
	Size        int64
	ModTime     time.Time
	IsDir       bool
	ContentType string
	Metadata    map[string]string
}

// File is an alias for FileInfo for backward compatibility
// Deprecated: Use FileInfo instead
type File = FileInfo

// ============================================================================
// Core Interfaces (Interface Segregation)
// ============================================================================

// FileReader provides read-only filesystem access.
// Use this type in function signatures to enforce read-only at compile time.
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

// FileWriter provides write filesystem operations.
type FileWriter interface {
	// Write writes content from reader to path.
	// Use bytes.NewReader(data) for []byte, os.Open() for local files.
	Write(ctx context.Context, path string, r io.Reader, opts ...Option) error

	// Delete removes a file.
	Delete(ctx context.Context, path string) error

	// CreateDir creates a directory (and parents if needed).
	CreateDir(ctx context.Context, path string) error

	// DeleteDir removes a directory and all contents.
	DeleteDir(ctx context.Context, path string) error
}

// FileSystem provides full read-write filesystem access.
type FileSystem interface {
	FileReader
	FileWriter
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================
// These interfaces allow drivers to expose optional capabilities.
// Use type assertion to check if a driver supports a capability:
//
//	if copier, ok := fs.(CanCopy); ok {
//	    copier.Copy(ctx, src, dst)
//	}

// CanCopy indicates the filesystem supports native copy operations.
// Native copy is more efficient than read+write for same-backend operations.
type CanCopy interface {
	Copy(ctx context.Context, src, dst string) error
}

// CanMove indicates the filesystem supports native move/rename operations.
// Native move is more efficient than copy+delete for same-backend operations.
type CanMove interface {
	Move(ctx context.Context, src, dst string) error
}

// Copier is an alias for CanCopy for backward compatibility.
// Deprecated: Use CanCopy instead.
type Copier = CanCopy

// Mover is an alias for CanMove for backward compatibility.
// Deprecated: Use CanMove instead.
type Mover = CanMove

// ============================================================================
// Checksum Interface (FileKit Security Feature)
// ============================================================================

// ChecksumAlgorithm represents a supported checksum algorithm
type ChecksumAlgorithm string

const (
	// ChecksumMD5 is the MD5 hash algorithm (128-bit, fast but not cryptographically secure)
	ChecksumMD5 ChecksumAlgorithm = "md5"
	// ChecksumSHA1 is the SHA-1 hash algorithm (160-bit, legacy)
	ChecksumSHA1 ChecksumAlgorithm = "sha1"
	// ChecksumSHA256 is the SHA-256 hash algorithm (256-bit, recommended)
	ChecksumSHA256 ChecksumAlgorithm = "sha256"
	// ChecksumSHA512 is the SHA-512 hash algorithm (512-bit, most secure)
	ChecksumSHA512 ChecksumAlgorithm = "sha512"
	// ChecksumCRC32 is the CRC32 checksum (32-bit, fastest, for integrity only)
	ChecksumCRC32 ChecksumAlgorithm = "crc32"
	// ChecksumXXHash is the xxHash algorithm (64-bit, extremely fast)
	ChecksumXXHash ChecksumAlgorithm = "xxhash"
)

// CanChecksum indicates the filesystem supports integrity verification.
// This is a FileKit security feature for:
// - File integrity verification
// - Deduplication
// - Change detection
// - Security validation
//
// Example:
//
//	if cs, ok := fs.(CanChecksum); ok {
//	    hash, err := cs.Checksum(ctx, "file.txt", ChecksumSHA256)
//	    fmt.Printf("SHA256: %s\n", hash)
//	}
type CanChecksum interface {
	// Checksum calculates the checksum of a file using the specified algorithm.
	// Returns the checksum as a hex-encoded string.
	Checksum(ctx context.Context, path string, algorithm ChecksumAlgorithm) (string, error)

	// Checksums calculates multiple checksums in a single read pass.
	// Returns a map of algorithm to hex-encoded checksum.
	Checksums(ctx context.Context, path string, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error)
}

// Checksummer is an alias for CanChecksum for backward compatibility.
// Deprecated: Use CanChecksum instead.
type Checksummer = CanChecksum

// ============================================================================
// URL Generation Interfaces
// ============================================================================

// CanSignURL indicates the filesystem can generate pre-signed URLs.
// Useful for S3, GCS, Azure - allows direct client access without proxying.
type CanSignURL interface {
	// SignedURL creates a pre-signed URL for downloading a file.
	SignedURL(ctx context.Context, path string, expires time.Duration) (string, error)

	// SignedUploadURL creates a pre-signed URL for uploading a file.
	SignedUploadURL(ctx context.Context, path string, expires time.Duration) (string, error)
}

// URLGenerator is an alias for CanSignURL for backward compatibility.
// Deprecated: Use CanSignURL instead.
type URLGenerator interface {
	GenerateDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error)
	GenerateUploadURL(ctx context.Context, path string, expires time.Duration) (string, error)
}

// ============================================================================
// File Watching Interface (ChangeToken Pattern)
// ============================================================================
// This follows Microsoft's IChangeToken pattern from ASP.NET Core.
// Benefits:
// - Simple interface (one method)
// - Works for all backends (native events OR polling)
// - Consumer decides how to react (poll HasChanged OR register callback)
// - Composable (combine multiple tokens)
// - Integrates with caching

// ChangeToken represents a change notification token.
// It provides a mechanism to be notified when a change occurs.
//
// Consumers can either:
// 1. Poll HasChanged() periodically
// 2. Register a callback via RegisterChangeCallback()
//
// Check ActiveChangeCallbacks() to know which approach is more efficient
// for the underlying implementation.
type ChangeToken interface {
	// HasChanged returns true if a change has occurred.
	// Once true, it remains true (tokens are single-use).
	HasChanged() bool

	// ActiveChangeCallbacks indicates if the token proactively raises callbacks.
	// If true, RegisterChangeCallback is efficient.
	// If false, consumers should poll HasChanged instead.
	ActiveChangeCallbacks() bool

	// RegisterChangeCallback registers a callback to be invoked when change occurs.
	// Returns a function to unregister the callback.
	// The callback receives no arguments - check the source for what changed.
	RegisterChangeCallback(callback func()) (unregister func())
}

// CanWatch indicates the filesystem supports file change notifications.
// Not all backends support watching - check with type assertion.
//
// Example:
//
//	if watcher, ok := fs.(CanWatch); ok {
//	    token, err := watcher.Watch(ctx, "**/*.json")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Option 1: Poll
//	    if token.HasChanged() {
//	        reloadConfig()
//	    }
//
//	    // Option 2: Callback (if supported)
//	    if token.ActiveChangeCallbacks() {
//	        unregister := token.RegisterChangeCallback(func() {
//	            log.Println("Change detected!")
//	            reloadConfig()
//	        })
//	        defer unregister()
//	    }
//	}
type CanWatch interface {
	// Watch creates a change token for the specified filter pattern.
	// Supports glob patterns: "**/*.txt", "config/*", "*.json", etc.
	// The token signals when any matching file is created, modified, or deleted.
	Watch(ctx context.Context, pattern string) (ChangeToken, error)
}

// Watcher is an alias for CanWatch for backward compatibility.
// Deprecated: Use CanWatch instead.
type Watcher = CanWatch
