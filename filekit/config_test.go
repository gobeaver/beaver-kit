package filekit

import (
	"os"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			want: Config{
				Driver:              "local",
				LocalBasePath:       "./storage",
				S3Region:            "us-east-1",
				DefaultVisibility:   "private",
				MaxFileSize:         10485760,
				EncryptionAlgorithm: "AES-256-GCM",
			},
		},
		{
			name: "s3 configuration",
			envVars: map[string]string{
				"BEAVER_FILEKIT_DRIVER":               "s3",
				"BEAVER_FILEKIT_S3_BUCKET":            "test-bucket",
				"BEAVER_FILEKIT_S3_PREFIX":            "test-prefix/",
				"BEAVER_FILEKIT_S3_REGION":            "us-west-2",
				"BEAVER_FILEKIT_S3_ACCESS_KEY_ID":     "test-key",
				"BEAVER_FILEKIT_S3_SECRET_ACCESS_KEY": "test-secret",
				"BEAVER_FILEKIT_S3_ENDPOINT":          "http://localhost:9000",
				"BEAVER_FILEKIT_S3_FORCE_PATH_STYLE":  "true",
			},
			want: Config{
				Driver:              "s3",
				LocalBasePath:       "./storage",
				S3Bucket:            "test-bucket",
				S3Prefix:            "test-prefix/",
				S3Region:            "us-west-2",
				S3AccessKeyID:       "test-key",
				S3SecretAccessKey:   "test-secret",
				S3Endpoint:          "http://localhost:9000",
				S3ForcePathStyle:    true,
				DefaultVisibility:   "private",
				MaxFileSize:         10485760,
				EncryptionAlgorithm: "AES-256-GCM",
			},
		},
		{
			name: "local configuration with options",
			envVars: map[string]string{
				"BEAVER_FILEKIT_DRIVER":                    "local",
				"BEAVER_FILEKIT_LOCAL_BASE_PATH":           "/custom/path",
				"BEAVER_FILEKIT_DEFAULT_VISIBILITY":        "public",
				"BEAVER_FILEKIT_DEFAULT_CACHE_CONTROL":     "max-age=7200",
				"BEAVER_FILEKIT_DEFAULT_OVERWRITE":         "true",
				"BEAVER_FILEKIT_DEFAULT_PRESERVE_FILENAME": "true",
			},
			want: Config{
				Driver:                  "local",
				LocalBasePath:           "/custom/path",
				S3Region:                "us-east-1",
				DefaultVisibility:       "public",
				DefaultCacheControl:     "max-age=7200",
				DefaultOverwrite:        true,
				DefaultPreserveFilename: true,
				MaxFileSize:             10485760,
				EncryptionAlgorithm:     "AES-256-GCM",
			},
		},
		{
			name: "file validation configuration",
			envVars: map[string]string{
				"BEAVER_FILEKIT_MAX_FILE_SIZE":      "5242880",
				"BEAVER_FILEKIT_ALLOWED_MIME_TYPES": "image/jpeg,image/png",
				"BEAVER_FILEKIT_BLOCKED_MIME_TYPES": "application/x-executable",
				"BEAVER_FILEKIT_ALLOWED_EXTENSIONS": ".jpg,.png",
				"BEAVER_FILEKIT_BLOCKED_EXTENSIONS": ".exe,.bat",
			},
			want: Config{
				Driver:              "local",
				LocalBasePath:       "./storage",
				S3Region:            "us-east-1",
				DefaultVisibility:   "private",
				MaxFileSize:         5242880,
				AllowedMimeTypes:    "image/jpeg,image/png",
				BlockedMimeTypes:    "application/x-executable",
				AllowedExtensions:   ".jpg,.png",
				BlockedExtensions:   ".exe,.bat",
				EncryptionAlgorithm: "AES-256-GCM",
			},
		},
		{
			name: "encryption configuration",
			envVars: map[string]string{
				"BEAVER_FILEKIT_ENCRYPTION_ENABLED":   "true",
				"BEAVER_FILEKIT_ENCRYPTION_ALGORITHM": "AES-128-GCM",
				"BEAVER_FILEKIT_ENCRYPTION_KEY":       "dGVzdC1rZXktdGVzdC1rZXktdGVzdC1rZXk=", // base64 encoded "test-key-test-key-test-key"
			},
			want: Config{
				Driver:              "local",
				LocalBasePath:       "./storage",
				S3Region:            "us-east-1",
				DefaultVisibility:   "private",
				MaxFileSize:         10485760,
				EncryptionEnabled:   true,
				EncryptionAlgorithm: "AES-128-GCM",
				EncryptionKey:       "dGVzdC1rZXktdGVzdC1rZXktdGVzdC1rZXk=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg, err := GetConfig()
			if err != nil {
				t.Fatalf("GetConfig() error = %v", err)
			}

			// Compare configs
			if cfg.Driver != tt.want.Driver {
				t.Errorf("Driver = %v, want %v", cfg.Driver, tt.want.Driver)
			}
			if cfg.LocalBasePath != tt.want.LocalBasePath {
				t.Errorf("LocalBasePath = %v, want %v", cfg.LocalBasePath, tt.want.LocalBasePath)
			}
			if cfg.S3Bucket != tt.want.S3Bucket {
				t.Errorf("S3Bucket = %v, want %v", cfg.S3Bucket, tt.want.S3Bucket)
			}
			if cfg.S3Prefix != tt.want.S3Prefix {
				t.Errorf("S3Prefix = %v, want %v", cfg.S3Prefix, tt.want.S3Prefix)
			}
			if cfg.S3Region != tt.want.S3Region {
				t.Errorf("S3Region = %v, want %v", cfg.S3Region, tt.want.S3Region)
			}
			if cfg.S3AccessKeyID != tt.want.S3AccessKeyID {
				t.Errorf("S3AccessKeyID = %v, want %v", cfg.S3AccessKeyID, tt.want.S3AccessKeyID)
			}
			if cfg.S3SecretAccessKey != tt.want.S3SecretAccessKey {
				t.Errorf("S3SecretAccessKey = %v, want %v", cfg.S3SecretAccessKey, tt.want.S3SecretAccessKey)
			}
			if cfg.S3Endpoint != tt.want.S3Endpoint {
				t.Errorf("S3Endpoint = %v, want %v", cfg.S3Endpoint, tt.want.S3Endpoint)
			}
			if cfg.S3ForcePathStyle != tt.want.S3ForcePathStyle {
				t.Errorf("S3ForcePathStyle = %v, want %v", cfg.S3ForcePathStyle, tt.want.S3ForcePathStyle)
			}
			if cfg.DefaultVisibility != tt.want.DefaultVisibility {
				t.Errorf("DefaultVisibility = %v, want %v", cfg.DefaultVisibility, tt.want.DefaultVisibility)
			}
			if cfg.DefaultCacheControl != tt.want.DefaultCacheControl {
				t.Errorf("DefaultCacheControl = %v, want %v", cfg.DefaultCacheControl, tt.want.DefaultCacheControl)
			}
			if cfg.DefaultOverwrite != tt.want.DefaultOverwrite {
				t.Errorf("DefaultOverwrite = %v, want %v", cfg.DefaultOverwrite, tt.want.DefaultOverwrite)
			}
			if cfg.DefaultPreserveFilename != tt.want.DefaultPreserveFilename {
				t.Errorf("DefaultPreserveFilename = %v, want %v", cfg.DefaultPreserveFilename, tt.want.DefaultPreserveFilename)
			}
			if cfg.MaxFileSize != tt.want.MaxFileSize {
				t.Errorf("MaxFileSize = %v, want %v", cfg.MaxFileSize, tt.want.MaxFileSize)
			}
			if cfg.AllowedMimeTypes != tt.want.AllowedMimeTypes {
				t.Errorf("AllowedMimeTypes = %v, want %v", cfg.AllowedMimeTypes, tt.want.AllowedMimeTypes)
			}
			if cfg.BlockedMimeTypes != tt.want.BlockedMimeTypes {
				t.Errorf("BlockedMimeTypes = %v, want %v", cfg.BlockedMimeTypes, tt.want.BlockedMimeTypes)
			}
			if cfg.AllowedExtensions != tt.want.AllowedExtensions {
				t.Errorf("AllowedExtensions = %v, want %v", cfg.AllowedExtensions, tt.want.AllowedExtensions)
			}
			if cfg.BlockedExtensions != tt.want.BlockedExtensions {
				t.Errorf("BlockedExtensions = %v, want %v", cfg.BlockedExtensions, tt.want.BlockedExtensions)
			}
			if cfg.EncryptionEnabled != tt.want.EncryptionEnabled {
				t.Errorf("EncryptionEnabled = %v, want %v", cfg.EncryptionEnabled, tt.want.EncryptionEnabled)
			}
			if cfg.EncryptionAlgorithm != tt.want.EncryptionAlgorithm {
				t.Errorf("EncryptionAlgorithm = %v, want %v", cfg.EncryptionAlgorithm, tt.want.EncryptionAlgorithm)
			}
			if cfg.EncryptionKey != tt.want.EncryptionKey {
				t.Errorf("EncryptionKey = %v, want %v", cfg.EncryptionKey, tt.want.EncryptionKey)
			}
		})
	}
}
