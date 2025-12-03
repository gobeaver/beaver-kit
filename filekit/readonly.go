package filekit

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrReadOnly is returned when a write operation is attempted on a read-only filesystem.
var ErrReadOnly = errors.New("filesystem is read-only")

// ============================================================================
// ReadOnlyFileSystem Decorator
// ============================================================================

// ReadOnlyFileSystem wraps a FileSystem to prevent all write operations.
// This is useful for:
// - Providing safe read-only access to sensitive data
// - Creating temporary read-only views of filesystems
// - Testing scenarios where writes should be prevented
// - Exposing filesystems to untrusted code
//
// Example:
//
//	fs := local.New("/data")
//	readOnly := filekit.NewReadOnlyFileSystem(fs)
//
//	// Read operations work normally
//	reader, _ := readOnly.Read(ctx, "file.txt")
//
//	// Write operations return ErrReadOnly
//	err := readOnly.Write(ctx, "file.txt", reader)
//	// err wraps ErrReadOnly
type ReadOnlyFileSystem struct {
	fs   FileSystem
	opts ReadOnlyOptions
}

// ReadOnlyOptions configures the ReadOnlyFileSystem behavior.
// This struct is designed for future extensibility.
type ReadOnlyOptions struct {
	// AllowCreateDir permits directory creation even in read-only mode.
	// Useful for temporary directories or staging areas.
	// Default: false
	AllowCreateDir bool

	// AllowDelete permits file deletion in read-only mode.
	// Use with caution - typically you want this false.
	// Default: false
	AllowDelete bool

	// OnWriteAttempt is called when a write operation is attempted.
	// If nil, the default behavior returns ErrReadOnly.
	// This can be used for logging, metrics, or custom error handling.
	// If this function returns nil, the write is allowed (use carefully).
	OnWriteAttempt func(op, path string) error

	// ErrorWrapper allows customizing the error returned for write attempts.
	// If nil, wraps with PathError containing ErrReadOnly.
	// Useful for providing context-specific error messages.
	ErrorWrapper func(op, path string, err error) error
}

// ReadOnlyOption is a functional option for configuring ReadOnlyFileSystem.
type ReadOnlyOption func(*ReadOnlyOptions)

// WithAllowCreateDir allows directory creation in read-only mode.
func WithAllowCreateDir(allow bool) ReadOnlyOption {
	return func(o *ReadOnlyOptions) {
		o.AllowCreateDir = allow
	}
}

// WithAllowDelete allows file deletion in read-only mode.
func WithAllowDelete(allow bool) ReadOnlyOption {
	return func(o *ReadOnlyOptions) {
		o.AllowDelete = allow
	}
}

// WithWriteAttemptHandler sets a custom handler for write attempts.
func WithWriteAttemptHandler(handler func(op, path string) error) ReadOnlyOption {
	return func(o *ReadOnlyOptions) {
		o.OnWriteAttempt = handler
	}
}

// WithErrorWrapper sets a custom error wrapper for write attempts.
func WithErrorWrapper(wrapper func(op, path string, err error) error) ReadOnlyOption {
	return func(o *ReadOnlyOptions) {
		o.ErrorWrapper = wrapper
	}
}

// NewReadOnlyFileSystem creates a read-only wrapper around a FileSystem.
// All write operations (Write, Delete, CreateDir, DeleteDir) will fail
// with ErrReadOnly unless configured otherwise via options.
func NewReadOnlyFileSystem(fs FileSystem, opts ...ReadOnlyOption) *ReadOnlyFileSystem {
	options := ReadOnlyOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &ReadOnlyFileSystem{
		fs:   fs,
		opts: options,
	}
}

// Unwrap returns the underlying FileSystem.
// This allows access to the original filesystem if needed.
func (r *ReadOnlyFileSystem) Unwrap() FileSystem {
	return r.fs
}

// IsReadOnly returns true, indicating this is a read-only filesystem.
func (r *ReadOnlyFileSystem) IsReadOnly() bool {
	return true
}

