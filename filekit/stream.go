package filekit

import (
	"context"
	"io"
)

// streamManager implements the Streamer interface
type streamManager struct {
	fs FileSystem
}

func NewStreamer(fs FileSystem) Streamer {
	return &streamManager{fs: fs}
}

func (s *streamManager) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.fs.Download(ctx, path)
}

func (s *streamManager) StreamWrite(ctx context.Context, path string, reader io.Reader) error {
	return s.fs.Upload(ctx, path, reader)
}
