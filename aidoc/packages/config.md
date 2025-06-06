---
title: "Config Package API Reference"
tags: ["config", "environment", "configuration", "struct-tags"]
prerequisites: []
relatedDocs:
  - "getting-started"
  - "database"
  - "cache"
---

# Config Package

## Overview

The config package provides environment variable configuration loading with struct tag support and default values. It serves as the foundation for all other Beaver Kit packages, enabling consistent environment-based configuration across the framework.

```json
{
  "@context": "http://schema.org",
  "@type": "TechArticle",
  "name": "Config Package API Reference",
  "about": "Environment variable configuration loader with struct tag support",
  "programmingLanguage": "Go",
  "codeRepository": "https://github.com/gobeaver/beaver-kit",
  "keywords": ["config", "environment", "configuration", "struct-tags", "defaults"]
}
```

## Key Features

- **Struct Tag Support** - Load environment variables directly into Go structs
- **Type Conversion** - Automatic conversion for string, int, int64, and bool fields
- **Default Values** - Specify default values via struct tags
- **Zero Dependencies** - Simple, lightweight implementation
- **Beaver Kit Foundation** - Used by all other packages for configuration

## Quick Start

### Basic Usage

```go
// Purpose: Load environment variables into Go struct with defaults
// Prerequisites: Environment variables optionally set
// Expected outcome: Struct populated with environment values or defaults

package main

import (
    "fmt"
    "github.com/gobeaver/beaver-kit/config"
)

type AppConfig struct {
    DatabaseURL string `env:"DATABASE_URL,default:postgres://localhost/myapp"`
    Port        int    `env:"PORT,default:8080"`
    Debug       bool   `env:"DEBUG,default:false"`
    APIKey      string `env:"API_KEY"`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        panic(err)
    }
    
    fmt.Printf("Database: %s\n", cfg.DatabaseURL)
    fmt.Printf("Port: %d\n", cfg.Port)
    fmt.Printf("Debug: %t\n", cfg.Debug)
    fmt.Printf("API Key: %s\n", cfg.APIKey)
}
```

### Environment Variables

```bash
# Purpose: Set environment variables for configuration
# Prerequisites: Application using config package
# Expected outcome: Values loaded into struct fields

DATABASE_URL=postgres://prod-server/myapp
PORT=3000
DEBUG=true
API_KEY=secret-api-key
```

## API Reference

### Load Function

```go
// Purpose: Load environment variables into struct using reflection
// Prerequisites: Struct with env tags defined
// Expected outcome: Struct fields populated with environment values or defaults

func Load(config interface{}) error

// Example usage
type Config struct {
    Field string `env:"ENV_VAR_NAME,default:defaultvalue"`
}

cfg := &Config{}
err := config.Load(cfg)
```

## Struct Tag Format

```go
// Purpose: Define environment variable mapping and defaults
// Prerequisites: Understanding of struct tags
// Expected outcome: Proper field mapping configuration

type Config struct {
    // Basic mapping
    Field1 string `env:"ENV_VAR_NAME"`
    
    // With default value
    Field2 string `env:"ENV_VAR_NAME,default:defaultvalue"`
    
    // Different types
    Port   int    `env:"PORT,default:8080"`
    Enabled bool  `env:"ENABLED,default:true"`
    Size   int64  `env:"SIZE,default:1024"`
}
```

### Tag Components

- **env**: Required prefix
- **ENV_VAR_NAME**: Environment variable to read from
- **default:value**: Optional default value if environment variable is not set

## Supported Types

### String Fields

```go
// Purpose: Map string environment variables
// Prerequisites: String environment variable set
// Expected outcome: Direct string assignment

type Config struct {
    Name        string `env:"APP_NAME,default:MyApp"`
    Description string `env:"APP_DESC"`
    Version     string `env:"VERSION,default:1.0.0"`
}
```

### Integer Fields

```go
// Purpose: Map integer environment variables with type conversion
// Prerequisites: Numeric environment variable set
// Expected outcome: Parsed integer value

type Config struct {
    Port     int   `env:"PORT,default:8080"`
    MaxUsers int   `env:"MAX_USERS,default:100"`
    FileSize int64 `env:"FILE_SIZE,default:1048576"` // 1MB
}
```

### Boolean Fields

