package filekit

import (
	"context"
	"io"
	"sync"
	"time"
)

// ============================================================================
// Cache Interface
// ============================================================================

// Cache defines the interface for cache backends.
// This interface is designed to be simple and backend-agnostic,
// allowing implementations for in-memory, Redis, Memcached, etc.
//
// Implementations should be thread-safe.
type Cache interface {
	// Get retrieves a value from the cache.
	// Returns the value and true if found, nil and false otherwise.
	Get(key string) (interface{}, bool)

	// Set stores a value in the cache with the given TTL.
	// A TTL of 0 means no expiration.
	Set(key string, value interface{}, ttl time.Duration)

	// Delete removes a value from the cache.
	Delete(key string)

	// Clear removes all values from the cache.
	Clear()
}

// CacheStats provides statistics about cache usage.
// Implementations may optionally support this interface.
type CacheStats interface {
	// Stats returns cache statistics.
	Stats() CacheStatistics
}

// CacheStatistics contains cache performance metrics.
type CacheStatistics struct {
	Hits       int64
	Misses     int64
	Size       int64
	Evictions  int64
	HitRate    float64
}

// ============================================================================
// In-Memory Cache Implementation
// ============================================================================

// cacheEntry represents a single cache entry with expiration.
type cacheEntry struct {
	value      interface{}
	expiration time.Time
	hasExpiry  bool
}

// MemoryCache is a simple in-memory cache implementation.
// It is thread-safe and supports TTL-based expiration.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	hits    int64
	misses  int64
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]*cacheEntry),
	}
}

// Get retrieves a value from the cache.
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	// Check expiration
	if entry.hasExpiry && time.Now().After(entry.expiration) {
		c.mu.Lock()
		delete(c.entries, key)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
	return entry.value, true
}

// Set stores a value in the cache.
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &cacheEntry{value: value}
	if ttl > 0 {
		entry.expiration = time.Now().Add(ttl)
		entry.hasExpiry = true
	}
	c.entries[key] = entry
}

// Delete removes a value from the cache.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all values from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() CacheStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStatistics{
		Hits:    c.hits,
		Misses:  c.misses,
		Size:    int64(len(c.entries)),
		HitRate: hitRate,
	}
}

// Cleanup removes expired entries from the cache.
// Call this periodically to prevent memory leaks from expired entries.
func (c *MemoryCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if entry.hasExpiry && now.After(entry.expiration) {
			delete(c.entries, key)
		}
	}
}

// Ensure MemoryCache implements Cache and CacheStats
var (
	_ Cache      = (*MemoryCache)(nil)
	_ CacheStats = (*MemoryCache)(nil)
)

// ============================================================================
// CachingFileSystem Decorator
// ============================================================================

// CachingFileSystem wraps a FileSystem to cache metadata operations.
// This is useful for:
// - Reducing latency for repeated metadata queries
// - Reducing API calls to cloud storage (cost savings)
// - Improving performance for high-traffic scenarios
//
// Note: Only metadata is cached (FileExists, DirExists, Stat, ListContents), not file content.
// File content caching would consume too much memory for large files.
//
// Example:
//
//	fs := s3driver.New(client, "bucket")
//	cache := filekit.NewMemoryCache()
//	cachedFS := filekit.NewCachingFileSystem(fs, cache,
//	    filekit.WithCacheTTL(5 * time.Minute),
//	    filekit.WithCacheExists(true),
//	    filekit.WithCacheFileInfo(true),
//	)
//
//	// First call hits the backend
//	info, _ := cachedFS.Stat(ctx, "file.txt")
//
//	// Second call returns cached result
//	info, _ = cachedFS.Stat(ctx, "file.txt")
type CachingFileSystem struct {
	fs    FileSystem
	cache Cache
	opts  CacheOptions
}

