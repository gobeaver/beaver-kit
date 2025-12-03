package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/filekit"
)

// Mode represents the ZIP adapter mode
type Mode int

const (
	// ModeRead opens an existing ZIP file for reading
	ModeRead Mode = iota
	// ModeWrite creates a new ZIP file for writing
	ModeWrite
	// ModeReadWrite opens or creates a ZIP file for both reading and writing
	ModeReadWrite
)

// Adapter provides a ZIP archive implementation of filekit.FileSystem
type Adapter struct {
	mu       sync.RWMutex
	path     string
	mode     Mode
	reader   *zip.ReadCloser
	writer   *zip.Writer
	file     *os.File
	files    map[string]*zipEntry // In-memory index for read mode
	pending  map[string]*zipEntry // Pending writes for write mode
	modified bool
}

// zipEntry represents a file or directory in the ZIP
type zipEntry struct {
	header  *zip.FileHeader
	content []byte
	isDir   bool
}

// Open opens an existing ZIP file for reading
func Open(zipPath string) (*Adapter, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	a := &Adapter{
		path:   zipPath,
		mode:   ModeRead,
		reader: reader,
		files:  make(map[string]*zipEntry),
	}

	// Build file index
	for _, f := range reader.File {
		name := normalizePath(f.Name)
		a.files[name] = &zipEntry{
			header: &f.FileHeader,
			isDir:  f.FileInfo().IsDir(),
		}

		// Also add parent directories
		a.ensureParentDirs(name)
	}

	return a, nil
}

// Create creates a new ZIP file for writing
func Create(zipPath string) (*Adapter, error) {
	file, err := os.Create(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	return &Adapter{
		path:    zipPath,
		mode:    ModeWrite,
		file:    file,
		writer:  zip.NewWriter(file),
		pending: make(map[string]*zipEntry),
		files:   make(map[string]*zipEntry),
	}, nil
}

// OpenOrCreate opens an existing ZIP or creates a new one
func OpenOrCreate(zipPath string) (*Adapter, error) {
	// Check if file exists
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return Create(zipPath)
	}

	// Open existing file and load into memory for read-write
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	a := &Adapter{
		path:    zipPath,
		mode:    ModeReadWrite,
		reader:  reader,
		files:   make(map[string]*zipEntry),
		pending: make(map[string]*zipEntry),
	}

	// Load existing files into memory
	for _, f := range reader.File {
		name := normalizePath(f.Name)

		// Read file content
		var content []byte
		if !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				reader.Close()
				return nil, fmt.Errorf("failed to read zip entry: %w", err)
			}
			content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				reader.Close()
				return nil, fmt.Errorf("failed to read zip entry content: %w", err)
			}
		}

		a.files[name] = &zipEntry{
			header:  &f.FileHeader,
			content: content,
			isDir:   f.FileInfo().IsDir(),
		}

		// Also add parent directories
		a.ensureParentDirs(name)
	}

	return a, nil
}

