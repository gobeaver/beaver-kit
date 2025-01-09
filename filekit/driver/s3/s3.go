package s3

import (
	"bytes"
	"context"
	"errors"
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

// Upload implements filekit.FileSystem
func (a *Adapter) Upload(ctx context.Context, filePath string, content io.Reader, options ...filekit.Option) error {
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
	if opts.Metadata != nil && len(opts.Metadata) > 0 {
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
		return mapS3Error("upload", filePath, err)
	}

	return nil
}

// Download implements filekit.FileSystem
func (a *Adapter) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Get object
	resp, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapS3Error("download", filePath, err)
	}

	return resp.Body, nil
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

// Exists implements filekit.FileSystem
func (a *Adapter) Exists(ctx context.Context, filePath string) (bool, error) {
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
		return false, mapS3Error("exists", filePath, err)
	}

	return true, nil
}

// FileInfo implements filekit.FileSystem
func (a *Adapter) FileInfo(ctx context.Context, filePath string) (*filekit.File, error) {
	// Combine prefix and path
	key := path.Join(a.prefix, filePath)

	// Get object metadata
	resp, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapS3Error("fileinfo", filePath, err)
	}

	// Extract metadata
	metadata := make(map[string]string)
	for k, v := range resp.Metadata {
		metadata[k] = v
	}

	// Determine if it's a directory
	isDir := strings.HasSuffix(key, "/")

	return &filekit.File{
		Name:        filepath.Base(filePath),
		Path:        filePath,
		Size:        *resp.ContentLength,
		ModTime:     aws.ToTime(resp.LastModified),
		IsDir:       isDir,
		ContentType: aws.ToString(resp.ContentType),
		Metadata:    metadata,
	}, nil
}

// List implements filekit.FileSystem
func (a *Adapter) List(ctx context.Context, prefix string) ([]filekit.File, error) {
	// Prepare prefix for listing
	listPrefix := path.Join(a.prefix, prefix)
	if !strings.HasSuffix(listPrefix, "/") {
		listPrefix += "/"
	}

	// List objects with the prefix
	resp, err := a.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(a.bucket),
		Prefix:    aws.String(listPrefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, mapS3Error("list", prefix, err)
	}

	// Prepare result
	files := make([]filekit.File, 0, len(resp.CommonPrefixes)+len(resp.Contents))

	// Add directories (common prefixes)
	for _, p := range resp.CommonPrefixes {
		dirName := strings.TrimPrefix(aws.ToString(p.Prefix), listPrefix)
		dirName = strings.TrimSuffix(dirName, "/")
		if dirName == "" {
			continue
		}

		files = append(files, filekit.File{
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

		files = append(files, filekit.File{
			Name:    fileName,
			Path:    path.Join(prefix, fileName),
			Size:    *obj.Size,
			ModTime: aws.ToTime(obj.LastModified),
			IsDir:   false,
		})
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

// UploadFile implements filekit.Uploader
func (a *Adapter) UploadFile(ctx context.Context, path string, localPath string, options ...filekit.Option) error {
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
			Op:   "uploadfile",
			Path: localPath,
			Err:  err,
		}
	}
	defer file.Close()

	// Upload the file
	return a.Upload(ctx, path, file, options...)
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

	_, err := a.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(a.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(int32(partNumber)),
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
func mapS3Error(op, path string, err error) error {
	var nsk *types.NoSuchKey
	var notFound *types.NotFound

	if errors.As(err, &nsk) || errors.As(err, &notFound) {
		return &filekit.PathError{
			Op:   op,
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	// Map other specific errors here

	return &filekit.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