// readOnlyError creates an appropriate error for write operations.
func (r *ReadOnlyFileSystem) readOnlyError(op, path string) error {
	// Check custom handler first
	if r.opts.OnWriteAttempt != nil {
		if err := r.opts.OnWriteAttempt(op, path); err != nil {
			if r.opts.ErrorWrapper != nil {
				return r.opts.ErrorWrapper(op, path, err)
			}
			return &PathError{Op: op, Path: path, Err: err}
		}
		// Handler returned nil, allow the operation
		return nil
	}

	// Default: return ErrReadOnly
	if r.opts.ErrorWrapper != nil {
		return r.opts.ErrorWrapper(op, path, ErrReadOnly)
	}
	return &PathError{Op: op, Path: path, Err: ErrReadOnly}
}

// ============================================================================
// FileSystem Interface - Read Operations (Delegated)
// ============================================================================

// Read delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return r.fs.Read(ctx, path)
}

// ReadAll delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) ReadAll(ctx context.Context, path string) ([]byte, error) {
	return r.fs.ReadAll(ctx, path)
}

// FileExists delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) FileExists(ctx context.Context, path string) (bool, error) {
	return r.fs.FileExists(ctx, path)
}

// DirExists delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) DirExists(ctx context.Context, path string) (bool, error) {
	return r.fs.DirExists(ctx, path)
}

// Stat delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) Stat(ctx context.Context, path string) (*FileInfo, error) {
	return r.fs.Stat(ctx, path)
}

// ListContents delegates to the underlying filesystem.
func (r *ReadOnlyFileSystem) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	return r.fs.ListContents(ctx, path, recursive)
}

// ============================================================================
// FileSystem Interface - Write Operations (Blocked)
// ============================================================================

// Write returns ErrReadOnly.
func (r *ReadOnlyFileSystem) Write(ctx context.Context, path string, content io.Reader, options ...Option) error {
	if err := r.readOnlyError("write", path); err != nil {
		return err
	}
	// Handler allowed the operation
	return r.fs.Write(ctx, path, content, options...)
}

// Delete returns ErrReadOnly unless AllowDelete is enabled.
func (r *ReadOnlyFileSystem) Delete(ctx context.Context, path string) error {
	if !r.opts.AllowDelete {
		if err := r.readOnlyError("delete", path); err != nil {
			return err
		}
	}
	return r.fs.Delete(ctx, path)
}

// CreateDir returns ErrReadOnly unless AllowCreateDir is enabled.
func (r *ReadOnlyFileSystem) CreateDir(ctx context.Context, path string) error {
	if !r.opts.AllowCreateDir {
		if err := r.readOnlyError("createdir", path); err != nil {
			return err
		}
	}
	return r.fs.CreateDir(ctx, path)
}

// DeleteDir returns ErrReadOnly.
func (r *ReadOnlyFileSystem) DeleteDir(ctx context.Context, path string) error {
	if err := r.readOnlyError("deletedir", path); err != nil {
		return err
	}
	// Handler allowed the operation
	return r.fs.DeleteDir(ctx, path)
}

// ============================================================================
// Optional Interface Delegation
// ============================================================================

// Copy returns ErrReadOnly for write operations.
// If the underlying filesystem supports Copier, delegates for read-only checks.
func (r *ReadOnlyFileSystem) Copy(ctx context.Context, src, dst string) error {
	if err := r.readOnlyError("copy", dst); err != nil {
		return err
	}
	// Handler allowed the operation
	if copier, ok := r.fs.(Copier); ok {
		return copier.Copy(ctx, src, dst)
	}
	return &PathError{Op: "copy", Path: src, Err: ErrNotSupported}
}

// Move returns ErrReadOnly for write operations.
func (r *ReadOnlyFileSystem) Move(ctx context.Context, src, dst string) error {
	if err := r.readOnlyError("move", dst); err != nil {
		return err
	}
	// Handler allowed the operation
	if mover, ok := r.fs.(Mover); ok {
		return mover.Move(ctx, src, dst)
	}
	return &PathError{Op: "move", Path: src, Err: ErrNotSupported}
}

// Move also handles rename operations - returns ErrReadOnly for write operations.
// Note: Rename is now handled by the Move method which covers both operations.

