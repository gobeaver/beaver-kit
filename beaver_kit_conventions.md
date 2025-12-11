# Beaver Kit Module Conventions

## Overview


This document defines the standard patterns and conventions for developing beaver-kit modules. All service modules that require configuration should follow these patterns to ensure consistency across the ecosystem.

## Core Pattern

Every service module implements this structure:

```go
package packagename

import (
    "sync"
    "github.com/gobeaver/beaver-kit/config"
)

// Global instance management
var (
    defaultInstance *Service
    defaultOnce     sync.Once
    defaultErr      error
)

// Config defines package configuration
type Config struct {
    // No prefix in struct tags (prefix applied via functional options)
    Field1 string `env:"FIELD1" envDefault:"value"`
    Field2 int    `env:"FIELD2" envDefault:"10"`
}

// Service is the main package type
type Service struct {
    // Internal fields
}

// GetConfig returns config loaded from environment
func GetConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}

// Init initializes the global instance with optional config
func Init(configs ...Config) error {
    defaultOnce.Do(func() {
        var cfg *Config
        if len(configs) > 0 {
            cfg = &configs[0]
        } else {
            cfg, defaultErr = GetConfig()
            if defaultErr != nil {
                return
            }
        }
        
        defaultInstance, defaultErr = New(*cfg)
    })
    
    return defaultErr
}

// New creates a new instance with given config
func New(cfg Config) (*Service, error) {
    // Validation
    if err := validateConfig(cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    // Initialization
    return &Service{
        // Initialize fields
    }, nil
}

// validateConfig checks configuration validity
func validateConfig(cfg Config) error {
    // Package-specific validation
    return nil
}

// Reset clears the global instance (for testing)
func Reset() {
    defaultInstance = nil
    defaultOnce = sync.Once{}
    defaultErr = nil
}

// Package-specific accessor (e.g., DB(), JWT(), Service())
func ServiceName() *Service {
    if defaultInstance == nil {
        Init() // Initialize with defaults if needed
    }
    return defaultInstance
}
```

## Environment Variable Convention

All environment variables use the `BEAVER_` prefix by default, but this is configurable:

```go
type Config struct {
    // No prefix in struct tags (prefix applied via functional options)
    Driver   string `env:"DB_DRIVER" envDefault:"sqlite"`
    Host     string `env:"DB_HOST" envDefault:"localhost"`
    Port     string `env:"DB_PORT" envDefault:"5432"`
}
```

### Configurable Prefix Pattern

Each package must support configurable prefixes via a Builder pattern:

```go
// Builder pattern for custom prefixes
type Builder struct {
    prefix string
}

func WithPrefix(prefix string) *Builder {
    return &Builder{prefix: prefix}
}

func (b *Builder) Init() error {
    cfg := &Config{}
    if err := config.Load(cfg, config.WithPrefix(b.prefix)); err != nil {
        return err
    }
    return Init(*cfg)
}
```

### GetConfig Updates

The `GetConfig()` function must accept functional options:

```go
func GetConfig(opts ...config.Option) (*Config, error) {
    cfg := &Config{}
    // Apply default prefix if not specified
    if len(opts) == 0 {
        opts = append(opts, config.WithPrefix("BEAVER_PACKAGE_"))
    }
    if err := config.Load(cfg, opts...); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

## Error Handling Convention

Define package-specific errors and wrap with context:

```go
// Define standard errors for the package
var (
    ErrInvalidConfig = errors.New("invalid configuration")
    ErrNotInitialized = errors.New("service not initialized")
    ErrConnectionFailed = errors.New("connection failed")
)

