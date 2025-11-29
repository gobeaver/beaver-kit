# FileValidator

A comprehensive and flexible file validation package for Go applications. Part of the [Beaver Kit](https://github.com/gobeaver/beaver-kit) collection.

## Features

- **Comprehensive Validation**: File size, MIME type, filename, and extension validation
- **Highly Configurable**: Customize validation rules to fit your exact requirements
- **Media Type Groups**: Use predefined or custom groups for easy acceptance of file categories
- **Context Support**: Cancel long-running validations with context
- **Stream Validation**: Validate large files without loading them entirely into memory
- **Detailed Errors**: Specific error types for better error handling and user feedback
- **Helper Functions**: Format file sizes, detect MIME types, and more
- **No External Dependencies**: Pure Go implementation
- **Content-Based Validation**: Deep file content validation to prevent malicious uploads

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
    // Create a validator with default constraints
    validator := filevalidator.NewDefault()
    
    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        // Parse the multipart form
        if err := r.ParseMultipartForm(10 << 20); err != nil {
            http.Error(w, "Could not parse form", http.StatusBadRequest)
            return
        }
        
        // Get the uploaded file
        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "Error retrieving file", http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        // Validate the file
        if err := validator.Validate(header); err != nil {
            http.Error(w, "File validation error: "+err.Error(), http.StatusBadRequest)
            return
        }
        
        // File is valid, proceed with processing
        fmt.Fprintf(w, "File '%s' uploaded successfully!", header.Filename)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Configuring Validation Constraints

```go
// Using predefined constraints for images
validator := filevalidator.New(filevalidator.ImageOnlyConstraints())

// Or build your own constraints with the builder pattern
constraints := filevalidator.NewConstraintsBuilder().
    WithMaxFileSize(5 * filevalidator.MB).
    WithMinFileSize(1 * filevalidator.KB).
    WithAcceptedTypes([]string{"image/jpeg", "image/png"}).
    WithAllowedExtensions([]string{".jpg", ".jpeg", ".png"}).
    WithBlockedExtensions([]string{".exe", ".php", ".js"}).
    WithMaxNameLength(100).
    WithRequireExtension(true).
    Build()

customValidator := filevalidator.New(constraints)
```

## Validation Methods

The package provides multiple ways to validate files:

```go
// Validate a multipart.FileHeader (from form uploads)
err := validator.Validate(header)

// Validate with context for potential cancellation
err := validator.ValidateWithContext(ctx, header)

// Validate a file from an io.Reader
err := validator.ValidateReader(reader, "example.jpg", fileSize)

// Validate from a byte slice
err := validator.ValidateBytes(fileContent, "example.jpg")
```

## Error Handling

The package provides specific error types for better error handling:

```go
if err := validator.Validate(header); err != nil {
    if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize) {
        fmt.Println("File size error:", err)
    } else if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeMIME) {
        fmt.Println("MIME type error:", err)
    } else if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeFileName) {
        fmt.Println("Filename error:", err)
    } else if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeExtension) {
        fmt.Println("Extension error:", err)
    } else if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeContent) {
        fmt.Println("Content validation error:", err)
    } else {
        fmt.Println("Other error:", err)
    }
}
```

## Media Type Groups

The package provides predefined media type groups for easy configuration:

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

// Add custom media types to a group
filevalidator.AddCustomMediaTypeGroupMapping(
    filevalidator.AllowAllDocuments,
    []string{"application/x-custom-doc"},
)
```

## Custom Validation Rules

You can create custom validation rules by extending the Constraints struct:

```go
// Create a validator that only accepts PDF files under 5MB
constraints := filevalidator.Constraints{
    MaxFileSize: 5 * filevalidator.MB,
    AcceptedTypes: []string{"application/pdf"},
    AllowedExts: []string{".pdf"},
}
validator := filevalidator.New(constraints)
```

## Content-Based Validation

The package now includes deep content validation to protect against sophisticated attacks like zip bombs, malicious images, and dangerous PDFs:

### Archive Validation (Zip Bomb Protection)

```go
// The default constraints include archive validation for zip bomb protection
validator := filevalidator.NewDefault()

// Or configure your own archive validator
archiveValidator := &filevalidator.ArchiveValidator{
    MaxCompressionRatio: 100.0,  // Maximum 100:1 compression ratio
    MaxFiles:            1000,   // Maximum 1000 files per archive
    MaxUncompressedSize: 100 * filevalidator.GB,
    MaxNestedArchives:   5,
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/zip", archiveValidator)
```

### Image Validation

```go
// Using predefined image constraints includes content validation
validator := filevalidator.New(filevalidator.ImageOnlyConstraints())

// Or configure your own image validator
imageValidator := &filevalidator.ImageValidator{
    MaxWidth:       10000,
    MaxHeight:      10000,
    MaxPixels:      50000000, // 50 megapixels
    AllowSVG:       true,
    MaxSVGSize:     5 * filevalidator.MB,
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("image/jpeg", imageValidator)
constraints.ContentValidatorRegistry.Register("image/png", imageValidator)
```

### PDF Validation

```go
// Using predefined document constraints includes PDF validation
validator := filevalidator.New(filevalidator.DocumentOnlyConstraints())

// Or configure your own PDF validator
pdfValidator := &filevalidator.PDFValidator{
    AllowJavaScript:    false,
    AllowEmbeddedFiles: false,
    AllowForms:         true,
    AllowActions:       false,
    MaxSize:            50 * filevalidator.MB,
}

constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/pdf", pdfValidator)
```

### Custom Content Validators

You can implement your own content validators:

```go
type CustomValidator struct{}

func (v *CustomValidator) ValidateContent(reader io.Reader, size int64) error {
    // Implement your validation logic
    return nil
}

func (v *CustomValidator) SupportedMIMETypes() []string {
    return []string{"application/custom"}
}

// Register your custom validator
constraints := filevalidator.DefaultConstraints()
constraints.ContentValidatorRegistry.Register("application/custom", &CustomValidator{})
```

## Large File Validation

For large files, use stream validation to avoid memory issues:

```go
func validateLargeFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    validator := filevalidator.NewDefault()
    
    // Validate using the streaming validator
    return filevalidator.StreamValidate(
        file,
        filepath.Base(filePath),
        validator,
        8192, // 8KB buffer size
    )
}
```

## Integration with FileKit

The FileValidator package works seamlessly with the Beaver Kit FileKit package:

```go
package main

import (
    "context"
    "io"
    
    "github.com/gobeaver/beaver-kit/filekit"
    "github.com/gobeaver/beaver-kit/filevalidator"
)

// Create a file uploader with validation
type ValidatedUploader struct {
    uploader filekit.Uploader
    validator filevalidator.Validator
}

func NewValidatedUploader(uploader filekit.Uploader) *ValidatedUploader {
    return &ValidatedUploader{
        uploader: uploader,
        validator: filevalidator.NewDefault(),
    }
}

func (v *ValidatedUploader) Upload(ctx context.Context, path string, content io.Reader, options ...filekit.Option) error {
    // Create a temporary buffer to validate the content
    var buf bytes.Buffer
    tee := io.TeeReader(content, &buf)
    
    // Get the filename from the path
    filename := filepath.Base(path)
    
    // Validate the content
    if err := v.validator.ValidateReader(tee, filename, -1); err != nil {
        return err
    }
    
    // If validation passes, upload the content
    return v.uploader.Upload(ctx, path, &buf, options...)
}
```

## License

This package is licensed under the Apache 2.0 License. See the LICENSE file for details.
