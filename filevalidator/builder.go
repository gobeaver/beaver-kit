package filevalidator

import "regexp"

// Builder provides a fluent API for constructing validators
type Builder struct {
	constraints Constraints
}

// New creates a new validator builder with sensible defaults
func NewBuilder() *Builder {
	return &Builder{
		constraints: DefaultConstraints(),
	}
}

// Empty creates a builder with minimal defaults (no restrictions)
func Empty() *Builder {
	return &Builder{
		constraints: Constraints{
			ContentValidatorRegistry: NewContentValidatorRegistry(),
		},
	}
}

// --- Size constraints ---

// MaxSize sets the maximum allowed file size
func (b *Builder) MaxSize(size int64) *Builder {
	b.constraints.MaxFileSize = size
	return b
}

// MinSize sets the minimum required file size
func (b *Builder) MinSize(size int64) *Builder {
	b.constraints.MinFileSize = size
	return b
}

// SizeRange sets both minimum and maximum file size
func (b *Builder) SizeRange(minSize, maxSize int64) *Builder {
	b.constraints.MinFileSize = minSize
	b.constraints.MaxFileSize = maxSize
	return b
}

// --- MIME type constraints ---

// Accept adds accepted MIME types (e.g., "image/png", "image/*")
func (b *Builder) Accept(mimeTypes ...string) *Builder {
	b.constraints.AcceptedTypes = append(b.constraints.AcceptedTypes, mimeTypes...)
	return b
}

// AcceptImages allows all image types
func (b *Builder) AcceptImages() *Builder {
	return b.Accept(string(AllowAllImages))
}

// AcceptDocuments allows all document types
func (b *Builder) AcceptDocuments() *Builder {
	return b.Accept(string(AllowAllDocuments))
}

// AcceptAudio allows all audio types
func (b *Builder) AcceptAudio() *Builder {
	return b.Accept(string(AllowAllAudio))
}

// AcceptVideo allows all video types
func (b *Builder) AcceptVideo() *Builder {
	return b.Accept(string(AllowAllVideo))
}

// AcceptMedia allows all audio and video types
func (b *Builder) AcceptMedia() *Builder {
	return b.AcceptAudio().AcceptVideo()
}

// AcceptAll allows all file types
func (b *Builder) AcceptAll() *Builder {
	return b.Accept(string(AllowAll))
}

// StrictMIME enables strict MIME type validation (extension must match content)
func (b *Builder) StrictMIME() *Builder {
	b.constraints.StrictMIMETypeValidation = true
	return b
}

// --- Extension constraints ---

// Extensions sets the allowed file extensions (e.g., ".jpg", ".png")
func (b *Builder) Extensions(exts ...string) *Builder {
	b.constraints.AllowedExts = append(b.constraints.AllowedExts, exts...)
	return b
}

// BlockExtensions adds extensions to the blocklist
func (b *Builder) BlockExtensions(exts ...string) *Builder {
	b.constraints.BlockedExts = append(b.constraints.BlockedExts, exts...)
	return b
}

// RequireExtension requires files to have an extension
func (b *Builder) RequireExtension() *Builder {
	b.constraints.RequireExtension = true
	return b
}

// AllowNoExtension allows files without extensions
func (b *Builder) AllowNoExtension() *Builder {
	b.constraints.RequireExtension = false
	return b
}

// --- Filename constraints ---

// MaxNameLength sets the maximum filename length
func (b *Builder) MaxNameLength(length int) *Builder {
	b.constraints.MaxNameLength = length
	return b
}

// FileNamePattern sets a regex pattern for valid filenames
func (b *Builder) FileNamePattern(pattern *regexp.Regexp) *Builder {
	b.constraints.FileNameRegex = pattern
	return b
}

// FileNamePatternString sets a regex pattern from a string
func (b *Builder) FileNamePatternString(pattern string) *Builder {
	b.constraints.FileNameRegex = regexp.MustCompile(pattern)
	return b
}

// DangerousChars sets characters to block in filenames
func (b *Builder) DangerousChars(chars ...string) *Builder {
	b.constraints.DangerousChars = chars
	return b
}

