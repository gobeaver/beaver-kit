# FileKit Research: Filesystem Abstraction Patterns

This document captures research findings from analyzing mature filesystem abstraction libraries to guide FileKit's development with future-proof APIs.

## Libraries Analyzed

1. **Microsoft IFileProvider** (ASP.NET Core) - .NET ecosystem standard
2. **Apache Commons VFS** (Java) - Enterprise-grade virtual filesystem
3. **Flysystem** (PHP) - Most popular PHP filesystem abstraction

---

## 1. Core Interface Patterns

### Microsoft IFileProvider

```csharp
interface IFileProvider {
    IFileInfo GetFileInfo(string subpath);
    IDirectoryContents GetDirectoryContents(string subpath);
    IChangeToken Watch(string filter);
}

interface IChangeToken {
    bool HasChanged { get; }
    bool ActiveChangeCallbacks { get; }
    IDisposable RegisterChangeCallback(Action<object> callback, object state);
}
```

**Key Design Decisions:**
- Minimal interface (3 methods)
- Change detection via tokens (single-use, composable)
- Implementations: PhysicalFileProvider, EmbeddedFileProvider, CompositeFileProvider

### Apache Commons VFS

```java
interface FileSystemManager {
    FileObject resolveFile(String name, FileSystemOptions opts);
    FileSystemCapabilities[] getCapabilities(String scheme);
}

interface FileObject {
    FileName getName();
    FileType getType();
    FileContent getContent();
    FileObject[] getChildren();
    FileObject[] findFiles(FileSelector selector);
    void copyFrom(FileObject src, FileSelector selector);
}

interface FileSelector {
    boolean includeFile(FileSelectInfo info);
    boolean traverseDescendants(FileSelectInfo info);
}
```

**Key Design Decisions:**
- Rich FileObject with navigation (getParent, getChildren, resolveFile)
- FileSelector for powerful filtering during traversal
- Capabilities discovery per filesystem type
- Junctions for mounting filesystems at paths

### Flysystem

```php
interface FilesystemOperator {
    // Read operations
    public function read(string $path): string;
    public function readStream(string $path): resource;
    public function fileExists(string $path): bool;
    public function directoryExists(string $path): bool;
    public function lastModified(string $path): int;
    public function fileSize(string $path): int;
    public function mimeType(string $path): string;
    public function visibility(string $path): string;
    public function listContents(string $path, bool $deep): iterable;

    // Write operations
    public function write(string $path, string $contents, array $config = []): void;
    public function writeStream(string $path, $contents, array $config = []): void;
    public function setVisibility(string $path, string $visibility): void;
    public function delete(string $path): void;
    public function deleteDirectory(string $path): void;
    public function createDirectory(string $path, array $config = []): void;
    public function move(string $source, string $destination, array $config = []): void;
    public function copy(string $source, string $destination, array $config = []): void;
}

// Optional interfaces (type-check before use)
interface PublicUrlGenerator {
    public function publicUrl(string $path, array $config = []): string;
}

interface TemporaryUrlGenerator {
    public function temporaryUrl(string $path, DateTimeInterface $expiresAt, array $config = []): string;
}

interface ChecksumProvider {
    public function checksum(string $path, array $config = []): string;
}
```

**Key Design Decisions:**
- Separate stream methods (readStream, writeStream)
- Optional interfaces via type assertion
- Config array for per-operation options
- Visibility as first-class concept

---

## 2. Decorator/Middleware Patterns

### ReadOnly Pattern

**Flysystem's ReadOnlyFilesystemAdapter:**
```php
class ReadOnlyFilesystemAdapter implements FilesystemAdapter {
    public function __construct(private FilesystemAdapter $adapter) {}

    public function write(...): void {
        throw UnableToWriteFile::atLocation($path, 'This is a readonly adapter.');
    }

    public function read(string $path): string {
        return $this->adapter->read($path);
    }
    // ... delegates read ops, throws on write ops
}
```