// Close closes the ZIP adapter and finalizes any writes
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var errs []error

	// For write or read-write mode, finalize the ZIP
	if a.mode == ModeWrite || (a.mode == ModeReadWrite && a.modified) {
		if a.mode == ModeReadWrite {
			// Need to rewrite the entire ZIP
			if err := a.rewriteZip(); err != nil {
				errs = append(errs, err)
			}
		} else if a.writer != nil {
			if err := a.writer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if a.reader != nil {
		if err := a.reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.file != nil {
		if err := a.file.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing zip: %v", errs)
	}

	return nil
}

// rewriteZip rewrites the ZIP file with all changes
func (a *Adapter) rewriteZip() error {
	// Close the reader first
	if a.reader != nil {
		a.reader.Close()
		a.reader = nil
	}

	// Create a temporary file
	tmpPath := a.path + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	writer := zip.NewWriter(tmpFile)

	// Merge files and pending into the new ZIP
	allFiles := make(map[string]*zipEntry)
	for k, v := range a.files {
		allFiles[k] = v
	}
	for k, v := range a.pending {
		allFiles[k] = v
	}

	// Sort keys for consistent output
	var keys []string
	for k := range allFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write all entries
	for _, name := range keys {
		entry := allFiles[name]
		if entry == nil {
			continue // Deleted entry
		}

		header := &zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}

		if entry.isDir {
			header.Name = name + "/"
			header.SetMode(os.ModeDir | 0755)
			_, err := writer.CreateHeader(header)
			if err != nil {
				writer.Close()
				tmpFile.Close()
				os.Remove(tmpPath)
				return err
			}
		} else {
			header.SetMode(0644)
			w, err := writer.CreateHeader(header)
			if err != nil {
				writer.Close()
				tmpFile.Close()
				os.Remove(tmpPath)
				return err
			}
			if _, err := w.Write(entry.content); err != nil {
				writer.Close()
				tmpFile.Close()
				os.Remove(tmpPath)
				return err
			}
		}
	}

	if err := writer.Close(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Replace original with temp
	if err := os.Rename(tmpPath, a.path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// Write implements filekit.FileWriter
func (a *Adapter) Write(ctx context.Context, filePath string, content io.Reader, options ...filekit.Option) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	filePath = normalizePath(filePath)

	if !isValidPath(filePath) {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	opts := processOptions(options...)

	// Check if file exists
	if !opts.Overwrite {
		if _, exists := a.files[filePath]; exists {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  filekit.ErrExist,
			}
		}
		if _, exists := a.pending[filePath]; exists {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  filekit.ErrExist,
			}
		}
	}

	// Read content
	data, err := io.ReadAll(content)
	if err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  err,
		}
	}

	// For write mode, write directly to the ZIP
	if a.mode == ModeWrite && a.writer != nil {
		header := &zip.FileHeader{
			Name:     filePath,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		header.SetMode(0644)

		w, err := a.writer.CreateHeader(header)
		if err != nil {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  err,
			}
		}

		if _, err := w.Write(data); err != nil {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  err,
			}
		}

		// Add to index
		a.files[filePath] = &zipEntry{
			header:  header,
			content: data,
			isDir:   false,
		}
		a.ensureParentDirs(filePath)
	} else {
		// For read-write mode, store in pending
		a.pending[filePath] = &zipEntry{
			content: data,
			isDir:   false,
		}
		a.modified = true
		a.ensureParentDirsPending(filePath)
	}

	return nil
}

// Read implements filekit.FileReader
func (a *Adapter) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	filePath = normalizePath(filePath)

	// Check pending first (for read-write mode)
	if entry, exists := a.pending[filePath]; exists {
		if entry == nil || entry.isDir {
			return nil, &filekit.PathError{
				Op:   "read",
				Path: filePath,
				Err:  filekit.ErrNotExist,
			}
		}
		return io.NopCloser(bytes.NewReader(entry.content)), nil
	}

	// Check files index
	entry, exists := a.files[filePath]
	if !exists {
		return nil, &filekit.PathError{
			Op:   "read",
			Path: filePath,
			Err:  filekit.ErrNotExist,
		}
	}

	if entry.isDir {
		return nil, &filekit.PathError{
			Op:   "read",
			Path: filePath,
			Err:  filekit.ErrIsDir,
		}
	}

	// If we have content in memory (read-write mode)
	if entry.content != nil {
		return io.NopCloser(bytes.NewReader(entry.content)), nil
	}

	// Read from ZIP file (read mode)
	if a.reader != nil {
		for _, f := range a.reader.File {
			if normalizePath(f.Name) == filePath {
				return f.Open()
			}
		}
	}

	return nil, &filekit.PathError{
		Op:   "read",
		Path: filePath,
		Err:  filekit.ErrNotExist,
	}
}

// ReadAll reads the entire file contents
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{
			Op:   "delete",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	filePath = normalizePath(filePath)

	// Check if exists
	_, inFiles := a.files[filePath]
	_, inPending := a.pending[filePath]

	if !inFiles && !inPending {
		return &filekit.PathError{
			Op:   "delete",
			Path: filePath,
			Err:  filekit.ErrNotExist,
		}
	}

	// Mark as deleted
	delete(a.files, filePath)
	delete(a.pending, filePath)
	a.modified = true

	return nil
}

