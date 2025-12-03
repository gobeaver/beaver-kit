package filekit

import (
	"bytes"
	"context"
	"fmt"
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

// Write implements FileSystem with validation
func (v *ValidatedFileSystem) Write(ctx context.Context, path string, content io.Reader, options ...Option) error {
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
		// Optimization: If the reader is an io.Seeker (like os.File), use it directly!
		// filevalidator handles seeking automatically.
		if seeker, ok := content.(io.ReadSeeker); ok {
			// We need to know the size for validation
			size, err := getStreamSize(seeker)
			if err != nil {
				return err
			}

			// Validate using the seeker
			if err := validator.ValidateReader(content, filepath.Base(path), size); err != nil {
				return err
			}

			// Reset seeker to start for write
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				return err
			}
		} else {
			// For non-seekable streams (like HTTP body), we have to be careful.
			// We can't validate the *entire* content (like Zip structure) without buffering it all.
			// But we CAN validate MIME type and size efficiently.

			// 1. Read the header for MIME detection (512 bytes)
			header := make([]byte, 512)
			n, err := io.ReadFull(content, header)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				return err
			}
			header = header[:n]

			// 2. Perform "Best Effort" validation using the header
			if err := validator.ValidateBytes(header, filepath.Base(path)); err != nil {
				return err
			}

			// 3. Reconstruct the reader for Write
			// We stitch the header back with the rest of the stream
			content = io.MultiReader(bytes.NewReader(header), content)

			// 4. Enforce MaxFileSize for streams
			// Since we can't know the size upfront, we wrap the reader to error if it exceeds the limit.
			constraints := validator.GetConstraints()
			if constraints.MaxFileSize > 0 {
				content = &SizeLimitReader{
					R:     content,
					Limit: constraints.MaxFileSize,
				}
			}
		}
	}

	// Pass through to the underlying filesystem
	return v.fs.Write(ctx, path, content, options...)
}

// SizeLimitReader restricts the number of bytes read and returns an error if the limit is exceeded.
type SizeLimitReader struct {
	R     io.Reader
	Limit int64
	N     int64
}

func (l *SizeLimitReader) Read(p []byte) (n int, err error) {
	n, err = l.R.Read(p)
	l.N += int64(n)
	if l.N > l.Limit {
		return n, fmt.Errorf("file size exceeds limit of %d bytes", l.Limit)
	}
	return n, err
}

// getStreamSize tries to get the size of a seekable stream
func getStreamSize(seeker io.ReadSeeker) (int64, error) {
	current, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	end, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = seeker.Seek(current, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return end - current, nil
}

// Read implements FileSystem
func (v *ValidatedFileSystem) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return v.fs.Read(ctx, path)
}

// ReadAll implements FileSystem
func (v *ValidatedFileSystem) ReadAll(ctx context.Context, path string) ([]byte, error) {
	return v.fs.ReadAll(ctx, path)
}

// Delete implements FileSystem
func (v *ValidatedFileSystem) Delete(ctx context.Context, path string) error {
	return v.fs.Delete(ctx, path)
}

// FileExists implements FileSystem
func (v *ValidatedFileSystem) FileExists(ctx context.Context, path string) (bool, error) {
	return v.fs.FileExists(ctx, path)
}

// DirExists implements FileSystem
func (v *ValidatedFileSystem) DirExists(ctx context.Context, path string) (bool, error) {
	return v.fs.DirExists(ctx, path)
}

// Stat implements FileSystem
func (v *ValidatedFileSystem) Stat(ctx context.Context, path string) (*FileInfo, error) {
	return v.fs.Stat(ctx, path)
}

// ListContents implements FileSystem
func (v *ValidatedFileSystem) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	return v.fs.ListContents(ctx, path, recursive)
}

// CreateDir implements FileSystem
func (v *ValidatedFileSystem) CreateDir(ctx context.Context, path string) error {
	return v.fs.CreateDir(ctx, path)
}

// DeleteDir implements FileSystem
func (v *ValidatedFileSystem) DeleteDir(ctx context.Context, path string) error {
	return v.fs.DeleteDir(ctx, path)
}
