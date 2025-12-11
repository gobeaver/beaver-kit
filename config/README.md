# Config Package

Environment variable configuration loader with struct tag support, configurable prefixes, and automatic .env file loading.

This package wraps [github.com/caarlos0/env](https://github.com/caarlos0/env) with sensible defaults for Beaver Kit applications.

## Features

- Load environment variables into Go structs using reflection
- **Configurable environment variable prefixes** for multi-tenant applications
- Support for default values via `envDefault` struct tag
- **Full type support**: strings, ints, floats, bools, durations, slices, maps, nested structs
- **Required field validation** with `required` tag
- **Automatic .env file loading** (via vendored joho/godotenv)
- Zero external dependencies (all dependencies are vendored)
- Zero configuration with sensible defaults

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "time"
    "github.com/gobeaver/beaver-kit/config"
)

type AppConfig struct {
    DatabaseURL string        `env:"DATABASE_URL" envDefault:"postgres://localhost/myapp"`
    Port        int           `env:"PORT" envDefault:"8080"`
    Debug       bool          `env:"DEBUG" envDefault:"false"`
    Timeout     time.Duration `env:"TIMEOUT" envDefault:"30s"`
    Hosts       []string      `env:"HOSTS" envSeparator:","`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        panic(err)
    }

    fmt.Printf("Database: %s\n", cfg.DatabaseURL)
    fmt.Printf("Port: %d\n", cfg.Port)
    fmt.Printf("Debug: %t\n", cfg.Debug)
    fmt.Printf("Timeout: %v\n", cfg.Timeout)
}
```

### Environment Variables (Default Prefix)

By default, all environment variables use the `BEAVER_` prefix:

```bash
BEAVER_DATABASE_URL=postgres://prod-server/myapp
BEAVER_PORT=3000
BEAVER_DEBUG=true
BEAVER_TIMEOUT=60s
BEAVER_HOSTS=host1,host2,host3
```

### Custom Environment Variable Prefixes

For multi-tenant applications or custom configurations:

```go
// Load with custom prefix
cfg := &AppConfig{}
if err := config.Load(cfg, config.WithPrefix("MYAPP_")); err != nil {
    panic(err)
}

// Now it reads: MYAPP_DATABASE_URL, MYAPP_PORT, etc.

// Load with no prefix
if err := config.Load(cfg, config.WithPrefix("")); err != nil {
    panic(err)
}

// Now it reads: DATABASE_URL, PORT, etc.
```

## Struct Tag Format

```go
type Config struct {
    // Basic field with default
    Host string `env:"HOST" envDefault:"localhost"`

    // Required field (error if not set)
    APIKey string `env:"API_KEY,required"`

    // Slice with custom separator
    Hosts []string `env:"HOSTS" envSeparator:","`

    // Map field
    Metadata map[string]string `env:"METADATA"`

    // Nested struct with prefix
    Database DatabaseConfig `envPrefix:"DB_"`

    // Ignored field
    Internal string `env:"-"`
}
```

### Tag Reference

| Tag | Description |
|-----|-------------|
| `env:"NAME"` | Environment variable name (without prefix) |
| `env:"NAME,required"` | Field is required (error if not set) |
| `env:"NAME,notEmpty"` | Field must not be empty string |
| `env:"NAME,file"` | Value is a file path, read contents from file |
| `env:"NAME,expand"` | Expand $VAR or ${VAR} in value |
| `env:"NAME,unset"` | Unset variable after reading |
| `env:"-"` | Ignore this field |
| `envDefault:"value"` | Default value if not set |
| `envSeparator:","` | Separator for slice types (default: ",") |
| `envKeyValSeparator:":"` | Separator for map key:value pairs (default: ":") |
| `envPrefix:"PREFIX_"` | Prefix for nested struct fields |

## Supported Types

- `string`: Direct assignment
- `int`, `int8`, `int16`, `int32`, `int64`: Integer types
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`: Unsigned integer types
- `float32`, `float64`: Floating point types
- `bool`: Parsed using `strconv.ParseBool` (accepts: true, false, 1, 0, t, f, TRUE, FALSE)
- `time.Duration`: Parsed using `time.ParseDuration` (e.g., "10s", "5m", "1h30m")
- `url.URL`: Parsed using `url.Parse`
- `[]T`: Slices of any supported type
- `map[K]V`: Maps with supported key and value types
- Custom types implementing `encoding.TextUnmarshaler`

