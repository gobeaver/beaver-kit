package gcs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gobeaver/beaver-kit/filekit"
	"google.golang.org/api/iterator"
)

// Adapter provides a Google Cloud Storage implementation of filekit.FileSystem
type Adapter struct {
	client *storage.Client
	bucket string
	prefix string
}

// AdapterOption is a function that configures GCS Adapter
type AdapterOption func(*Adapter)

// WithPrefix sets the prefix for GCS objects
func WithPrefix(prefix string) AdapterOption {
	return func(a *Adapter) {
		// Ensure prefix ends with a slash if it's not empty
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		a.prefix = prefix
	}
}

// New creates a new GCS filesystem adapter
func New(client *storage.Client, bucket string, options ...AdapterOption) *Adapter {
	adapter := &Adapter{
		client: client,
		bucket: bucket,
	}

	// Apply options
	for _, option := range options {
		option(adapter)
	}

	return adapter
}

// Write implements filekit.FileWriter
func (a *Adapter) Write(ctx context.Context, filePath string, content io.Reader, options ...filekit.Option) error {
	opts := processOptions(options...)

	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Get bucket handle
	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	// Check if file exists and overwrite is not allowed
	if !opts.Overwrite {
		_, err := obj.Attrs(ctx)
		if err == nil {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  filekit.ErrExist,
			}
		}
		if !errors.Is(err, storage.ErrObjectNotExist) {
			return mapGCSError("write", filePath, err)
		}
	}

	// Create a writer
	writer := obj.NewWriter(ctx)

	// Set content type if provided
	if opts.ContentType != "" {
		writer.ContentType = opts.ContentType
	} else {
		// Try to detect content type from extension
		writer.ContentType = detectContentType(filePath)
	}

	// Set cache control if provided
	if opts.CacheControl != "" {
		writer.CacheControl = opts.CacheControl
	}

	// Set metadata if provided
	if len(opts.Metadata) > 0 {
		writer.Metadata = opts.Metadata
	}

	// Set ACL based on visibility
	if opts.Visibility == filekit.Public {
		writer.ACL = []storage.ACLRule{
			{Entity: storage.AllUsers, Role: storage.RoleReader},
		}
	}

	// Copy content to writer
	if _, err := io.Copy(writer, content); err != nil {
		writer.Close()
		return mapGCSError("write", filePath, err)
	}

	// Close the writer to complete the write
	if err := writer.Close(); err != nil {
		return mapGCSError("write", filePath, err)
	}

	return nil
}

// Read implements filekit.FileReader
func (a *Adapter) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	key := path.Join(a.prefix, filePath)

	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, mapGCSError("read", filePath, err)
	}

	return reader, nil
}

// ReadAll implements filekit.FileReader
func (a *Adapter) ReadAll(ctx context.Context, path string) ([]byte, error) {
	rc, err := a.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// Delete implements filekit.FileSystem
func (a *Adapter) Delete(ctx context.Context, filePath string) error {
	key := path.Join(a.prefix, filePath)

	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	if err := obj.Delete(ctx); err != nil {
		return mapGCSError("delete", filePath, err)
	}

	return nil
}

// FileExists checks if a file exists (not a directory)
func (a *Adapter) FileExists(ctx context.Context, filePath string) (bool, error) {
	key := path.Join(a.prefix, filePath)

	// Ensure it's not a directory marker
	if strings.HasSuffix(key, "/") {
		return false, nil
	}

	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, mapGCSError("fileexists", filePath, err)
	}

	// Check if it's a directory marker
	if strings.HasSuffix(attrs.Name, "/") || attrs.ContentType == "application/x-directory" {
		return false, nil
	}

	return true, nil
}

// DirExists checks if a directory prefix exists
func (a *Adapter) DirExists(ctx context.Context, dirPath string) (bool, error) {
	dirKey := path.Join(a.prefix, dirPath)
	if !strings.HasSuffix(dirKey, "/") {
		dirKey += "/"
	}

	bkt := a.client.Bucket(a.bucket)

	// Check if the directory marker exists
	obj := bkt.Object(dirKey)
	_, err := obj.Attrs(ctx)
	if err == nil {
		return true, nil
	}

	// If directory marker doesn't exist, check if any objects with this prefix exist
	if errors.Is(err, storage.ErrObjectNotExist) {
		query := &storage.Query{
			Prefix: dirKey,
		}
		it := bkt.Objects(ctx, query)
		_, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return false, nil
		}
		if err != nil {
			return false, mapGCSError("direxists", dirPath, err)
		}
		return true, nil
	}

	return false, mapGCSError("direxists", dirPath, err)
}