```go
// Purpose: Map boolean environment variables with flexible parsing
// Prerequisites: Boolean environment variable set
// Expected outcome: Parsed boolean value

type Config struct {
    Debug   bool `env:"DEBUG,default:false"`
    Enabled bool `env:"ENABLED,default:true"`
    Verbose bool `env:"VERBOSE"`
}

// Supported boolean values:
// true:  "true", "1", "t", "TRUE", "True"
// false: "false", "0", "f", "FALSE", "False"
```

## Real-World Examples

### Database Configuration

```go
// Purpose: Configure database connection using environment variables
// Prerequisites: Database connection details available
// Expected outcome: Complete database configuration

type DatabaseConfig struct {
    Driver   string `env:"DB_DRIVER,default:postgres"`
    Host     string `env:"DB_HOST,default:localhost"`
    Port     int    `env:"DB_PORT,default:5432"`
    Database string `env:"DB_NAME,default:myapp"`
    Username string `env:"DB_USER,default:postgres"`
    Password string `env:"DB_PASSWORD"`
    
    // Connection pool settings
    MaxOpenConns int `env:"DB_MAX_OPEN_CONNS,default:25"`
    MaxIdleConns int `env:"DB_MAX_IDLE_CONNS,default:5"`
    
    // SSL settings
    SSLMode string `env:"DB_SSL_MODE,default:disable"`
    
    // Debug settings
    Debug bool `env:"DB_DEBUG,default:false"`
}

func loadDatabaseConfig() (*DatabaseConfig, error) {
    cfg := &DatabaseConfig{}
    if err := config.Load(cfg); err != nil {
        return nil, fmt.Errorf("failed to load database config: %w", err)
    }
    return cfg, nil
}
```

### Application Configuration

```go
// Purpose: Configure application settings from environment
// Prerequisites: Application environment variables set
// Expected outcome: Complete application configuration

type AppConfig struct {
    // Basic settings
    AppName string `env:"APP_NAME,default:MyApplication"`
    Version string `env:"APP_VERSION,default:1.0.0"`
    
    // Server settings
    Host string `env:"HOST,default:0.0.0.0"`
    Port int    `env:"PORT,default:8080"`
    
    // Feature flags
    EnableMetrics bool `env:"ENABLE_METRICS,default:true"`
    EnableTracing bool `env:"ENABLE_TRACING,default:false"`
    
    // External services
    RedisURL    string `env:"REDIS_URL"`
    DatabaseURL string `env:"DATABASE_URL,default:postgres://localhost/myapp"`
    
    // Security
    JWTSecret  string `env:"JWT_SECRET"`
    APIKey     string `env:"API_KEY"`
    
    // Performance
    WorkerCount    int   `env:"WORKER_COUNT,default:4"`
    MaxRequestSize int64 `env:"MAX_REQUEST_SIZE,default:10485760"` // 10MB
    
    // Logging
    LogLevel  string `env:"LOG_LEVEL,default:info"`
    LogFormat string `env:"LOG_FORMAT,default:json"`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        log.Fatal("Configuration error:", err)
    }
    
    // Validate required fields
    if cfg.JWTSecret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }
    
    // Use configuration
    server := &http.Server{
        Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
    }
    
    log.Printf("Starting %s v%s on %s", cfg.AppName, cfg.Version, server.Addr)
}
```

### Multi-Environment Configuration

```go
// Purpose: Handle different environment configurations
// Prerequisites: Environment-specific variables set
// Expected outcome: Environment-appropriate configuration

type Config struct {
    Environment string `env:"ENVIRONMENT,default:development"`
    
    // Database settings vary by environment
    DatabaseURL string `env:"DATABASE_URL"`
    
    // Cache settings
    CacheDriver string `env:"CACHE_DRIVER,default:memory"`
    RedisURL    string `env:"REDIS_URL"`
    
    // External service URLs
    APIBaseURL      string `env:"API_BASE_URL"`
    WebhookURL      string `env:"WEBHOOK_URL"`
    
    // Debug settings
    Debug   bool `env:"DEBUG,default:false"`
    Verbose bool `env:"VERBOSE,default:false"`
    
    // Performance settings
    RateLimit int `env:"RATE_LIMIT,default:100"`
    Timeout   int `env:"TIMEOUT,default:30"`
}

func loadEnvironmentConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, err
    }
    
    // Environment-specific validations
    switch cfg.Environment {
    case "production":
        if cfg.DatabaseURL == "" {
            return nil, errors.New("DATABASE_URL required in production")
        }
        if cfg.Debug {
            log.Warn("Debug mode enabled in production")
        }
    case "development":
        if cfg.DatabaseURL == "" {
            cfg.DatabaseURL = "postgres://localhost/myapp_dev"
        }
    case "test":
        if cfg.DatabaseURL == "" {
            cfg.DatabaseURL = "postgres://localhost/myapp_test"
        }
        cfg.CacheDriver = "memory" // Force memory cache for tests
    }
    
    return cfg, nil
}
```

