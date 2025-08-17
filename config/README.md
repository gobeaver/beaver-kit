# Config Package

Environment variable configuration loader with struct tag support, configurable prefixes, and automatic .env file loading.

## Features

- Load environment variables into Go structs using reflection
- **Configurable environment variable prefixes** for multi-tenant applications
- Support for default values via struct tags
- Type conversion for string, int, int64, bool, and **time.Duration** fields
- **Automatic .env file loading** (via github.com/joho/godotenv)
- **Debug mode** for development environments
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
    DatabaseURL string        `env:"DATABASE_URL,default:postgres://localhost/myapp"`
    Port        int           `env:"PORT,default:8080"`
    Debug       bool          `env:"DEBUG,default:false"`
    Timeout     time.Duration `env:"TIMEOUT,default:30s"`
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
```

### Custom Environment Variable Prefixes

For multi-tenant applications or custom configurations:

```go
// Load with custom prefix
cfg := &AppConfig{}
if err := config.Load(cfg, config.LoadOptions{Prefix: "MYAPP_"}); err != nil {
    panic(err)
}

// Now it reads: MYAPP_DATABASE_URL, MYAPP_PORT, etc.
```

## Struct Tag Format

```go
type Config struct {
    Field string `env:"ENV_VAR_NAME,default:defaultvalue"`
}
```

- `ENV_VAR_NAME`: Environment variable name (**without prefix** - prefix is applied automatically)
- `default:value`: Optional default value if environment variable is not set

## Supported Types

- `string`: Direct assignment
- `int` and `int64`: Parsed using `strconv.ParseInt`
- `bool`: Parsed using `strconv.ParseBool` (accepts: true, false, 1, 0, t, f, TRUE, FALSE, True, False)
- `time.Duration`: Parsed using `time.ParseDuration` (e.g., "10s", "5m", "1h30m")

## Debug Mode

Enable debug output to see loaded configuration values:

```bash
BEAVER_CONFIG_DEBUG=true ./myapp
```

This will print all loaded configuration values in the format:
```
[BEAVER] BEAVER_DATABASE_URL=postgres://localhost/myapp
[BEAVER] BEAVER_PORT=8080
[BEAVER] BEAVER_DEBUG=false
```

Debug output is also automatically enabled when `env` is set to `development`, `dev`, or `test`.

## Package Integration Pattern

### Basic Package Configuration

```go
package mypackage

type Config struct {
    APIKey string `env:"API_KEY"`     // Will be prefixed as BEAVER_API_KEY
    Host   string `env:"HOST,default:localhost"`
    Port   int    `env:"PORT,default:8080"`
}

func GetConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

### Multi-Tenant Package Configuration

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
    if err := config.Load(cfg, config.LoadOptions{Prefix: db.prefix}); err != nil {
        return nil, err
    }
    // ... create connection
}

// Usage:
prodDB := database.WithPrefix("PROD_").Connect()
testDB := database.WithPrefix("TEST_").Connect()
```

## Advanced Usage

### .env File Support

The package automatically loads `.env` files if present in the working directory:

```bash
# .env file
BEAVER_DATABASE_URL=postgres://localhost/dev
BEAVER_DEBUG=true
```

### Multiple Configuration Structs

```go
type DatabaseConfig struct {
    URL      string `env:"DATABASE_URL"`
    MaxConns int    `env:"DATABASE_MAX_CONNS,default:10"`
}

type CacheConfig struct {
    RedisURL string        `env:"REDIS_URL,default:redis://localhost:6379"`
    TTL      time.Duration `env:"CACHE_TTL,default:5m"`
}

// Load multiple configs with same prefix
dbCfg := &DatabaseConfig{}
cacheCfg := &CacheConfig{}

config.Load(dbCfg)    // Uses BEAVER_ prefix by default
config.Load(cacheCfg) // Uses BEAVER_ prefix by default
```

### Custom Prefix for Different Environments

```go
// Determine prefix based on environment
prefix := "BEAVER_"
if env := os.Getenv("APP_ENV"); env == "production" {
    prefix = "PROD_"
}

cfg := &Config{}
config.Load(cfg, config.LoadOptions{Prefix: prefix})
```

## Implementation Details

The `config.Load()` function:
1. Automatically loads `.env` files if present (errors ignored)
2. Iterates through struct fields using reflection
3. Reads `env` tags to get environment variable names
4. Applies the configured prefix to environment variable names
5. Loads values from environment or uses defaults
6. Converts string values to appropriate field types
7. Supports debug output for development environments

## Best Practices

1. **Use struct tags without prefixes** - Let the config package handle prefixing
2. **Always provide defaults** for optional configuration
3. **Use custom prefixes** for multi-tenant applications
4. **Enable debug mode** during development to verify configuration
5. **Keep sensitive values** in environment variables, not in code
6. **Use .env files** for local development only, not in production