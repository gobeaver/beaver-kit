// config.go
package captcha

import (
	"github.com/gobeaver/beaver-kit/config"
)

// Config defines captcha service configuration
type Config struct {
	// Provider specifies the captcha service provider (recaptcha, hcaptcha, turnstile)
	Provider string `env:"CAPTCHA_PROVIDER" envDefault:"recaptcha"`

	// SiteKey is the public key for the captcha service
	SiteKey string `env:"CAPTCHA_SITE_KEY"`

	// SecretKey is the private key for server-side validation
	SecretKey string `env:"CAPTCHA_SECRET_KEY"`

	// Version specifies the captcha version (only used for recaptcha: 2 or 3)
	Version int `env:"CAPTCHA_VERSION" envDefault:"2"`

	// Enabled determines if captcha validation is active
	Enabled bool `env:"CAPTCHA_ENABLED" envDefault:"false"`
}

// GetConfig returns config loaded from environment
func GetConfig(opts ...config.Option) (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg, opts...); err != nil {
		return nil, err
	}
	return cfg, nil
}
