package memory

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/filekit"
	"github.com/gobwas/glob"
)

// memoryFile represents a file stored in memory
type memoryFile struct {
	content     []byte
	contentType string
	metadata    map[string]string
	modTime     time.Time
	visibility  filekit.Visibility
}

// memoryDir represents a directory in memory
type memoryDir struct {
	modTime time.Time
}

// watchEntry represents a single watch subscription
type watchEntry struct {
	filter string
	token  *filekit.CallbackChangeToken
}

// Adapter provides an in-memory implementation of filekit.FileSystem
// Useful for testing and caching scenarios
type Adapter struct {
	mu      sync.RWMutex
	files   map[string]*memoryFile
	dirs    map[string]*memoryDir
	maxSize int64 // Maximum total storage size (0 = unlimited)
	size    int64 // Current total size

	// Watch support
	watchMu sync.RWMutex
	watches []*watchEntry
}

// Config holds configuration for the memory adapter
type Config struct {
	// MaxSize is the maximum total storage size in bytes (0 = unlimited)
	MaxSize int64
}

// New creates a new in-memory filesystem adapter
func New(cfg ...Config) *Adapter {
	var maxSize int64
	if len(cfg) > 0 {
		maxSize = cfg[0].MaxSize
	}

	a := &Adapter{
		files:   make(map[string]*memoryFile),
		dirs:    make(map[string]*memoryDir),
		maxSize: maxSize,
	}

	// Create root directory
	a.dirs[""] = &memoryDir{modTime: time.Now()}
	a.dirs["/"] = &memoryDir{modTime: time.Now()}

	return a
}

// Write implements filekit.FileWriter
func (a *Adapter) Write(ctx context.Context, path string, content io.Reader, options ...filekit.Option) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path = normalizePath(path)

	// Validate path
	if !isValidPath(path) {
		return &filekit.PathError{
			Op:   "write",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Read content into memory
	data, err := io.ReadAll(content)
	if err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: path,
			Err:  err,
		}
	}

	opts := processOptions(options...)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if file exists and overwrite is not allowed
	if existing, exists := a.files[path]; exists {
		if !opts.Overwrite {
			return &filekit.PathError{
				Op:   "write",
				Path: path,
				Err:  filekit.ErrExist,
			}
		}
		// Subtract old file size
		a.size -= int64(len(existing.content))
	}

	// Check max size limit
	newSize := a.size + int64(len(data))
	if a.maxSize > 0 && newSize > a.maxSize {
		return &filekit.PathError{
			Op:   "write",
			Path: path,
			Err:  filekit.ErrInvalidSize,
		}
	}

	// Ensure parent directories exist
	a.ensureParentDirs(path)

	// Determine content type
	contentType := opts.ContentType
	if contentType == "" {
		contentType = detectContentType(path, data)
	}

	// Store the file
	a.files[path] = &memoryFile{
		content:     data,
		contentType: contentType,
		metadata:    opts.Metadata,
		modTime:     time.Now(),
		visibility:  opts.Visibility,
	}
	a.size = newSize

	// Notify watchers of the change
	go a.notifyWatchers(path)

	return nil
}

