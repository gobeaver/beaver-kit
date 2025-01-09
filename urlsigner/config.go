package urlsigner

import (
	"time"
)

// Config defines the configuration for URL signer
type Config struct {
	// SecretKey is the HMAC secret key for signing URLs
	SecretKey string `env:"BEAVER_URLSIGNER_SECRET_KEY,required"`

	// DefaultExpiry is the default expiration duration for signed URLs
	DefaultExpiry time.Duration `env:"BEAVER_URLSIGNER_DEFAULT_EXPIRY,default:30m"`

	// Algorithm is the hashing algorithm to use (currently only sha256 supported)
	Algorithm string `env:"BEAVER_URLSIGNER_ALGORITHM,default:sha256"`

	// SignatureParam is the query parameter name for signature
	SignatureParam string `env:"BEAVER_URLSIGNER_SIGNATURE_PARAM,default:sig"`

	// ExpiresParam is the query parameter name for expiration
	ExpiresParam string `env:"BEAVER_URLSIGNER_EXPIRES_PARAM,default:expires"`

	// PayloadParam is the query parameter name for payload
	PayloadParam string `env:"BEAVER_URLSIGNER_PAYLOAD_PARAM,default:payload"`
}
