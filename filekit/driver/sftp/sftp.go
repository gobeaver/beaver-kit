package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/filekit"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Adapter provides an SFTP implementation of filekit.FileSystem
type Adapter struct {
	mu       sync.Mutex
	client   *sftp.Client
	sshConn  *ssh.Client
	basePath string
	config   Config
}

// Config holds SFTP connection configuration
type Config struct {
	Host       string
	Port       int
	Username   string
	Password   string
	PrivateKey []byte // PEM encoded private key
	BasePath   string
}

// AdapterOption is a function that configures SFTP Adapter
type AdapterOption func(*Adapter)

// WithBasePath sets the base path for SFTP operations
func WithBasePath(basePath string) AdapterOption {
	return func(a *Adapter) {
		a.basePath = basePath
	}
}

// New creates a new SFTP filesystem adapter
func New(cfg Config, options ...AdapterOption) (*Adapter, error) {
	adapter := &Adapter{
		config:   cfg,
		basePath: cfg.BasePath,
	}

	// Apply options
	for _, option := range options {
		option(adapter)
	}

	// Establish connection
	if err := adapter.connect(); err != nil {
		return nil, err
	}

	return adapter, nil
}

// connect establishes SSH and SFTP connections
func (a *Adapter) connect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Build SSH config
	sshConfig := &ssh.ClientConfig{
		User:            a.config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use known_hosts in production
	}

	// Add authentication method
	if len(a.config.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(a.config.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	if a.config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(a.config.Password))
	}

	if len(sshConfig.Auth) == 0 {
		return fmt.Errorf("no authentication method provided")
	}

	// Connect to SSH
	port := a.config.Port
	if port == 0 {
		port = 22
	}

	addr := fmt.Sprintf("%s:%d", a.config.Host, port)
	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH: %w", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	a.sshConn = sshConn
	a.client = sftpClient

	return nil
}

// Close closes the SFTP and SSH connections
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var errs []error

	if a.client != nil {
		if err := a.client.Close(); err != nil {
			errs = append(errs, err)
		}
		a.client = nil
	}

	if a.sshConn != nil {
		if err := a.sshConn.Close(); err != nil {
			errs = append(errs, err)
		}
		a.sshConn = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// ensureConnected ensures the SFTP connection is alive
func (a *Adapter) ensureConnected() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client == nil {
		a.mu.Unlock()
		err := a.connect()
		a.mu.Lock()
		return err
	}

	// Test connection with a simple operation
	_, err := a.client.Getwd()
	if err != nil {
		// Connection lost, reconnect
		a.client = nil
		a.sshConn = nil
		a.mu.Unlock()
		err = a.connect()
		a.mu.Lock()
		return err
	}

	return nil
}

// fullPath returns the full path combining base path and relative path
func (a *Adapter) fullPath(relativePath string) string {
	cleanPath := path.Clean(relativePath)
	if a.basePath == "" {
		return cleanPath
	}
	return path.Join(a.basePath, cleanPath)
}

// isPathSafe checks if the path is safe (doesn't escape base path)
func (a *Adapter) isPathSafe(relativePath string) bool {
	fullPath := a.fullPath(relativePath)
	if a.basePath == "" {
		return !strings.HasPrefix(fullPath, "..")
	}
	return strings.HasPrefix(fullPath, a.basePath)
}

