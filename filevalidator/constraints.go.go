package filevalidator

import (
	"regexp"
)

// Size constants for easier file size configuration
const (
	KB = int64(1024)
	MB = KB * 1024
	GB = MB * 1024
)

// Constraints defines the configuration for file validation
type Constraints struct {
	// MaxFileSize is the maximum allowed file size in bytes
	// Use the provided constants for readable configuration, e.g., 10 * MB for 10 megabytes
	MaxFileSize int64

	// MinFileSize is the minimum allowed file size in bytes
	// Use the provided constants for readable configuration, e.g., 1 * KB for 1 kilobyte
	MinFileSize int64

	// AcceptedTypes is a list of allowed MIME types (e.g., "image/jpeg", "application/pdf")
	// Special media type groups like "image/*" are also supported
	AcceptedTypes []string

	// AllowedExts is a list of allowed file extensions including the dot (e.g., ".jpg", ".pdf")
	// If empty, all extensions are allowed unless blocked by BlockedExts
	AllowedExts []string

	// BlockedExts is a list of blocked file extensions including the dot (e.g., ".exe", ".php")
	// These extensions will be blocked regardless of AllowedExts configuration
	BlockedExts []string

	// MaxNameLength is the maximum allowed length for filenames (including extension)
	// If set to 0, no length limit will be enforced
	MaxNameLength int

	// FileNameRegex is an optional regular expression pattern for validating filenames
	// If nil, no pattern matching will be performed
	FileNameRegex *regexp.Regexp

	// DangerousChars is a list of characters considered dangerous in filenames
	DangerousChars []string

	// RequireExtension enforces that files must have an extension
	RequireExtension bool

	// StrictMIMETypeValidation requires that both the MIME type and extension match
	StrictMIMETypeValidation bool

	// ContentValidationEnabled enables deep content validation
	ContentValidationEnabled bool

	// RequireContentValidation makes content validation mandatory
	RequireContentValidation bool

	// ContentValidatorRegistry holds content validators for different file types
	ContentValidatorRegistry *ContentValidatorRegistry
}

// DefaultConstraints creates a new set of constraints with sensible defaults
func DefaultConstraints() Constraints {
	registry := NewContentValidatorRegistry()
	// Register default content validators for high-risk formats
	archiveValidator := DefaultArchiveValidator()
	for _, mimeType := range archiveValidator.SupportedMIMETypes() {
		registry.Register(mimeType, archiveValidator)
	}

	return Constraints{
		MaxFileSize:              10 * MB,
		MinFileSize:              1, // 1 byte
		MaxNameLength:            255,
		DangerousChars:           []string{"../", "\\", ";", "&", "|", ">", "<", "$", "`", "!", "*"},
		BlockedExts:              []string{".exe", ".bat", ".cmd", ".sh", ".php", ".phtml", ".pl", ".cgi", ".386", ".dll", ".com", ".torrent", ".app", ".jar", ".pif", ".vb", ".vbs", ".vbe", ".js", ".jse", ".msc", ".ws", ".wsf", ".wsc", ".wsh", ".ps1", ".ps1xml", ".ps2", ".ps2xml", ".psc1", ".psc2", ".msh", ".msh1", ".msh2", ".mshxml", ".msh1xml", ".msh2xml", ".scf", ".lnk", ".inf", ".reg", ".docm", ".dotm", ".xlsm", ".xltm", ".xlam", ".pptm", ".potm", ".ppam", ".ppsm", ".sldm"},
		RequireExtension:         true,
		ContentValidationEnabled: true,
		RequireContentValidation: false,
		ContentValidatorRegistry: registry,
	}
}

// ImageOnlyConstraints creates constraints that only allow image files with sensible defaults
func ImageOnlyConstraints() Constraints {
	constraints := DefaultConstraints()
	constraints.AcceptedTypes = []string{"image/*"}
	constraints.AllowedExts = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif"}

	// Add image validator
	imageValidator := DefaultImageValidator()
	for _, mimeType := range imageValidator.SupportedMIMETypes() {
		constraints.ContentValidatorRegistry.Register(mimeType, imageValidator)
	}

	return constraints
}

