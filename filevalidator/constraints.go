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
