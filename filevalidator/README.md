# FileValidator

A high-performance, memory-efficient file validation package for Go. Validates file types without loading entire files into memory.

## Features

- **Memory Efficient** - Header-only validation (500MB file uses ~2KB RAM)
- **60+ File Formats** - Images, documents, archives, audio, video, text
- **Zero Dependencies** - Pure Go, no CGO
- **Fluent API** - Clean, chainable configuration
- **Content Validation** - Zip bomb protection, path traversal detection
- **Magic Bytes Detection** - Accurate MIME detection beyond `http.DetectContentType`

## Installation

```bash
go get github.com/gobeaver/beaver-kit/filevalidator
```

## Quick Start

```go
// One-liner with preset
validator := filevalidator.ForImages().Build()

// Or customize
validator := filevalidator.NewBuilder().
    MaxSize(10 * filevalidator.MB).
    Accept("image/*").
    Extensions(".jpg", ".png", ".webp").
    WithContentValidation().
    Build()

// Validate
err := validator.Validate(fileHeader)
```

## Fluent Builder API

### Basic Usage

```go
validator := filevalidator.NewBuilder().
    MaxSize(10 * filevalidator.MB).
    MinSize(1 * filevalidator.KB).
    Accept("image/png", "image/jpeg").
    Extensions(".png", ".jpg").
    BlockExtensions(".exe", ".php").
    StrictMIME().
    RequireExtension().
    WithContentValidation().
    Build()
```

### Presets

```go
// Images (jpg, png, gif, webp, svg, bmp, tiff) - 10MB max
validator := filevalidator.ForImages().Build()

// Documents (pdf, doc, docx, xls, xlsx, ppt, pptx, txt, csv) - 50MB max
validator := filevalidator.ForDocuments().Build()

// Audio/Video (mp3, wav, mp4, webm, avi, mov, mkv, etc.) - 500MB max
validator := filevalidator.ForMedia().Build()

// Archives (zip, tar, gz, tgz) - 1GB max
validator := filevalidator.ForArchives().Build()

// Web uploads (images + documents) - 25MB max
validator := filevalidator.ForWeb().Build()

// Strict mode (strict MIME, required extension, required content validation)
validator := filevalidator.Strict().Accept("image/*").Build()
```

### Customize Presets

```go
validator := filevalidator.ForImages().
    MaxSize(5 * filevalidator.MB).  // Override default 10MB
    Extensions(".png", ".jpg").      // More restrictive
    Build()
```

### All Builder Methods

| Category | Methods |
|----------|---------|
| **Size** | `MaxSize(int64)`, `MinSize(int64)`, `SizeRange(min, max int64)` |
| **MIME** | `Accept(...string)`, `AcceptImages()`, `AcceptDocuments()`, `AcceptAudio()`, `AcceptVideo()`, `AcceptMedia()`, `AcceptAll()`, `StrictMIME()` |
| **Extensions** | `Extensions(...string)`, `BlockExtensions(...string)`, `RequireExtension()`, `AllowNoExtension()` |
| **Filename** | `MaxNameLength(int)`, `FileNamePattern(*regexp.Regexp)`, `FileNamePatternString(string)`, `DangerousChars(...string)` |
| **Content** | `WithContentValidation()`, `WithoutContentValidation()`, `RequireContentValidation()`, `WithRegistry(*ContentValidatorRegistry)`, `WithDefaultRegistry()`, `WithMinimalRegistry()` |

## Validation Methods

```go
// From multipart.FileHeader (HTTP uploads)
err := validator.Validate(header)

// With context (cancellable)
err := validator.ValidateWithContext(ctx, header)

// From io.Reader
err := validator.ValidateReader(reader, "file.jpg", size)

// From bytes
err := validator.ValidateBytes(data, "file.jpg")

// Local file
err := filevalidator.ValidateLocalFile(validator, "/path/to/file.jpg")
```

## Validation Result (Detailed)

For detailed validation information:

```go
builder := filevalidator.NewResultBuilder("photo.png", 1024)
result := builder.
    SetDetectedMIME("image/png").
    AddCheck("size", true, "size within limits").
    AddCheck("mime", true, "valid image type").
    Build()

fmt.Println(result.Summary())
// ✓ photo.png (image/png, 1 KB) validated in 50µs

fmt.Println(result.Valid)        // true
fmt.Println(result.DetectedMIME) // image/png
fmt.Println(result.Duration)     // 50µs
```

## Content Validators

All validators read only headers - never full file content.

### Supported Formats

| Category | Formats | Validation |
|----------|---------|------------|
| **Archives** | ZIP, TAR, GZIP, TAR.GZ | Zip bomb, path traversal, nested archives |
| **Images** | JPEG, PNG, GIF, WebP, BMP, TIFF, SVG, ICO | Dimensions, decompression bombs |
| **Documents** | PDF | Header/trailer structure |
| **Office** | DOCX, XLSX, PPTX | ZIP structure, macro detection |
| **Video** | MP4, WebM, MKV, AVI, MOV, FLV | Magic bytes validation |
| **Audio** | MP3, WAV, OGG, FLAC, AAC, M4A | Magic bytes validation |
| **Text** | JSON, XML, CSV | Structure, depth limits, XXE protection |