// Checksum delegates to the underlying filesystem if supported.
func (r *ReadOnlyFileSystem) Checksum(ctx context.Context, path string, algorithm ChecksumAlgorithm) (string, error) {
	if checksummer, ok := r.fs.(CanChecksum); ok {
		return checksummer.Checksum(ctx, path, algorithm)
	}
	return "", &PathError{Op: "checksum", Path: path, Err: ErrNotSupported}
}

// Checksums delegates to the underlying filesystem if supported.
func (r *ReadOnlyFileSystem) Checksums(ctx context.Context, path string, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error) {
	if checksummer, ok := r.fs.(CanChecksum); ok {
		return checksummer.Checksums(ctx, path, algorithms)
	}
	return nil, &PathError{Op: "checksums", Path: path, Err: ErrNotSupported}
}

// SignedURL delegates to the underlying filesystem if supported.
func (r *ReadOnlyFileSystem) SignedURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	if urlGen, ok := r.fs.(CanSignURL); ok {
		return urlGen.SignedURL(ctx, path, expires)
	}
	return "", &PathError{Op: "signed-url", Path: path, Err: ErrNotSupported}
}

// SignedUploadURL returns ErrReadOnly (upload URLs enable writes).
func (r *ReadOnlyFileSystem) SignedUploadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	// Upload URLs enable writes, so they're blocked in read-only mode
	if err := r.readOnlyError("signed-upload-url", path); err != nil {
		return "", err
	}
	if urlGen, ok := r.fs.(CanSignURL); ok {
		return urlGen.SignedUploadURL(ctx, path, expires)
	}
	return "", &PathError{Op: "signed-upload-url", Path: path, Err: ErrNotSupported}
}

// GenerateDownloadURL delegates to the underlying filesystem if supported.
// Deprecated: Use SignedURL instead.
func (r *ReadOnlyFileSystem) GenerateDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	if urlGen, ok := r.fs.(URLGenerator); ok {
		return urlGen.GenerateDownloadURL(ctx, path, expires)
	}
	return "", &PathError{Op: "generate-download-url", Path: path, Err: ErrNotSupported}
}

// GenerateUploadURL returns ErrReadOnly (upload URLs enable writes).
// Deprecated: Use SignedUploadURL instead.
func (r *ReadOnlyFileSystem) GenerateUploadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	// Upload URLs enable writes, so they're blocked in read-only mode
	if err := r.readOnlyError("generate-upload-url", path); err != nil {
		return "", err
	}
	if urlGen, ok := r.fs.(URLGenerator); ok {
		return urlGen.GenerateUploadURL(ctx, path, expires)
	}
	return "", &PathError{Op: "generate-upload-url", Path: path, Err: ErrNotSupported}
}

// Watch delegates to the underlying filesystem if supported.
func (r *ReadOnlyFileSystem) Watch(ctx context.Context, filter string) (ChangeToken, error) {
	if watcher, ok := r.fs.(CanWatch); ok {
		return watcher.Watch(ctx, filter)
	}
	return CancelledChangeToken{}, nil
}

// ============================================================================
// Interface Assertions
// ============================================================================

// Ensure ReadOnlyFileSystem implements FileSystem and optional interfaces
var (
	_ FileSystem   = (*ReadOnlyFileSystem)(nil)
	_ FileReader   = (*ReadOnlyFileSystem)(nil)
	_ FileWriter   = (*ReadOnlyFileSystem)(nil)
	_ CanCopy      = (*ReadOnlyFileSystem)(nil)
	_ CanMove      = (*ReadOnlyFileSystem)(nil)
	_ CanChecksum  = (*ReadOnlyFileSystem)(nil)
	_ CanSignURL   = (*ReadOnlyFileSystem)(nil)
	_ CanWatch     = (*ReadOnlyFileSystem)(nil)
	_ URLGenerator = (*ReadOnlyFileSystem)(nil) // Deprecated interface compatibility
)

// ============================================================================
// Helper Functions
// ============================================================================

// IsReadOnlyError checks if an error is due to read-only restrictions.
func IsReadOnlyError(err error) bool {
	return errors.Is(err, ErrReadOnly)
}
