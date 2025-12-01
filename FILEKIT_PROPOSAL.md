# FileKit v2.0: The Premier Go Filesystem Abstraction Library

## Executive Summary

This proposal outlines enhancements to make `beaver-kit/filekit` the **most comprehensive filesystem abstraction library for Go**, surpassing PHP's Flysystem, Go's Afero, Java's Apache Commons VFS, and ASP.NET Core's IFileProvider.

### Current State

| Package | Status | Highlights |
|---------|--------|------------|
| **filekit** | Production-ready | S3, Local, Encryption, Streaming, Progress tracking |
| **filevalidator** | Production-ready | 60+ formats, Zip bomb protection, Fluent API |

### Vision

**"One interface, any storage, with security built-in."**

---

## Competitive Analysis

### Feature Gap Analysis

| Feature | Flysystem | Afero | Commons VFS | IFileProvider | **FileKit (Current)** | **FileKit (Proposed)** |
|---------|-----------|-------|-------------|---------------|------------------------|------------------------|
| Unified API | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Local FS | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| In-Memory | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| S3 | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ |
| Azure Blob | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
| GCS | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
| FTP/SFTP | ✅ | ❌ | ✅ | ❌ | ❌ | ✅ |
| ZIP Archive | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ |
| Mount Manager | ✅ | ⚠️ | ✅ | ✅ | ❌ | ✅ |
| File Watching | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| **Built-in Validation** | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Encryption** | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Progress Tracking** | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| **Zip Bomb Protection** | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |

### Our Unique Differentiators (Already Implemented)

1. **Integrated File Validation** - 60+ format support with security checks
2. **AES-256-GCM Encryption** - Transparent encryption layer
3. **Zip Bomb Protection** - Decompression bomb detection
4. **Progress Tracking** - Built-in upload progress callbacks
5. **Chunked Uploads** - Multipart upload support for S3
6. **Pure Go** - Zero CGO dependencies

---

## Proposed Enhancements

### Phase 1: Storage Driver Expansion (High Priority)

#### 1.1 In-Memory Driver
```go
// Perfect for testing and caching
fs := filekit.NewMemoryFS()
fs.Upload(ctx, "test.txt", strings.NewReader("hello"))

// Configuration
type MemoryConfig struct {
    MaxSize      int64         // Max total storage (default: 100MB)
    MaxFiles     int           // Max file count (default: 10000)
    PersistPath  string        // Optional: persist to disk on shutdown
}
```

**Priority**: Critical (testing support)
**Effort**: 2-3 days

#### 1.2 Google Cloud Storage Driver
```go
fs, _ := filekit.NewGCSDriver(filekit.GCSConfig{
    Bucket:      "my-bucket",
    Credentials: "path/to/service-account.json", // or auto-detect
    Prefix:      "uploads/",
})
```

**Priority**: High (major cloud provider)
**Effort**: 3-4 days

#### 1.3 Azure Blob Storage Driver
```go
fs, _ := filekit.NewAzureDriver(filekit.AzureConfig{
    AccountName:   "myaccount",
    AccountKey:    "...",
    ContainerName: "uploads",
    // Or use connection string
    ConnectionString: "DefaultEndpointsProtocol=https;...",
})
```

**Priority**: High (enterprise demand)
**Effort**: 3-4 days

#### 1.4 SFTP Driver
```go
fs, _ := filekit.NewSFTPDriver(filekit.SFTPConfig{
    Host:       "sftp.example.com",
    Port:       22,
    Username:   "user",
    Password:   "pass", // or PrivateKey
    PrivateKey: "/path/to/key",
    BasePath:   "/uploads",
})
```

**Priority**: Medium (legacy system integration)
**Effort**: 3-4 days

#### 1.5 ZIP Archive Driver (Read/Write)
```go
// Read from ZIP
fs, _ := filekit.NewZIPDriver("/path/to/archive.zip")
files, _ := fs.List(ctx, "/")

// Write to ZIP
writer := filekit.NewZIPWriter("/path/to/new.zip")
writer.Upload(ctx, "file.txt", reader)
writer.Close()
```

**Priority**: Medium (archive operations)
**Effort**: 2-3 days

---

### Phase 2: Mount Manager & Composite FS