### Archive Validation (Zip Bomb Protection)

```go
validator := filevalidator.DefaultArchiveValidator()
// MaxCompressionRatio: 100:1
// MaxFiles: 1000
// MaxUncompressedSize: 1GB
// MaxNestedArchives: 3
```

Detects:
- Zip bombs (high compression ratios)
- Nested archive attacks
- Path traversal (`../`, absolute paths)
- File count bombs

### Image Validation

```go
validator := filevalidator.DefaultImageValidator()
// MaxWidth: 10000
// MaxHeight: 10000
// MaxPixels: 50 megapixels
// MaxSVGSize: 5MB
```

Uses `image.DecodeConfig()` - reads only header bytes.

### Office Document Validation

```go
validator := filevalidator.DefaultOfficeValidator()
// AllowMacros: false (blocks .docm, .xlsm, .pptm)
```

Validates ZIP structure and required Office files.

### XML Validation (XXE Protection)

```go
validator := filevalidator.DefaultXMLValidator()
// AllowDTD: false (blocks XXE attacks)
// MaxDepth: 100
```

### Custom Content Validator

```go
type MyValidator struct{}

func (v *MyValidator) ValidateContent(reader io.Reader, size int64) error {
    // Read only what you need
    header := make([]byte, 512)
    io.ReadFull(reader, header)

    // Validate...
    return nil
}

func (v *MyValidator) SupportedMIMETypes() []string {
    return []string{"application/x-custom"}
}

// Register
registry := filevalidator.DefaultRegistry()
registry.Register("application/x-custom", &MyValidator{})
```

## MIME Detection

Enhanced detection using magic bytes (60+ signatures):

```go
// From reader
mime, err := filevalidator.DetectMIME(reader)

// From bytes
mime := filevalidator.DetectMIMEFromBytes(data)

// Helpers
filevalidator.IsBinaryMIME("image/png")       // true
filevalidator.IsExecutableMIME("application/x-msdownload") // true
filevalidator.GetMIMECategory("video/mp4")    // "video"
```

Detects: Images, video, audio, archives, documents, executables, fonts, and more.

## Registry

### Default (All Validators)

```go
registry := filevalidator.DefaultRegistry()
// ~4.6µs to create, 38ns per lookup, ~1.5KB memory
```

### Specialized Registries

```go
registry := filevalidator.MinimalRegistry()      // ZIP, Image, PDF only
registry := filevalidator.ImageOnlyRegistry()    // Images only
registry := filevalidator.DocumentOnlyRegistry() // PDF + Office only
registry := filevalidator.MediaOnlyRegistry()    // Audio + Video only
```

### Registry Operations

```go
registry.HasValidator("image/png")     // true
registry.Count()                       // number of validators
registry.RegisteredMIMETypes()         // []string of all types
registry.Unregister("image/png")       // remove validator
registry.Clone()                       // copy registry
```

## Error Handling

```go
err := validator.Validate(header)
if err != nil {
    switch {
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize):
        // File too large or too small
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeMIME):
        // Invalid MIME type
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeExtension):
        // Blocked extension
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeFileName):
        // Invalid filename
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeContent):
        // Content validation failed
    }
}
```

## Size Constants

```go
filevalidator.KB  // 1024
filevalidator.MB  // 1024 * 1024
filevalidator.GB  // 1024 * 1024 * 1024
```

## HTTP Example

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Auth check (instant)
    // 2. Rate limit (instant)

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Bad request", 400)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "No file", 400)
        return
    }
    defer file.Close()

    // 3. Validate (fast - header only)
    validator := filevalidator.ForImages().Build()
    if err := validator.Validate(header); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }

    // 4. (Optional) Malware scan with ClamAV
    // 5. Save file

    w.WriteHeader(200)
}
```

## Performance

| Operation | Time | Memory |
|-----------|------|--------|
| Create default registry | 4.6µs | 10KB |
| Validator lookup | 38ns | 0 allocs |
| Image validation (header) | ~50µs | ~2KB |
| ZIP validation (central dir) | ~100µs | ~2KB |

Large files (500MB+) use the same ~2KB memory - only headers are read.

## Design Philosophy

This package does **type validation**, not security scanning.

```
┌─────────────────────────────────────────┐
│ 1. Auth + Rate Limit        (instant)   │
│ 2. Size + Extension Check   (instant)   │
│ 3. Type Validation (this)   (fast)      │  ← You are here
│ 4. Malware Scan (ClamAV)    (slow)      │
│ 5. Storage                  (I/O)       │
└─────────────────────────────────────────┘
```

**What this package does:**
- ✅ Validates file types
- ✅ Validates file structure
- ✅ Detects zip bombs
- ✅ Blocks path traversal

**What this package doesn't do:**
- ❌ Malware detection (use ClamAV)
- ❌ Virus scanning
- ❌ Deep content analysis

## License

Apache 2.0 License. See [LICENSE](../LICENSE) for details.