// DocumentOnlyConstraints creates constraints that only allow document files with sensible defaults
func DocumentOnlyConstraints() Constraints {
	constraints := DefaultConstraints()
	constraints.AcceptedTypes = []string{"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/plain"}
	constraints.AllowedExts = []string{".pdf", ".doc", ".docx", ".txt", ".rtf"}

	// Add PDF validator
	pdfValidator := DefaultPDFValidator()
	for _, mimeType := range pdfValidator.SupportedMIMETypes() {
		constraints.ContentValidatorRegistry.Register(mimeType, pdfValidator)
	}

	return constraints
}

// MediaOnlyConstraints creates constraints that only allow media files with sensible defaults
func MediaOnlyConstraints() Constraints {
	constraints := DefaultConstraints()
	constraints.AcceptedTypes = []string{"audio/*", "video/*"}
	constraints.AllowedExts = []string{".mp3", ".wav", ".ogg", ".mp4", ".webm", ".avi", ".mov", ".wmv", ".flac", ".aac", ".m4a"}
	constraints.MaxFileSize = 500 * MB
	return constraints
}

// ConstraintsBuilder is a builder for creating validation constraints
type ConstraintsBuilder struct {
	constraints Constraints
}

// NewConstraintsBuilder creates a new constraints builder starting with default constraints
func NewConstraintsBuilder() *ConstraintsBuilder {
	return &ConstraintsBuilder{
		constraints: DefaultConstraints(),
	}
}

// WithMaxFileSize sets the maximum file size
func (b *ConstraintsBuilder) WithMaxFileSize(size int64) *ConstraintsBuilder {
	b.constraints.MaxFileSize = size
	return b
}

// WithMinFileSize sets the minimum file size
func (b *ConstraintsBuilder) WithMinFileSize(size int64) *ConstraintsBuilder {
	b.constraints.MinFileSize = size
	return b
}

// WithAcceptedTypes sets the accepted MIME types
func (b *ConstraintsBuilder) WithAcceptedTypes(types []string) *ConstraintsBuilder {
	b.constraints.AcceptedTypes = types
	return b
}

// WithAllowedExtensions sets the allowed file extensions
func (b *ConstraintsBuilder) WithAllowedExtensions(exts []string) *ConstraintsBuilder {
	b.constraints.AllowedExts = exts
	return b
}

// WithBlockedExtensions sets the blocked file extensions
func (b *ConstraintsBuilder) WithBlockedExtensions(exts []string) *ConstraintsBuilder {
	b.constraints.BlockedExts = exts
	return b
}

// WithMaxNameLength sets the maximum filename length
func (b *ConstraintsBuilder) WithMaxNameLength(length int) *ConstraintsBuilder {
	b.constraints.MaxNameLength = length
	return b
}

// WithFileNameRegex sets the filename regex pattern
func (b *ConstraintsBuilder) WithFileNameRegex(pattern *regexp.Regexp) *ConstraintsBuilder {
	b.constraints.FileNameRegex = pattern
	return b
}

// WithDangerousChars sets the dangerous characters list
func (b *ConstraintsBuilder) WithDangerousChars(chars []string) *ConstraintsBuilder {
	b.constraints.DangerousChars = chars
	return b
}

// WithRequireExtension sets whether an extension is required
func (b *ConstraintsBuilder) WithRequireExtension(require bool) *ConstraintsBuilder {
	b.constraints.RequireExtension = require
	return b
}

// WithStrictMIMETypeValidation sets whether strict MIME type validation is required
func (b *ConstraintsBuilder) WithStrictMIMETypeValidation(strict bool) *ConstraintsBuilder {
	b.constraints.StrictMIMETypeValidation = strict
	return b
}

// Build returns the built constraints
func (b *ConstraintsBuilder) Build() Constraints {
	return b.constraints
}