#### 2.1 Mount Manager
```go
mounts := filekit.NewMountManager()

// Mount different backends under virtual paths
mounts.Mount("/local", localDriver)
mounts.Mount("/cloud", s3Driver)
mounts.Mount("/cache", memoryDriver)

// Transparent access
mounts.Upload(ctx, "/local/file.txt", reader)
mounts.Download(ctx, "/cloud/image.png")

// Copy between mounts
mounts.Copy(ctx, "/local/file.txt", "/cloud/backup/file.txt")
```

**Features**:
- Virtual path namespacing
- Cross-driver operations (copy/move)
- Path resolution and validation
- Unmount support

**Priority**: High (multi-backend scenarios)
**Effort**: 4-5 days

#### 2.2 Fallback/Mirror Driver
```go
// Try primary, fallback to secondary
fs := filekit.NewFallbackDriver(primaryFS, fallbackFS)

// Or mirror writes to multiple backends
fs := filekit.NewMirrorDriver(primaryFS, backupFS, archiveFS)
```

**Priority**: Medium (redundancy)
**Effort**: 2-3 days

---

### Phase 3: File Watching & Events

#### 3.1 File Watcher Interface
```go
type FileWatcher interface {
    Watch(ctx context.Context, path string, callback WatchCallback) error
    WatchRecursive(ctx context.Context, path string, callback WatchCallback) error
    Unwatch(path string) error
}

type WatchEvent struct {
    Type      EventType // Created, Modified, Deleted, Renamed
    Path      string
    OldPath   string    // For rename events
    Timestamp time.Time
}

type WatchCallback func(event WatchEvent)
```

#### 3.2 Implementation Options

**Local Driver**: Use `fsnotify`
```go
localFS.Watch(ctx, "/uploads", func(e WatchEvent) {
    log.Printf("File %s: %s", e.Type, e.Path)
})
```

**S3/Cloud Drivers**: Polling or event integration
```go
// Polling mode (configurable interval)
s3FS.WatchWithPolling(ctx, "/uploads", 30*time.Second, callback)

// Or AWS S3 Event Notifications (SNS/SQS)
s3FS.WatchWithSNS(ctx, snsTopicARN, callback)
```

**Priority**: Medium (real-time applications)
**Effort**: 5-7 days

---

### Phase 4: Enhanced Features

#### 4.1 Atomic Operations
```go
// Transaction-like operations
tx := fs.BeginTransaction()
tx.Upload(ctx, "file1.txt", reader1)
tx.Upload(ctx, "file2.txt", reader2)
tx.Delete(ctx, "old-file.txt")
if err := tx.Commit(); err != nil {
    tx.Rollback() // Clean up partial uploads
}
```

**Priority**: Medium (data integrity)
**Effort**: 4-5 days

#### 4.2 Caching Layer
```go
// Add caching to any driver
cachedFS := filekit.WithCache(s3FS, filekit.CacheConfig{
    Backend:    memoryDriver, // or Redis driver
    TTL:        1 * time.Hour,
    MaxSize:    500 * 1024 * 1024, // 500MB
    CacheReads: true,
    CacheList:  true,
})
```

**Priority**: Medium (performance)
**Effort**: 3-4 days

#### 4.3 Rate Limiting & Throttling
```go
// Limit bandwidth per driver
throttledFS := filekit.WithRateLimit(s3FS, filekit.RateLimitConfig{
    MaxBytesPerSecond: 10 * 1024 * 1024, // 10MB/s
    MaxConcurrent:     5,
    MaxRequestsPerMin: 100,
})
```

**Priority**: Low (resource management)
**Effort**: 2-3 days

#### 4.4 Retry & Resilience
```go
// Automatic retry with exponential backoff
resilientFS := filekit.WithRetry(s3FS, filekit.RetryConfig{
    MaxRetries:     3,
    InitialDelay:   100 * time.Millisecond,
    MaxDelay:       5 * time.Second,
    RetryableErrors: []error{ErrTimeout, ErrNetworkError},
})
```

**Priority**: Medium (production reliability)
**Effort**: 2-3 days

---

### Phase 5: API Enhancements

