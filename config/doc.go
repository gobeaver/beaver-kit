// Package config provides flexible configuration loading from environment variables
// with support for custom prefixes, automatic type conversion, and .env file loading.
//
// This package follows the twelve-factor app methodology for configuration management,
// allowing applications to be easily configured across different environments without
// code changes. It supports struct-based configuration with field tags and provides
// builder patterns for advanced usage scenarios.
//
// # Basic Usage
//
// Define a configuration struct with environment variable tags:
//
//	type Config struct {
//	    DatabaseURL string `env:"DATABASE_URL"`
//	    Port        int    `env:"PORT" envDefault:"8080"`
//	    Debug       bool   `env:"DEBUG" envDefault:"false"`
//	    Timeout     time.Duration `env:"TIMEOUT" envDefault:"30s"`
//	}
//
// Load configuration from environment variables:
//
//	import "github.com/gobeaver/beaver-kit/config"
//
//	var cfg Config
//	err := config.Load(&cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Custom Prefixes
//
// Use custom prefixes to avoid environment variable conflicts:
//
//	// Load with custom prefix (will look for MYAPP_DATABASE_URL, MYAPP_PORT, etc.)
//	err := config.Load(&cfg, config.LoadOptions{Prefix: "MYAPP_"})
//
// # Builder Pattern
//
// Many beaver-kit packages support the builder pattern for prefix configuration:
//
//	// OAuth with custom prefix
//	err := oauth.WithPrefix("MYAPP_OAUTH_").Init()
//
//	// Database with custom prefix  
//	err := database.WithPrefix("MYAPP_DB_").Init()
//
// # Supported Types
//
// The config package automatically handles type conversion for:
//   - string: Direct string values
//   - int, int8, int16, int32, int64: Integer conversion with validation
//   - uint, uint8, uint16, uint32, uint64: Unsigned integer conversion
//   - float32, float64: Floating point conversion
//   - bool: Boolean conversion ("true", "false", "1", "0", "yes", "no")
//   - time.Duration: Duration parsing ("1h30m", "45s", etc.)
//   - []string: Comma-separated values
//   - Custom types implementing encoding.TextUnmarshaler
//
// # Field Tags
//
// Configure field behavior using struct tags:
//   - `env:"VAR_NAME"`: Specify environment variable name
//   - `envDefault:"value"`: Set default value if environment variable is not set
//   - `envRequired:"true"`: Mark field as required (will error if not provided)
//   - `envSeparator:"|"`: Use custom separator for slice types (default: ",")
//
// Example with all tags:
//
//	type DatabaseConfig struct {
//	    Host     string   `env:"DB_HOST" envRequired:"true"`
//	    Port     int      `env:"DB_PORT" envDefault:"5432"`
//	    Database string   `env:"DB_NAME" envRequired:"true"`
//	    Options  []string `env:"DB_OPTIONS" envSeparator:"|"`
//	}
//
// # Environment File Support
//
// The package automatically loads .env files from the current directory:
//
//	# .env file
//	DATABASE_URL=postgres://localhost:5432/myapp
//	PORT=8080
//	DEBUG=true
//
// Environment variables take precedence over .env file values.
//
// # Debug Mode
//
// Enable debug logging to see configuration loading details:
//
//	// Enable debug via environment variable
//	export BEAVER_CONFIG_DEBUG=true
//
//	// Or programmatically
//	err := config.Load(&cfg, config.LoadOptions{Debug: true})
//
// Debug mode shows:
//   - Which .env files are loaded
//   - Environment variable lookups and values
//   - Type conversions and default value usage
//   - Configuration validation results
//
// # Error Handling
//
// The package provides detailed error information for:
//   - Missing required environment variables
//   - Type conversion failures
//   - Invalid default values
//   - Struct validation errors
//
// Example error handling:
//
//	var cfg Config
//	if err := config.Load(&cfg); err != nil {
//	    fmt.Printf("Configuration error: %v\n", err)
//	    // Handle specific error types if needed
//	    return fmt.Errorf("failed to load config: %w", err)
//	}
//
// # Multi-Environment Support
//
// Use different prefixes for different environments:
//
//	switch os.Getenv("ENVIRONMENT") {
//	case "production":
//	    err = config.Load(&cfg, config.LoadOptions{Prefix: "PROD_"})
//	case "staging":
//	    err = config.Load(&cfg, config.LoadOptions{Prefix: "STAGE_"})
//	default:
//	    err = config.Load(&cfg, config.LoadOptions{Prefix: "DEV_"})
//	}
//
// # Best Practices
//
//   - Use descriptive environment variable names
//   - Provide sensible defaults for non-critical settings
//   - Mark security-sensitive variables as required
//   - Use the builder pattern for package-specific prefixes
//   - Enable debug mode during development
//   - Validate loaded configuration before using
//
// # Integration with Beaver Kit
//
// All beaver-kit packages use this config system internally:
//   - OAuth: BEAVER_OAUTH_* variables with oauth.WithPrefix() support
//   - Database: BEAVER_DB_* variables with database.WithPrefix() support  
//   - Cache: BEAVER_CACHE_* variables with cache.WithPrefix() support
//   - Slack: BEAVER_SLACK_* variables with slack.WithPrefix() support
//
// This ensures consistent configuration patterns across the entire toolkit.
package config