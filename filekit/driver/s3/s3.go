package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gobeaver/beaver-kit/filekit"
)

// Adapter provides an S3 implementation of filekit.FileSystem
type Adapter struct {
	client *s3.Client
	bucket string
	prefix string
}

// AdapterOption is a function that configures S3Adapter
type AdapterOption func(*Adapter)

// WithPrefix sets the prefix for S3 objects
func WithPrefix(prefix string) AdapterOption {
	return func(a *Adapter) {
		// Ensure prefix ends with a slash if it's not empty
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		a.prefix = prefix
	}
}

// New creates a new S3 filesystem adapter
func New(client *s3.Client, bucket string, options ...AdapterOption) *Adapter {
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
	// Process options
	opts := processOptions(options...)

	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Prepare upload input
	input := &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
		Body:   content,
	}

	// Set content type if provided
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	// Set cache control if provided
	if opts.CacheControl != "" {
		input.CacheControl = aws.String(opts.CacheControl)
	}

	// Set metadata if provided
	if len(opts.Metadata) > 0 {
		metadata := make(map[string]string, len(opts.Metadata))
		for k, v := range opts.Metadata {
			metadata[k] = v
		}
		input.Metadata = metadata
	}

	// Set ACL based on visibility
	if opts.Visibility == filekit.Public {
		input.ACL = types.ObjectCannedACLPublicRead
	} else if opts.Visibility == filekit.Private {
		input.ACL = types.ObjectCannedACLPrivate
	}

	// Upload the object
	_, err := a.client.PutObject(ctx, input)
	if err != nil {
		return mapS3Error("write", filePath, err)
	}

	return nil
}

// Read implements filekit.FileReader
func (a *Adapter) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Get object
	resp, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapS3Error("read", filePath, err)
	}

	return resp.Body, nil
}

// ReadAll implements filekit.FileReader
func (a *Adapter) ReadAll(ctx context.Context, filePath string) ([]byte, error) {
	rc, err := a.Read(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// Delete implements filekit.FileSystem
func (a *Adapter) Delete(ctx context.Context, filePath string) error {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Delete the object
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return mapS3Error("delete", filePath, err)
	}

	// Wait for the object to be deleted
	waiter := s3.NewObjectNotExistsWaiter(a.client)
	err = waiter.Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}, 30*time.Second)
	if err != nil {
		return mapS3Error("delete", filePath, err)
	}

	return nil
}

// FileExists implements filekit.FileReader
func (a *Adapter) FileExists(ctx context.Context, filePath string) (bool, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Check if the object exists
	_, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		var notFound *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &notFound) {
			return false, nil
		}
		return false, mapS3Error("fileexists", filePath, err)
	}

	// Check if it's not a directory (doesn't end with /)
	return !strings.HasSuffix(key, "/"), nil
}

// DirExists implements filekit.FileReader
func (a *Adapter) DirExists(ctx context.Context, dirPath string) (bool, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, dirPath)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	// Check if the directory marker exists or if there are any objects with this prefix
	resp, err := a.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(a.bucket),
		Prefix:  aws.String(key),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, mapS3Error("direxists", dirPath, err)
	}

	return len(resp.Contents) > 0 || len(resp.CommonPrefixes) > 0, nil
}

// Stat implements filekit.FileReader
func (a *Adapter) Stat(ctx context.Context, filePath string) (*filekit.FileInfo, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Get object metadata
	resp, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapS3Error("stat", filePath, err)
	}

	// Extract metadata
	metadata := make(map[string]string)
	for k, v := range resp.Metadata {
		metadata[k] = v
	}

	// Determine if it's a directory
	isDir := strings.HasSuffix(key, "/")

	return &filekit.FileInfo{
		Name:        filepath.Base(filePath),
		Path:        filePath,
		Size:        *resp.ContentLength,
		ModTime:     aws.ToTime(resp.LastModified),
		IsDir:       isDir,
		ContentType: aws.ToString(resp.ContentType),
		Metadata:    metadata,
	}, nil
}