#### 5.1 Extended File Operations
```go
type FileSystem interface {
    // Existing methods...

    // New operations
    Copy(ctx context.Context, src, dst string) error
    Move(ctx context.Context, src, dst string) error
    Rename(ctx context.Context, old, new string) error

    // Metadata operations
    SetMetadata(ctx context.Context, path string, meta map[string]string) error
    GetMetadata(ctx context.Context, path string) (map[string]string, error)

    // Permissions (where supported)
    SetVisibility(ctx context.Context, path string, visibility Visibility) error
    GetVisibility(ctx context.Context, path string) (Visibility, error)

    // Directory operations
    Walk(ctx context.Context, path string, fn WalkFunc) error
    ListRecursive(ctx context.Context, path string) ([]File, error)

    // Timestamps
    Touch(ctx context.Context, path string) error
    SetModTime(ctx context.Context, path string, t time.Time) error
}
```

**Priority**: High (API completeness)
**Effort**: 3-4 days

#### 5.2 Presigned URLs (Generalized)
```go
type URLGenerator interface {
    GenerateDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error)
    GenerateUploadURL(ctx context.Context, path string, expires time.Duration) (string, error)
}

// Works for S3, GCS, Azure, etc.
url, _ := fs.GenerateDownloadURL(ctx, "file.pdf", 1*time.Hour)
```

**Priority**: High (direct client uploads)
**Effort**: Already partial, 1-2 days to generalize

---

### Phase 6: FileValidator Integration Enhancements

#### 6.1 Missing Format Validators
```go
// Add full structure validation for:
- RAR archives (beyond magic bytes)
- 7-Zip archives
- HEIC/AVIF dimensions
- RTF document structure
- MIDI file structure
- HTML sanitization warnings
```

**Priority**: Low-Medium
**Effort**: 3-4 days

#### 6.2 Streaming Large File Validation
```go
// For non-seekable streams over 1GB
validator.ValidateStreaming(ctx, reader, filename, progressCallback)
```

**Priority**: Medium
**Effort**: 2-3 days

#### 6.3 Custom Validator SDK
```go
// Easy custom format support
type CustomValidator struct{}

func (v *CustomValidator) ValidateContent(reader io.ReaderAt, size int64) error {
    // Custom validation logic
}

func (v *CustomValidator) SupportedMIMETypes() []string {
    return []string{"application/x-custom"}
}

// Register
registry.Register(&CustomValidator{})
```

**Already Implemented** - Document better

---

### Phase 7: Developer Experience

#### 7.1 CLI Tool
```bash
# File operations
filekit cp local://./file.txt s3://bucket/file.txt
filekit mv s3://bucket/old.txt s3://bucket/new.txt
filekit ls s3://bucket/prefix/

# Validation
filekit validate ./uploads/*.pdf --strict

# Sync directories
filekit sync local://./uploads s3://bucket/uploads
```

**Priority**: Low (nice-to-have)
**Effort**: 5-7 days

#### 7.2 Improved Documentation
- Migration guides from Afero, Flysystem
- Performance benchmarks vs competitors
- Security best practices
- Video tutorials

**Priority**: Medium
**Effort**: Ongoing

---

## Implementation Roadmap

### Quarter 1: Foundation

| Week | Deliverable | Effort |
|------|-------------|--------|
| 1-2 | In-Memory Driver | 3 days |
| 2-3 | Google Cloud Storage Driver | 4 days |
| 3-4 | Azure Blob Storage Driver | 4 days |
| 4 | Extended File Operations (Copy, Move, Walk) | 3 days |

### Quarter 2: Advanced Features

| Week | Deliverable | Effort |
|------|-------------|--------|
| 5-6 | Mount Manager | 5 days |
| 6-7 | SFTP Driver | 4 days |
| 7-8 | ZIP Archive Driver | 3 days |
| 8-9 | Caching Layer | 4 days |

### Quarter 3: Enterprise Features

| Week | Deliverable | Effort |
|------|-------------|--------|
| 9-10 | File Watching (Local) | 4 days |
| 10-11 | Retry & Resilience | 3 days |
| 11-12 | Atomic Transactions | 5 days |
| 12 | Rate Limiting | 2 days |

### Quarter 4: Polish & Ecosystem

| Week | Deliverable | Effort |
|------|-------------|--------|
| 13-14 | CLI Tool | 6 days |
| 14-15 | Cloud File Watching (Polling) | 4 days |
| 15-16 | Documentation & Benchmarks | Ongoing |

---

## Technical Design Decisions

### 1. Driver Interface Extension

