// Package config provides environment variable configuration loading with
// automatic .env file support and configurable prefixes.
//
// This package wraps github.com/caarlos0/env with sensible defaults for
// Beaver Kit applications.
//
// # Basic Usage
//
//	type Config struct {
//		Host string `env:"HOST" envDefault:"localhost"`
//		Port int    `env:"PORT" envDefault:"8080"`
//	}
//
//	var cfg Config
//	config.Load(&cfg) // Uses BEAVER_ prefix, loads .env
//
// # Environment Variables
//
// By default, all environment variables are prefixed with "BEAVER_":
//
//	BEAVER_HOST=example.com
//	BEAVER_PORT=3000
//
// # Custom Prefixes
//
// Use WithPrefix for multi-instance configurations:
//
//	// Primary database: PRIMARY_DB_HOST, PRIMARY_DB_PORT
//	config.Load(&dbCfg, config.WithPrefix("PRIMARY_DB_"))
//
//	// Replica database: REPLICA_DB_HOST, REPLICA_DB_PORT
//	config.Load(&dbCfg, config.WithPrefix("REPLICA_DB_"))
//
//	// No prefix: DB_HOST, DB_PORT
//	config.Load(&dbCfg, config.WithPrefix(""))
//
// # Supported Types
//
// All types supported by caarlos0/env are available:
//
//	type Config struct {
//		// Basic types
//		String   string        `env:"STRING"`
//		Int      int           `env:"INT"`
//		Bool     bool          `env:"BOOL"`
//		Duration time.Duration `env:"DURATION"`
//
//		// Slices
//		Hosts []string `env:"HOSTS" envSeparator:","`
//
//		// Required fields
//		APIKey string `env:"API_KEY,required"`
//
//		// Defaults
//		Port int `env:"PORT" envDefault:"8080"`
//
//		// Nested structs
//		Database DatabaseConfig `envPrefix:"DB_"`
//	}
//
// # .env File Support
//
// The package automatically loads .env files before parsing.
// Use WithEnvFiles to specify custom files:
//
//	config.Load(&cfg, config.WithEnvFiles(".env", ".env.local"))
//
// Use WithoutDotEnv to disable automatic loading:
//
//	config.Load(&cfg, config.WithoutDotEnv())
//
// # Multi-Instance Pattern
//
// The prefix pattern enables running multiple instances with different configs:
//
//	// Environment:
//	// DEV_SLACK_WEBHOOK_URL=https://hooks.slack.com/dev
//	// PROD_SLACK_WEBHOOK_URL=https://hooks.slack.com/prod
//
//	devSlack := slack.WithPrefix("DEV_").New()
//	prodSlack := slack.WithPrefix("PROD_").New()
//
// This pattern avoids YAML complexity while remaining 12-factor compliant.
//
// # Tag Reference
//
// Available struct tags:
//
//	env:"NAME"              - Environment variable name (required)
//	env:"NAME,required"     - Field is required (error if not set)
//	env:"NAME,notEmpty"     - Field must not be empty string
//	env:"NAME,file"         - Value is a file path, read contents from file
//	env:"NAME,expand"       - Expand $VAR or ${VAR} in value
//	env:"NAME,unset"        - Unset variable after reading
//	env:"-"                 - Ignore this field
//	envDefault:"value"      - Default value if not set
//	envSeparator:","        - Separator for slice types (default: ",")
//	envKeyValSeparator:":"  - Separator for map key:value pairs (default: ":")
//	envPrefix:"PREFIX_"     - Prefix for nested struct fields
//
// # Required Fields
//
// Use WithRequired to make all fields without defaults required:
//
//	config.Load(&cfg, config.WithRequired())
//
// Or mark individual fields:
//
//	type Config struct {
//		APIKey string `env:"API_KEY,required"`
//	}
//
// # Error Handling
//
// Use MustLoad for fail-fast behavior:
//
//	config.MustLoad(&cfg) // Panics on error
//
// Or handle errors explicitly:
//
//	if err := config.Load(&cfg); err != nil {
//		log.Fatalf("config error: %v", err)
//	}
package config
