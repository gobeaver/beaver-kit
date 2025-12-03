package filekit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
)

var (
	// ErrMountNotFound is returned when no mount point matches the path
	ErrMountNotFound = errors.New("no mount point found for path")
	// ErrMountExists is returned when trying to mount at an existing path
	ErrMountExists = errors.New("mount point already exists")
	// ErrInvalidMountPath is returned when the mount path is invalid
	ErrInvalidMountPath = errors.New("invalid mount path")
	// ErrEmptyMountPath is returned when the mount path is empty
	ErrEmptyMountPath = errors.New("mount path cannot be empty")
	// ErrNilDriver is returned when trying to mount a nil driver
	ErrNilDriver = errors.New("driver cannot be nil")
	// ErrCrossMount is returned when an operation cannot cross mount boundaries
	ErrCrossMount = errors.New("operation cannot cross mount boundaries")
)

// MountManager provides virtual path namespacing for multiple filesystems.
// It allows mounting different storage backends under virtual paths and
// provides a unified interface to access them all.
type MountManager struct {
	mu     sync.RWMutex
	mounts map[string]FileSystem
	// sorted mount paths for longest-prefix matching
	sortedPaths []string
}

// NewMountManager creates a new mount manager instance.
func NewMountManager() *MountManager {
	return &MountManager{
		mounts: make(map[string]FileSystem),
	}
}

// Mount attaches a filesystem at the specified virtual path.
// The path must start with "/" and be unique.
//
// Example:
//
//	mounts.Mount("/local", localDriver)
//	mounts.Mount("/cloud", s3Driver)
//	mounts.Mount("/cloud/archive", glacierDriver) // nested mounts supported
func (m *MountManager) Mount(mountPath string, fs FileSystem) error {
	if fs == nil {
		return ErrNilDriver
	}

	mountPath = normalizeMountPath(mountPath)
	if mountPath == "" {
		return ErrEmptyMountPath
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mounts[mountPath]; exists {
		return fmt.Errorf("%w: %s", ErrMountExists, mountPath)
	}

	m.mounts[mountPath] = fs
	m.updateSortedPaths()

	return nil
}

// Unmount removes the filesystem at the specified path.
func (m *MountManager) Unmount(mountPath string) error {
	mountPath = normalizeMountPath(mountPath)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mounts[mountPath]; !exists {
		return fmt.Errorf("%w: %s", ErrMountNotFound, mountPath)
	}

	delete(m.mounts, mountPath)
	m.updateSortedPaths()

	return nil
}

// Mounts returns a copy of all current mount points and their filesystems.
func (m *MountManager) Mounts() map[string]FileSystem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]FileSystem, len(m.mounts))
	for k, v := range m.mounts {
		result[k] = v
	}
	return result
}

// MountPaths returns all mount paths in sorted order (longest first).
func (m *MountManager) MountPaths() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.sortedPaths))
	copy(result, m.sortedPaths)
	return result
}

// GetMount returns the filesystem mounted at the exact path.
func (m *MountManager) GetMount(mountPath string) (FileSystem, error) {
	mountPath = normalizeMountPath(mountPath)

	m.mu.RLock()
	defer m.mu.RUnlock()

	fs, exists := m.mounts[mountPath]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrMountNotFound, mountPath)
	}
	return fs, nil
}

// resolve finds the correct mount and relative path for an absolute path.
// Uses longest-prefix matching to support nested mounts.
func (m *MountManager) resolve(absPath string) (FileSystem, string, error) {
	absPath = normalizeMountPath(absPath)
	if absPath == "" {
		return nil, "", ErrEmptyMountPath
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the longest matching mount path
	for _, mountPath := range m.sortedPaths {
		if absPath == mountPath || strings.HasPrefix(absPath, mountPath+"/") {
			fs := m.mounts[mountPath]
			relativePath := strings.TrimPrefix(absPath, mountPath)
			relativePath = strings.TrimPrefix(relativePath, "/")
			return fs, relativePath, nil
		}
	}

	return nil, "", fmt.Errorf("%w: %s", ErrMountNotFound, absPath)
}

// updateSortedPaths updates the sorted paths slice for longest-prefix matching.
// Must be called with lock held.
func (m *MountManager) updateSortedPaths() {
	paths := make([]string, 0, len(m.mounts))
	for p := range m.mounts {
		paths = append(paths, p)
	}
	// Sort by length descending for longest-prefix matching
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) > len(paths[j])
	})
	m.sortedPaths = paths
}

// normalizeMountPath ensures the path starts with "/" and has no trailing slash.
func normalizeMountPath(p string) string {
	if p == "" {
		return ""
	}
	// Ensure leading slash
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Clean the path
	p = path.Clean(p)
	// Remove trailing slash (except for root)
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// ============================================================================
// FileSystem Interface Implementation
// ============================================================================

// Write writes content to the path, routing to the appropriate mount.
func (m *MountManager) Write(ctx context.Context, filePath string, content io.Reader, options ...Option) error {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return err
	}
	return fs.Write(ctx, relativePath, content, options...)
}