// CacheOptions configures the CachingFileSystem behavior.
type CacheOptions struct {
	// TTL is the default time-to-live for cache entries.
	// Default: 5 minutes
	TTL time.Duration

	// CacheExists enables caching of FileExists() and DirExists() results.
	// Default: true
	CacheExists bool

	// CacheFileInfo enables caching of Stat() results.
	// Default: true
	CacheFileInfo bool

	// CacheList enables caching of ListContents() results.
	// Default: false (lists can be large and change frequently)
	CacheList bool

	// InvalidateOnWrite clears relevant cache entries when write operations occur.
	// Default: true
	InvalidateOnWrite bool

	// PathFilter optionally filters which paths should be cached.
	// If nil, all paths are cached.
	// Return true to cache the path, false to skip caching.
	PathFilter func(path string) bool

	// KeyPrefix is prepended to all cache keys.
	// Useful when sharing a cache between multiple filesystems.
	// Default: "filekit:"
	KeyPrefix string

	// OnCacheHit is called when a cache hit occurs.
	// Useful for metrics and debugging.
	OnCacheHit func(op, path string)

	// OnCacheMiss is called when a cache miss occurs.
	// Useful for metrics and debugging.
	OnCacheMiss func(op, path string)
}

// CacheOption is a functional option for configuring CachingFileSystem.
type CacheOption func(*CacheOptions)

// WithCacheTTL sets the default TTL for cache entries.
func WithCacheTTL(ttl time.Duration) CacheOption {
	return func(o *CacheOptions) {
		o.TTL = ttl
	}
}

// WithCacheExists enables or disables caching of FileExists() and DirExists() results.
func WithCacheExists(enabled bool) CacheOption {
	return func(o *CacheOptions) {
		o.CacheExists = enabled
	}
}

// WithCacheFileInfo enables or disables caching of Stat() results.
func WithCacheFileInfo(enabled bool) CacheOption {
	return func(o *CacheOptions) {
		o.CacheFileInfo = enabled
	}
}

// WithCacheList enables or disables caching of ListContents() results.
func WithCacheList(enabled bool) CacheOption {
	return func(o *CacheOptions) {
		o.CacheList = enabled
	}
}

// WithInvalidateOnWrite enables or disables cache invalidation on write operations.
func WithInvalidateOnWrite(enabled bool) CacheOption {
	return func(o *CacheOptions) {
		o.InvalidateOnWrite = enabled
	}
}

// WithCachePathFilter sets a filter function for which paths should be cached.
func WithCachePathFilter(filter func(path string) bool) CacheOption {
	return func(o *CacheOptions) {
		o.PathFilter = filter
	}
}

// WithCacheKeyPrefix sets the prefix for cache keys.
func WithCacheKeyPrefix(prefix string) CacheOption {
	return func(o *CacheOptions) {
		o.KeyPrefix = prefix
	}
}

// WithCacheHitCallback sets the callback for cache hits.
func WithCacheHitCallback(callback func(op, path string)) CacheOption {
	return func(o *CacheOptions) {
		o.OnCacheHit = callback
	}
}

// WithCacheMissCallback sets the callback for cache misses.
func WithCacheMissCallback(callback func(op, path string)) CacheOption {
	return func(o *CacheOptions) {
		o.OnCacheMiss = callback
	}
}