**Future-Proof API Considerations:**
- Should wrap any FileSystem implementation
- Clear error messages indicating read-only mode
- Optionally configurable (allow some writes, deny others)
- Support checking if wrapped filesystem supports specific operations

### Caching Pattern

**Flysystem V1 CachedAdapter (deprecated but instructive):**
```php
class CachedAdapter implements AdapterInterface {
    protected $adapter;
    protected $cache;

    public function __construct(AdapterInterface $adapter, CacheInterface $cache) {}

    public function has($path) {
        if ($this->cache->has($path)) {
            return $this->cache->get($path) !== false;
        }
        $result = $this->adapter->has($path);
        $this->cache->set($path, $result);
        return $result;
    }
}
```

**Apache Commons VFS Caching:**
- FileSystemManager maintains cache of opened file systems
- Cache keyed by URI/scheme
- Configurable cache strategies

**Future-Proof API Considerations:**
- Pluggable cache backend (interface-based)
- TTL support for cache entries
- Selective caching (metadata only, not content)
- Integration with ChangeToken for invalidation
- Cache warming and preloading options
- Statistics/metrics exposure

### Path Prefixing Pattern

**Flysystem's PathPrefixedAdapter:**
```php
class PathPrefixedFilesystem implements FilesystemOperator {
    public function __construct(
        private FilesystemOperator $filesystem,
        private string $prefix
    ) {}

    public function read(string $path): string {
        return $this->filesystem->read($this->preparePath($path));
    }

    private function preparePath(string $path): string {
        return ltrim($this->prefix . '/' . ltrim($path, '/'), '/');
    }
}
```

**Already in FileKit:** MountManager provides superior path namespacing

---

## 3. File Selection & Filtering

### Commons VFS Selectors

```java
// Built-in selectors
PatternFileSelector   // Regex patterns
WildcardFileSelector  // Glob patterns (*, ?)
FileDepthSelector     // Depth-limited traversal
FileFilterSelector    // Wraps FileFilter implementations

// Custom selector interface
interface FileSelector {
    boolean includeFile(FileSelectInfo fileInfo) throws Exception;
    boolean traverseDescendants(FileSelectInfo fileInfo) throws Exception;
}

// Usage
FileObject[] files = dir.findFiles(new PatternFileSelector(".*\\.txt$"));
```

**Future-Proof API Considerations:**
- FileSelector interface for custom filtering logic
- Composable selectors (AND, OR, NOT)
- Pre-built selectors for common cases
- Integration with List() and recursive operations
- Streaming results for large directories

---

## 4. Visibility & Permissions

### Flysystem Visibility

```php
// Simple string-based visibility
const VISIBILITY_PUBLIC = 'public';
const VISIBILITY_PRIVATE = 'private';

// Visibility converter for fine-grained control
interface VisibilityConverter {
    public function forFile(string $visibility): int;      // Returns Unix permissions
    public function forDirectory(string $visibility): int;
    public function inverseForFile(int $visibility): string;
    public function inverseForDirectory(int $visibility): string;
    public function defaultForDirectories(): int;
}

// Per-operation visibility
$filesystem->write('path/to/file.txt', 'contents', [
    Config::OPTION_VISIBILITY => Visibility::PRIVATE,
]);
```

**Future-Proof API Considerations:**
- String-based visibility (extensible beyond public/private)
- VisibilityConverter interface for backend-specific mapping
- Per-operation visibility override
- Default visibility configuration

---

## 5. Metadata Handling

### Flysystem Metadata

```php
// Rich metadata via StorageAttributes
interface StorageAttributes {
    public function path(): string;
    public function type(): string;            // 'file' or 'dir'
    public function visibility(): ?string;
    public function lastModified(): ?int;
    public function extraMetadata(): array;
}

interface FileAttributes extends StorageAttributes {
    public function fileSize(): ?int;
    public function mimeType(): ?string;
}
```

