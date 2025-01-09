package filekit

import (
	"github.com/gobeaver/beaver-kit/config"
)

type Config struct {
	// Default driver to use (local, s3)
	Driver string `env:"BEAVER_FILEKIT_DRIVER,default:local"`

	// Local driver configuration
	LocalBasePath string `env:"BEAVER_FILEKIT_LOCAL_BASE_PATH,default:./storage"`

	// S3 driver configuration
	S3Region          string `env:"BEAVER_FILEKIT_S3_REGION,default:us-east-1"`
	S3Bucket          string `env:"BEAVER_FILEKIT_S3_BUCKET"`
	S3Prefix          string `env:"BEAVER_FILEKIT_S3_PREFIX"`
	S3Endpoint        string `env:"BEAVER_FILEKIT_S3_ENDPOINT"`
	S3AccessKeyID     string `env:"BEAVER_FILEKIT_S3_ACCESS_KEY_ID"`
	S3SecretAccessKey string `env:"BEAVER_FILEKIT_S3_SECRET_ACCESS_KEY"`
	S3ForcePathStyle  bool   `env:"BEAVER_FILEKIT_S3_FORCE_PATH_STYLE,default:false"`

	// Default upload options
	DefaultVisibility       string `env:"BEAVER_FILEKIT_DEFAULT_VISIBILITY,default:private"`
	DefaultCacheControl     string `env:"BEAVER_FILEKIT_DEFAULT_CACHE_CONTROL"`
	DefaultOverwrite        bool   `env:"BEAVER_FILEKIT_DEFAULT_OVERWRITE,default:false"`
	DefaultPreserveFilename bool   `env:"BEAVER_FILEKIT_DEFAULT_PRESERVE_FILENAME,default:false"`

	// File validation defaults
	MaxFileSize       int64  `env:"BEAVER_FILEKIT_MAX_FILE_SIZE,default:10485760"` // 10MB default
	AllowedMimeTypes  string `env:"BEAVER_FILEKIT_ALLOWED_MIME_TYPES"`             // comma-separated
	BlockedMimeTypes  string `env:"BEAVER_FILEKIT_BLOCKED_MIME_TYPES"`             // comma-separated
	AllowedExtensions string `env:"BEAVER_FILEKIT_ALLOWED_EXTENSIONS"`             // comma-separated
	BlockedExtensions string `env:"BEAVER_FILEKIT_BLOCKED_EXTENSIONS"`             // comma-separated

	// Encryption settings
	EncryptionEnabled   bool   `env:"BEAVER_FILEKIT_ENCRYPTION_ENABLED,default:false"`
	EncryptionAlgorithm string `env:"BEAVER_FILEKIT_ENCRYPTION_ALGORITHM,default:AES-256-GCM"`
	EncryptionKey       string `env:"BEAVER_FILEKIT_ENCRYPTION_KEY"`
}

// GetConfig returns config loaded from environment
func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
