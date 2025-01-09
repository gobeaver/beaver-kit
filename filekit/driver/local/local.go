package local

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobeaver/beaver-kit/filekit"
)

// Adapter provides a local filesystem implementation of filekit.FileSystem
type Adapter struct {
	root string
}

// New creates a new local filesystem adapter
func New(root string) (*Adapter, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Ensure the root directory exists
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, err
	}

	return &Adapter{
		root: absRoot,
	}, nil
}

// Upload implements filekit.FileSystem
func (a *Adapter) Upload(ctx context.Context, path string, content io.Reader, options ...filekit.Option) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return &filekit.PathError{
			Op:   "upload",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &filekit.PathError{
			Op:   "upload",
			Path: path,
			Err:  err,
		}
	}

	// Create the file
	f, err := os.Create(fullPath)
	if err != nil {
		return &filekit.PathError{
			Op:   "upload",
			Path: path,
			Err:  err,
		}
	}
	defer f.Close()

	// Copy the content to the file
	_, err = io.Copy(f, content)
	if err != nil {
		return &filekit.PathError{
			Op:   "upload",
			Path: path,
			Err:  err,
		}
	}

	// Apply file options (permissions, etc.) if needed
	opts := processOptions(options...)

	// Set file permissions based on visibility
	if opts.Visibility == filekit.Public {
		if err := os.Chmod(fullPath, 0644); err != nil {
			return &filekit.PathError{
				Op:   "upload",
				Path: path,
				Err:  err,
			}
		}
	} else if opts.Visibility == filekit.Private {
		if err := os.Chmod(fullPath, 0600); err != nil {
			return &filekit.PathError{
				Op:   "upload",
				Path: path,
				Err:  err,
			}
		}
	}

	return nil
}

// Download implements filekit.FileSystem
func (a *Adapter) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return nil, &filekit.PathError{
			Op:   "download",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Open the file
	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &filekit.PathError{
				Op:   "download",
				Path: path,
				Err:  filekit.ErrNotExist,
			}
		}
		return nil, &filekit.PathError{
			Op:   "download",
			Path: path,
			Err:  err,
		}
	}

	return f, nil
}

// Delete implements filekit.FileSystem
func (a *Adapter) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return &filekit.PathError{
			Op:   "delete",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Delete the file
	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &filekit.PathError{
				Op:   "delete",
				Path: path,
				Err:  filekit.ErrNotExist,
			}
		}
		return &filekit.PathError{
			Op:   "delete",
			Path: path,
			Err:  err,
		}
	}

	return nil
}

// Exists implements filekit.FileSystem
func (a *Adapter) Exists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return false, &filekit.PathError{
			Op:   "exists",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &filekit.PathError{
			Op:   "exists",
			Path: path,
			Err:  err,
		}
	}

	return true, nil
}

// FileInfo implements filekit.FileSystem
func (a *Adapter) FileInfo(ctx context.Context, path string) (*filekit.File, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return nil, &filekit.PathError{
			Op:   "fileinfo",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &filekit.PathError{
				Op:   "fileinfo",
				Path: path,
				Err:  filekit.ErrNotExist,
			}
		}
		return nil, &filekit.PathError{
			Op:   "fileinfo",
			Path: path,
			Err:  err,
		}
	}

	// Get content type
	contentType := ""
	if !info.IsDir() {
		contentType = getContentType(fullPath)
	}

	return &filekit.File{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		ContentType: contentType,
	}, nil
}

// List implements filekit.FileSystem
func (a *Adapter) List(ctx context.Context, prefix string) ([]filekit.File, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(prefix))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return nil, &filekit.PathError{
			Op:   "list",
			Path: prefix,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Check if the directory exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &filekit.PathError{
				Op:   "list",
				Path: prefix,
				Err:  filekit.ErrNotExist,
			}
		}
		return nil, &filekit.PathError{
			Op:   "list",
			Path: prefix,
			Err:  err,
		}
	}

	// If it's not a directory, return an error
	if !info.IsDir() {
		return nil, &filekit.PathError{
			Op:   "list",
			Path: prefix,
			Err:  filekit.ErrNotDir,
		}
	}

	// Read the directory
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, &filekit.PathError{
			Op:   "list",
			Path: prefix,
			Err:  err,
		}
	}

	// Convert entries to File structs
	files := make([]filekit.File, 0, len(entries))
	for _, entry := range entries {
		entryPath := filepath.Join(prefix, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		contentType := ""
		if !info.IsDir() {
			contentType = getContentType(filepath.Join(a.root, entryPath))
		}

		files = append(files, filekit.File{
			Name:        entry.Name(),
			Path:        entryPath,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			IsDir:       info.IsDir(),
			ContentType: contentType,
		})
	}

	return files, nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return &filekit.PathError{
			Op:   "createdir",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Create the directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return &filekit.PathError{
			Op:   "createdir",
			Path: path,
			Err:  err,
		}
	}

	return nil
}

// DeleteDir implements filekit.FileSystem
func (a *Adapter) DeleteDir(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue
	}

	fullPath := filepath.Join(a.root, filepath.Clean(path))

	// Check if the path is under the root
	if !isPathUnderRoot(a.root, fullPath) {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Check if the directory exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &filekit.PathError{
				Op:   "deletedir",
				Path: path,
				Err:  filekit.ErrNotExist,
			}
		}
		return &filekit.PathError{
			Op:   "deletedir",
			Path: path,
			Err:  err,
		}
	}

	// Check if it's a directory
	if !info.IsDir() {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: path,
			Err:  filekit.ErrNotDir,
		}
	}

	// Delete the directory
	if err := os.RemoveAll(fullPath); err != nil {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: path,
			Err:  err,
		}
	}

	return nil
}

// UploadFile implements filekit.Uploader
func (a *Adapter) UploadFile(ctx context.Context, path string, localPath string, options ...filekit.Option) error {
	// Open the local file
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

// isPathUnderRoot checks if a path is under a given root directory
func isPathUnderRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "../")
}

// getContentType tries to determine the content type of a file
func getContentType(path string) string {
	// Try to determine content type from extension
	ext := filepath.Ext(path)
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}

	// Try to determine content type by reading file header
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	// Read a small slice of the file to detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return ""
	}

	return http.DetectContentType(buffer[:n])
}

// processOptions processes the provided options
func processOptions(options ...filekit.Option) *filekit.Options {
	opts := &filekit.Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}