// Stat implements filekit.FileReader
func (a *Adapter) Stat(ctx context.Context, filePath string) (*filekit.FileInfo, error) {
	key := path.Join(a.prefix, filePath)

	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, mapGCSError("stat", filePath, err)
	}

	// Determine if it's a directory
	isDir := strings.HasSuffix(key, "/") || attrs.ContentType == "application/x-directory"

	return &filekit.FileInfo{
		Name:        filepath.Base(filePath),
		Path:        filePath,
		Size:        attrs.Size,
		ModTime:     attrs.Updated,
		IsDir:       isDir,
		ContentType: attrs.ContentType,
		Metadata:    attrs.Metadata,
	}, nil
}

// ListContents lists files and directories at the specified path
func (a *Adapter) ListContents(ctx context.Context, path string, recursive bool) ([]filekit.FileInfo, error) {
	// Prepare prefix for listing
	listPrefix := path
	if a.prefix != "" {
		listPrefix = strings.TrimPrefix(path, "/")
		listPrefix = strings.TrimSuffix(a.prefix, "/") + "/" + listPrefix
	}
	if listPrefix != "" && !strings.HasSuffix(listPrefix, "/") {
		listPrefix += "/"
	}

	bkt := a.client.Bucket(a.bucket)

	// Create query with or without delimiter based on recursive flag
	query := &storage.Query{
		Prefix: listPrefix,
	}
	if !recursive {
		query.Delimiter = "/"
	}

	var files []filekit.FileInfo
	it := bkt.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, mapGCSError("listcontents", path, err)
		}

		// Handle "directory" prefixes (only when not recursive)
		if attrs.Prefix != "" {
			dirName := strings.TrimPrefix(attrs.Prefix, listPrefix)
			dirName = strings.TrimSuffix(dirName, "/")
			if dirName == "" {
				continue
			}

			files = append(files, filekit.FileInfo{
				Name:  filepath.Base(dirName),
				Path:  strings.TrimPrefix(attrs.Prefix, a.prefix),
				IsDir: true,
			})
			continue
		}

		// Skip the directory itself
		if attrs.Name == listPrefix {
			continue
		}

		// Get the file name relative to the prefix
		relPath := strings.TrimPrefix(attrs.Name, listPrefix)
		if relPath == "" {
			continue
		}

		// For non-recursive, skip items with slashes (deeper nested items)
		if !recursive && strings.Contains(relPath, "/") {
			continue
		}

		isDir := strings.HasSuffix(attrs.Name, "/") || attrs.ContentType == "application/x-directory"

		files = append(files, filekit.FileInfo{
			Name:        filepath.Base(strings.TrimSuffix(attrs.Name, "/")),
			Path:        strings.TrimPrefix(attrs.Name, a.prefix),
			Size:        attrs.Size,
			ModTime:     attrs.Updated,
			IsDir:       isDir,
			ContentType: attrs.ContentType,
			Metadata:    attrs.Metadata,
		})
	}

	return files, nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, dirPath string) error {
	// GCS doesn't have real directories, but we can create an empty object with a trailing slash
	key := path.Join(a.prefix, dirPath)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	bkt := a.client.Bucket(a.bucket)
	obj := bkt.Object(key)

	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/x-directory"

	if err := writer.Close(); err != nil {
		return mapGCSError("createdir", dirPath, err)
	}

	return nil
}

// DeleteDir implements filekit.FileSystem
func (a *Adapter) DeleteDir(ctx context.Context, dirPath string) error {
	// Prepare directory path
	dirKey := path.Join(a.prefix, dirPath)
	if !strings.HasSuffix(dirKey, "/") {
		dirKey += "/"
	}

	bkt := a.client.Bucket(a.bucket)

	// List all objects with the prefix
	query := &storage.Query{Prefix: dirKey}
	it := bkt.Objects(ctx, query)

	var found bool
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return mapGCSError("deletedir", dirPath, err)
		}

		found = true

		// Delete the object
		if err := bkt.Object(attrs.Name).Delete(ctx); err != nil {
			return mapGCSError("deletedir", dirPath, err)
		}
	}

	if !found {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotExist,
		}
	}

	return nil
}