// Wrap errors with context
func validateConfig(cfg Config) error {
    if cfg.Field1 == "" {
        return fmt.Errorf("%w: field1 required", ErrInvalidConfig)
    }
    return nil
}
```

## Resource Management

### When to Include Shutdown()

Include a `Shutdown()` function only for packages that manage resources:

```go
func Shutdown(ctx context.Context) error {
    if defaultInstance == nil {
        return nil
    }
    // Package-specific shutdown logic
    return defaultInstance.Close(ctx)
}
```

**Include Shutdown() for:**
- Database connections
- Cache connections
- Message queue clients
- WebSocket managers
- Background workers
- File handles
- Network listeners

**Skip Shutdown() for:**
- Stateless utilities (crypto, validators)
- Simple HTTP clients
- Configuration parsers
- Pure computation packages

### Health Check Pattern

For services that manage connections:

```go
func Health() error {
    if defaultInstance == nil {
        return ErrNotInitialized
    }
    return defaultInstance.Ping()
}
```

## API Flexibility

While packages must implement the core pattern, they are encouraged to provide additional functionality:

### Additional Constructors

```go
// Core pattern (required)
func New(cfg Config) (*Service, error)

// Additional constructors (encouraged)
func NewGoogleCaptcha(siteKey, secretKey string, version int) *GoogleCaptchaService
func NewWithDefaults() (*Service, error)
func NewFromURL(url string) (*Service, error)
```

### Factory Functions

When multiple implementations exist:

```go
func NewFromConfig() (CaptchaService, error) {
    cfg, err := GetConfig()
    if err != nil {
        return nil, err
    }
    
    switch cfg.Provider {
    case "recaptcha":
        return NewGoogleCaptcha(cfg.SiteKey, cfg.SecretKey, cfg.Version), nil
    case "hcaptcha":
        return NewHCaptcha(cfg.SiteKey, cfg.SecretKey), nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
```

### Interface-Based Design

Use interfaces to allow multiple implementations:

```go
type CaptchaService interface {
    Validate(ctx context.Context, token string, remoteIP string) (bool, error)
    GenerateHTML() string
}
```

### The Convention Is The Foundation, Not The Limit

- ✅ DO implement the core pattern for consistency
- ✅ DO add domain-specific methods and types
- ✅ DO provide convenience constructors
- ✅ DO support multiple implementations via interfaces
- ❌ DON'T break the core pattern
- ❌ DON'T make the API unnecessarily complex

## Usage Examples

### Database Package Implementation

```go
package database

import (
    "context"
    "errors"
    "sync"
    "github.com/gobeaver/beaver-kit/config"
    "gorm.io/gorm"
)

var (
    defaultDB   *gorm.DB
    defaultOnce sync.Once
    defaultErr  error
)

type Config struct {
    Driver   string `env:"DB_DRIVER" envDefault:"sqlite"`
    Host     string `env:"DB_HOST" envDefault:"localhost"`
    Port     string `env:"DB_PORT"`
    Database string `env:"DB_DATABASE" envDefault:"beaver.db"`
    Username string `env:"DB_USERNAME"`
    Password string `env:"DB_PASSWORD"`
}

func GetConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}

func Init(configs ...Config) error {
    defaultOnce.Do(func() {
        var cfg *Config
        if len(configs) > 0 {
            cfg = &configs[0]
        } else {
            cfg, defaultErr = GetConfig()
            if defaultErr != nil {
                return
            }
        }
        
        defaultDB, defaultErr = New(*cfg)
    })
    
    return defaultErr
}

func New(cfg Config) (*gorm.DB, error) {
    if cfg.Driver == "" {
        return nil, errors.New("database driver required")
    }
    
    // Create connection based on driver
    // Configure pool settings
    // Return initialized DB
    return db, nil
}

func DB() *gorm.DB {
    if defaultDB == nil {
        Init()
    }
    return defaultDB
}

func Reset() {
    defaultDB = nil
    defaultOnce = sync.Once{}
    defaultErr = nil
}

func Shutdown(ctx context.Context) error {
    if defaultDB == nil {
        return nil
    }
    sqlDB, err := defaultDB.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}
```

### Captcha Package Implementation

```go
package captcha

import (
    "sync"
    "github.com/gobeaver/beaver-kit/config"
)

var (
    defaultService CaptchaService
    defaultOnce    sync.Once
    defaultErr     error
)

type Config struct {
    Provider  string `env:"CAPTCHA_PROVIDER" envDefault:"recaptcha"`
    SiteKey   string `env:"CAPTCHA_SITE_KEY"`
    SecretKey string `env:"CAPTCHA_SECRET_KEY"`
    Version   int    `env:"CAPTCHA_VERSION" envDefault:"2"`
}

func GetConfig() (*Config, error) {
    cfg := &Config{}
    if err := config.Load(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}

func Init(configs ...Config) error {
    defaultOnce.Do(func() {
        var cfg *Config
        if len(configs) > 0 {
            cfg = &configs[0]
        } else {
            cfg, defaultErr = GetConfig()
            if defaultErr != nil {
                return
            }
        }
        
        defaultService, defaultErr = New(*cfg)
    })
    
    return defaultErr
}

func New(cfg Config) (CaptchaService, error) {
    switch cfg.Provider {
    case "recaptcha":
        return NewGoogleCaptcha(cfg.SiteKey, cfg.SecretKey, cfg.Version), nil
    case "hcaptcha":
        return NewHCaptcha(cfg.SiteKey, cfg.SecretKey), nil
    case "turnstile":
        return NewTurnstile(cfg.SiteKey, cfg.SecretKey), nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}

func Service() CaptchaService {
    if defaultService == nil {
        Init()
    }
    return defaultService
}

func Reset() {
    defaultService = nil
    defaultOnce = sync.Once{}
    defaultErr = nil
}

// No Shutdown() needed - stateless service
```

## Testing Guidelines

### Using Reset in Tests

```go
func TestDatabaseConnection(t *testing.T) {
    defer database.Reset() // Clean up after test
    
    testConfig := database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
    }
    
    err := database.Init(testConfig)
    if err != nil {
        t.Fatal(err)
    }
    
    // Test logic
    db := database.DB()
    // ...
}
```

### Testing Multiple Configurations

```go
func TestMultipleInstances(t *testing.T) {
    // Test primary instance
    primary, err := database.New(database.Config{
        Driver:   "postgres",
        Host:     "primary.db",
        Database: "test",
    })
    assert.NoError(t, err)
    
    // Test replica instance
    replica, err := database.New(database.Config{
        Driver:   "postgres",
        Host:     "replica.db",
        Database: "test",
    })
    assert.NoError(t, err)
    
    // Verify they're different instances
    assert.NotEqual(t, primary, replica)
}
```

## Documentation Requirements

Each package must include:

1. **README.md** with:
    - Purpose and features
    - Installation instructions
    - Usage examples for all initialization methods
    - Configuration options table

2. **Config struct documentation**:
   ```go
   type Config struct {
       // Driver specifies the database driver (postgres, mysql, sqlite)
       Driver string `env:"DB_DRIVER" envDefault:"sqlite"`

       // Host is the database server hostname
       Host string `env:"DB_HOST" envDefault:"localhost"`
   }
   ```

3. **Example files** showing:
    - Zero-config usage
    - Environment-based configuration
    - Direct configuration
    - Common use cases

## Debugging Configuration

Enable configuration debugging with:

```bash
BEAVER_CONFIG_DEBUG=true ./myapp
```

This will print all loaded configuration values:
```
[BEAVER] BEAVER_DB_DRIVER=postgres
[BEAVER] BEAVER_DB_HOST=localhost
[BEAVER] BEAVER_DB_DATABASE=myapp
```

## Context Handling

### Context Should Be Optional

Keep the existing simple signatures for initialization:

```go
// Keep these simple - most services don't need context during init
func Init(configs ...Config) error
func New(cfg Config) (*Service, error)

// Add context variants only where needed
func InitWithContext(ctx context.Context, configs ...Config) error
func NewWithContext(ctx context.Context, cfg Config) (*Service, error)
```

**Rationale:**
- Many services don't need context during initialization (JWT, validators, captcha)
- Keeps the simple API simple - most users just want `Init()`
- Backwards compatible
- Context is more useful for operations than initialization

### When to Add Context Variants

Add `WithContext` variants only for:
- Services making network calls during init (database connections, API clients)
- Services that might hang during initialization
- Services needing cancellable startup

### Better Approach for Context

Add context to methods that actually need it:

```go
// Init stays simple
db := database.DB()

// Operations use context
err := db.WithContext(ctx).Find(&users)
valid, err := captcha.ValidateWithContext(ctx, token, ip)
msg, err := slack.SendMessageWithContext(ctx, channel, text)
```

This preserves the zero-config philosophy while supporting advanced use cases.

## Best Practices

1. **Remove prefixes from struct tags** - let the config package apply prefixes
2. **Provide sensible defaults** via `envDefault` tag
3. **Support configurable prefixes** via Builder pattern and functional options
4. **Validate configuration** in `New()` with descriptive errors
5. **Document all config fields** with clear comments
6. **Use fmt.Errorf with %w** for error wrapping
7. **Keep global state minimal** - only the instance, once, and error
8. **Make zero-config work** - `Init()` with no args should succeed
9. **Test with Reset()** to ensure clean state between tests
10. **Keep context optional** - add `WithContext` variants only when necessary

## Migration Guide

### For New Packages

To adopt the configurable prefix pattern:

1. Add Config struct with `env` and `envDefault` tags (no prefix in tags)
2. Implement `GetConfig(opts ...config.Option)` using `config.Load()`
3. Rename constructors to `New()`
4. Add `Init()` with `sync.Once`
5. Add `Builder` type with `WithPrefix()` method
6. Add package-specific accessor (e.g., `DB()`, `Service()`)
7. Add `Reset()` for testing
8. Add `Shutdown()` if managing resources
9. Update documentation and examples

### For Existing Packages (Migration to New Config API)

To migrate existing packages:

1. **Update struct tags to new format:**
   ```go
   // Before
   type Config struct {
       Driver string `env:"BEAVER_DB_DRIVER,default:sqlite"`
   }

   // After
   type Config struct {
       Driver string `env:"DB_DRIVER" envDefault:"sqlite"`
   }
   ```

2. **Update GetConfig to accept functional options:**
   ```go
   // Before
   func GetConfig(opts ...config.LoadOptions) (*Config, error) {
       cfg := &Config{}
       if err := config.Load(cfg, opts...); err != nil {
           return nil, err
       }
       return cfg, nil
   }

   // After
   func GetConfig(opts ...config.Option) (*Config, error) {
       cfg := &Config{}
       // Apply default prefix if not specified
       if len(opts) == 0 {
           opts = append(opts, config.WithPrefix("BEAVER_DB_"))
       }
       if err := config.Load(cfg, opts...); err != nil {
           return nil, err
       }
       return cfg, nil
   }
   ```

3. **Update Builder pattern:**
   ```go
   type Builder struct {
       prefix string
   }

   func WithPrefix(prefix string) *Builder {
       return &Builder{prefix: prefix}
   }

   func (b *Builder) Init() error {
       cfg := &Config{}
       if err := config.Load(cfg, config.WithPrefix(b.prefix)); err != nil {
           return err
       }
       return Init(*cfg)
   }
   ```

4. **Update tests to use new API:**
   ```go
   // In test files
   cfg, err := packagename.GetConfig(config.WithPrefix(""))
   ```

This migration maintains backward compatibility while enabling the new functional options API.

## Non-Service Modules

Utility packages that don't require global state (like validators, pure functions) don't need to follow this pattern. They can simply export functions directly:

```go
package validator

// No global state needed
func ValidateEmail(email string) error {
    // Direct function implementation
}
```

This convention applies only to service modules that benefit from configuration and global instance management.

## Multi-Instance Pattern

For packages that commonly need multiple instances:

```go
type Manager struct {
    instances map[string]*Service
    mu        sync.RWMutex
}

func NewManager() *Manager {
    return &Manager{
        instances: make(map[string]*Service),
    }
}

func (m *Manager) Get(name string) (*Service, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    service, ok := m.instances[name]
    if !ok {
        return nil, fmt.Errorf("instance %s not found", name)
    }
    return service, nil
}

func (m *Manager) Add(name string, cfg Config) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    service, err := New(cfg)
    if err != nil {
        return err
    }
    
    m.instances[name] = service
    return nil
}
```