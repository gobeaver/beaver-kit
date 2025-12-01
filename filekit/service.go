package filekit

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/gobeaver/beaver-kit/config"
	"github.com/gobeaver/beaver-kit/filevalidator"
)

// Global instance
var (
	defaultFS   FileSystem
	defaultOnce sync.Once
	defaultErr  error
)

// Builder provides a way to create FileSystem instances with custom prefixes
type Builder struct {
	prefix string
}

// WithPrefix creates a new Builder with the specified prefix
func WithPrefix(prefix string) *Builder {
	return &Builder{prefix: prefix}
}

// Init initializes the global FileSystem instance using the builder's prefix
func (b *Builder) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return err
	}
	return Init(*cfg)
}

// New creates a new FileSystem instance using the builder's prefix
func (b *Builder) New() (FileSystem, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	return New(*cfg)
}

// Init initializes the global file system instance
func Init(configs ...Config) error {
	defaultOnce.Do(func() {
		var cfg *Config
		if len(configs) > 0 {
			cfg = &configs[0]
		} else {
			cfg, defaultErr = GetConfig()
			if defaultErr != nil {
				return
			}
		}

		defaultFS, defaultErr = New(*cfg)
	})

	return defaultErr
}

// New creates a new file system instance with given config
func New(cfg Config) (FileSystem, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create base filesystem using factory
	fs, err := CreateDriver(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// Wrap with encryption if enabled
	if cfg.EncryptionEnabled && cfg.EncryptionKey != "" {
		// Decode the key from base64
		key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("encryption key must be 32 bytes (got %d bytes)", len(key))
		}
		fs = NewEncryptedFS(fs, key)
	}

	// Wrap with validator if needed
	if validator := createValidator(cfg); validator != nil {
		fs = NewValidatedFileSystem(fs, validator)
	}

	// Apply default options if configured
	if shouldApplyDefaults(cfg) {
		fs = &defaultOptionsFS{
			fs:      fs,
			options: createDefaultOptions(cfg),
		}
	}

	return fs, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
	if cfg.Driver == "" {
		return errors.New("driver is required")
	}

	switch cfg.Driver {
	case "local":
		if cfg.LocalBasePath == "" {
			return errors.New("local base path is required for local driver")
		}
	case "s3":
		if cfg.S3Bucket == "" {
			return errors.New("S3 bucket is required for S3 driver")
		}
		// Access keys can be provided via IAM roles, so not always required
	default:
		return fmt.Errorf("unknown driver: %s", cfg.Driver)
	}

	return nil
}

// createValidator creates a file validator from config
func createValidator(cfg Config) filevalidator.Validator {
	// Start with default constraints
	constraints := filevalidator.DefaultConstraints()

	// Set size constraint
	if cfg.MaxFileSize > 0 {
		constraints.MaxFileSize = cfg.MaxFileSize
	}

	// Set MIME type constraints
	if cfg.AllowedMimeTypes != "" {
		types := strings.Split(cfg.AllowedMimeTypes, ",")
		for i := range types {
			types[i] = strings.TrimSpace(types[i])
		}
		constraints.AcceptedTypes = types
	}

	// Set extension constraints
	if cfg.AllowedExtensions != "" {
		exts := strings.Split(cfg.AllowedExtensions, ",")
		for i := range exts {
			exts[i] = strings.TrimSpace(exts[i])
		}
		constraints.AllowedExts = exts
	}

	if cfg.BlockedExtensions != "" {
		exts := strings.Split(cfg.BlockedExtensions, ",")
		for i := range exts {
			exts[i] = strings.TrimSpace(exts[i])
		}
		// Append to existing blocked extensions
		constraints.BlockedExts = append(constraints.BlockedExts, exts...)
	}

	// Note: We don't support BlockedMimeTypes in the current filevalidator API
	// but we could add this feature later if needed

	return filevalidator.New(constraints)
}

// shouldApplyDefaults checks if any default options are configured
func shouldApplyDefaults(cfg Config) bool {
	return cfg.DefaultVisibility != "" ||
		cfg.DefaultCacheControl != "" ||
		cfg.DefaultOverwrite ||
		cfg.DefaultPreserveFilename
}

// createDefaultOptions creates default options from config
func createDefaultOptions(cfg Config) []Option {
	var options []Option

	if cfg.DefaultVisibility != "" {
		visibility := Visibility(cfg.DefaultVisibility)
		options = append(options, WithVisibility(visibility))
	}

	if cfg.DefaultCacheControl != "" {
		options = append(options, WithCacheControl(cfg.DefaultCacheControl))
	}

	if cfg.DefaultOverwrite {
		options = append(options, WithOverwrite(true))
	}

	if cfg.DefaultPreserveFilename {
		options = append(options, WithPreserveFilename(true))
	}

	return options
}

// FS returns the global file system instance
func FS() FileSystem {
	if defaultFS == nil {
		_ = Init()
	}
	return defaultFS
}

// Default returns the global instance, initializing if needed with error handling
func Default() (FileSystem, error) {
	if defaultFS == nil {
		if err := Init(); err != nil {
			return nil, err
		}
	}
	return defaultFS, nil
}

// NewFromEnv creates instance from environment variables (convenience constructor)
func NewFromEnv() (FileSystem, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return New(*cfg)
}

// InitFromEnv initializes the global instance from environment variables (convenience method)
func InitFromEnv() error {
	return Init()
}

// Reset clears the global instance (for testing)
func Reset() {
	defaultFS = nil
	defaultOnce = sync.Once{}
	defaultErr = nil
}

// defaultOptionsFS wraps a FileSystem to apply default options
type defaultOptionsFS struct {
	fs      FileSystem
	options []Option
}

func (d *defaultOptionsFS) Upload(ctx context.Context, path string, content io.Reader, options ...Option) error {
	// Merge default options with provided options (provided options take precedence)
	allOptions := append(d.options, options...)
	return d.fs.Upload(ctx, path, content, allOptions...)
}

func (d *defaultOptionsFS) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return d.fs.Download(ctx, path)
}

func (d *defaultOptionsFS) Delete(ctx context.Context, path string) error {
	return d.fs.Delete(ctx, path)
}

func (d *defaultOptionsFS) Exists(ctx context.Context, path string) (bool, error) {
	return d.fs.Exists(ctx, path)
}

func (d *defaultOptionsFS) FileInfo(ctx context.Context, path string) (*File, error) {
	return d.fs.FileInfo(ctx, path)
}

func (d *defaultOptionsFS) List(ctx context.Context, prefix string) ([]File, error) {
	return d.fs.List(ctx, prefix)
}

func (d *defaultOptionsFS) CreateDir(ctx context.Context, path string) error {
	return d.fs.CreateDir(ctx, path)
}

func (d *defaultOptionsFS) DeleteDir(ctx context.Context, path string) error {
	return d.fs.DeleteDir(ctx, path)
}