// WriteFile writes a local file to the filesystem
func (a *Adapter) WriteFile(ctx context.Context, destPath string, localPath string, options ...filekit.Option) error {
	file, err := os.Open(localPath)
	if err != nil {
		return &filekit.PathError{
			Op:   "writefile",
			Path: localPath,
			Err:  err,
		}
	}
	defer file.Close()

	// Detect content type if not provided
	opts := processOptions(options...)
	if opts.ContentType == "" {
		contentType := detectContentType(localPath)
		options = append(options, filekit.WithContentType(contentType))
	}

	return a.Write(ctx, destPath, file, options...)
}

// SignedURL generates a signed URL for downloading a file
func (a *Adapter) SignedURL(ctx context.Context, filePath string, expiry time.Duration) (string, error) {
	return a.GenerateSignedGetURL(ctx, filePath, expiry)
}

// SignedUploadURL generates a signed URL for uploading a file
func (a *Adapter) SignedUploadURL(ctx context.Context, filePath string, expiry time.Duration) (string, error) {
	return a.GenerateSignedPutURL(ctx, filePath, expiry, "application/octet-stream")
}

// GenerateSignedGetURL generates a signed URL for downloading a file
func (a *Adapter) GenerateSignedGetURL(ctx context.Context, filePath string, expiry time.Duration) (string, error) {
	key := path.Join(a.prefix, filePath)

	opts := &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	}

	url, err := a.client.Bucket(a.bucket).SignedURL(key, opts)
	if err != nil {
		return "", mapGCSError("signed-get-url", filePath, err)
	}

	return url, nil
}

// GenerateSignedPutURL generates a signed URL for uploading a file
func (a *Adapter) GenerateSignedPutURL(ctx context.Context, filePath string, expiry time.Duration, contentType string) (string, error) {
	key := path.Join(a.prefix, filePath)

	opts := &storage.SignedURLOptions{
		Method:      "PUT",
		Expires:     time.Now().Add(expiry),
		ContentType: contentType,
	}

	url, err := a.client.Bucket(a.bucket).SignedURL(key, opts)
	if err != nil {
		return "", mapGCSError("signed-put-url", filePath, err)
	}

	return url, nil
}