// Read reads content from the path, routing to the appropriate mount.
func (m *MountManager) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return nil, err
	}
	return fs.Read(ctx, relativePath)
}

// ReadAll reads all content from the path and returns it as a byte slice.
func (m *MountManager) ReadAll(ctx context.Context, filePath string) ([]byte, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return nil, err
	}
	return fs.ReadAll(ctx, relativePath)
}

// Delete deletes the file at the path, routing to the appropriate mount.
func (m *MountManager) Delete(ctx context.Context, filePath string) error {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return err
	}
	return fs.Delete(ctx, relativePath)
}

// FileExists checks if a file exists at the path.
func (m *MountManager) FileExists(ctx context.Context, filePath string) (bool, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return false, err
	}
	return fs.FileExists(ctx, relativePath)
}

// DirExists checks if a directory exists at the path.
func (m *MountManager) DirExists(ctx context.Context, dirPath string) (bool, error) {
	fs, relativePath, err := m.resolve(dirPath)
	if err != nil {
		return false, err
	}
	return fs.DirExists(ctx, relativePath)
}

// Stat returns information about a file.
func (m *MountManager) Stat(ctx context.Context, filePath string) (*FileInfo, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(ctx, relativePath)
	if err != nil {
		return nil, err
	}
	// Adjust path to include mount prefix
	if info != nil {
		mountPath := m.getMountPathForFile(filePath)
		info.Path = path.Join(mountPath, info.Path)
	}
	return info, nil
}

// ListContents lists files under the given prefix.
// If the prefix matches a mount point exactly, it lists from that mount.
// If the prefix is "/", it returns virtual directories for each mount point.
func (m *MountManager) ListContents(ctx context.Context, prefix string, recursive bool) ([]FileInfo, error) {
	prefix = normalizeMountPath(prefix)

	// Special case: listing root shows mount points
	if prefix == "/" {
		return m.listMountPoints(), nil
	}

	fs, relativePath, err := m.resolve(prefix)
	if err != nil {
		// If no mount found, check if we should list mount point directories
		return m.listMountPointDirs(prefix)
	}

	files, err := fs.ListContents(ctx, relativePath, recursive)
	if err != nil {
		return nil, err
	}

	// Adjust paths to include mount prefix
	mountPath := m.getMountPathForFile(prefix)
	for i := range files {
		files[i].Path = path.Join(mountPath, files[i].Path)
	}

	return files, nil
}

// CreateDir creates a directory at the path.
func (m *MountManager) CreateDir(ctx context.Context, dirPath string) error {
	fs, relativePath, err := m.resolve(dirPath)
	if err != nil {
		return err
	}
	return fs.CreateDir(ctx, relativePath)
}

// DeleteDir deletes a directory at the path.
func (m *MountManager) DeleteDir(ctx context.Context, dirPath string) error {
	fs, relativePath, err := m.resolve(dirPath)
	if err != nil {
		return err
	}
	return fs.DeleteDir(ctx, relativePath)
}

// ============================================================================
// Cross-Mount Operations
// ============================================================================

// Copy copies a file from source to destination.
// Supports cross-mount copying (downloads from source, uploads to destination).
func (m *MountManager) Copy(ctx context.Context, srcPath, dstPath string) error {
	srcFS, srcRelative, err := m.resolve(srcPath)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}

	dstFS, dstRelative, err := m.resolve(dstPath)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	// If same mount, try native copy if supported
	if srcFS == dstFS {
		if copier, ok := srcFS.(Copier); ok {
			return copier.Copy(ctx, srcRelative, dstRelative)
		}
	}

	// Cross-mount copy: read and write
	reader, err := srcFS.Read(ctx, srcRelative)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	defer reader.Close()

	// Get source file info for metadata
	srcInfo, err := srcFS.Stat(ctx, srcRelative)
	if err != nil {
		return fmt.Errorf("get source info: %w", err)
	}

	opts := []Option{}
	if srcInfo.ContentType != "" {
		opts = append(opts, WithContentType(srcInfo.ContentType))
	}
	if len(srcInfo.Metadata) > 0 {
		opts = append(opts, WithMetadata(srcInfo.Metadata))
	}

	if err := dstFS.Write(ctx, dstRelative, reader, opts...); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}

	return nil
}

