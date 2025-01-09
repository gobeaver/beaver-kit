package filekit

import (
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/gobeaver/beaver-kit/filevalidator"
)

// ValidatedFileSystem wraps a FileSystem with validation support
type ValidatedFileSystem struct {
	fs        FileSystem
	validator filevalidator.Validator
}

// NewValidatedFileSystem creates a new FileSystem with validation
func NewValidatedFileSystem(fs FileSystem, validator filevalidator.Validator) *ValidatedFileSystem {
	return &ValidatedFileSystem{
		fs:        fs,
		validator: validator,
	}
}

// Upload implements FileSystem with validation
func (v *ValidatedFileSystem) Upload(ctx context.Context, path string, content io.Reader, options ...Option) error {
	// Process options
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}

	// If a validator is provided in options, use it; otherwise use the default validator
	validator := v.validator
	if opts.Validator != nil {
		validator = opts.Validator
	}

	// If we have a validator, perform validation
	if validator != nil {
		// We need to buffer the content to validate it
		// This is necessary because we need to read the content for validation
		// but also pass it to the underlying filesystem
		data, err := io.ReadAll(content)
		if err != nil {
			return err
		}

		// Use ValidateReader instead of ValidateWithContext
		if err := validator.ValidateReader(bytes.NewReader(data), filepath.Base(path), int64(len(data))); err != nil {
			return err
		}

		// Create a new reader from the buffered data
		content = bytes.NewReader(data)
	}

	// Pass through to the underlying filesystem
	return v.fs.Upload(ctx, path, content, options...)
}

// Download implements FileSystem
func (v *ValidatedFileSystem) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return v.fs.Download(ctx, path)
}

// Delete implements FileSystem
func (v *ValidatedFileSystem) Delete(ctx context.Context, path string) error {
	return v.fs.Delete(ctx, path)
}

// Exists implements FileSystem
func (v *ValidatedFileSystem) Exists(ctx context.Context, path string) (bool, error) {
	return v.fs.Exists(ctx, path)
}

// FileInfo implements FileSystem
func (v *ValidatedFileSystem) FileInfo(ctx context.Context, path string) (*File, error) {
	return v.fs.FileInfo(ctx, path)
}

// List implements FileSystem
func (v *ValidatedFileSystem) List(ctx context.Context, prefix string) ([]File, error) {
	return v.fs.List(ctx, prefix)
}

// CreateDir implements FileSystem
func (v *ValidatedFileSystem) CreateDir(ctx context.Context, path string) error {
	return v.fs.CreateDir(ctx, path)
}

// DeleteDir implements FileSystem
func (v *ValidatedFileSystem) DeleteDir(ctx context.Context, path string) error {
	return v.fs.DeleteDir(ctx, path)
}
