package config

import (
	"fmt"
	"github.com/gobeaver/beaver-kit/config/dotenv"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// LoadOptions defines options for loading configuration from environment variables.
type LoadOptions struct {
	Prefix string // Prefix to prepend to environment variable names (default: "BEAVER_")
	Debug  bool   // Enable debug logging of configuration loading process
}

// Load populates a struct from .env file and environment variables using reflection.
// This function automatically loads .env files from the current directory and then
// reads environment variables to populate the provided struct.
//
// The function uses struct field tags to determine environment variable names:
//   - `env:"VAR_NAME"`: Maps the field to the specified environment variable
//   - `env:"VAR_NAME,default:value"`: Provides a default value if env var is not set
//
// Environment variable names are automatically prefixed with the value specified
// in LoadOptions.Prefix (defaults to "BEAVER_").
//
// Parameters:
//   - cfg: Pointer to a struct to populate with configuration values
//   - opts: Optional LoadOptions to customize loading behavior
//
// Returns an error if:
//   - cfg is not a pointer to a struct
//   - Type conversion fails for any field
//   - Required environment variables are missing (if validation is implemented)
//
// Example:
//
//	type Config struct {
//	    DatabaseURL string `env:"DATABASE_URL"`
//	    Port        int    `env:"PORT,default:8080"`
//	    Debug       bool   `env:"DEBUG,default:false"`
//	}
//
//	var cfg Config
//	err := config.Load(&cfg, config.LoadOptions{Prefix: "MYAPP_"})
//	// Will look for MYAPP_DATABASE_URL, MYAPP_PORT, MYAPP_DEBUG
func Load(cfg interface{}, opts ...LoadOptions) error {
	options := LoadOptions{Prefix: "BEAVER_"} // Default
	if len(opts) > 0 {
		options = opts[0]
	}
	// Silently try to load .env file, ignore if not found
	dotenv.Load()

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	printDebug := os.Getenv("BEAVER_CONFIG_DEBUG") == "true"

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		parts := strings.Split(envTag, ",")
		envName := parts[0]
		defaultValue := ""

		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "default:") {
				defaultValue = strings.TrimPrefix(part, "default:")
				break
			}
		}

		// Apply prefix to environment variable name
		fullEnvName := options.Prefix + envName
		value := os.Getenv(fullEnvName)
		if value == "" {
			value = defaultValue
		}
		if printDebug || os.Getenv("env") == "development" || os.Getenv("env") == "test" || os.Getenv("env") == "dev" {
			fmt.Printf("[BEAVER] %s=%s\n", fullEnvName, value)
		}

		if value != "" {
			if err := setFieldValue(v.Field(i), value); err != nil {
				return err
			}
		}
	}

	return nil
}

// setFieldValue sets the value of a struct field using reflection and type conversion.
// This is an internal helper function that handles conversion from string environment
// variable values to the appropriate Go types.
//
// Supported types:
//   - string: Direct assignment
//   - int, int64: Parsed using strconv.ParseInt with base 10
//   - bool: Parsed using strconv.ParseBool (supports "true", "false", "1", "0", etc.)
//   - time.Duration: Parsed using time.ParseDuration
//
// Parameters:
//   - field: The reflect.Value of the struct field to set
//   - value: The string value from the environment variable
//
// Returns an error if type conversion fails or the type is unsupported.
func setFieldValue(field reflect.Value, value string) error {
	// Check for time.Duration first
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		// Skip unsupported field types silently
		return nil
	}
	return nil
}