// ListContents implements filekit.FileReader
func (a *Adapter) ListContents(ctx context.Context, prefix string, recursive bool) ([]filekit.FileInfo, error) {
	// Prepare prefix for listing
	listPrefix := path.Join(a.prefix, prefix)
	if listPrefix != "" && !strings.HasSuffix(listPrefix, "/") {
		listPrefix += "/"
	}

	var files []filekit.FileInfo

	if recursive {
		// List all objects recursively (no delimiter)
		paginator := s3.NewListObjectsV2Paginator(a.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(a.bucket),
			Prefix: aws.String(listPrefix),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, mapS3Error("listcontents", prefix, err)
			}

			for _, obj := range page.Contents {
				// Skip the directory itself
				if aws.ToString(obj.Key) == listPrefix {
					continue
				}

				relPath := strings.TrimPrefix(aws.ToString(obj.Key), a.prefix)
				if strings.HasPrefix(relPath, "/") {
					relPath = relPath[1:]
				}

				isDir := strings.HasSuffix(aws.ToString(obj.Key), "/")

				files = append(files, filekit.FileInfo{
					Name:    filepath.Base(relPath),
					Path:    relPath,
					Size:    aws.ToInt64(obj.Size),
					ModTime: aws.ToTime(obj.LastModified),
					IsDir:   isDir,
				})
			}
		}
	} else {
		// List objects with delimiter for immediate children only
		resp, err := a.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(a.bucket),
			Prefix:    aws.String(listPrefix),
			Delimiter: aws.String("/"),
		})
		if err != nil {
			return nil, mapS3Error("listcontents", prefix, err)
		}

		// Add directories (common prefixes)
		for _, p := range resp.CommonPrefixes {
			dirName := strings.TrimPrefix(aws.ToString(p.Prefix), listPrefix)
			dirName = strings.TrimSuffix(dirName, "/")
			if dirName == "" {
				continue
			}

			files = append(files, filekit.FileInfo{
				Name:  dirName,
				Path:  path.Join(prefix, dirName),
				IsDir: true,
			})
		}

		// Add files
		for _, obj := range resp.Contents {
			// Skip the directory itself
			if aws.ToString(obj.Key) == listPrefix {
				continue
			}

			fileName := strings.TrimPrefix(aws.ToString(obj.Key), listPrefix)
			if fileName == "" || strings.Contains(fileName, "/") {
				continue
			}

			files = append(files, filekit.FileInfo{
				Name:    fileName,
				Path:    path.Join(prefix, fileName),
				Size:    aws.ToInt64(obj.Size),
				ModTime: aws.ToTime(obj.LastModified),
				IsDir:   false,
			})
		}
	}

	return files, nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, dirPath string) error {
	// S3 doesn't have real directories, but we can create an empty object with a trailing slash
	key := path.Join(a.prefix, dirPath)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	// Create an empty object with a trailing slash
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte{}),
		ContentType: aws.String("application/x-directory"),
	})
	if err != nil {
		return mapS3Error("createdir", dirPath, err)
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

	// List all objects with the prefix
	resp, err := a.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
		Prefix: aws.String(dirKey),
	})
	if err != nil {
		return mapS3Error("deletedir", dirPath, err)
	}

	// If no objects found, the directory doesn't exist
	if len(resp.Contents) == 0 {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotExist,
		}
	}

	// Delete all objects with the prefix
	objectsToDelete := make([]types.ObjectIdentifier, len(resp.Contents))
	for i, obj := range resp.Contents {
		objectsToDelete[i] = types.ObjectIdentifier{
			Key: obj.Key,
		}
	}

	// Delete the objects
	_, err = a.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(a.bucket),
		Delete: &types.Delete{
			Objects: objectsToDelete,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		return mapS3Error("deletedir", dirPath, err)
	}

	return nil
}

