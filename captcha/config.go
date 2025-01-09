// config.go
package captcha

import (
	"github.com/gobeaver/beaver-kit/config"
)

// Config defines captcha service configuration
type Config struct {
	// Provider specifies the captcha service provider (recaptcha, hcaptcha, turnstile)
	Provider string `env:"BEAVER_CAPTCHA_PROVIDER,default:recaptcha"`

	// SiteKey is the public key for the captcha service
	SiteKey string `env:"BEAVER_CAPTCHA_SITE_KEY"`

	// SecretKey is the private key for server-side validation
	SecretKey string `env:"BEAVER_CAPTCHA_SECRET_KEY"`

	// Version specifies the captcha version (only used for recaptcha: 2 or 3)
	Version int `env:"BEAVER_CAPTCHA_VERSION,default:2"`

	// Enabled determines if captcha validation is active
	Enabled bool `env:"BEAVER_CAPTCHA_ENABLED,default:false"`
}

// GetConfig returns config loaded from environment
func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