## Functional Options

```go
// Load with BEAVER_ prefix (default)
config.Load(&cfg)

// Load with custom prefix
config.Load(&cfg, config.WithPrefix("MYAPP_"))

// Load with no prefix
config.Load(&cfg, config.WithPrefix(""))

// Load specific .env files
config.Load(&cfg, config.WithEnvFiles(".env", ".env.local"))

// Disable .env file loading
config.Load(&cfg, config.WithoutDotEnv())

// Make all fields required unless they have defaults
config.Load(&cfg, config.WithRequired())

// Combine options
config.Load(&cfg,
    config.WithPrefix("APP_"),
    config.WithEnvFiles(".env.production"),
)
```

## Package Integration Pattern

### Basic Package Configuration

```go
package mypackage

type Config struct {
    APIKey string `env:"API_KEY"`     // Will be prefixed as BEAVER_API_KEY
    Host   string `env:"HOST" envDefault:"localhost"`
    Port   int    `env:"PORT" envDefault:"8080"`
}

func GetConfig(opts ...config.Option) (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg, opts...); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

### Multi-Instance Pattern

For packages that support multiple instances with different configurations:

```go
package database

// Builder pattern for configurable prefix
type Database struct {
    prefix string
}

func WithPrefix(prefix string) *Database {
    return &Database{prefix: prefix}
}

func (db *Database) Connect() (*Connection, error) {
    cfg := &Config{}
    if err := config.Load(cfg, config.WithPrefix(db.prefix)); err != nil {
        return nil, err
    }
    // ... create connection
}

// Usage:
// Environment: PROD_DB_HOST, PROD_DB_PORT, TEST_DB_HOST, TEST_DB_PORT
prodDB := database.WithPrefix("PROD_").Connect()
testDB := database.WithPrefix("TEST_").Connect()
```

## .env File Support

The package automatically loads `.env` files if present in the working directory:

```bash
# .env file
BEAVER_DATABASE_URL=postgres://localhost/dev
BEAVER_DEBUG=true
```

### Multiple .env Files

```go
config.Load(&cfg, config.WithEnvFiles(".env", ".env.local", ".env.secrets"))
```

### Disable .env Loading

```go
config.Load(&cfg, config.WithoutDotEnv())
```

## Error Handling

### Fail-Fast with MustLoad

```go
// Panics on error
config.MustLoad(&cfg)
config.MustLoad(&cfg, config.WithPrefix("MYAPP_"))
```

### Required Fields

```go
type Config struct {
    APIKey string `env:"API_KEY,required"`
}

// Returns error: required environment variable "BEAVER_API_KEY" is not set
```

### Make All Fields Required

```go
type Config struct {
    Host string `env:"HOST"`                    // Required
    Port int    `env:"PORT" envDefault:"8080"`  // Has default, not required
}

config.Load(&cfg, config.WithRequired())
```

## Nested Structs

```go
type DatabaseConfig struct {
    Host string `env:"HOST" envDefault:"localhost"`
    Port int    `env:"PORT" envDefault:"5432"`
}

type Config struct {
    App      string         `env:"APP_NAME" envDefault:"myapp"`
    Database DatabaseConfig `envPrefix:"DB_"`
}

// Environment variables:
// BEAVER_APP_NAME=myapp
// BEAVER_DB_HOST=db.example.com
// BEAVER_DB_PORT=5432
```

## Migration from Old API

If you're migrating from the old beaver-kit/config API:

| Old | New |
|-----|-----|
| `env:"NAME,default:value"` | `env:"NAME" envDefault:"value"` |
| `config.LoadOptions{Prefix: "X_"}` | `config.WithPrefix("X_")` |
| `config.Load(cfg, opts...)` where opts is `LoadOptions` | `config.Load(cfg, opts...)` where opts is `Option` |

## Security

This package vendors its dependencies (caarlos0/env and joho/godotenv) to:
- Prevent supply chain attacks
- Ensure reproducible builds
- Allow for code auditing

See `env/CREDITS.md` and `dotenv/CREDITS.md` for version information and audit history.

## Best Practices

1. **Use `envDefault` for optional configuration** - Don't make everything required
2. **Use custom prefixes** for multi-tenant applications
3. **Keep sensitive values** in environment variables, not in code
4. **Use .env files** for local development only, not in production
5. **Validate configuration** early in application startup
