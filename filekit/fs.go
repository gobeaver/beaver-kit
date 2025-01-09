package filekit

import (
	"context"
	"io"
	"time"
)

// File represents a file in the filesystem
type File struct {
	Name        string
	Path        string
	Size        int64
	ModTime     time.Time
	IsDir       bool
	ContentType string
	Metadata    map[string]string
}

// FileSystem defines the main interface for file operations
type FileSystem interface {
	// Core operations
	Upload(ctx context.Context, path string, content io.Reader, options ...Option) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error

	// File operations
	Exists(ctx context.Context, path string) (bool, error)
	FileInfo(ctx context.Context, path string) (*File, error)

	// Directory operations
	List(ctx context.Context, prefix string) ([]File, error)
	CreateDir(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string) error
}

// Uploader interface specifically for upload operations
type Uploader interface {
	Upload(ctx context.Context, path string, content io.Reader, options ...Option) error
	UploadFile(ctx context.Context, path string, localPath string, options ...Option) error
}

// Streamer interface for streaming operations
type Streamer interface {
	Stream(ctx context.Context, path string) (io.ReadCloser, error)
	StreamWrite(ctx context.Context, path string, reader io.Reader) error
}