// FileExists checks if a file exists at the given path
func (a *Adapter) FileExists(ctx context.Context, filePath string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	filePath = normalizePath(filePath)

	if entry, exists := a.pending[filePath]; exists {
		return entry != nil && !entry.isDir, nil
	}

	if entry, exists := a.files[filePath]; exists {
		return !entry.isDir, nil
	}

	return false, nil
}

// DirExists checks if a directory exists at the given path
func (a *Adapter) DirExists(ctx context.Context, dirPath string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	dirPath = normalizePath(dirPath)

	if entry, exists := a.pending[dirPath]; exists {
		return entry != nil && entry.isDir, nil
	}

	if entry, exists := a.files[dirPath]; exists {
		return entry.isDir, nil
	}

	return false, nil
}

// Stat implements filekit.FileReader
func (a *Adapter) Stat(ctx context.Context, filePath string) (*filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	filePath = normalizePath(filePath)

	// Check pending first
	if entry, exists := a.pending[filePath]; exists && entry != nil {
		return &filekit.FileInfo{
			Name:        filepath.Base(filePath),
			Path:        filePath,
			Size:        int64(len(entry.content)),
			ModTime:     time.Now(),
			IsDir:       entry.isDir,
			ContentType: detectContentType(filePath, entry.content),
		}, nil
	}

	// Check files
	entry, exists := a.files[filePath]
	if !exists {
		return nil, &filekit.PathError{
			Op:   "stat",
			Path: filePath,
			Err:  filekit.ErrNotExist,
		}
	}

	var size int64
	var modTime time.Time
	if entry.header != nil {
		size = int64(entry.header.UncompressedSize64)
		modTime = entry.header.Modified
	}
	if entry.content != nil {
		size = int64(len(entry.content))
	}

	return &filekit.FileInfo{
		Name:        filepath.Base(filePath),
		Path:        filePath,
		Size:        size,
		ModTime:     modTime,
		IsDir:       entry.isDir,
		ContentType: detectContentType(filePath, entry.content),
	}, nil
}