## Integration Patterns

### Package Configuration Pattern

```go
// Purpose: Standard pattern used by all Beaver Kit packages
// Prerequisites: Understanding of package initialization
// Expected outcome: Consistent configuration across packages

package mypackage

type Config struct {
    Enabled bool   `env:"MYPACKAGE_ENABLED,default:true"`
    Host    string `env:"MYPACKAGE_HOST,default:localhost"`
    Port    int    `env:"MYPACKAGE_PORT,default:8080"`
    APIKey  string `env:"MYPACKAGE_API_KEY"`
    Timeout int    `env:"MYPACKAGE_TIMEOUT,default:30"`
    Debug   bool   `env:"MYPACKAGE_DEBUG,default:false"`
}

func GetConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, fmt.Errorf("mypackage config: %w", err)
    }
    return cfg, nil
}

// Package initialization function
func Init(configs ...Config) error {
    var cfg Config
    if len(configs) > 0 {
        cfg = configs[0]
    } else {
        loadedCfg, err := GetConfig()
        if err != nil {
            return err
        }
        cfg = *loadedCfg
    }
    
    // Initialize package with config
    return initializeWithConfig(cfg)
}
```

### Validation Pattern

```go
// Purpose: Validate configuration after loading
// Prerequisites: Config struct with validation requirements
// Expected outcome: Validated configuration or error

type Config struct {
    DatabaseURL string `env:"DATABASE_URL"`
    Port        int    `env:"PORT,default:8080"`
    APIKey      string `env:"API_KEY"`
}

func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("DATABASE_URL is required")
    }
    
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("invalid port: %d", c.Port)
    }
    
    if c.APIKey == "" {
        return errors.New("API_KEY is required")
    }
    
    return nil
}

func LoadAndValidateConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, fmt.Errorf("config load failed: %w", err)
    }
    
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return cfg, nil
}
```

## Error Handling

### Common Errors

```go
// Purpose: Handle configuration loading errors appropriately
// Prerequisites: Understanding of potential failure modes
// Expected outcome: Robust error handling

func loadConfigWithErrorHandling() (*Config, error) {
    cfg := &Config{}
    err := config.Load(cfg)
    if err != nil {
        // Check for specific error types
        if strings.Contains(err.Error(), "invalid syntax") {
            return nil, fmt.Errorf("configuration parsing error: %w", err)
        }
        
        if strings.Contains(err.Error(), "strconv") {
            return nil, fmt.Errorf("type conversion error in configuration: %w", err)
        }
        
        return nil, fmt.Errorf("configuration error: %w", err)
    }
    
    return cfg, nil
}
```

### Type Conversion Errors

```go
// Purpose: Handle type conversion errors gracefully
// Prerequisites: Environment variables with invalid types
// Expected outcome: Clear error messages for debugging

type Config struct {
    Port    int  `env:"PORT,default:8080"`
    Enabled bool `env:"ENABLED,default:true"`
    Size    int64 `env:"SIZE,default:1024"`
}

func handleTypeErrors() {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        // Examples of type conversion errors:
        // PORT=abc -> strconv.ParseInt error
        // ENABLED=maybe -> strconv.ParseBool error
        // SIZE=huge -> strconv.ParseInt error
        
        log.Printf("Configuration error: %v", err)
        log.Println("Check environment variable types:")
        log.Println("PORT should be a number")
        log.Println("ENABLED should be true/false")
        log.Println("SIZE should be a number")
    }
}
```

## Testing Patterns

### Test Configuration

