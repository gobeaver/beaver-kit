package config

import (
	"github.com/gobeaver/beaver-kit/config/dotenv"
	"github.com/gobeaver/beaver-kit/config/env"
)

// DefaultPrefix is the default environment variable prefix
const DefaultPrefix = "BEAVER_"

// Options configures the config loader
type Options struct {
	// Prefix for environment variable names (default: "BEAVER_")
	Prefix string

	// EnvFiles to load before parsing (default: ".env")
	EnvFiles []string

	// SkipDotEnv skips loading .env files
	SkipDotEnv bool

	// Required makes all fields required unless marked optional
	Required bool
}

// Load parses environment variables into a struct.
// Automatically loads .env file first (can be disabled via options).
//
// Example:
//
//	type Config struct {
//		Host string `env:"HOST" envDefault:"localhost"`
//		Port int    `env:"PORT" envDefault:"8080"`
//	}
//
//	var cfg Config
//	config.Load(&cfg)                           // Uses BEAVER_ prefix
//	config.Load(&cfg, config.WithPrefix(""))    // No prefix
//	config.Load(&cfg, config.WithPrefix("APP_")) // Custom prefix
func Load(cfg interface{}, opts ...Option) error {
	options := Options{
		Prefix:   DefaultPrefix,
		EnvFiles: []string{".env"},
	}

	for _, opt := range opts {
		opt(&options)
	}

	// Load .env files (vendored joho/godotenv)
	if !options.SkipDotEnv {
		for _, file := range options.EnvFiles {
			_ = dotenv.Load(file) // Ignore errors - file may not exist
		}
	}

	// Parse environment variables (vendored caarlos0/env)
	envOpts := env.Options{
		Prefix: options.Prefix,
	}

	if options.Required {
		envOpts.RequiredIfNoDef = true
	}

	return env.ParseWithOptions(cfg, envOpts)
}

// MustLoad is like Load but panics on error
func MustLoad(cfg interface{}, opts ...Option) {
	if err := Load(cfg, opts...); err != nil {
		panic("config: " + err.Error())
	}
}

// Option configures Load behavior
type Option func(*Options)

// WithPrefix sets a custom environment variable prefix
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithEnvFiles sets custom .env files to load
func WithEnvFiles(files ...string) Option {
	return func(o *Options) {
		o.EnvFiles = files
	}
}

// WithoutDotEnv disables automatic .env file loading
func WithoutDotEnv() Option {
	return func(o *Options) {
		o.SkipDotEnv = true
	}
}

// WithRequired makes all fields required unless they have defaults
func WithRequired() Option {
	return func(o *Options) {
		o.Required = true
	}
}