// NewCachingFileSystem creates a caching wrapper around a FileSystem.
func NewCachingFileSystem(fs FileSystem, cache Cache, opts ...CacheOption) *CachingFileSystem {
	options := CacheOptions{
		TTL:               5 * time.Minute,
		CacheExists:       true,
		CacheFileInfo:     true,
		CacheList:         false,
		InvalidateOnWrite: true,
		KeyPrefix:         "filekit:",
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &CachingFileSystem{
		fs:    fs,
		cache: cache,
		opts:  options,
	}
}

// Unwrap returns the underlying FileSystem.
func (c *CachingFileSystem) Unwrap() FileSystem {
	return c.fs
}

// Cache returns the underlying Cache.
func (c *CachingFileSystem) Cache() Cache {
	return c.cache
}

// cacheKey generates a cache key for the given operation and path.
func (c *CachingFileSystem) cacheKey(op, path string) string {
	return c.opts.KeyPrefix + op + ":" + path
}

// shouldCache returns true if the path should be cached.
func (c *CachingFileSystem) shouldCache(path string) bool {
	if c.opts.PathFilter == nil {
		return true
	}
	return c.opts.PathFilter(path)
}

// invalidatePath removes cache entries for a path.
func (c *CachingFileSystem) invalidatePath(path string) {
	if !c.opts.InvalidateOnWrite {
		return
	}
	c.cache.Delete(c.cacheKey("fileexists", path))
	c.cache.Delete(c.cacheKey("direxists", path))
	c.cache.Delete(c.cacheKey("stat", path))
	// Note: List cache invalidation is more complex (would need prefix matching)
	// For simplicity, we clear all list caches when any write occurs
}

// invalidateAll clears all cache entries.
func (c *CachingFileSystem) invalidateAll() {
	if c.opts.InvalidateOnWrite {
		c.cache.Clear()
	}
}

// ============================================================================
// FileSystem Interface - Cached Operations
// ============================================================================

// FileExists checks if a file exists, using cache when available.
func (c *CachingFileSystem) FileExists(ctx context.Context, path string) (bool, error) {
	if !c.opts.CacheExists || !c.shouldCache(path) {
		return c.fs.FileExists(ctx, path)
	}

	key := c.cacheKey("fileexists", path)

	// Try cache first
	if cached, ok := c.cache.Get(key); ok {
		if c.opts.OnCacheHit != nil {
			c.opts.OnCacheHit("fileexists", path)
		}
		return cached.(bool), nil
	}

	if c.opts.OnCacheMiss != nil {
		c.opts.OnCacheMiss("fileexists", path)
	}

	// Cache miss, call underlying filesystem
	exists, err := c.fs.FileExists(ctx, path)
	if err != nil {
		return false, err
	}

	// Cache the result
	c.cache.Set(key, exists, c.opts.TTL)
	return exists, nil
}

// DirExists checks if a directory exists, using cache when available.
func (c *CachingFileSystem) DirExists(ctx context.Context, path string) (bool, error) {
	if !c.opts.CacheExists || !c.shouldCache(path) {
		return c.fs.DirExists(ctx, path)
	}

	key := c.cacheKey("direxists", path)

	// Try cache first
	if cached, ok := c.cache.Get(key); ok {
		if c.opts.OnCacheHit != nil {
			c.opts.OnCacheHit("direxists", path)
		}
		return cached.(bool), nil
	}

	if c.opts.OnCacheMiss != nil {
		c.opts.OnCacheMiss("direxists", path)
	}

	// Cache miss, call underlying filesystem
	exists, err := c.fs.DirExists(ctx, path)
	if err != nil {
		return false, err
	}

	// Cache the result
	c.cache.Set(key, exists, c.opts.TTL)
	return exists, nil
}

// Stat returns file information, using cache when available.
func (c *CachingFileSystem) Stat(ctx context.Context, path string) (*FileInfo, error) {
	if !c.opts.CacheFileInfo || !c.shouldCache(path) {
		return c.fs.Stat(ctx, path)
	}

	key := c.cacheKey("stat", path)

	// Try cache first
	if cached, ok := c.cache.Get(key); ok {
		if c.opts.OnCacheHit != nil {
			c.opts.OnCacheHit("stat", path)
		}
		// Return a copy to prevent mutation
		info := cached.(*FileInfo)
		return &FileInfo{
			Name:        info.Name,
			Path:        info.Path,
			Size:        info.Size,
			ModTime:     info.ModTime,
			IsDir:       info.IsDir,
			ContentType: info.ContentType,
			Metadata:    info.Metadata,
		}, nil
	}

	if c.opts.OnCacheMiss != nil {
		c.opts.OnCacheMiss("stat", path)
	}

	// Cache miss, call underlying filesystem
	info, err := c.fs.Stat(ctx, path)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache.Set(key, info, c.opts.TTL)
	return info, nil
}

// ListContents returns directory contents, using cache when available.
func (c *CachingFileSystem) ListContents(ctx context.Context, path string, recursive bool) ([]FileInfo, error) {
	if !c.opts.CacheList || !c.shouldCache(path) {
		return c.fs.ListContents(ctx, path, recursive)
	}

	// Include recursive flag in cache key
	cacheOp := "list"
	if recursive {
		cacheOp = "list-recursive"
	}
	key := c.cacheKey(cacheOp, path)

	// Try cache first
	if cached, ok := c.cache.Get(key); ok {
		if c.opts.OnCacheHit != nil {
			c.opts.OnCacheHit(cacheOp, path)
		}
		// Return a copy to prevent mutation
		files := cached.([]FileInfo)
		result := make([]FileInfo, len(files))
		copy(result, files)
		return result, nil
	}

	if c.opts.OnCacheMiss != nil {
		c.opts.OnCacheMiss(cacheOp, path)
	}

	// Cache miss, call underlying filesystem
	files, err := c.fs.ListContents(ctx, path, recursive)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache.Set(key, files, c.opts.TTL)
	return files, nil
}

// ============================================================================
// FileSystem Interface - Pass-through Operations
// ============================================================================

// Read delegates to the underlying filesystem (content is not cached).
func (c *CachingFileSystem) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return c.fs.Read(ctx, path)
}

