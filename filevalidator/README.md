# FileValidator

A comprehensive, security-focused file validation package for Go applications. Part of the [Beaver Kit](https://github.com/gobeaver/beaver-kit) collection.

## Features

- **Comprehensive Validation** - File size, MIME type, filename, and extension validation
- **Content-Based Security** - Deep inspection to detect zip bombs, malicious images, and dangerous PDFs
- **Zero External Dependencies** - Pure Go implementation with no CGO requirements
- **Context Support** - Cancel long-running validations with context
- **Stream Validation** - Validate large files without loading them entirely into memory
- **Structured Errors** - Specific error types for precise error handling
- **Highly Configurable** - Builder pattern and predefined constraint sets

## Installation

```bash
go get github.com/gobeaver/beaver-kit/filevalidator
```

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/gobeaver/beaver-kit/filevalidator"
)

func main() {
    validator := filevalidator.NewDefault()

    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        if err := r.ParseMultipartForm(10 << 20); err != nil {
            http.Error(w, "Could not parse form", http.StatusBadRequest)
            return
        }

        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "Error retrieving file", http.StatusBadRequest)
            return
        }
        defer file.Close()

        if err := validator.Validate(header); err != nil {
            http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
            return
        }

        fmt.Fprintf(w, "File '%s' uploaded successfully!", header.Filename)
    })

    http.ListenAndServe(":8080", nil)
}
```

## Default Constraints

`NewDefault()` provides secure defaults suitable for most applications:

| Constraint | Default Value |
|------------|---------------|
| Max file size | 10 MB |
| Min file size | 1 byte |
| Max filename length | 255 characters |
| Require extension | Yes |
| Content validation | Enabled (non-blocking) |
| Blocked extensions | `.exe`, `.bat`, `.sh`, `.php`, `.js`, and 40+ others |
| Dangerous characters | `../`, `\`, `;`, `&`, `\|`, `>`, `<`, `$`, `` ` ``, `!`, `*` |

## Predefined Constraint Sets

```go
// Images only (jpg, png, gif, webp, svg, bmp, tiff)
validator := filevalidator.New(filevalidator.ImageOnlyConstraints())

// Documents only (pdf, doc, docx, txt, rtf)
validator := filevalidator.New(filevalidator.DocumentOnlyConstraints())

// Media only (audio + video, max 500MB)
validator := filevalidator.New(filevalidator.MediaOnlyConstraints())
```

## Custom Constraints with Builder

```go
import (
    "regexp"

    "github.com/gobeaver/beaver-kit/filevalidator"
)

constraints := filevalidator.NewConstraintsBuilder().
    WithMaxFileSize(5 * filevalidator.MB).
    WithMinFileSize(1 * filevalidator.KB).
    WithAcceptedTypes([]string{"image/jpeg", "image/png"}).
    WithAllowedExtensions([]string{".jpg", ".jpeg", ".png"}).
    WithBlockedExtensions([]string{".exe", ".php", ".js"}).
    WithMaxNameLength(100).
    WithRequireExtension(true).
    WithFileNameRegex(regexp.MustCompile(`^[a-zA-Z0-9_-]+\.[a-z]+$`)).
    WithStrictMIMETypeValidation(true).
    Build()

validator := filevalidator.New(constraints)
```

### Builder Methods

| Method | Description |
|--------|-------------|
| `WithMaxFileSize(int64)` | Maximum file size in bytes |
| `WithMinFileSize(int64)` | Minimum file size in bytes |
| `WithAcceptedTypes([]string)` | Allowed MIME types |
| `WithAllowedExtensions([]string)` | Allowed file extensions |
| `WithBlockedExtensions([]string)` | Blocked file extensions |
| `WithMaxNameLength(int)` | Maximum filename length |
| `WithFileNameRegex(*regexp.Regexp)` | Filename pattern validation |
| `WithDangerousChars([]string)` | Characters to block in filenames |
| `WithRequireExtension(bool)` | Require files to have extensions |
| `WithStrictMIMETypeValidation(bool)` | Require MIME and extension match |

### Size Constants

```go
filevalidator.KB  // 1024 bytes
filevalidator.MB  // 1024 KB
filevalidator.GB  // 1024 MB
```

## Validation Methods

```go
// Validate multipart.FileHeader (from HTTP uploads)
err := validator.Validate(header)

// Validate with context for cancellation
err := validator.ValidateWithContext(ctx, header)

// Validate from io.Reader (must be io.Seeker for MIME detection)
err := validator.ValidateReader(reader, "example.jpg", fileSize)

// Validate from byte slice
err := validator.ValidateBytes(content, "example.jpg")

// Validate local file path
err := filevalidator.ValidateLocalFile(validator, "/path/to/file.jpg")

// Stream validation for large files (memory-efficient)
err := filevalidator.StreamValidate(reader, "large-file.zip", validator, 8192)
```