```go
// Base interface (required)
type Driver interface {
    Upload(ctx context.Context, path string, content io.Reader, opts ...Option) error
    Download(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    FileInfo(ctx context.Context, path string) (*File, error)
    List(ctx context.Context, prefix string) ([]File, error)
}

// Optional capabilities (check with type assertion)
type DirectoryOperator interface {
    CreateDir(ctx context.Context, path string) error
    DeleteDir(ctx context.Context, path string) error
}

type Copier interface {
    Copy(ctx context.Context, src, dst string) error
}

type Mover interface {
    Move(ctx context.Context, src, dst string) error
}

type URLGenerator interface {
    GenerateDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error)
    GenerateUploadURL(ctx context.Context, path string, expires time.Duration) (string, error)
}

type Watcher interface {
    Watch(ctx context.Context, path string, callback WatchCallback) error
    Unwatch(path string) error
}

type ChunkedUploader interface {
    InitiateUpload(ctx context.Context, path string, opts ...Option) (uploadID string, err error)
    UploadPart(ctx context.Context, uploadID string, partNumber int, content io.Reader) (partInfo PartInfo, err error)
    CompleteUpload(ctx context.Context, uploadID string, parts []PartInfo) error
    AbortUpload(ctx context.Context, uploadID string) error
}
```

### 2. Capability Discovery

```go
// Check driver capabilities at runtime
if copier, ok := fs.(Copier); ok {
    copier.Copy(ctx, src, dst)
} else {
    // Fallback: download + upload
    reader, _ := fs.Download(ctx, src)
    fs.Upload(ctx, dst, reader)
}

// Or use helper
filekit.Copy(ctx, fs, src, dst) // Auto-detects best method
```

### 3. Middleware Pattern (Current)

```
Request → Validation → Encryption → Default Options → Driver → Response
```

Keep this pattern, add more middleware:
- Caching
- Rate Limiting
- Retry
- Logging/Metrics

---

## Competitive Positioning

### Tagline
**"The only Go filesystem library with built-in security."**

### Key Messages

1. **Security First**: Built-in validation, encryption, zip bomb protection
2. **Cloud Native**: First-class S3, GCS, Azure support
3. **Pure Go**: No CGO, easy cross-compilation
4. **Battle Tested**: Production-ready with comprehensive tests
5. **Developer Friendly**: Fluent API, sensible defaults

### Comparison Table for README

```markdown
| Feature | FileKit | Afero | spf13/viper |
|---------|---------|-------|-------------|
| Cloud Storage | ✅ S3, GCS, Azure | ⚠️ Third-party | ❌ |
| Encryption | ✅ AES-256-GCM | ❌ | ❌ |
| File Validation | ✅ 60+ formats | ❌ | ❌ |
| Zip Bomb Protection | ✅ | ❌ | ❌ |
| Progress Tracking | ✅ | ❌ | ❌ |
| Pure Go | ✅ | ✅ | ✅ |
```

---

## Success Metrics

1. **GitHub Stars**: Target 1000+ within first year
2. **Weekly Downloads**: Target 5000+ on pkg.go.dev
3. **Production Users**: Target 50+ companies
4. **Community**: Active Discord/Slack, responsive issues

---

## Immediate Next Steps

### High-Priority Items (This Week)

1. **Fix S3 Multipart Upload** - Complete the TODO items for upload metadata storage
2. **Add In-Memory Driver** - Critical for testing
3. **Add Copy/Move Operations** - Basic API completeness

### Medium-Priority Items (This Month)

4. **Google Cloud Storage Driver**
5. **Azure Blob Storage Driver**
6. **Mount Manager**

### Documentation Updates

7. Update README with feature matrix
8. Add migration guide from Afero
9. Performance benchmarks

---

## Appendix: Current Package Analysis

### FileKit Strengths
- Clean interface design
- Encryption built-in
- Validation integration
- S3 multipart uploads
- Progress tracking
- Path traversal protection

### FileKit Gaps (To Address)
- Missing cloud drivers (GCS, Azure)
- No in-memory driver
- No file watching
- No mount manager
- Incomplete S3 multipart metadata

### FileValidator Strengths
- 60+ format support
- Memory-efficient header-only validation
- Fluent builder API
- Security-focused (zip bombs, XXE, path traversal)
- Zero dependencies

### FileValidator Gaps (Minor)
- Some formats only have magic byte detection
- No timeout mechanism for large files
- Limited audio/video codec verification

---

*This proposal positions beaver-kit/filekit as the most comprehensive, secure, and developer-friendly filesystem abstraction for Go.*