// WriteFile writes a local file to S3
func (a *Adapter) WriteFile(ctx context.Context, destPath string, localPath string, options ...filekit.Option) error {
	// Determine content type from file extension
	contentType := ""
	ext := filepath.Ext(localPath)
	if ext != "" {
		contentType = http.DetectContentType([]byte(ext))
	}

	// Add content type option if it's not already specified
	hasContentType := false
	for _, option := range options {
		_ = option
		// This is a simplistic check that assumes that if there are any options,
		// content type might be one of them. In a real implementation,
		// you would need to check if content type is actually set.
		hasContentType = true
		break
	}

	if !hasContentType && contentType != "" {
		options = append(options, filekit.WithContentType(contentType))
	}

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return &filekit.PathError{
			Op:   "writefile",
			Path: localPath,
			Err:  err,
		}
	}
	defer file.Close()

	// Write the file
	return a.Write(ctx, destPath, file, options...)
}

// InitiateUpload implements filekit.ChunkedUploader
func (a *Adapter) InitiateUpload(ctx context.Context, filePath string) (string, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Initiate multipart upload
	resp, err := a.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", mapS3Error("initiate-upload", filePath, err)
	}

	return aws.ToString(resp.UploadId), nil
}

// UploadPart implements filekit.ChunkedUploader
func (a *Adapter) UploadPart(ctx context.Context, uploadID string, partNumber int, data []byte) error {
	// For simplicity, we store info about the upload in memory
	// In a real implementation, you would likely use a database or cache

	// TODO: Implement a proper way to store upload metadata
	key := "demo" // This should be retrieved from uploadID

	// Upload the part

	// Validate partNumber is within int32 range (AWS S3 supports 1-10000 parts)
	if partNumber < 1 || partNumber > 10000 {
		return fmt.Errorf("part number must be between 1 and 10000, got %d", partNumber)
	}
	_, err := a.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(a.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(int32(partNumber)), //nolint:gosec // validated above
		Body:       bytes.NewReader(data),
	})
	if err != nil {
		return mapS3Error("upload-part", key, err)
	}

	return nil
}

// CompleteUpload implements filekit.ChunkedUploader
func (a *Adapter) CompleteUpload(ctx context.Context, uploadID string) error {
	// This is a simplified implementation
	// In a real implementation, you would retrieve the part info from a database

	// TODO: Implement a proper way to retrieve parts
	key := "demo"                    // This should be retrieved from uploadID
	parts := []types.CompletedPart{} // These should be retrieved from a store

	// Complete the multipart upload
	_, err := a.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(a.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return mapS3Error("complete-upload", key, err)
	}

	return nil
}

// AbortUpload implements filekit.ChunkedUploader
func (a *Adapter) AbortUpload(ctx context.Context, uploadID string) error {
	// This is a simplified implementation
	// In a real implementation, you would retrieve the key from a database

	// TODO: Implement a proper way to retrieve the key
	key := "demo" // This should be retrieved from uploadID

	// Abort the multipart upload
	_, err := a.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(a.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return mapS3Error("abort-upload", key, err)
	}

	return nil
}

// processOptions processes the provided options
func processOptions(options ...filekit.Option) *filekit.Options {
	opts := &filekit.Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

func (a *Adapter) GeneratePresignedGetURL(ctx context.Context, filePath string, expiry time.Duration) (string, error) {
	key := path.Join(a.prefix, filePath)

	presignClient := s3.NewPresignClient(a.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", mapS3Error("presign-get", filePath, err)
	}

	return request.URL, nil
}

func (a *Adapter) GeneratePresignedPutURL(ctx context.Context, filePath string, expiry time.Duration, options ...filekit.Option) (string, error) {
	key := path.Join(a.prefix, filePath)
	opts := processOptions(options...)

	presignClient := s3.NewPresignClient(a.client)
	input := &s3.PutObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	}

	// Set content type if provided
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	request, err := presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", mapS3Error("presign-put", filePath, err)
	}

	return request.URL, nil
}