### S3-Specific Metadata (Flysystem)
- ACL, CacheControl, ContentDisposition, ContentEncoding
- ContentLength, ContentType, Expires
- Metadata (custom key-value pairs)
- ServerSideEncryption, StorageClass, Tagging

**Future-Proof API Considerations:**
- Standardized metadata fields in File struct
- Custom metadata via map (current approach is good)
- Optional MetadataProvider interface for get/set
- Backend-specific metadata through Options

---

## 6. Stream Handling

### Flysystem Stream Pattern

```php
// Separate methods for content vs stream
public function read(string $path): string;           // Returns full content
public function readStream(string $path): resource;   // Returns stream handle

public function write(string $path, string $contents, array $config = []): void;
public function writeStream(string $path, $contents, array $config = []): void;
```

**Future-Proof API Considerations:**
- Current Download() returns io.ReadCloser (good)
- Consider ReadAt(path, offset, length) for partial reads
- WriteStream for efficient large file handling
- Progress callbacks for streaming operations

---

## 7. Composite/Fallback Patterns

### Microsoft CompositeFileProvider

```csharp
// Combines multiple providers, first match wins
class CompositeFileProvider : IFileProvider {
    public CompositeFileProvider(IEnumerable<IFileProvider> fileProviders) {}

    public IFileInfo GetFileInfo(string subpath) {
        foreach (var provider in _providers) {
            var fileInfo = provider.GetFileInfo(subpath);
            if (fileInfo != null && fileInfo.Exists)
                return fileInfo;
        }
        return new NotFoundFileInfo(subpath);
    }
}
```

**Difference from MountManager:**
- MountManager: Path namespacing (/local/..., /cloud/...)
- CompositeFileSystem: Overlays, first match wins (same paths, multiple backends)

**Future-Proof API Considerations:**
- Configurable resolution strategy (First, Last, Priority)
- Merge listings across all providers
- Fallback write behavior (primary fails, try secondary)

---

## 8. Error Handling Patterns

### Flysystem Exception Hierarchy

```php
FilesystemException (base)
├── UnableToReadFile
├── UnableToWriteFile
├── UnableToDeleteFile
├── UnableToCreateDirectory
├── UnableToDeleteDirectory
├── UnableToMoveFile
├── UnableToCopyFile
├── UnableToRetrieveMetadata
├── UnableToSetVisibility
├── UnableToCheckFileExistence
└── UnableToCheckDirectoryExistence
```

**Future-Proof API Considerations:**
- Current PathError is good foundation
- Add operation-specific error types
- Retryable vs permanent error distinction
- Error codes for programmatic handling

---

## 9. Configuration Patterns

### Per-Operation Config (Flysystem)

```php
$filesystem->write('file.txt', 'content', [
    Config::OPTION_VISIBILITY => 'private',
    Config::OPTION_DIRECTORY_VISIBILITY => 'public',
    'CacheControl' => 'max-age=3600',
    'Metadata' => ['author' => 'john'],
]);
```

### FileSystemOptions (Commons VFS)

```java
// Scheme-specific configuration builders
FtpFileSystemConfigBuilder builder = FtpFileSystemConfigBuilder.getInstance();
FileSystemOptions opts = new FileSystemOptions();
builder.setPassiveMode(opts, true);
builder.setUserDirIsRoot(opts, false);

FileObject file = manager.resolveFile("ftp://host/path", opts);
```

**Future-Proof API Considerations:**
- Current Options pattern is good
- Add timeout, retries, retry backoff
- Per-operation context (trace ID, etc.)
- Backend-specific options through extension

---

## 10. Capabilities Discovery

### Commons VFS Capabilities