## Error Handling

The package provides structured errors with specific types:

```go
import "github.com/gobeaver/beaver-kit/filevalidator"

err := validator.Validate(header)
if err != nil {
    switch {
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize):
        // File too large or too small
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeMIME):
        // Invalid or unaccepted MIME type
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeFileName):
        // Invalid filename (length, dangerous chars, pattern)
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeExtension):
        // Blocked or unallowed extension
    case filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeContent):
        // Content validation failed (zip bomb, malicious image, etc.)
    default:
        // Other error
    }
}
```

### Error Helper Functions

```go
// Check if error is a ValidationError
filevalidator.IsValidationError(err) bool

// Check error type
filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize) bool

// Get error type
filevalidator.GetErrorType(err) ValidationErrorType

// Get error message
filevalidator.GetErrorMessage(err) string
```

## Media Type Groups

Use predefined groups to accept categories of files:

```go
// Accept all image types
validator := filevalidator.New(filevalidator.Constraints{
    AcceptedTypes: []string{string(filevalidator.AllowAllImages)},
})

// Accept multiple groups
validator := filevalidator.New(filevalidator.Constraints{
    AcceptedTypes: []string{
        string(filevalidator.AllowAllImages),
        string(filevalidator.AllowAllDocuments),
    },
})
```

### Available Groups

| Group | MIME Types Included |
|-------|---------------------|
| `AllowAllImages` | jpeg, png, gif, webp, svg+xml, tiff, bmp, heic, heif |
| `AllowAllDocuments` | pdf, msword, docx, xlsx, pptx, txt, csv, rtf |
| `AllowAllAudio` | mpeg, wav, ogg, midi, aac, flac, mp4, webm, wma |
| `AllowAllVideo` | mp4, mpeg, webm, quicktime, avi, wmv, 3gpp, flv |
| `AllowAllText` | plain, html, css, csv, javascript, xml, markdown |
| `AllowAll` | All MIME types (`*/*`) |

### Custom MIME Mappings

```go
// Add custom extension mapping
filevalidator.AddCustomMediaTypeMapping(".custom", "application/x-custom")

// Add custom types to a group
filevalidator.AddCustomMediaTypeGroupMapping(
    filevalidator.AllowAllDocuments,
    []string{"application/x-custom-doc"},
)
```

## Content-Based Validation

Deep content inspection protects against sophisticated attacks. Content validators are automatically registered for high-risk formats.

### Archive Validation (Zip Bomb Protection)

```go
// Default constraints include archive validation
validator := filevalidator.NewDefault()

// Custom archive validator
archiveValidator := &filevalidator.ArchiveValidator{
    MaxCompressionRatio: 100.0,           // Maximum 100:1 ratio
    MaxFiles:            1000,            // Maximum files per archive
    MaxUncompressedSize: 100 * filevalidator.GB,
    MaxNestedArchives:   5,               // Nested archive limit
    MaxDepth:            10,              // Directory depth limit
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/zip", archiveValidator)
validator := filevalidator.New(constraints)
```

**Protected formats:** zip, jar, war, ear, rar, 7z, tar, gz, bz2, xz

**Detects:**
- Zip bombs (excessive compression ratios)
- Too many files in archive
- Nested archive attacks
- Directory traversal attempts (`../`, absolute paths)

### Image Validation

```go
// ImageOnlyConstraints() includes image validation
validator := filevalidator.New(filevalidator.ImageOnlyConstraints())

// Custom image validator
imageValidator := &filevalidator.ImageValidator{
    MaxWidth:       10000,
    MaxHeight:      10000,
    MaxPixels:      50000000,  // 50 megapixels
    MinWidth:       1,
    MinHeight:      1,
    AllowSVG:       true,
    MaxSVGSize:     5 * filevalidator.MB,
    ValidatePixels: true,
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("image/jpeg", imageValidator)
constraints.ContentValidatorRegistry.Register("image/png", imageValidator)
validator := filevalidator.New(constraints)
```

**Protected formats:** JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, SVG

**Detects:**
- Decompression bombs (excessive dimensions)
- Malicious SVG content (scripts, event handlers, iframes)
- Embedded scripts in image metadata
- Invalid file structure

### PDF Validation

