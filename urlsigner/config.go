package urlsigner

import (
	"time"
)

// Config defines the configuration for URL signer
type Config struct {
	// SecretKey is the HMAC secret key for signing URLs
	SecretKey string `env:"URLSIGNER_SECRET_KEY,required"`

	// DefaultExpiry is the default expiration duration for signed URLs
	DefaultExpiry time.Duration `env:"URLSIGNER_DEFAULT_EXPIRY" envDefault:"30m"`

	// Algorithm is the hashing algorithm to use (currently only sha256 supported)
	Algorithm string `env:"URLSIGNER_ALGORITHM" envDefault:"sha256"`

	// SignatureParam is the query parameter name for signature
	SignatureParam string `env:"URLSIGNER_SIGNATURE_PARAM" envDefault:"sig"`

	// ExpiresParam is the query parameter name for expiration
	ExpiresParam string `env:"URLSIGNER_EXPIRES_PARAM" envDefault:"expires"`

	// PayloadParam is the query parameter name for payload
	PayloadParam string `env:"URLSIGNER_PAYLOAD_PARAM" envDefault:"payload"`
}
