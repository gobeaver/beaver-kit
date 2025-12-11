# [Refactor] Config Package - Vendor caarlos0/env Wrapper - 2025-12-11

## Impact
**High** - Breaking API changes

## Components
- Backend (all packages using config)

## Summary
Complete refactor of the config package to wrap vendored `caarlos0/env` (v11.3.1) and `joho/godotenv` (v1.5.1) libraries. This provides a more robust, feature-rich configuration system with zero external runtime dependencies. All dependent packages have been migrated to the new API.

## Breaking Changes

### Tag Format
```go
// OLD
env:"NAME,default:value"

// NEW
env:"NAME" envDefault:"value"
```

### API Changes
```go
// OLD
config.Load(cfg, config.LoadOptions{Prefix: "X_"})

// NEW
config.Load(cfg, config.WithPrefix("X_"))
```

### GetConfig Signature
```go
// OLD
func GetConfig(opts ...config.LoadOptions) (*Config, error)

// NEW
func GetConfig(opts ...config.Option) (*Config, error)
```

## New Features

### Functional Options
- `config.WithPrefix(prefix)` - Set environment variable prefix
- `config.WithEnvFiles(files...)` - Load specific .env files
- `config.WithoutDotEnv()` - Skip automatic .env loading
- `config.WithRequired()` - Make all fields required unless they have defaults

### Enhanced Tag Support
| Tag | Description |
|-----|-------------|
| `env:"NAME"` | Environment variable name |
| `env:"NAME,required"` | Required field |
| `env:"NAME,notEmpty"` | Must not be empty string |
| `env:"NAME,file"` | Read contents from file path |
| `env:"NAME,expand"` | Expand $VAR or ${VAR} |
| `envDefault:"value"` | Default value |
| `envSeparator:","` | Separator for slices |
| `envPrefix:"PREFIX_"` | Prefix for nested structs |

### New Type Support
- Slices: `[]string`, `[]int`, etc. with custom separators
- Maps: `map[string]string` with key:value format
- Nested structs with `envPrefix` tag
- `url.URL` type
- Custom types implementing `encoding.TextUnmarshaler`

## Files Modified

### Config Package (New Implementation)
| File | Description |
|------|-------------|
| `config/config.go` | New 105-line wrapper with functional options |
| `config/config_test.go` | 315 lines, 13 comprehensive test functions |
| `config/doc.go` | 124 lines of documentation |
| `config/env/` | Vendored caarlos0/env v11.3.1 (4 files) |
| `config/env/CREDITS.md` | Version tracking for vendored code |
| `config/dotenv/CREDITS.md` | Updated version tracking |

### Dependent Packages Updated
| Package | Files |
|---------|-------|
| database | config.go, service.go |
| cache | config.go, service.go |
| oauth | config.go, service.go, multi_config.go, multi_provider_service.go, monitoring.go, middleware.go |
| slack | config.go, service.go |
| captcha | config.go, service.go |
| urlsigner | config.go, service.go |

### Documentation
| File | Description |
|------|-------------|
| `CLAUDE.md` | Updated config architecture section |
| `README.md` | Updated config package examples |
| `integration_test/config_integration_test.go` | Migrated to new API |

## Migration Guide

### Step 1: Update Struct Tags
```go
// Before
type Config struct {
    Host string `env:"HOST,default:localhost"`
    Port int    `env:"PORT,default:8080"`
}

// After
type Config struct {
    Host string `env:"HOST" envDefault:"localhost"`
    Port int    `env:"PORT" envDefault:"8080"`
}
```

### Step 2: Update Load Calls
```go
// Before
config.Load(cfg, config.LoadOptions{Prefix: "MYAPP_"})

// After
config.Load(cfg, config.WithPrefix("MYAPP_"))
```

### Step 3: Update GetConfig Functions
```go
// Before
func GetConfig(opts ...config.LoadOptions) (*Config, error)

// After
func GetConfig(opts ...config.Option) (*Config, error)
```

## Testing
- [x] Unit Tests (13/13 config tests passing)
- [x] Integration Tests (6/6 passing)
- [x] Build Verification (`go build ./...` passes)
- [x] Dependent Package Tests (database, cache, oauth, captcha, urlsigner)

## Security Notes
- All dependencies vendored (no external runtime fetches)
- Version tracking via CREDITS.md files
- Security-first approach: lag 3-6 months behind upstream releases

## Performance
No significant performance changes. The new implementation delegates to caarlos0/env which uses reflection similarly to the previous custom implementation.