// ReadAll delegates to the underlying filesystem (content is not cached).
func (c *CachingFileSystem) ReadAll(ctx context.Context, path string) ([]byte, error) {
	return c.fs.ReadAll(ctx, path)
}

// Write delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) Write(ctx context.Context, path string, content io.Reader, options ...Option) error {
	err := c.fs.Write(ctx, path, content, options...)
	if err == nil {
		c.invalidatePath(path)
	}
	return err
}

// Delete delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) Delete(ctx context.Context, path string) error {
	err := c.fs.Delete(ctx, path)
	if err == nil {
		c.invalidatePath(path)
	}
	return err
}

// CreateDir delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) CreateDir(ctx context.Context, path string) error {
	err := c.fs.CreateDir(ctx, path)
	if err == nil {
		c.invalidatePath(path)
	}
	return err
}

// DeleteDir delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) DeleteDir(ctx context.Context, path string) error {
	err := c.fs.DeleteDir(ctx, path)
	if err == nil {
		c.invalidateAll() // Directory deletion affects many paths
	}
	return err
}

// ============================================================================
// Optional Interface Delegation
// ============================================================================

// Copy delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) Copy(ctx context.Context, src, dst string) error {
	if copier, ok := c.fs.(Copier); ok {
		err := copier.Copy(ctx, src, dst)
		if err == nil {
			c.invalidatePath(dst)
		}
		return err
	}
	return &PathError{Op: "copy", Path: src, Err: ErrNotSupported}
}

// Move delegates to the underlying filesystem and invalidates cache.
func (c *CachingFileSystem) Move(ctx context.Context, src, dst string) error {
	if mover, ok := c.fs.(Mover); ok {
		err := mover.Move(ctx, src, dst)
		if err == nil {
			c.invalidatePath(src)
			c.invalidatePath(dst)
		}
		return err
	}
	return &PathError{Op: "move", Path: src, Err: ErrNotSupported}
}

// Checksum delegates to the underlying filesystem.
func (c *CachingFileSystem) Checksum(ctx context.Context, path string, algorithm ChecksumAlgorithm) (string, error) {
	if checksummer, ok := c.fs.(Checksummer); ok {
		return checksummer.Checksum(ctx, path, algorithm)
	}
	return "", &PathError{Op: "checksum", Path: path, Err: ErrNotSupported}
}

