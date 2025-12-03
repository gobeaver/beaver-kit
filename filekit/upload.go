package filekit

import (
	"context"
	"io"
)

// ProgressFunc is a callback function for upload progress
type ProgressFunc func(bytesTransferred int64, totalBytes int64)

// UploadOptions contains options for uploading files
type UploadOptions struct {
	// ContentType specifies the MIME type of the file
	ContentType string

	// ChunkSize defines the size of chunks for chunked upload
	// If zero, a default chunk size will be used
	ChunkSize int64

	// Progress is a callback function for upload progress
	Progress ProgressFunc

	// Metadata contains additional metadata for the file
	Metadata map[string]string

	// Visibility defines the file visibility (public or private)
	Visibility Visibility
}

// ChunkedUploader is the interface for filesystems that support chunked uploads
type ChunkedUploader interface {
	// InitiateUpload starts a chunked upload process and returns an upload ID
	InitiateUpload(ctx context.Context, path string) (string, error)

	// UploadPart uploads a part of a file in a chunked upload process
	UploadPart(ctx context.Context, uploadID string, partNumber int, data []byte) error

	// CompleteUpload finalizes a chunked upload
	CompleteUpload(ctx context.Context, uploadID string) error

	// AbortUpload cancels a chunked upload
	AbortUpload(ctx context.Context, uploadID string) error
}

// Upload uploads a file to the filesystem with the given options
func Upload(ctx context.Context, fs FileSystem, path string, r io.Reader, size int64, opts *UploadOptions) error {
	if opts == nil {
		opts = &UploadOptions{}
	}

	// Prepare options for the upload
	options := []Option{}

	if opts.ContentType != "" {
		options = append(options, WithContentType(opts.ContentType))
	}

	if opts.Metadata != nil {
		options = append(options, WithMetadata(opts.Metadata))
	}

	if opts.Visibility != "" {
		options = append(options, WithVisibility(opts.Visibility))
	}

	// If a progress callback is provided, we need to wrap the reader
	if opts.Progress != nil {
		r = &progressReader{
			reader:   r,
			progress: opts.Progress,
			size:     size,
		}
	}

	// Check if the filesystem supports chunked uploads
	if chunkedFS, ok := fs.(ChunkedUploader); ok && size > 0 && opts.ChunkSize > 0 {
		return uploadChunked(ctx, chunkedFS, path, r, size, opts)
	}

	// Use regular write
	return fs.Write(ctx, path, r, options...)
}

// uploadChunked uploads a file using chunked upload
func uploadChunked(ctx context.Context, fs ChunkedUploader, path string, r io.Reader, size int64, opts *UploadOptions) error {
	// Validate size
	if size <= 0 {
		return ErrInvalidSize
	}

	// Initiate upload
	uploadID, err := fs.InitiateUpload(ctx, path)
	if err != nil {
		return err
	}

	// Make sure to abort upload on error
	defer func() {
		if err != nil {
			_ = fs.AbortUpload(ctx, uploadID)
		}
	}()

	// Determine chunk size
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 5 * 1024 * 1024 // Default to 5MB
	}

	// Upload parts
	partNumber := 1
	buffer := make([]byte, chunkSize)
	for {
		// Read chunk
		n, err := io.ReadFull(r, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}

		// If we read something, upload it
		if n > 0 {
			if err := fs.UploadPart(ctx, uploadID, partNumber, buffer[:n]); err != nil {
				return err
			}
			partNumber++
		}

		// If we reached EOF, break
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	// Complete upload
	return fs.CompleteUpload(ctx, uploadID)
}

// progressReader is a reader that reports progress
type progressReader struct {
	reader        io.Reader
	progress      ProgressFunc
	size          int64
	bytesRead     int64
	lastReported  int64
	reportingStep int64
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.bytesRead += int64(n)

		// Report progress if we've read enough since the last report
		// or if we're at the end
		if r.progress != nil &&
			(r.bytesRead-r.lastReported >= r.reportingStep || err == io.EOF) {
			r.progress(r.bytesRead, r.size)
			r.lastReported = r.bytesRead
		}
	}
	return n, err
}