// processOptions processes the provided options
func processOptions(options ...filekit.Option) *filekit.Options {
	opts := &filekit.Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

// detectContentType determines the content type from file extension
func detectContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext != "" {
		// Read a bit of content to detect type
		contentType := http.DetectContentType([]byte{})
		if contentType != "application/octet-stream" {
			return contentType
		}
	}

	// Common extension mappings
	switch strings.ToLower(ext) {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz", ".gzip":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// mapGCSError maps GCS errors to filekit errors
func mapGCSError(op, path string, err error) error {
	if errors.Is(err, storage.ErrObjectNotExist) {
		return &filekit.PathError{
			Op:   op,
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	if errors.Is(err, storage.ErrBucketNotExist) {
		return &filekit.PathError{
			Op:   op,
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	return &filekit.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

func init() {
	filekit.RegisterDriver("gcs", func(cfg filekit.Config) (filekit.FileSystem, error) {
		ctx := context.Background()

		// Create client - uses GOOGLE_APPLICATION_CREDENTIALS env var or default credentials
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, err
		}

		var options []AdapterOption
		if cfg.GCSPrefix != "" {
			options = append(options, WithPrefix(cfg.GCSPrefix))
		}

		return New(client, cfg.GCSBucket, options...), nil
	})
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// Copy implements filekit.Copier using GCS's native CopierFrom.
func (a *Adapter) Copy(ctx context.Context, src, dst string) error {
	srcKey := path.Join(a.prefix, src)
	dstKey := path.Join(a.prefix, dst)

	srcObj := a.client.Bucket(a.bucket).Object(srcKey)
	dstObj := a.client.Bucket(a.bucket).Object(dstKey)

	// Use GCS native copy
	_, err := dstObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		return mapGCSError("copy", src, err)
	}

	return nil
}

// Move implements filekit.Mover using GCS's copy + delete.
func (a *Adapter) Move(ctx context.Context, src, dst string) error {
	// Copy the object
	if err := a.Copy(ctx, src, dst); err != nil {
		return err
	}

	// Delete the source
	srcKey := path.Join(a.prefix, src)
	if err := a.client.Bucket(a.bucket).Object(srcKey).Delete(ctx); err != nil {
		return mapGCSError("move", src, err)
	}

	return nil
}

// GenerateDownloadURL implements filekit.URLGenerator.
func (a *Adapter) GenerateDownloadURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GenerateSignedGetURL(ctx, filePath, expires)
}

// GenerateUploadURL implements filekit.URLGenerator.
func (a *Adapter) GenerateUploadURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GenerateSignedPutURL(ctx, filePath, expires, "application/octet-stream")
}

// Checksum implements filekit.Checksummer by reading and hashing the file.
func (a *Adapter) Checksum(ctx context.Context, filePath string, algorithm filekit.ChecksumAlgorithm) (string, error) {
	reader, err := a.Read(ctx, filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	checksum, err := filekit.CalculateChecksum(reader, algorithm)
	if err != nil {
		return "", &filekit.PathError{Op: "checksum", Path: filePath, Err: err}
	}

	return checksum, nil
}

// Checksums implements filekit.MultiChecksummer for efficient multi-hash calculation.
func (a *Adapter) Checksums(ctx context.Context, filePath string, algorithms []filekit.ChecksumAlgorithm) (map[filekit.ChecksumAlgorithm]string, error) {
	reader, err := a.Read(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	checksums, err := filekit.CalculateChecksums(reader, algorithms)
	if err != nil {
		return nil, &filekit.PathError{Op: "checksums", Path: filePath, Err: err}
	}

	return checksums, nil
}

// ============================================================================
// Watcher Implementation (Polling-based)
// ============================================================================

// Watch implements filekit.Watcher using a polling approach.
// GCS doesn't have native file system events, so we poll for changes.
// The filter pattern supports glob patterns like "**/*.json", "config/*".
// Default polling interval is 30 seconds.
func (a *Adapter) Watch(ctx context.Context, filter string) (filekit.ChangeToken, error) {
	// Get initial state of matching files
	initialState, err := a.getMatchingFilesState(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Create a polling change token that checks for changes
	token := filekit.NewPollingChangeToken(ctx, filekit.PollingConfig{
		Interval: 30 * time.Second,
		CheckFunc: func() bool {
			currentState, err := a.getMatchingFilesState(ctx, filter)
			if err != nil {
				return false // Can't determine change, don't signal
			}
			return !gcsStatesEqual(initialState, currentState)
		},
	})

	return token, nil
}

// gcsFileState represents the state of a file for change detection
type gcsFileState struct {
	path    string
	modTime time.Time
	size    int64
}

// getMatchingFilesState returns the current state of files matching the filter
func (a *Adapter) getMatchingFilesState(ctx context.Context, filter string) (map[string]gcsFileState, error) {
	state := make(map[string]gcsFileState)

	// List all objects with the prefix
	it := a.client.Bucket(a.bucket).Objects(ctx, &storage.Query{
		Prefix: a.prefix,
	})

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, &filekit.PathError{Op: "watch", Path: filter, Err: err}
		}

		// Remove prefix from key to get relative path
		relPath := strings.TrimPrefix(attrs.Name, a.prefix)

		// Check if path matches filter
		if gcsMatchesGlobFilter(relPath, filter) {
			state[relPath] = gcsFileState{
				path:    relPath,
				modTime: attrs.Updated,
				size:    attrs.Size,
			}
		}
	}

	return state, nil
}

// gcsStatesEqual checks if two file states are equal
func gcsStatesEqual(a, b map[string]gcsFileState) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false // File was deleted
		}
		if v.modTime != bv.modTime || v.size != bv.size {
			return false // File was modified
		}
	}
	return true
}

// gcsMatchesGlobFilter checks if a path matches a glob pattern
func gcsMatchesGlobFilter(filePath, filter string) bool {
	// Handle ** patterns for recursive matching
	if strings.Contains(filter, "**") {
		parts := strings.Split(filter, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			hasPrefix := prefix == "" || strings.HasPrefix(filePath, prefix)
			hasSuffix := suffix == "" || strings.HasSuffix(filePath, suffix)

			return hasPrefix && hasSuffix
		}
	}

	// Simple glob matching
	matched, _ := path.Match(filter, filePath)
	return matched
}

// Ensure Adapter implements required and optional interfaces
var (
	_ filekit.FileSystem  = (*Adapter)(nil)
	_ filekit.FileReader  = (*Adapter)(nil)
	_ filekit.FileWriter  = (*Adapter)(nil)
	_ filekit.CanCopy     = (*Adapter)(nil)
	_ filekit.CanMove     = (*Adapter)(nil)
	_ filekit.CanSignURL  = (*Adapter)(nil)
	_ filekit.CanChecksum = (*Adapter)(nil)
	_ filekit.CanWatch    = (*Adapter)(nil)
)