```go
// DocumentOnlyConstraints() includes PDF validation
validator := filevalidator.New(filevalidator.DocumentOnlyConstraints())

// Custom PDF validator
pdfValidator := &filevalidator.PDFValidator{
    AllowJavaScript:    false,  // Block JavaScript
    AllowEmbeddedFiles: false,  // Block embedded files
    AllowForms:         true,   // Allow form fields
    AllowActions:       false,  // Block actions
    MaxSize:            50 * filevalidator.MB,
    ValidateStructure:  true,
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/pdf", pdfValidator)
validator := filevalidator.New(constraints)
```

**Detects:**
- JavaScript execution
- Embedded executable files
- Launch actions (always blocked)
- Suspicious URLs and executable references
- Excessive obfuscation

### Custom Content Validators

Implement `ContentValidator` for custom file types:

```go
type ContentValidator interface {
    ValidateContent(reader io.Reader, size int64) error
    SupportedMIMETypes() []string
}
```

```go
type CustomValidator struct{}

func (v *CustomValidator) ValidateContent(reader io.Reader, size int64) error {
    data, err := io.ReadAll(reader)
    if err != nil {
        return filevalidator.NewValidationError(
            filevalidator.ErrorTypeContent,
            "failed to read content",
        )
    }
    // Your validation logic
    return nil
}

func (v *CustomValidator) SupportedMIMETypes() []string {
    return []string{"application/x-custom"}
}

// Register
constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/x-custom", &CustomValidator{})
```

### Content Validation Modes

```go
constraints := filevalidator.DefaultConstraints()

// Enable content validation (default: true)
constraints.ContentValidationEnabled = true

// Make content validation mandatory (default: false)
// When false, content validation failures are warnings
// When true, content validation failures block the upload
constraints.RequireContentValidation = true
```

## Helper Functions

```go
// Format bytes as human-readable string
filevalidator.FormatSizeReadable(1536)     // "1.5 KB"
filevalidator.FormatSizeReadable(2097152)  // "2 MB"

// Detect content type from bytes
contentType := filevalidator.DetectContentType(data)

// Detect content type from file path
contentType, err := filevalidator.DetectContentTypeFromFile("/path/to/file")

// Check file type categories
filevalidator.IsImage("image/jpeg")           // true
filevalidator.IsDocument("application/pdf")  // true

// Check extension support
filevalidator.HasSupportedImageExtension("photo.jpg")     // true
filevalidator.HasSupportedDocumentExtension("report.pdf") // true

// Get MIME type for extension
filevalidator.MIMETypeForExtension(".jpg")  // "image/jpeg"
```

## Integration with FileKit

```go
package main

import (
    "bytes"
    "context"
    "io"
    "path/filepath"

    "github.com/gobeaver/beaver-kit/filekit"
    "github.com/gobeaver/beaver-kit/filevalidator"
)

type ValidatedUploader struct {
    uploader  filekit.Uploader
    validator filevalidator.Validator
}

func NewValidatedUploader(uploader filekit.Uploader) *ValidatedUploader {
    return &ValidatedUploader{
        uploader:  uploader,
        validator: filevalidator.NewDefault(),
    }
}

func (v *ValidatedUploader) Upload(ctx context.Context, path string, content io.Reader, options ...filekit.Option) error {
    var buf bytes.Buffer
    tee := io.TeeReader(content, &buf)

    if err := v.validator.ValidateReader(tee, filepath.Base(path), -1); err != nil {
        return err
    }

    return v.uploader.Upload(ctx, path, &buf, options...)
}
```

## Design Philosophy

FileValidator follows Go's philosophy of keeping libraries focused and composable. Several concerns are intentionally left to the caller:

### Concurrency

The validator performs synchronous validation. Handle concurrency at the application level:

```go
import (
    "mime/multipart"
    "sync"

    "github.com/gobeaver/beaver-kit/filevalidator"
)

// Concurrent validation of multiple files
func validateFiles(files []*multipart.FileHeader) []error {
    validator := filevalidator.NewDefault()
    errors := make([]error, len(files))
    var wg sync.WaitGroup

    for i, file := range files {
        wg.Add(1)
        go func(idx int, f *multipart.FileHeader) {
            defer wg.Done()
            errors[idx] = validator.Validate(f)
        }(i, file)
    }

    wg.Wait()
    return errors
}

// Async validation with channels
func validateAsync(file *multipart.FileHeader) <-chan error {
    result := make(chan error, 1)
    go func() {
        validator := filevalidator.NewDefault()
        result <- validator.Validate(file)
        close(result)
    }()
    return result
}
```

### Logging