// --- Content validation ---

// WithContentValidation enables content validation
func (b *Builder) WithContentValidation() *Builder {
	b.constraints.ContentValidationEnabled = true
	return b
}

// WithoutContentValidation disables content validation
func (b *Builder) WithoutContentValidation() *Builder {
	b.constraints.ContentValidationEnabled = false
	return b
}

// RequireContentValidation makes content validation mandatory
func (b *Builder) RequireContentValidation() *Builder {
	b.constraints.ContentValidationEnabled = true
	b.constraints.RequireContentValidation = true
	return b
}

// WithRegistry sets a custom content validator registry
func (b *Builder) WithRegistry(registry *ContentValidatorRegistry) *Builder {
	b.constraints.ContentValidatorRegistry = registry
	return b
}

// WithDefaultRegistry uses the default registry with all validators
func (b *Builder) WithDefaultRegistry() *Builder {
	b.constraints.ContentValidatorRegistry = DefaultRegistry()
	return b
}

// WithMinimalRegistry uses a minimal registry (ZIP, Image, PDF only)
func (b *Builder) WithMinimalRegistry() *Builder {
	b.constraints.ContentValidatorRegistry = MinimalRegistry()
	return b
}

// --- Build ---

// Build creates the validator with the configured constraints
func (b *Builder) Build() *FileValidator {
	return New(b.constraints)
}

// Constraints returns the current constraints (for inspection)
func (b *Builder) Constraints() Constraints {
	return b.constraints
}

// --- Presets ---

// ForImages creates a builder pre-configured for image uploads
func ForImages() *Builder {
	return NewBuilder().
		AcceptImages().
		Extensions(".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif").
		MaxSize(10 * MB).
		WithRegistry(ImageOnlyRegistry()).
		WithContentValidation()
}

// ForDocuments creates a builder pre-configured for document uploads
func ForDocuments() *Builder {
	return NewBuilder().
		AcceptDocuments().
		Extensions(".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv").
		MaxSize(50 * MB).
		WithRegistry(DocumentOnlyRegistry()).
		WithContentValidation()
}

// ForMedia creates a builder pre-configured for audio/video uploads
func ForMedia() *Builder {
	return NewBuilder().
		AcceptMedia().
		Extensions(".mp3", ".wav", ".ogg", ".flac", ".aac", ".mp4", ".webm", ".avi", ".mov", ".mkv").
		MaxSize(500 * MB).
		WithRegistry(MediaOnlyRegistry()).
		WithContentValidation()
}

// ForArchives creates a builder pre-configured for archive uploads
func ForArchives() *Builder {
	registry := NewContentValidatorRegistry()
	archiveValidator := DefaultArchiveValidator()
	for _, mime := range archiveValidator.SupportedMIMETypes() {
		registry.Register(mime, archiveValidator)
	}
	tarValidator := DefaultTarValidator()
	for _, mime := range tarValidator.SupportedMIMETypes() {
		registry.Register(mime, tarValidator)
	}
	gzipValidator := DefaultGzipValidator()
	for _, mime := range gzipValidator.SupportedMIMETypes() {
		registry.Register(mime, gzipValidator)
	}

	return NewBuilder().
		Accept("application/zip", "application/gzip", "application/x-tar", "application/x-gtar").
		Extensions(".zip", ".tar", ".gz", ".tgz", ".tar.gz").
		MaxSize(1 * GB).
		WithRegistry(registry).
		WithContentValidation()
}

// ForWeb creates a builder for typical web uploads (images + documents)
func ForWeb() *Builder {
	return NewBuilder().
		AcceptImages().
		AcceptDocuments().
		Extensions(
			// Images
			".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg",
			// Documents
			".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt",
		).
		MaxSize(25 * MB).
		WithDefaultRegistry().
		WithContentValidation()
}

// Strict creates a builder with strict validation settings
func Strict() *Builder {
	return NewBuilder().
		StrictMIME().
		RequireExtension().
		RequireContentValidation().
		WithDefaultRegistry()
}