// Move moves a file from source to destination.
// Supports cross-mount moving (copy + delete).
func (m *MountManager) Move(ctx context.Context, srcPath, dstPath string) error {
	srcFS, srcRelative, err := m.resolve(srcPath)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}

	dstFS, dstRelative, err := m.resolve(dstPath)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	// If same mount, try native move if supported
	if srcFS == dstFS {
		if mover, ok := srcFS.(Mover); ok {
			return mover.Move(ctx, srcRelative, dstRelative)
		}
	}

	// Cross-mount move: copy then delete
	if err := m.Copy(ctx, srcPath, dstPath); err != nil {
		return err
	}

	if err := srcFS.Delete(ctx, srcRelative); err != nil {
		return fmt.Errorf("delete source after move: %w", err)
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// getMountPathForFile returns the mount path for a given file path.
func (m *MountManager) getMountPathForFile(filePath string) string {
	filePath = normalizeMountPath(filePath)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, mountPath := range m.sortedPaths {
		if filePath == mountPath || strings.HasPrefix(filePath, mountPath+"/") {
			return mountPath
		}
	}
	return ""
}

// listMountPoints returns virtual directory entries for each root-level mount.
func (m *MountManager) listMountPoints() []FileInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get unique root-level mount names
	seen := make(map[string]bool)
	var files []FileInfo

	for mountPath := range m.mounts {
		// Get the first path component after /
		parts := strings.SplitN(strings.TrimPrefix(mountPath, "/"), "/", 2)
		if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
			seen[parts[0]] = true
			files = append(files, FileInfo{
				Name:  parts[0],
				Path:  "/" + parts[0],
				IsDir: true,
			})
		}
	}

	// Sort by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files
}

// listMountPointDirs returns virtual directories for nested mount paths.
func (m *MountManager) listMountPointDirs(prefix string) ([]FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]bool)
	var files []FileInfo

	for mountPath := range m.mounts {
		if strings.HasPrefix(mountPath, prefix+"/") {
			// Get the next path component after prefix
			remaining := strings.TrimPrefix(mountPath, prefix+"/")
			parts := strings.SplitN(remaining, "/", 2)
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				seen[parts[0]] = true
				files = append(files, FileInfo{
					Name:  parts[0],
					Path:  path.Join(prefix, parts[0]),
					IsDir: true,
				})
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrMountNotFound, prefix)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// ============================================================================
// Optional Interface Implementations
// ============================================================================

// Checksum implements Checksummer by delegating to the underlying mount.
func (m *MountManager) Checksum(ctx context.Context, filePath string, algorithm ChecksumAlgorithm) (string, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return "", err
	}

	// Check if the underlying filesystem supports checksums
	if checksummer, ok := fs.(Checksummer); ok {
		return checksummer.Checksum(ctx, relativePath, algorithm)
	}

	return "", &PathError{
		Op:   "checksum",
		Path: filePath,
		Err:  ErrNotSupported,
	}
}

// Checksums implements Checksummer by delegating to the underlying mount.
func (m *MountManager) Checksums(ctx context.Context, filePath string, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error) {
	fs, relativePath, err := m.resolve(filePath)
	if err != nil {
		return nil, err
	}

	// Check if the underlying filesystem supports checksums
	if checksummer, ok := fs.(Checksummer); ok {
		return checksummer.Checksums(ctx, relativePath, algorithms)
	}

	return nil, &PathError{
		Op:   "checksums",
		Path: filePath,
		Err:  ErrNotSupported,
	}
}

// ============================================================================
// Watcher Implementation
// ============================================================================

// Watch implements Watcher by delegating to the underlying mount.
// For cross-mount watching (filter patterns that span multiple mounts),
// use CompositeChangeToken to combine multiple watch tokens.
func (m *MountManager) Watch(ctx context.Context, filter string) (ChangeToken, error) {
	filter = normalizeMountPath(filter)

	// Try to resolve the filter as a path to a specific mount
	fs, relativeFilter, err := m.resolve(filter)
	if err != nil {
		// If we can't resolve to a specific mount, check if the filter
		// could match files across multiple mounts (e.g., "**/*.json")
		if strings.Contains(filter, "**") || !strings.HasPrefix(filter, "/") {
			return m.watchAllMounts(ctx, filter)
		}
		return nil, err
	}

	// Check if the underlying filesystem supports watching
	if watcher, ok := fs.(Watcher); ok {
		return watcher.Watch(ctx, relativeFilter)
	}

	// Return a CancelledChangeToken to indicate watching is not supported
	return CancelledChangeToken{}, nil
}

// watchAllMounts creates a composite token that watches across all mounts
func (m *MountManager) watchAllMounts(ctx context.Context, filter string) (ChangeToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tokens []ChangeToken

	for _, fs := range m.mounts {
		if watcher, ok := fs.(Watcher); ok {
			token, err := watcher.Watch(ctx, filter)
			if err != nil {
				// Skip mounts that fail to watch
				continue
			}
			tokens = append(tokens, token)
		}
	}

	if len(tokens) == 0 {
		return CancelledChangeToken{}, nil
	}

	return NewCompositeChangeToken(tokens...), nil
}

// Ensure MountManager implements FileSystem and optional interfaces
var (
	_ FileSystem  = (*MountManager)(nil)
	_ Copier      = (*MountManager)(nil)
	_ Mover       = (*MountManager)(nil)
	_ Checksummer = (*MountManager)(nil)
	_ Watcher     = (*MountManager)(nil)
)