// ListContents lists files and directories at the given path
func (a *Adapter) ListContents(ctx context.Context, prefix string, recursive bool) ([]filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	prefix = normalizePath(prefix)

	// Check if prefix is a directory
	if prefix != "" {
		entry, exists := a.files[prefix]
		pendingEntry, pendingExists := a.pending[prefix]

		if exists && !entry.isDir {
			return nil, &filekit.PathError{
				Op:   "listcontents",
				Path: prefix,
				Err:  filekit.ErrNotDir,
			}
		}
		if pendingExists && !pendingEntry.isDir {
			return nil, &filekit.PathError{
				Op:   "listcontents",
				Path: prefix,
				Err:  filekit.ErrNotDir,
			}
		}
		if !exists && !pendingExists {
			return nil, &filekit.PathError{
				Op:   "listcontents",
				Path: prefix,
				Err:  filekit.ErrNotExist,
			}
		}
	}

	seen := make(map[string]bool)
	var files []filekit.FileInfo

	// Helper to process entries
	processEntry := func(entryPath string, entry *zipEntry) {
		if entry == nil {
			return
		}

		var relPath string
		if prefix == "" {
			relPath = entryPath
		} else {
			if !strings.HasPrefix(entryPath, prefix+"/") {
				return
			}
			relPath = strings.TrimPrefix(entryPath, prefix+"/")
		}

		if relPath == "" {
			return
		}

		if recursive {
			// In recursive mode, include all descendants
			if seen[entryPath] {
				return
			}
			seen[entryPath] = true

			var size int64
			var modTime time.Time
			if entry.header != nil {
				size = int64(entry.header.UncompressedSize64)
				modTime = entry.header.Modified
			}
			if entry.content != nil {
				size = int64(len(entry.content))
			}

			files = append(files, filekit.FileInfo{
				Name:        filepath.Base(entryPath),
				Path:        entryPath,
				Size:        size,
				ModTime:     modTime,
				IsDir:       entry.isDir,
				ContentType: detectContentType(entryPath, entry.content),
			})
		} else {
			// Non-recursive: only immediate children
			parts := strings.SplitN(relPath, "/", 2)
			childName := parts[0]

			if seen[childName] {
				return
			}
			seen[childName] = true

			// If there are more parts, this is nested - skip (will be listed as dir)
			if len(parts) > 1 {
				// Add as directory
				files = append(files, filekit.FileInfo{
					Name:  childName,
					Path:  path.Join(prefix, childName),
					IsDir: true,
				})
				return
			}

			var size int64
			var modTime time.Time
			if entry.header != nil {
				size = int64(entry.header.UncompressedSize64)
				modTime = entry.header.Modified
			}
			if entry.content != nil {
				size = int64(len(entry.content))
			}

			files = append(files, filekit.FileInfo{
				Name:        childName,
				Path:        path.Join(prefix, childName),
				Size:        size,
				ModTime:     modTime,
				IsDir:       entry.isDir,
				ContentType: detectContentType(childName, entry.content),
			})
		}
	}

	// Process files
	for entryPath, entry := range a.files {
		processEntry(entryPath, entry)
	}

	// Process pending
	for entryPath, entry := range a.pending {
		processEntry(entryPath, entry)
	}

	// Sort by path
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, dirPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{
			Op:   "createdir",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	dirPath = normalizePath(dirPath)

	if !isValidPath(dirPath) {
		return &filekit.PathError{
			Op:   "createdir",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	// Check if file exists at path
	if entry, exists := a.files[dirPath]; exists && !entry.isDir {
		return &filekit.PathError{
			Op:   "createdir",
			Path: dirPath,
			Err:  filekit.ErrExist,
		}
	}

	// For write mode, write directory entry to ZIP
	if a.mode == ModeWrite && a.writer != nil {
		header := &zip.FileHeader{
			Name:     dirPath + "/",
			Method:   zip.Store,
			Modified: time.Now(),
		}
		header.SetMode(os.ModeDir | 0755)

		_, err := a.writer.CreateHeader(header)
		if err != nil {
			return &filekit.PathError{
				Op:   "createdir",
				Path: dirPath,
				Err:  err,
			}
		}

		a.files[dirPath] = &zipEntry{
			header: header,
			isDir:  true,
		}
	} else {
		// For read-write mode
		a.pending[dirPath] = &zipEntry{isDir: true}
		a.modified = true
	}

	// Ensure parent directories
	a.ensureParentDirs(dirPath)

	return nil
}

// DeleteDir implements filekit.FileSystem
func (a *Adapter) DeleteDir(ctx context.Context, dirPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	dirPath = normalizePath(dirPath)

	// Check if directory exists
	entry, inFiles := a.files[dirPath]
	pendingEntry, inPending := a.pending[dirPath]

	if !inFiles && !inPending {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotExist,
		}
	}

	if (inFiles && !entry.isDir) || (inPending && !pendingEntry.isDir) {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotDir,
		}
	}

	// Delete directory and all contents
	prefix := dirPath + "/"

	for p := range a.files {
		if p == dirPath || strings.HasPrefix(p, prefix) {
			delete(a.files, p)
		}
	}

	for p := range a.pending {
		if p == dirPath || strings.HasPrefix(p, prefix) {
			delete(a.pending, p)
		}
	}

	a.modified = true

	return nil
}

// UploadFile implements filekit.Uploader
func (a *Adapter) UploadFile(ctx context.Context, destPath string, localPath string, options ...filekit.Option) error {
	file, err := os.Open(localPath)
	if err != nil {
		return &filekit.PathError{
			Op:   "uploadfile",
			Path: localPath,
			Err:  err,
		}
	}
	defer file.Close()

	return a.Write(ctx, destPath, file, options...)
}

// ensureParentDirs creates parent directory entries
func (a *Adapter) ensureParentDirs(filePath string) {
	dir := path.Dir(filePath)
	for dir != "" && dir != "." && dir != "/" {
		if _, exists := a.files[dir]; !exists {
			a.files[dir] = &zipEntry{isDir: true}
		}
		dir = path.Dir(dir)
	}
}

// ensureParentDirsPending creates parent directory entries in pending
func (a *Adapter) ensureParentDirsPending(filePath string) {
	dir := path.Dir(filePath)
	for dir != "" && dir != "." && dir != "/" {
		if _, exists := a.files[dir]; !exists {
			if _, exists := a.pending[dir]; !exists {
				a.pending[dir] = &zipEntry{isDir: true}
			}
		}
		dir = path.Dir(dir)
	}
}

// normalizePath normalizes a file path
func normalizePath(p string) string {
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	if p == "" || p == "." {
		return ""
	}
	return path.Clean(p)
}

// isValidPath checks if path is valid (no traversal)
func isValidPath(p string) bool {
	return !strings.Contains(p, "..")
}

// detectContentType determines content type from path and content
func detectContentType(filePath string, content []byte) string {
	ext := filepath.Ext(filePath)
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}

	if len(content) > 0 {
		return http.DetectContentType(content)
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
	filekit.RegisterDriver("zip", func(cfg filekit.Config) (filekit.FileSystem, error) {
		// ZIP driver requires a path - use LocalBasePath as the ZIP file path
		if cfg.LocalBasePath == "" {
			return nil, fmt.Errorf("zip driver requires LocalBasePath to be set to the ZIP file path")
		}

		return OpenOrCreate(cfg.LocalBasePath)
	})
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// Copy implements filekit.Copier for in-memory ZIP file copying.
func (a *Adapter) Copy(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotAllowed}
	}

	src = normalizePath(src)
	dst = normalizePath(dst)

	if !isValidPath(src) || !isValidPath(dst) {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotAllowed}
	}

	// Get source content
	var content []byte
	if entry, exists := a.pending[src]; exists {
		content = make([]byte, len(entry.content))
		copy(content, entry.content)
	} else if entry, exists := a.files[src]; exists {
		if entry.content != nil {
			content = make([]byte, len(entry.content))
			copy(content, entry.content)
		} else if entry.header != nil {
			// Read from ZIP
			rc, err := a.reader.Open(entry.header.Name)
			if err != nil {
				return &filekit.PathError{Op: "copy", Path: src, Err: err}
			}
			content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return &filekit.PathError{Op: "copy", Path: src, Err: err}
			}
		}
	} else {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotExist}
	}

	// Create destination
	a.pending[dst] = &zipEntry{
		content: content,
		isDir:   false,
	}
	a.ensureParentDirsPending(dst)
	a.modified = true

	return nil
}

