package filekit

import (
	"time"

	"github.com/gobeaver/beaver-kit/filevalidator"
)

// Option represents a configuration option
type Option func(*Options)

// Options contains all possible options for file operations
type Options struct {
	// ContentType specifies the MIME type of the file
	ContentType string

	// Metadata contains additional metadata for the file
	Metadata map[string]string

	// Visibility defines the file visibility (public or private)
	Visibility Visibility

	// CacheControl sets the Cache-Control header for the file
	CacheControl string

	// Overwrite determines whether to overwrite existing files
	Overwrite bool

	// Encryption specifies encryption settings for the file
	Encryption *EncryptionOptions

	// Expires sets when the file should expire
	Expires *time.Time

	// PreserveFilename determines whether to keep the original filename
	PreserveFilename bool

	// Headers contains additional HTTP headers to set
	Headers map[string]string

	// SkipExistingCheck skips checking if a file already exists before upload
	SkipExistingCheck bool

	// ContentDisposition sets the Content-Disposition header
	ContentDisposition string

	// ACL sets specific access control list settings
	ACL string

	// Validator is an optional file validator to use before upload
	Validator filevalidator.Validator
}

// Visibility represents file visibility
type Visibility string

const (
	// Private means the file is only accessible by authenticated users
	Private Visibility = "private"

	// Public means the file is publicly accessible
	Public Visibility = "public"

	// Protected means the file is accessible with specific permissions
	Protected Visibility = "protected"
)

// EncryptionOptions contains options for file encryption
type EncryptionOptions struct {
	// Algorithm specifies the encryption algorithm to use
	Algorithm string

	// Key is the encryption key
	Key []byte

	// KeyID is an identifier for the encryption key (for key rotation)
	KeyID string
}

// WithContentType sets the content type of the file
func WithContentType(contentType string) Option {
	return func(o *Options) {
		o.ContentType = contentType
	}
}

// WithMetadata sets additional metadata for the file
func WithMetadata(metadata map[string]string) Option {
	return func(o *Options) {
		o.Metadata = metadata
	}
}

// WithVisibility sets the file visibility
func WithVisibility(visibility Visibility) Option {
	return func(o *Options) {
		o.Visibility = visibility
	}
}

// WithCacheControl sets the Cache-Control header
func WithCacheControl(cacheControl string) Option {
	return func(o *Options) {
		o.CacheControl = cacheControl
	}
}

// WithOverwrite enables or disables overwriting existing files
func WithOverwrite(overwrite bool) Option {
	return func(o *Options) {
		o.Overwrite = overwrite
	}
}

// WithEncryption enables encryption for the file
func WithEncryption(algorithm string, key []byte) Option {
	return func(o *Options) {
		o.Encryption = &EncryptionOptions{
			Algorithm: algorithm,
			Key:       key,
		}
	}
}

// WithEncryptionKeyID enables encryption with a specific key ID
func WithEncryptionKeyID(algorithm string, key []byte, keyID string) Option {
	return func(o *Options) {
		o.Encryption = &EncryptionOptions{
			Algorithm: algorithm,
			Key:       key,
			KeyID:     keyID,
		}
	}
}

// WithExpires sets when the file should expire
func WithExpires(expires time.Time) Option {
	return func(o *Options) {
		o.Expires = &expires
	}
}

// WithPreserveFilename enables or disables preserving the original filename
func WithPreserveFilename(preserve bool) Option {
	return func(o *Options) {
		o.PreserveFilename = preserve
	}
}

// WithHeaders sets additional HTTP headers
func WithHeaders(headers map[string]string) Option {
	return func(o *Options) {
		o.Headers = headers
	}
}

// WithContentDisposition sets the Content-Disposition header
func WithContentDisposition(disposition string) Option {
	return func(o *Options) {
		o.ContentDisposition = disposition
	}
}

// WithACL sets specific access control list settings
func WithACL(acl string) Option {
	return func(o *Options) {
		o.ACL = acl
	}
}

// WithValidator sets a file validator to use before upload
func WithValidator(validator filevalidator.Validator) Option {
	return func(o *Options) {
		o.Validator = validator
	}
}