// mapS3Error maps S3 errors to filekit errors
func mapS3Error(op, filePath string, err error) error {
	var nsk *types.NoSuchKey
	var notFound *types.NotFound

	if errors.As(err, &nsk) || errors.As(err, &notFound) {
		return &filekit.PathError{
			Op:   op,
			Path: filePath,
			Err:  filekit.ErrNotExist,
		}
	}

	// Map other specific errors here

	return &filekit.PathError{
		Op:   op,
		Path: filePath,
		Err:  err,
	}
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// Copy implements filekit.Copier using S3's native CopyObject API.
// This is more efficient than download+upload for same-bucket copies.
func (a *Adapter) Copy(ctx context.Context, src, dst string) error {
	srcKey := path.Join(a.prefix, src)
	dstKey := path.Join(a.prefix, dst)

	// S3 CopyObject requires source in "bucket/key" format
	copySource := fmt.Sprintf("%s/%s", a.bucket, srcKey)

	_, err := a.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(a.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dstKey),
	})
	if err != nil {
		return mapS3Error("copy", src, err)
	}

	return nil
}

// Move implements filekit.Mover using S3's CopyObject + DeleteObject.
// S3 doesn't have a native move/rename, so this is copy+delete.
func (a *Adapter) Move(ctx context.Context, src, dst string) error {
	// Copy the object
	if err := a.Copy(ctx, src, dst); err != nil {
		return err
	}

	// Delete the source
	srcKey := path.Join(a.prefix, src)
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(srcKey),
	})
	if err != nil {
		return mapS3Error("move", src, err)
	}

	return nil
}

// SignedURL implements filekit.CanSignURL.
func (a *Adapter) SignedURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GeneratePresignedGetURL(ctx, filePath, expires)
}

// SignedUploadURL implements filekit.CanSignURL.
func (a *Adapter) SignedUploadURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GeneratePresignedPutURL(ctx, filePath, expires)
}

// GenerateDownloadURL implements filekit.URLGenerator (deprecated, use SignedURL).
func (a *Adapter) GenerateDownloadURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GeneratePresignedGetURL(ctx, filePath, expires)
}

// GenerateUploadURL implements filekit.URLGenerator.
func (a *Adapter) GenerateUploadURL(ctx context.Context, filePath string, expires time.Duration) (string, error) {
	return a.GeneratePresignedPutURL(ctx, filePath, expires)
}

// Checksum implements filekit.CanChecksum by reading and hashing the file.
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

// Checksums implements filekit.CanChecksum for efficient multi-hash calculation.
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
// S3 doesn't have native file system events, so we poll for changes.
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
			return !statesEqual(initialState, currentState)
		},
	})

	return token, nil
}

// fileState represents the state of a file for change detection
type fileState struct {
	path    string
	modTime time.Time
	size    int64
}

// getMatchingFilesState returns the current state of files matching the filter
func (a *Adapter) getMatchingFilesState(ctx context.Context, filter string) (map[string]fileState, error) {
	state := make(map[string]fileState)

	// List all objects with the prefix
	paginator := s3.NewListObjectsV2Paginator(a.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
		Prefix: aws.String(a.prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &filekit.PathError{Op: "watch", Path: filter, Err: err}
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}

			// Remove prefix from key to get relative path
			relPath := strings.TrimPrefix(*obj.Key, a.prefix)

			// Check if path matches filter
			if matchesGlobFilter(relPath, filter) {
				var modTime time.Time
				if obj.LastModified != nil {
					modTime = *obj.LastModified
				}
				var size int64
				if obj.Size != nil {
					size = *obj.Size
				}
				state[relPath] = fileState{
					path:    relPath,
					modTime: modTime,
					size:    size,
				}
			}
		}
	}

	return state, nil
}

// statesEqual checks if two file states are equal
func statesEqual(a, b map[string]fileState) bool {
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

// matchesGlobFilter checks if a path matches a glob pattern
func matchesGlobFilter(filePath, filter string) bool {
	// Handle ** patterns for recursive matching
	if strings.Contains(filter, "**") {
		// Convert ** to .* for regex-like matching
		pattern := strings.ReplaceAll(filter, "**", ".*")
		pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
		pattern = strings.ReplaceAll(pattern, "?", ".")
		pattern = "^" + pattern + "$"

		matched, _ := path.Match(filter, filePath)
		if matched {
			return true
		}

		// Try matching with path.Match for simpler patterns
		// or use a more sophisticated glob library
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

// Ensure Adapter implements interfaces
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