```java
enum Capability {
    READ_CONTENT, WRITE_CONTENT, APPEND_CONTENT,
    CREATE, DELETE, RENAME,
    GET_TYPE, GET_LAST_MODIFIED, SET_LAST_MODIFIED,
    LIST_CHILDREN, SIGNING, URI, COMPRESS, VIRTUAL,
    RANDOM_ACCESS_READ, RANDOM_ACCESS_WRITE,
    // ... many more
}

// Query capabilities
FileSystemCapabilities caps = manager.getCapabilities("s3");
boolean canAppend = caps.hasCapability(Capability.APPEND_CONTENT);
```

**FileKit's Approach (Better):**
- Optional interfaces via type assertion
- No runtime capability enum needed
- Compile-time type safety

---

## 11. Recommended Features for FileKit

### Immediate Implementation

| Feature | Priority | API Design |
|---------|----------|------------|
| ReadOnlyFileSystem | HIGH | Decorator wrapping FileSystem |
| CachingFileSystem | HIGH | Decorator with Cache interface |
| FileSelector | MEDIUM | Interface for filtering during List |

### Future Roadmap

| Feature | Priority | API Design |
|---------|----------|------------|
| CompositeFileSystem | MEDIUM | Multiple filesystems, fallback strategy |
| VisibilityConverter | LOW | Interface for backend-specific mapping |
| BatchOperations | LOW | Optional interface for bulk ops |
| StreamControl | LOW | Offset/length options for reads |

---

## 12. API Design Principles

### 1. Interface Segregation
- Keep core FileSystem minimal
- Add capabilities via optional interfaces
- Type assertion for capability checking

### 2. Composition over Inheritance
- Decorators for orthogonal concerns
- Stack decorators: Encryption → Validation → Caching → ReadOnly
- Each decorator single responsibility

### 3. Configuration Flexibility
- Sensible defaults (zero-config works)
- Per-operation overrides via Options
- Backend-specific options through extension

### 4. Future Compatibility
- Interface methods should not change signature
- Add new optional interfaces for new capabilities
- Deprecate gracefully, never remove

### 5. Error Handling
- Wrap all errors with context (PathError)
- Distinguish retryable from permanent errors
- Provide actionable error messages

---

## 13. Implementation Notes

### ReadOnlyFileSystem

```go
// Future-proof interface
type ReadOnlyFileSystem struct {
    fs FileSystem
    // Options for future extension
    opts ReadOnlyOptions
}

type ReadOnlyOptions struct {
    // Future: allow specific write operations
    AllowCreateDir bool
    // Future: custom error handler
    OnWriteAttempt func(op, path string) error
}
```

### CachingFileSystem

```go
// Pluggable cache interface (future-proof)
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
}

// Caching decorator
type CachingFileSystem struct {
    fs    FileSystem
    cache Cache
    opts  CacheOptions
}

type CacheOptions struct {
    TTL               time.Duration
    CacheExists       bool  // Cache Exists() results
    CacheFileInfo     bool  // Cache FileInfo() results
    CacheList         bool  // Cache List() results
    InvalidateOnWrite bool  // Clear cache on write operations
    // Future: selective path caching
    PathFilter        func(path string) bool
}
```

### FileSelector

```go
// Future-proof selector interface
type FileSelector interface {
    // Match returns true if file should be included
    Match(file *File) bool
}

// Composable selectors
type AndSelector []FileSelector
type OrSelector []FileSelector
type NotSelector struct{ Selector FileSelector }

// Built-in selectors
type GlobSelector struct{ Pattern string }
type ExtensionSelector struct{ Extensions []string }
type SizeSelector struct{ Min, Max int64 }
type ModTimeSelector struct{ After, Before time.Time }
```

---

## References

- [Microsoft IFileProvider Documentation](https://learn.microsoft.com/en-us/aspnet/core/fundamentals/file-providers)
- [Apache Commons VFS Documentation](https://commons.apache.org/proper/commons-vfs/)
- [Flysystem Documentation](https://flysystem.thephpleague.com/docs/)
- [Flysystem GitHub Repository](https://github.com/thephpleague/flysystem)
