package filekit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Register test drivers
	RegisterDriver("local", newLocalDriver)
	RegisterDriver("s3", newS3Driver)
}

func newLocalDriver(cfg Config) (FileSystem, error) {
	if cfg.LocalBasePath == "" {
		return nil, fmt.Errorf("local base path is required")
	}
	return &testLocalFS{basePath: cfg.LocalBasePath}, nil
}

func newS3Driver(cfg Config) (FileSystem, error) {
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	return &testS3FS{bucket: cfg.S3Bucket, files: make(map[string]string)}, nil
}

// testLocalFS is a simple local filesystem implementation for testing
type testLocalFS struct {
	basePath string
}

func (fs *testLocalFS) Write(ctx context.Context, path string, reader io.Reader, options ...Option) error {
	fullPath := filepath.Join(fs.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0644)
}

func (fs *testLocalFS) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(fs.basePath, path)
	return os.Open(fullPath)
}

func (fs *testLocalFS) ReadAll(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(fs.basePath, path)
	return os.ReadFile(fullPath)
}

func (fs *testLocalFS) FileExists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(fs.basePath, path)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !stat.IsDir(), nil
}

func (fs *testLocalFS) DirExists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(fs.basePath, path)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stat.IsDir(), nil
}

func (fs *testLocalFS) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(fs.basePath, path)
	return os.Remove(fullPath)
}

func (fs *testLocalFS) Stat(ctx context.Context, path string) (*FileInfo, error) {
	fullPath := filepath.Join(fs.basePath, path)
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:    path,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
	}, nil
}

func (fs *testLocalFS) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	var files []FileInfo
	root := filepath.Join(fs.basePath, path)

	if recursive {
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(fs.basePath, p)
				if err != nil {
					return err
				}
				files = append(files, FileInfo{
					Path:    relPath,
					Size:    info.Size(),
					ModTime: info.ModTime(),
					IsDir:   info.IsDir(),
				})
			}
			return nil
		})
		return files, err
	}

	// Non-recursive: only immediate children
	entries, err := os.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		relPath := filepath.Join(path, entry.Name())
		files = append(files, FileInfo{
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		})
	}
	return files, nil
}

func (fs *testLocalFS) CreateDir(ctx context.Context, path string) error {
	fullPath := filepath.Join(fs.basePath, path)
	return os.MkdirAll(fullPath, 0755)
}

func (fs *testLocalFS) DeleteDir(ctx context.Context, path string) error {
	fullPath := filepath.Join(fs.basePath, path)
	return os.RemoveAll(fullPath)
}

// testS3FS is a mock S3 filesystem for testing
type testS3FS struct {
	bucket string
	files  map[string]string
}

func (fs *testS3FS) Write(ctx context.Context, path string, reader io.Reader, options ...Option) error {
	if fs.files == nil {
		fs.files = make(map[string]string)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	fs.files[path] = string(data)
	return nil
}

func (fs *testS3FS) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if fs.files == nil {
		return nil, os.ErrNotExist
	}
	content, exists := fs.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (fs *testS3FS) ReadAll(ctx context.Context, path string) ([]byte, error) {
	if fs.files == nil {
		return nil, os.ErrNotExist
	}
	content, exists := fs.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

func (fs *testS3FS) FileExists(ctx context.Context, path string) (bool, error) {
	if fs.files == nil {
		return false, nil
	}
	_, exists := fs.files[path]
	return exists, nil
}

func (fs *testS3FS) DirExists(ctx context.Context, path string) (bool, error) {
	// S3 doesn't have real directories
	return false, nil
}

func (fs *testS3FS) Delete(ctx context.Context, path string) error {
	if fs.files == nil {
		return nil
	}
	delete(fs.files, path)
	return nil
}

func (fs *testS3FS) Stat(ctx context.Context, path string) (*FileInfo, error) {
	if fs.files == nil {
		return nil, os.ErrNotExist
	}
	content, exists := fs.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return &FileInfo{
		Path: path,
		Size: int64(len(content)),
	}, nil
}

func (fs *testS3FS) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	var files []FileInfo
	if fs.files == nil {
		return files, nil
	}
	for filePath, content := range fs.files {
		if strings.HasPrefix(filePath, path) {
			files = append(files, FileInfo{
				Path: filePath,
				Size: int64(len(content)),
			})
		}
	}
	return files, nil
}

func (fs *testS3FS) CreateDir(ctx context.Context, path string) error {
	// S3 doesn't have real directories
	return nil
}

func (fs *testS3FS) DeleteDir(ctx context.Context, path string) error {
	if fs.files == nil {
		return nil
	}
	// Delete all files with the given prefix
	for filePath := range fs.files {
		if strings.HasPrefix(filePath, path+"/") {
			delete(fs.files, filePath)
		}
	}
	return nil
}
