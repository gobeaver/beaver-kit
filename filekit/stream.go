package filekit

import (
	"context"
	"io"
)

// Streamer provides streaming read/write operations for filesystems.
type Streamer interface {
	// Stream opens a file for streaming read.
	Stream(ctx context.Context, path string) (io.ReadCloser, error)

	// StreamWrite writes data to a file from a stream.
	StreamWrite(ctx context.Context, path string, reader io.Reader) error
}

// streamManager implements the Streamer interface
type streamManager struct {
	fs FileSystem
}

func NewStreamer(fs FileSystem) Streamer {
	return &streamManager{fs: fs}
}

func (s *streamManager) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.fs.Read(ctx, path)
}

func (s *streamManager) StreamWrite(ctx context.Context, path string, reader io.Reader) error {
	return s.fs.Write(ctx, path, reader)
}