The library returns structured errors but does not include logging. Integrate with your preferred logger:

```go
import (
    "log/slog"
    "mime/multipart"

    "github.com/gobeaver/beaver-kit/filevalidator"
    "go.uber.org/zap"
)

// With zap
func validateWithZap(logger *zap.Logger, validator filevalidator.Validator, file *multipart.FileHeader) error {
    err := validator.Validate(file)
    if err != nil {
        logger.Warn("file validation failed",
            zap.String("filename", file.Filename),
            zap.Int64("size", file.Size),
            zap.Error(err),
        )
    }
    return err
}

// With slog
func validateWithSlog(validator filevalidator.Validator, file *multipart.FileHeader) error {
    err := validator.Validate(file)
    if err != nil {
        slog.Warn("file validation failed",
            "filename", file.Filename,
            "size", file.Size,
            "error", err,
        )
    }
    return err
}
```

### Rate Limiting

Rate limiting belongs at the infrastructure or middleware layer:

```go
import (
    "net/http"

    "github.com/gobeaver/beaver-kit/filevalidator"
    "golang.org/x/time/rate"
)

func RateLimitedUploadHandler(validator filevalidator.Validator, limiter *rate.Limiter) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too many requests", http.StatusTooManyRequests)
            return
        }

        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "Bad request", http.StatusBadRequest)
            return
        }
        defer file.Close()

        if err := validator.Validate(header); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        // Process file...
    }
}
```

### Metrics

Wrap the validator for observability:

```go
import (
    "mime/multipart"
    "time"

    "github.com/gobeaver/beaver-kit/filevalidator"
    "github.com/prometheus/client_golang/prometheus"
)

type InstrumentedValidator struct {
    validator filevalidator.Validator
    total     prometheus.Counter
    errors    prometheus.Counter
    duration  prometheus.Histogram
}

func (v *InstrumentedValidator) Validate(file *multipart.FileHeader) error {
    start := time.Now()
    err := v.validator.Validate(file)

    v.total.Inc()
    v.duration.Observe(time.Since(start).Seconds())
    if err != nil {
        v.errors.Inc()
    }

    return err
}
```

### Responsibility Matrix

| Concern | Library | Caller |
|---------|:-------:|:------:|
| Validation logic | ✓ | |
| Structured errors | ✓ | |
| Content inspection | ✓ | |
| Concurrency | | ✓ |
| Logging | | ✓ |
| Rate limiting | | ✓ |
| Metrics | | ✓ |
| Retries | | ✓ |

This design:
- **Avoids dependency bloat** - No forced logging or metrics libraries
- **Stays composable** - Works with any framework or architecture
- **Follows Go idioms** - Libraries do one thing well
- **Enables flexibility** - Integrate with your existing infrastructure

## API Reference

### Types

```go
type Validator interface {
    Validate(file *multipart.FileHeader) error
    ValidateWithContext(ctx context.Context, file *multipart.FileHeader) error
    ValidateReader(reader io.Reader, filename string, size int64) error
    ValidateBytes(content []byte, filename string) error
    GetConstraints() Constraints
}

type Constraints struct {
    MaxFileSize              int64
    MinFileSize              int64
    AcceptedTypes            []string
    AllowedExts              []string
    BlockedExts              []string
    MaxNameLength            int
    FileNameRegex            *regexp.Regexp
    DangerousChars           []string
    RequireExtension         bool
    StrictMIMETypeValidation bool
    ContentValidationEnabled bool
    RequireContentValidation bool
    ContentValidatorRegistry *ContentValidatorRegistry
}

type ValidationError struct {
    Type    ValidationErrorType
    Message string
}

type ValidationErrorType string

const (
    ErrorTypeSize      ValidationErrorType = "size"
    ErrorTypeMIME      ValidationErrorType = "mime"
    ErrorTypeFileName  ValidationErrorType = "filename"
    ErrorTypeExtension ValidationErrorType = "extension"
    ErrorTypeContent   ValidationErrorType = "content"
)
```

### Constructors

```go
func New(constraints Constraints) *FileValidator
func NewDefault() *FileValidator
func NewConstraintsBuilder() *ConstraintsBuilder

func DefaultConstraints() Constraints
func ImageOnlyConstraints() Constraints
func DocumentOnlyConstraints() Constraints
func MediaOnlyConstraints() Constraints

func DefaultArchiveValidator() *ArchiveValidator
func DefaultImageValidator() *ImageValidator
func DefaultPDFValidator() *PDFValidator
```

## License

This package is licensed under the Apache 2.0 License. See the [LICENSE](../LICENSE) file for details.