```go
// Purpose: Use configuration in tests
// Prerequisites: Test environment setup
// Expected outcome: Isolated test configuration

func TestWithConfig(t *testing.T) {
    // Set test environment variables
    os.Setenv("TEST_HOST", "localhost")
    os.Setenv("TEST_PORT", "9999")
    os.Setenv("TEST_DEBUG", "true")
    defer func() {
        os.Unsetenv("TEST_HOST")
        os.Unsetenv("TEST_PORT")
        os.Unsetenv("TEST_DEBUG")
    }()
    
    type TestConfig struct {
        Host  string `env:"TEST_HOST,default:example.com"`
        Port  int    `env:"TEST_PORT,default:8080"`
        Debug bool   `env:"TEST_DEBUG,default:false"`
    }
    
    cfg := &TestConfig{}
    if err := config.Load(cfg); err != nil {
        t.Fatal("Config load failed:", err)
    }
    
    if cfg.Host != "localhost" {
        t.Errorf("Expected localhost, got %s", cfg.Host)
    }
    if cfg.Port != 9999 {
        t.Errorf("Expected 9999, got %d", cfg.Port)
    }
    if !cfg.Debug {
        t.Error("Expected debug to be true")
    }
}
```

### Environment Isolation

```go
// Purpose: Isolate environment variables between tests
// Prerequisites: Multiple tests using environment variables
// Expected outcome: Clean test environment

func TestEnvironmentIsolation(t *testing.T) {
    tests := []struct {
        name     string
        envVars  map[string]string
        expected Config
    }{
        {
            name: "production config",
            envVars: map[string]string{
                "APP_ENV":   "production",
                "APP_DEBUG": "false",
                "APP_PORT":  "80",
            },
            expected: Config{
                Environment: "production",
                Debug:       false,
                Port:        80,
            },
        },
        {
            name: "development config",
            envVars: map[string]string{
                "APP_ENV":   "development",
                "APP_DEBUG": "true",
                "APP_PORT":  "3000",
            },
            expected: Config{
                Environment: "development",
                Debug:       true,
                Port:        3000,
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set environment variables
            for key, value := range tt.envVars {
                os.Setenv(key, value)
            }
            defer func() {
                // Clean up environment variables
                for key := range tt.envVars {
                    os.Unsetenv(key)
                }
            }()
            
            cfg := &Config{}
            if err := config.Load(cfg); err != nil {
                t.Fatal("Config load failed:", err)
            }
            
            // Verify configuration
            if cfg.Environment != tt.expected.Environment {
                t.Errorf("Expected environment %s, got %s", tt.expected.Environment, cfg.Environment)
            }
        })
    }
}
```

## Best Practices

### Environment Variable Naming

```go
// Purpose: Follow consistent naming conventions
// Prerequisites: Understanding of naming best practices
// Expected outcome: Clear, maintainable configuration

type Config struct {
    // Use uppercase with underscores
    DatabaseURL string `env:"DATABASE_URL"`
    
    // Use descriptive prefixes for grouping
    RedisHost     string `env:"REDIS_HOST"`
    RedisPort     int    `env:"REDIS_PORT"`
    RedisPassword string `env:"REDIS_PASSWORD"`
    
    // Use boolean names that read clearly
    EnableMetrics bool `env:"ENABLE_METRICS,default:true"`
    DebugMode     bool `env:"DEBUG_MODE,default:false"`
    
    // Use consistent units
    TimeoutSeconds int   `env:"TIMEOUT_SECONDS,default:30"`
    MaxSizeBytes   int64 `env:"MAX_SIZE_BYTES,default:1048576"`
}
```

### Default Value Guidelines

```go
// Purpose: Provide sensible defaults for all configuration
// Prerequisites: Understanding of application requirements
// Expected outcome: Application works with minimal configuration

type Config struct {
    // Always provide defaults for optional settings
    Host string `env:"HOST,default:0.0.0.0"`
    Port int    `env:"PORT,default:8080"`
    
    // Provide safe defaults for security settings
    Debug             bool `env:"DEBUG,default:false"`
    EnableProfiling   bool `env:"ENABLE_PROFILING,default:false"`
    AllowInsecureMode bool `env:"ALLOW_INSECURE,default:false"`
    
    // Provide reasonable defaults for performance settings
    WorkerCount    int   `env:"WORKER_COUNT,default:4"`
    MaxConnections int   `env:"MAX_CONNECTIONS,default:100"`
    RequestTimeout int   `env:"REQUEST_TIMEOUT,default:30"`
    
    // Don't provide defaults for secrets (force explicit setting)
    APIKey    string `env:"API_KEY"`
    JWTSecret string `env:"JWT_SECRET"`
    DBPassword string `env:"DB_PASSWORD"`
}
```

The config package serves as the foundation for all Beaver Kit packages, providing a consistent and simple way to handle environment-based configuration throughout your applications.