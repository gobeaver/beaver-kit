package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gobeaver/beaver-kit/filekit"
)

func init() {
	filekit.RegisterDriver("s3", createS3FileSystem)
}

func createS3FileSystem(cfg filekit.Config) (filekit.FileSystem, error) {
	// Create S3 client
	s3Client, err := createS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create S3 file system with options
	opts := []AdapterOption{}
	if cfg.S3Prefix != "" {
		opts = append(opts, WithPrefix(cfg.S3Prefix))
	}

	return New(s3Client, cfg.S3Bucket, opts...), nil
}

// createS3Client creates an S3 client from config
func createS3Client(cfg filekit.Config) (*s3.Client, error) {
	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
	)
	if err != nil {
		return nil, err
	}

	// Override with explicit credentials if provided
	if cfg.S3AccessKeyID != "" && cfg.S3SecretAccessKey != "" {
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKeyID,
			cfg.S3SecretAccessKey,
			"",
		)
	}

	// Create S3 client options
	s3Options := func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		if cfg.S3ForcePathStyle {
			o.UsePathStyle = true
		}
	}

	return s3.NewFromConfig(awsCfg, s3Options), nil
}
