package slack

import (
	"time"

	"github.com/gobeaver/beaver-kit/config"
)

// Config defines slack configuration
type Config struct {
	WebhookURL string        `env:"BEAVER_SLACK_WEBHOOK_URL"`
	Channel    string        `env:"BEAVER_SLACK_CHANNEL"`
	Username   string        `env:"BEAVER_SLACK_USERNAME,default:Beaver"`
	IconEmoji  string        `env:"BEAVER_SLACK_ICON_EMOJI"`
	IconURL    string        `env:"BEAVER_SLACK_ICON_URL"`
	Timeout    time.Duration `env:"BEAVER_SLACK_TIMEOUT,default:10s"`
}

// GetConfig returns config loaded from environment
func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