// Checksums delegates to the underlying filesystem.
func (c *CachingFileSystem) Checksums(ctx context.Context, path string, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error) {
	if checksummer, ok := c.fs.(Checksummer); ok {
		return checksummer.Checksums(ctx, path, algorithms)
	}
	return nil, &PathError{Op: "checksums", Path: path, Err: ErrNotSupported}
}

// GenerateDownloadURL delegates to the underlying filesystem.
func (c *CachingFileSystem) GenerateDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	if urlGen, ok := c.fs.(URLGenerator); ok {
		return urlGen.GenerateDownloadURL(ctx, path, expires)
	}
	return "", &PathError{Op: "generate-download-url", Path: path, Err: ErrNotSupported}
}

// GenerateUploadURL delegates to the underlying filesystem.
func (c *CachingFileSystem) GenerateUploadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	if urlGen, ok := c.fs.(URLGenerator); ok {
		return urlGen.GenerateUploadURL(ctx, path, expires)
	}
	return "", &PathError{Op: "generate-upload-url", Path: path, Err: ErrNotSupported}
}

// Watch delegates to the underlying filesystem.
// When change is detected, relevant cache entries are invalidated.
func (c *CachingFileSystem) Watch(ctx context.Context, filter string) (ChangeToken, error) {
	if watcher, ok := c.fs.(Watcher); ok {
		token, err := watcher.Watch(ctx, filter)
		if err != nil {
			return nil, err
		}

		// Wrap the token to invalidate cache on change
		return &cacheInvalidatingToken{
			token: token,
			cache: c,
		}, nil
	}
	return CancelledChangeToken{}, nil
}

// cacheInvalidatingToken wraps a ChangeToken to invalidate cache on change.
type cacheInvalidatingToken struct {
	token ChangeToken
	cache *CachingFileSystem
}

func (t *cacheInvalidatingToken) HasChanged() bool {
	return t.token.HasChanged()
}

func (t *cacheInvalidatingToken) ActiveChangeCallbacks() bool {
	return t.token.ActiveChangeCallbacks()
}

func (t *cacheInvalidatingToken) RegisterChangeCallback(callback func()) (unregister func()) {
	return t.token.RegisterChangeCallback(func() {
		// Invalidate cache before calling the callback
		t.cache.invalidateAll()
		callback()
	})
}

// ============================================================================
// Interface Assertions
// ============================================================================

// Ensure CachingFileSystem implements FileSystem and optional interfaces
var (
	_ FileSystem   = (*CachingFileSystem)(nil)
	_ FileReader   = (*CachingFileSystem)(nil)
	_ FileWriter   = (*CachingFileSystem)(nil)
	_ Copier       = (*CachingFileSystem)(nil)
	_ Mover        = (*CachingFileSystem)(nil)
	_ Checksummer  = (*CachingFileSystem)(nil)
	_ URLGenerator = (*CachingFileSystem)(nil)
	_ Watcher      = (*CachingFileSystem)(nil)
)

// ============================================================================
// Cache Utilities
// ============================================================================

// WarmCache pre-populates the cache with metadata for files under a prefix.
// This is useful for warming the cache before high-traffic periods.
func WarmCache(ctx context.Context, fs *CachingFileSystem, prefix string) error {
	files, err := fs.fs.ListContents(ctx, prefix, false)
	if err != nil {
		return err
	}

	for _, file := range files {
		// Cache exists results
		if file.IsDir {
			fs.cache.Set(fs.cacheKey("direxists", file.Path), true, fs.opts.TTL)
		} else {
			fs.cache.Set(fs.cacheKey("fileexists", file.Path), true, fs.opts.TTL)
		}

		// Cache stat result
		fileCopy := file // Create copy to avoid pointer issues
		fs.cache.Set(fs.cacheKey("stat", file.Path), &fileCopy, fs.opts.TTL)

		// Recursively warm subdirectories
		if file.IsDir {
			if err := WarmCache(ctx, fs, file.Path); err != nil {
				return err
			}
		}
	}

	return nil
}
