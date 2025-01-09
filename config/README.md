# Config Package

Environment variable configuration loader with struct tag support and default values.

## Features

- Load environment variables into Go structs using reflection
- Support for default values via struct tags
- Type conversion for string, int, int64, and bool fields
- Simple, zero-dependency implementation

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/gobeaver/beaver-kit/config"
)

type AppConfig struct {
    DatabaseURL string `env:"DATABASE_URL,default:postgres://localhost/myapp"`
    Port        int    `env:"PORT,default:8080"`
    Debug       bool   `env:"DEBUG,default:false"`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        panic(err)
    }
    
    fmt.Printf("Database: %s\n", cfg.DatabaseURL)
    fmt.Printf("Port: %d\n", cfg.Port)
    fmt.Printf("Debug: %t\n", cfg.Debug)
}
```

### Environment Variables

```bash
DATABASE_URL=postgres://prod-server/myapp
PORT=3000
DEBUG=true
```

## Struct Tag Format

```go
type Config struct {
    Field string `env:"ENV_VAR_NAME,default:defaultvalue"`
}
```

- `ENV_VAR_NAME`: Environment variable to read from
- `default:value`: Optional default value if environment variable is not set

## Supported Types

- `string`: Direct assignment
- `int` and `int64`: Parsed using `strconv.ParseInt`
- `bool`: Parsed using `strconv.ParseBool` (accepts: true, false, 1, 0, t, f, TRUE, FALSE, True, False)

## Package Integration Pattern

Many packages use this config pattern for initialization:

```go
package mypackage

type Config struct {
    APIKey string `env:"MYPACKAGE_API_KEY"`
    Host   string `env:"MYPACKAGE_HOST,default:localhost"`
    Port   int    `env:"MYPACKAGE_PORT,default:8080"`
}

func GetConfig() *Config {
    cfg := &Config{}
    config.Load(cfg)
    return cfg
}
```

## Implementation

The `config.Load()` function uses reflection to:
1. Iterate through struct fields
2. Read `env` tags to get environment variable names and defaults
3. Load values from environment or use defaults
4. Convert string values to appropriate field types