// Read implements filekit.FileReader
func (a *Adapter) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	file, exists := a.files[path]
	if !exists {
		return nil, &filekit.PathError{
			Op:   "read",
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	// Return a copy of the content to prevent modification
	return io.NopCloser(bytes.NewReader(file.content)), nil
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
func (a *Adapter) Delete(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.Lock()
	defer a.mu.Unlock()

	file, exists := a.files[path]
	if !exists {
		return &filekit.PathError{
			Op:   "delete",
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	a.size -= int64(len(file.content))
	delete(a.files, path)

	// Notify watchers of the deletion
	go a.notifyWatchers(path)

	return nil
}

// FileExists implements filekit.FileSystem
func (a *Adapter) FileExists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	_, fileExists := a.files[path]

	return fileExists, nil
}

// DirExists checks if a directory exists
func (a *Adapter) DirExists(ctx context.Context, path string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	_, dirExists := a.dirs[path]

	return dirExists, nil
}

// Stat implements filekit.FileReader
func (a *Adapter) Stat(ctx context.Context, path string) (*filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check if it's a file
	if file, exists := a.files[path]; exists {
		return &filekit.FileInfo{
			Name:        filepath.Base(path),
			Path:        path,
			Size:        int64(len(file.content)),
			ModTime:     file.modTime,
			IsDir:       false,
			ContentType: file.contentType,
			Metadata:    file.metadata,
		}, nil
	}

	// Check if it's a directory
	if dir, exists := a.dirs[path]; exists {
		return &filekit.FileInfo{
			Name:    filepath.Base(path),
			Path:    path,
			Size:    0,
			ModTime: dir.modTime,
			IsDir:   true,
		}, nil
	}

	return nil, &filekit.PathError{
		Op:   "stat",
		Path: path,
		Err:  filekit.ErrNotExist,
	}
}

// ListContents implements filekit.FileSystem
func (a *Adapter) ListContents(ctx context.Context, path string, recursive bool) ([]filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check if directory exists
	if _, exists := a.dirs[path]; !exists {
		// Check if it's a file
		if _, isFile := a.files[path]; isFile {
			return nil, &filekit.PathError{
				Op:   "listcontents",
				Path: path,
				Err:  filekit.ErrNotDir,
			}
		}
		return nil, &filekit.PathError{
			Op:   "listcontents",
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	var files []filekit.FileInfo

	if recursive {
		// Recursive mode: return all files and directories under path
		prefixWithSlash := path
		if path != "" && path != "/" {
			prefixWithSlash = path + "/"
		}

		// For root directory, we match all items
		isRoot := path == "" || path == "/"

		// List files
		for filePath, file := range a.files {
			if isRoot || strings.HasPrefix(filePath, prefixWithSlash) {
				files = append(files, filekit.FileInfo{
					Name:        filepath.Base(filePath),
					Path:        filePath,
					Size:        int64(len(file.content)),
					ModTime:     file.modTime,
					IsDir:       false,
					ContentType: file.contentType,
					Metadata:    file.metadata,
				})
			}
		}

		// List directories
		for dirPath, dir := range a.dirs {
			if dirPath == path || dirPath == "" || dirPath == "/" {
				continue
			}
			if isRoot || strings.HasPrefix(dirPath, prefixWithSlash) {
				files = append(files, filekit.FileInfo{
					Name:    filepath.Base(dirPath),
					Path:    dirPath,
					Size:    0,
					ModTime: dir.modTime,
					IsDir:   true,
				})
			}
		}
	} else {
		// Non-recursive mode: return immediate children only
		seen := make(map[string]bool)

		// For root directory, we match all top-level items
		isRoot := path == "" || path == "/"

		// List files
		for filePath, file := range a.files {
			var relPath string
			if isRoot {
				relPath = filePath
			} else {
				if !strings.HasPrefix(filePath, path+"/") {
					continue
				}
				relPath = strings.TrimPrefix(filePath, path+"/")
			}

			if relPath == "" {
				continue
			}

			// Get immediate child name
			parts := strings.SplitN(relPath, "/", 2)
			childName := parts[0]

			if seen[childName] {
				continue
			}

			// If there are more parts, this is a nested file - skip (directory will be listed)
			if len(parts) > 1 {
				continue
			}

			seen[childName] = true
			childPath := filepath.Join(path, childName)

			files = append(files, filekit.FileInfo{
				Name:        childName,
				Path:        childPath,
				Size:        int64(len(file.content)),
				ModTime:     file.modTime,
				IsDir:       false,
				ContentType: file.contentType,
				Metadata:    file.metadata,
			})
		}

		// List directories
		for dirPath, dir := range a.dirs {
			if dirPath == path || dirPath == "" || dirPath == "/" {
				continue
			}

			var relPath string
			if isRoot {
				relPath = dirPath
			} else {
				if !strings.HasPrefix(dirPath, path+"/") {
					continue
				}
				relPath = strings.TrimPrefix(dirPath, path+"/")
			}

			if relPath == "" {
				continue
			}

			// Get immediate child name
			parts := strings.SplitN(relPath, "/", 2)
			childName := parts[0]

			if seen[childName] {
				continue
			}

			// If there are more parts, this is a nested directory - skip
			if len(parts) > 1 {
				continue
			}

			seen[childName] = true
			childPath := filepath.Join(path, childName)

			files = append(files, filekit.FileInfo{
				Name:    childName,
				Path:    childPath,
				Size:    0,
				ModTime: dir.modTime,
				IsDir:   true,
			})
		}
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path = normalizePath(path)

	if !isValidPath(path) {
		return &filekit.PathError{
			Op:   "createdir",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if file exists at this path
	if _, exists := a.files[path]; exists {
		return &filekit.PathError{
			Op:   "createdir",
			Path: path,
			Err:  filekit.ErrExist,
		}
	}

	// Ensure parent directories exist
	a.ensureParentDirs(path)

	// Create the directory
	a.dirs[path] = &memoryDir{modTime: time.Now()}

	return nil
}

// DeleteDir implements filekit.FileSystem
func (a *Adapter) DeleteDir(ctx context.Context, path string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if directory exists
	if _, exists := a.dirs[path]; !exists {
		// Check if it's a file
		if _, isFile := a.files[path]; isFile {
			return &filekit.PathError{
				Op:   "deletedir",
				Path: path,
				Err:  filekit.ErrNotDir,
			}
		}
		return &filekit.PathError{
			Op:   "deletedir",
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	// Delete directory and all contents
	prefixWithSlash := path
	if !strings.HasSuffix(path, "/") {
		prefixWithSlash = path + "/"
	}

	// Collect deleted file paths for notification
	var deletedPaths []string

	// Delete all files under this directory
	for filePath, file := range a.files {
		if strings.HasPrefix(filePath, prefixWithSlash) {
			a.size -= int64(len(file.content))
			deletedPaths = append(deletedPaths, filePath)
			delete(a.files, filePath)
		}
	}

	// Delete all subdirectories
	for dirPath := range a.dirs {
		if strings.HasPrefix(dirPath, prefixWithSlash) || dirPath == path {
			delete(a.dirs, dirPath)
		}
	}

	// Notify watchers of all deleted files
	if len(deletedPaths) > 0 {
		go func() {
			for _, p := range deletedPaths {
				a.notifyWatchers(p)
			}
		}()
	}

	return nil
}

// WriteFile implements filekit.FileWriter
func (a *Adapter) WriteFile(ctx context.Context, path string, localPath string, options ...filekit.Option) error {
	return &filekit.PathError{
		Op:   "writefile",
		Path: localPath,
		Err:  filekit.ErrNotSupported,
	}
}

// Clear removes all files and directories from the memory filesystem
// Useful for testing cleanup
func (a *Adapter) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.files = make(map[string]*memoryFile)
	a.dirs = make(map[string]*memoryDir)
	a.size = 0

	// Recreate root directory
	a.dirs[""] = &memoryDir{modTime: time.Now()}
	a.dirs["/"] = &memoryDir{modTime: time.Now()}
}

// Size returns the current total size of all stored files
func (a *Adapter) Size() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.size
}

// FileCount returns the number of files stored
func (a *Adapter) FileCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.files)
}

// ensureParentDirs creates all parent directories for a given path
// Must be called with lock held
func (a *Adapter) ensureParentDirs(path string) {
	dir := filepath.Dir(path)
	for dir != "" && dir != "." && dir != "/" {
		if _, exists := a.dirs[dir]; !exists {
			a.dirs[dir] = &memoryDir{modTime: time.Now()}
		}
		dir = filepath.Dir(dir)
	}
}

// normalizePath normalizes a file path
func normalizePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "." {
		return ""
	}
	path = filepath.Clean(path)
	return path
}

// isValidPath checks if a path is valid (no directory traversal)
func isValidPath(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	return true
}

// detectContentType determines the content type of a file
func detectContentType(path string, data []byte) string {
	// Try extension first
	ext := filepath.Ext(path)
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}

	// Fall back to content detection
	if len(data) > 0 {
		return http.DetectContentType(data)
	}

	return "application/octet-stream"
}

// processOptions processes the provided options
func processOptions(options ...filekit.Option) *filekit.Options {
	opts := &filekit.Options{}
	for _, option := range options {
		option(opts)
	}
	return opts
}

func init() {
	filekit.RegisterDriver("memory", func(cfg filekit.Config) (filekit.FileSystem, error) {
		return New(), nil
	})
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// Copy implements filekit.Copier for in-memory file copying.
func (a *Adapter) Copy(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	src = normalizePath(src)
	dst = normalizePath(dst)

	if !isValidPath(src) || !isValidPath(dst) {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotAllowed}
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Get source file
	srcFile, exists := a.files[src]
	if !exists {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotExist}
	}

	// Check size limit
	if a.maxSize > 0 && a.size+int64(len(srcFile.content)) > a.maxSize {
		return &filekit.PathError{Op: "copy", Path: dst, Err: filekit.ErrNoSpace}
	}

	// Ensure parent directories exist
	a.ensureParentDirs(dst)

	// Copy file data
	content := make([]byte, len(srcFile.content))
	copy(content, srcFile.content)

	metadata := make(map[string]string, len(srcFile.metadata))
	for k, v := range srcFile.metadata {
		metadata[k] = v
	}

	a.files[dst] = &memoryFile{
		content:     content,
		contentType: srcFile.contentType,
		modTime:     time.Now(),
		metadata:    metadata,
	}
	a.size += int64(len(content))

	// Notify watchers of the new file
	go a.notifyWatchers(dst)

	return nil
}

// Move implements filekit.Mover for in-memory file moving.
func (a *Adapter) Move(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	src = normalizePath(src)
	dst = normalizePath(dst)

	if !isValidPath(src) || !isValidPath(dst) {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotAllowed}
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Get source file
	srcFile, exists := a.files[src]
	if !exists {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotExist}
	}

	// Ensure parent directories exist
	a.ensureParentDirs(dst)

	// Move file (no size change)
	a.files[dst] = srcFile
	srcFile.modTime = time.Now()
	delete(a.files, src)

	// Notify watchers of both source deletion and destination creation
	go func() {
		a.notifyWatchers(src)
		a.notifyWatchers(dst)
	}()

	return nil
}

// Checksum implements filekit.Checksummer for in-memory files.
func (a *Adapter) Checksum(ctx context.Context, path string, algorithm filekit.ChecksumAlgorithm) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	file, exists := a.files[path]
	if !exists {
		return "", &filekit.PathError{Op: "checksum", Path: path, Err: filekit.ErrNotExist}
	}

	checksum, err := filekit.CalculateChecksum(bytes.NewReader(file.content), algorithm)
	if err != nil {
		return "", &filekit.PathError{Op: "checksum", Path: path, Err: err}
	}

	return checksum, nil
}

// Checksums implements filekit.MultiChecksummer for efficient multi-hash calculation.
func (a *Adapter) Checksums(ctx context.Context, path string, algorithms []filekit.ChecksumAlgorithm) (map[filekit.ChecksumAlgorithm]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path = normalizePath(path)

	a.mu.RLock()
	defer a.mu.RUnlock()

	file, exists := a.files[path]
	if !exists {
		return nil, &filekit.PathError{Op: "checksums", Path: path, Err: filekit.ErrNotExist}
	}

	checksums, err := filekit.CalculateChecksums(bytes.NewReader(file.content), algorithms)
	if err != nil {
		return nil, &filekit.PathError{Op: "checksums", Path: path, Err: err}
	}

	return checksums, nil
}

// ============================================================================
// Watcher Implementation
// ============================================================================

// Watch implements filekit.Watcher for in-memory file change detection.
// Supports glob patterns like "**/*.txt", "*.json", "config/*"
func (a *Adapter) Watch(ctx context.Context, filter string) (filekit.ChangeToken, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate the glob pattern
	_, err := glob.Compile(filter)
	if err != nil {
		return nil, &filekit.PathError{
			Op:   "watch",
			Path: filter,
			Err:  err,
		}
	}

	token := filekit.NewCallbackChangeToken()

	a.watchMu.Lock()
	a.watches = append(a.watches, &watchEntry{
		filter: filter,
		token:  token,
	})
	a.watchMu.Unlock()

	// Clean up when context is cancelled
	go func() {
		<-ctx.Done()
		a.removeWatch(token)
	}()

	return token, nil
}

// notifyWatchers signals all watchers whose filter matches the given path
func (a *Adapter) notifyWatchers(path string) {
	a.watchMu.RLock()
	defer a.watchMu.RUnlock()

	for _, entry := range a.watches {
		if matchesFilter(path, entry.filter) {
			entry.token.SignalChange()
		}
	}
}

// removeWatch removes a watch entry by token
func (a *Adapter) removeWatch(token *filekit.CallbackChangeToken) {
	a.watchMu.Lock()
	defer a.watchMu.Unlock()

	for i, entry := range a.watches {
		if entry.token == token {
			// Remove by swapping with last element
			a.watches[i] = a.watches[len(a.watches)-1]
			a.watches = a.watches[:len(a.watches)-1]
			return
		}
	}
}

// matchesFilter checks if a path matches a glob filter pattern
func matchesFilter(path, filter string) bool {
	g, err := glob.Compile(filter)
	if err != nil {
		return false
	}
	return g.Match(path)
}

// Ensure Adapter implements interfaces
var (
	_ filekit.FileSystem  = (*Adapter)(nil)
	_ filekit.FileReader  = (*Adapter)(nil)
	_ filekit.FileWriter  = (*Adapter)(nil)
	_ filekit.CanCopy     = (*Adapter)(nil)
	_ filekit.CanMove     = (*Adapter)(nil)
	_ filekit.CanChecksum = (*Adapter)(nil)
	_ filekit.CanWatch    = (*Adapter)(nil)
)