// Move implements filekit.Mover for in-memory ZIP file moving.
func (a *Adapter) Move(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.mode == ModeRead {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotAllowed}
	}

	src = normalizePath(src)
	dst = normalizePath(dst)

	if !isValidPath(src) || !isValidPath(dst) {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotAllowed}
	}

	// Get source entry
	var entry *zipEntry
	var fromPending bool
	if e, exists := a.pending[src]; exists {
		entry = e
		fromPending = true
	} else if e, exists := a.files[src]; exists {
		// Need to load content first
		var content []byte
		if e.content != nil {
			content = e.content
		} else if e.header != nil {
			rc, err := a.reader.Open(e.header.Name)
			if err != nil {
				return &filekit.PathError{Op: "move", Path: src, Err: err}
			}
			content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return &filekit.PathError{Op: "move", Path: src, Err: err}
			}
		}
		entry = &zipEntry{content: content, isDir: e.isDir}
	} else {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotExist}
	}

	// Add to destination
	a.pending[dst] = entry
	a.ensureParentDirsPending(dst)

	// Remove from source
	if fromPending {
		delete(a.pending, src)
	} else {
		delete(a.files, src)
	}

	a.modified = true

	return nil
}

// Checksum implements filekit.Checksummer for ZIP archive files.
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
// Watcher Implementation
// ============================================================================

// Watch implements filekit.Watcher.
// For ZIP archives, changes can only occur when the archive itself is modified.
// Returns a NeverChangeToken since in-memory ZIP contents don't change externally.
// For file-based ZIPs in read-write mode, consider polling the underlying file.
func (a *Adapter) Watch(ctx context.Context, filter string) (filekit.ChangeToken, error) {
	// ZIP archives don't have native file system events.
	// The contents only change through explicit API calls.
	// Return a NeverChangeToken since there's no external modification.
	return filekit.NeverChangeToken{}, nil
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