// Write implements filekit.FileWriter
func (a *Adapter) Write(ctx context.Context, filePath string, content io.Reader, options ...filekit.Option) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(filePath) {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  err,
		}
	}

	opts := processOptions(options...)
	fullPath := a.fullPath(filePath)

	// Check if file exists and overwrite is not allowed
	if !opts.Overwrite {
		_, err := a.client.Stat(fullPath)
		if err == nil {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  filekit.ErrExist,
			}
		}
		if !os.IsNotExist(err) {
			return &filekit.PathError{
				Op:   "write",
				Path: filePath,
				Err:  err,
			}
		}
	}

	// Ensure parent directory exists
	dir := path.Dir(fullPath)
	if err := a.client.MkdirAll(dir); err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  err,
		}
	}

	// Create file
	file, err := a.client.Create(fullPath)
	if err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  err,
		}
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, content); err != nil {
		return &filekit.PathError{
			Op:   "write",
			Path: filePath,
			Err:  err,
		}
	}

	// Set permissions based on visibility
	var perm os.FileMode = 0600 // Default: private
	if opts.Visibility == filekit.Public {
		perm = 0644
	}
	if err := a.client.Chmod(fullPath, perm); err != nil {
		// Non-fatal error, log and continue
		_ = err
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

	if !a.isPathSafe(filePath) {
		return nil, &filekit.PathError{
			Op:   "read",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return nil, &filekit.PathError{
			Op:   "read",
			Path: filePath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(filePath)

	file, err := a.client.Open(fullPath)
	if err != nil {
		return nil, mapSFTPError("read", filePath, err)
	}

	return file, nil
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(filePath) {
		return &filekit.PathError{
			Op:   "delete",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{
			Op:   "delete",
			Path: filePath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(filePath)

	if err := a.client.Remove(fullPath); err != nil {
		return mapSFTPError("delete", filePath, err)
	}

	return nil
}

// FileExists implements filekit.FileSystem
func (a *Adapter) FileExists(ctx context.Context, filePath string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	if !a.isPathSafe(filePath) {
		return false, &filekit.PathError{
			Op:   "fileexists",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return false, &filekit.PathError{
			Op:   "fileexists",
			Path: filePath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(filePath)

	info, err := a.client.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, mapSFTPError("fileexists", filePath, err)
	}

	// Check if it's a file, not a directory
	if info.IsDir() {
		return false, nil
	}

	return true, nil
}

// DirExists implements filekit.FileSystem
func (a *Adapter) DirExists(ctx context.Context, dirPath string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	if !a.isPathSafe(dirPath) {
		return false, &filekit.PathError{
			Op:   "direxists",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return false, &filekit.PathError{
			Op:   "direxists",
			Path: dirPath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(dirPath)

	info, err := a.client.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, mapSFTPError("direxists", dirPath, err)
	}

	// Check if it's a directory
	return info.IsDir(), nil
}

// Stat implements filekit.FileReader
func (a *Adapter) Stat(ctx context.Context, filePath string) (*filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !a.isPathSafe(filePath) {
		return nil, &filekit.PathError{
			Op:   "stat",
			Path: filePath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return nil, &filekit.PathError{
			Op:   "stat",
			Path: filePath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(filePath)

	info, err := a.client.Stat(fullPath)
	if err != nil {
		return nil, mapSFTPError("stat", filePath, err)
	}

	// Get content type from extension
	contentType := ""
	if !info.IsDir() {
		contentType = detectContentType(filePath)
	}

	return &filekit.FileInfo{
		Name:        filepath.Base(filePath),
		Path:        filePath,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		ContentType: contentType,
	}, nil
}

// ListContents implements filekit.FileSystem
func (a *Adapter) ListContents(ctx context.Context, path string, recursive bool) ([]filekit.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if !a.isPathSafe(path) {
		return nil, &filekit.PathError{
			Op:   "listcontents",
			Path: path,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return nil, &filekit.PathError{
			Op:   "listcontents",
			Path: path,
			Err:  err,
		}
	}

	fullPath := a.fullPath(path)

	// Check if path exists and is a directory
	info, err := a.client.Stat(fullPath)
	if err != nil {
		return nil, mapSFTPError("listcontents", path, err)
	}
	if !info.IsDir() {
		return nil, &filekit.PathError{
			Op:   "listcontents",
			Path: path,
			Err:  filekit.ErrNotDir,
		}
	}

	var files []filekit.FileInfo

	if recursive {
		// Recursive listing
		err = a.listRecursive(fullPath, path, &files)
		if err != nil {
			return nil, mapSFTPError("listcontents", path, err)
		}
	} else {
		// Non-recursive listing
		entries, err := a.client.ReadDir(fullPath)
		if err != nil {
			return nil, mapSFTPError("listcontents", path, err)
		}

		files = make([]filekit.FileInfo, 0, len(entries))
		for _, entry := range entries {
			contentType := ""
			if !entry.IsDir() {
				contentType = detectContentType(entry.Name())
			}

			files = append(files, filekit.FileInfo{
				Name:        entry.Name(),
				Path:        filepath.Join(path, entry.Name()),
				Size:        entry.Size(),
				ModTime:     entry.ModTime(),
				IsDir:       entry.IsDir(),
				ContentType: contentType,
			})
		}
	}

	return files, nil
}

// listRecursive recursively lists all files in a directory
func (a *Adapter) listRecursive(fullPath, relPath string, results *[]filekit.FileInfo) error {
	entries, err := a.client.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryRelPath := filepath.Join(relPath, entry.Name())
		entryFullPath := filepath.Join(fullPath, entry.Name())

		contentType := ""
		if !entry.IsDir() {
			contentType = detectContentType(entry.Name())
		}

		*results = append(*results, filekit.FileInfo{
			Name:        entry.Name(),
			Path:        entryRelPath,
			Size:        entry.Size(),
			ModTime:     entry.ModTime(),
			IsDir:       entry.IsDir(),
			ContentType: contentType,
		})

		if entry.IsDir() {
			if err := a.listRecursive(entryFullPath, entryRelPath, results); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateDir implements filekit.FileSystem
func (a *Adapter) CreateDir(ctx context.Context, dirPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(dirPath) {
		return &filekit.PathError{
			Op:   "createdir",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{
			Op:   "createdir",
			Path: dirPath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(dirPath)

	if err := a.client.MkdirAll(fullPath); err != nil {
		return mapSFTPError("createdir", dirPath, err)
	}

	return nil
}

// DeleteDir implements filekit.FileSystem
func (a *Adapter) DeleteDir(ctx context.Context, dirPath string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(dirPath) {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotAllowed,
		}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  err,
		}
	}

	fullPath := a.fullPath(dirPath)

	// Check if directory exists
	info, err := a.client.Stat(fullPath)
	if err != nil {
		return mapSFTPError("deletedir", dirPath, err)
	}
	if !info.IsDir() {
		return &filekit.PathError{
			Op:   "deletedir",
			Path: dirPath,
			Err:  filekit.ErrNotDir,
		}
	}

	// Recursively delete directory contents
	if err := a.removeAll(fullPath); err != nil {
		return mapSFTPError("deletedir", dirPath, err)
	}

	return nil
}

// removeAll recursively removes a directory and its contents
func (a *Adapter) removeAll(dirPath string) error {
	entries, err := a.client.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := path.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if err := a.removeAll(entryPath); err != nil {
				return err
			}
		} else {
			if err := a.client.Remove(entryPath); err != nil {
				return err
			}
		}
	}

	return a.client.RemoveDirectory(dirPath)
}

// WriteFile implements filekit.FileWriter
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
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}

	// Fallback to http detection
	return http.DetectContentType(nil)
}

// mapSFTPError maps SFTP errors to filekit errors
func mapSFTPError(op, path string, err error) error {
	if os.IsNotExist(err) {
		return &filekit.PathError{
			Op:   op,
			Path: path,
			Err:  filekit.ErrNotExist,
		}
	}

	if os.IsPermission(err) {
		return &filekit.PathError{
			Op:   op,
			Path: path,
			Err:  filekit.ErrPermission,
		}
	}

	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		if os.IsNotExist(pathErr.Err) {
			return &filekit.PathError{
				Op:   op,
				Path: path,
				Err:  filekit.ErrNotExist,
			}
		}
	}

	return &filekit.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

func init() {
	filekit.RegisterDriver("sftp", func(cfg filekit.Config) (filekit.FileSystem, error) {
		if cfg.SFTPHost == "" {
			return nil, fmt.Errorf("SFTP host is required")
		}

		sftpConfig := Config{
			Host:     cfg.SFTPHost,
			Port:     cfg.SFTPPort,
			Username: cfg.SFTPUsername,
			Password: cfg.SFTPPassword,
			BasePath: cfg.SFTPBasePath,
		}

		// Load private key if specified
		if cfg.SFTPPrivateKey != "" {
			keyData, err := os.ReadFile(cfg.SFTPPrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key: %w", err)
			}
			sftpConfig.PrivateKey = keyData
		}

		return New(sftpConfig)
	})
}

// ============================================================================
// Optional Capability Interfaces
// ============================================================================

// Copy implements filekit.Copier by reading and writing via SFTP.
// Note: SFTP doesn't have a native copy command, so this downloads and uploads.
func (a *Adapter) Copy(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(src) || !a.isPathSafe(dst) {
		return &filekit.PathError{Op: "copy", Path: src, Err: filekit.ErrNotAllowed}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{Op: "copy", Path: src, Err: err}
	}

	srcPath := a.fullPath(src)
	dstPath := a.fullPath(dst)

	// Open source file
	srcFile, err := a.client.Open(srcPath)
	if err != nil {
		return mapSFTPError("copy", src, err)
	}
	defer srcFile.Close()

	// Create destination directory if needed
	dstDir := path.Dir(dstPath)
	if err := a.client.MkdirAll(dstDir); err != nil {
		return mapSFTPError("copy", dst, err)
	}

	// Create destination file
	dstFile, err := a.client.Create(dstPath)
	if err != nil {
		return mapSFTPError("copy", dst, err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return mapSFTPError("copy", dst, err)
	}

	return nil
}

// Move implements filekit.Mover using SFTP's native Rename.
func (a *Adapter) Move(ctx context.Context, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !a.isPathSafe(src) || !a.isPathSafe(dst) {
		return &filekit.PathError{Op: "move", Path: src, Err: filekit.ErrNotAllowed}
	}

	if err := a.ensureConnected(); err != nil {
		return &filekit.PathError{Op: "move", Path: src, Err: err}
	}

	srcPath := a.fullPath(src)
	dstPath := a.fullPath(dst)

	// Create destination directory if needed
	dstDir := path.Dir(dstPath)
	if err := a.client.MkdirAll(dstDir); err != nil {
		return mapSFTPError("move", dst, err)
	}

	// Use native rename
	if err := a.client.Rename(srcPath, dstPath); err != nil {
		return mapSFTPError("move", src, err)
	}

	return nil
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
// SFTP doesn't have native file system events, so we poll for changes.
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
			return !sftpStatesEqual(initialState, currentState)
		},
	})

	return token, nil
}

// sftpFileState represents the state of a file for change detection
type sftpFileState struct {
	path    string
	modTime time.Time
	size    int64
}

// getMatchingFilesState returns the current state of files matching the filter
func (a *Adapter) getMatchingFilesState(ctx context.Context, filter string) (map[string]sftpFileState, error) {
	if err := a.ensureConnected(); err != nil {
		return nil, &filekit.PathError{Op: "watch", Path: filter, Err: err}
	}

	state := make(map[string]sftpFileState)

	// Walk the base path recursively
	err := a.walkDir(a.basePath, "", func(relPath string, info os.FileInfo) {
		if info.IsDir() {
			return
		}

		// Check if path matches filter
		if sftpMatchesGlobFilter(relPath, filter) {
			state[relPath] = sftpFileState{
				path:    relPath,
				modTime: info.ModTime(),
				size:    info.Size(),
			}
		}
	})

	if err != nil {
		return nil, &filekit.PathError{Op: "watch", Path: filter, Err: err}
	}

	return state, nil
}

// walkDir recursively walks a directory
func (a *Adapter) walkDir(fullPath, relPath string, fn func(string, os.FileInfo)) error {
	entries, err := a.client.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryRelPath := path.Join(relPath, entry.Name())
		entryFullPath := path.Join(fullPath, entry.Name())

		if entry.IsDir() {
			if err := a.walkDir(entryFullPath, entryRelPath, fn); err != nil {
				return err
			}
		} else {
			fn(entryRelPath, entry)
		}
	}

	return nil
}

// sftpStatesEqual checks if two file states are equal
func sftpStatesEqual(a, b map[string]sftpFileState) bool {
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

// sftpMatchesGlobFilter checks if a path matches a glob pattern
func sftpMatchesGlobFilter(filePath, filter string) bool {
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
	_ filekit.CanChecksum = (*Adapter)(nil)
	_ filekit.CanWatch    = (*Adapter)(nil)
)
