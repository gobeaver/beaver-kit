# Database Package

A flexible, SQL-first database package for Go applications with optional GORM support and zero CGO dependencies.

## Features

- **Pure Go Implementation** - No CGO required, all drivers are pure Go
- **SQL-First Design** - Direct access to `*sql.DB` for maximum control
- **Optional GORM Support** - Enable ORM functionality when needed
- **Multi-Database Support** - PostgreSQL, MySQL, SQLite, and Turso/LibSQL
- **Environment Configuration** - Easy setup via environment variables
- **Fluent Interface** - Chain configuration methods for clean API
- **Connection Pooling** - Built-in connection pool management
- **Custom Prefixes** - Configurable environment variable prefixes
- **Unified Database Wrapper** - Access both SQL and GORM through single interface

## Installation

```bash
go get github.com/gobeaver/beaver-kit/database
```

All database drivers are pure Go implementations, ensuring easy cross-compilation and deployment without CGO dependencies.

## Quick Start

### Fluent Interface API

The package provides a clean fluent interface for configuration and initialization:

```go
package main

import (
    "log"
    "github.com/gobeaver/beaver-kit/database"
)

func main() {
    // Global initialization with fluent interface
    if err := database.WithGORM().Init(); err != nil {
        log.Fatal(err)
    }
    
    // Custom prefix + GORM
    if err := database.WithPrefix("APP_").WithGORM().Init(); err != nil {
        log.Fatal(err)
    }
    
    // Create instance connections
    db, err := database.WithGORM().Connect()
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Access SQL database
    sqlDB := db.SQL()
    rows, err := sqlDB.Query("SELECT * FROM users")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    
    // Access GORM (if enabled)
    gormDB, err := db.GORM()
    if err != nil {
        log.Fatal(err)
    }
    
    var users []User
    gormDB.Find(&users)
}
```

### Traditional Global API

```go
// Traditional initialization (still supported)
if err := database.Init(); err != nil {
    log.Fatal(err)
}

// Get global instances
db := database.DB()
gormDB, err := database.GORM()

// Enable GORM explicitly
if err := database.InitWithGORM(); err != nil {
    log.Fatal(err)
}
```

## Configuration

### Environment Variables

```bash
# Database driver: postgres, mysql, sqlite, turso, libsql
BEAVER_DB_DRIVER=postgres

# Connection details (for traditional databases)
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=user
BEAVER_DB_PASSWORD=secret

# Or use a connection URL (overrides individual settings)
BEAVER_DB_URL=postgres://user:pass@localhost:5432/myapp

# For Turso/LibSQL
BEAVER_DB_URL=libsql://your-database.turso.io
BEAVER_DB_AUTH_TOKEN=your-auth-token

# Connection pool settings
BEAVER_DB_MAX_OPEN_CONNS=25
BEAVER_DB_MAX_IDLE_CONNS=5
BEAVER_DB_CONN_MAX_LIFETIME=300  # seconds
BEAVER_DB_CONN_MAX_IDLE_TIME=60  # seconds

# SSL/TLS
BEAVER_DB_SSL_MODE=require      # For PostgreSQL
BEAVER_DB_TLS_CONFIG=true       # For MySQL

# Debug mode
BEAVER_DB_DEBUG=false

# ORM Support (optional)
BEAVER_DB_ORM=gorm              # Enable GORM
BEAVER_DB_DISABLE_ORM_LOG=true  # Disable GORM logging

# Migrations (optional, for GORM)
BEAVER_DB_AUTO_MIGRATE=false
BEAVER_DB_MIGRATIONS_PATH=migrations
```

### Programmatic Configuration

```go
cfg := database.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     "5432",
    Database: "myapp",
    Username: "user",
    Password: "secret",
    MaxOpenConns: 25,
    MaxIdleConns: 5,
}

db, err := database.New(cfg)
```

## Database Drivers

All drivers are pure Go implementations:

- **PostgreSQL** - Uses `jackc/pgx/v5` (high-performance, pure Go)
- **MySQL** - Uses `go-sql-driver/mysql` (pure Go)
- **SQLite** - Uses `modernc.org/sqlite` (pure Go, no CGO)
- **Turso/LibSQL** - Uses `tursodatabase/libsql-client-go` (pure Go)

## API Reference

### Fluent Interface

The new fluent interface provides a clean, chainable API:

```go
// Top-level functions for fluent initialization
database.New()                    // Create with defaults
database.WithPrefix("APP_")       // Create with custom prefix  
database.WithGORM()              // Create with GORM enabled

// Fluent methods (chainable)
db := database.New().WithGORM().WithPrefix("CUSTOM_")

// Initialization methods
err := db.Init()                 // Global initialization
instance, err := db.Connect()    // Create new instance

// Database wrapper methods
sqlDB := db.SQL()                // Get *sql.DB
gormDB, err := db.GORM()         // Get *gorm.DB (with error)
gormDB := db.MustGORM()          // Get *gorm.DB (panic on error)
err := db.Close()                // Close connection
err := db.Ping()                 // Health check
stats := db.Stats()              // Connection stats
```

### Traditional API (Still Supported)

```go
// Global initialization
err := database.Init()                    // Basic init
err := database.InitWithGORM()           // With GORM
database.MustInitWithGORM()              // Panic on error

// Global instances  
db := database.DB()                      // Global SQL DB
gormDB, err := database.GORM()          // Global GORM
gormDB := database.MustGORM()           // Global GORM (panic)

// New instances
sqlDB, err := database.NewSQL(config)    // New SQL connection
gormDB, err := database.NewWithGORM()   // New GORM instance
gormDB, err := database.NewGORM(cfg, db) // GORM from existing SQL
```

### Custom Environment Variable Prefixes

The package supports configurable environment variable prefixes for multi-tenant applications:

```go
// Default prefix (BEAVER_)
database.New().Init()

// Custom prefix (APP_DB_DRIVER, APP_DB_HOST, etc.)
database.WithPrefix("APP_").Init()

// Multiple configurations
prodDB := database.WithPrefix("PROD_").WithGORM()
testDB := database.WithPrefix("TEST_").WithGORM()
```

### Error Handling

The package provides specific error types:

```go
var (
    ErrNotInitialized = errors.New("database not initialized")
    ErrInvalidDriver  = errors.New("invalid database driver")
    ErrInvalidConfig  = errors.New("invalid database configuration")
    ErrGORMNotEnabled = errors.New("GORM not enabled")
)

// Example usage
gormDB, err := db.GORM()
if err == database.ErrGORMNotEnabled {
    // Handle GORM not being enabled
}
```

## Examples

### PostgreSQL with SSL

```bash
BEAVER_DB_DRIVER=postgres
BEAVER_DB_HOST=prod-db.example.com
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=appuser
BEAVER_DB_PASSWORD=secret
BEAVER_DB_SSL_MODE=require
```

### MySQL with Custom Parameters

```bash
BEAVER_DB_DRIVER=mysql
BEAVER_DB_HOST=localhost
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=root
BEAVER_DB_PASSWORD=secret
BEAVER_DB_PARAMS="timeout=10s&readTimeout=30s"
```

### SQLite File Database

```bash
BEAVER_DB_DRIVER=sqlite
BEAVER_DB_DATABASE=/path/to/database.db
```

### Turso Cloud Database

```bash
BEAVER_DB_DRIVER=turso
BEAVER_DB_URL=libsql://my-db.turso.io
BEAVER_DB_AUTH_TOKEN=your-auth-token
```

### Fluent Interface Examples

```go
// Global initialization patterns
database.WithGORM().Init()                      // Global with GORM
database.WithPrefix("APP_").WithGORM().Init()   // Custom prefix + GORM
database.New().WithGORM().Init()                // Explicit new + GORM

// Instance creation patterns
db, err := database.WithGORM().Connect()        // Instance with GORM
db, err := database.WithPrefix("APP_").Connect() // Custom prefix instance

// Multi-environment setup
prodDB, err := database.WithPrefix("PROD_").WithGORM().Connect()
testDB, err := database.WithPrefix("TEST_").WithGORM().Connect()

// Traditional + fluent mixing
database.Init() // Initialize global
db := database.New().WithGORM() // Create fluent instance
instance, err := db.Connect()
```

### Using the Database Wrapper

```go
// Create connection with fluent interface
db, err := database.WithGORM().Connect()
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Access SQL database for custom queries
sqlDB := db.SQL()
rows, err := sqlDB.Query("SELECT * FROM users WHERE active = ?", true)
if err != nil {
    log.Fatal(err)
}
defer rows.Close()

// Access GORM for ORM operations
gormDB, err := db.GORM()
if err != nil {
    log.Fatal(err)
}

type User struct {
    ID   uint
    Name string
}

var users []User
gormDB.Find(&users)

// Health check
if err := db.Ping(); err != nil {
    log.Printf("Database unhealthy: %v", err)
}

// Get connection statistics
stats := db.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)
```

## Testing

For testing, create isolated database instances:

```go
func TestMyFunction(t *testing.T) {
    // Create test database instance (doesn't affect global state)
    db, err := database.New().Connect()
    if err != nil {
        // Fallback to in-memory SQLite for tests
        cfg := database.Config{
            Driver:   "sqlite",
            Database: ":memory:",
        }
        sqlDB, err := database.NewSQL(cfg)
        if err != nil {
            t.Fatal(err)
        }
        db = &database.Database{} // Create wrapper if needed
    }
    defer db.Close()
    
    // Run tests with isolated instance...
}
```

## Migration Support

While this package focuses on database connectivity, migrations should be handled by dedicated tools:

- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Database agnostic migration tool
- [pressly/goose](https://github.com/pressly/goose) - SQL migration tool
- GORM AutoMigrate - For simple schema updates when using GORM

## Best Practices

1. **Use Fluent Interface** - Prefer `database.WithGORM().Init()` for clean configuration
2. **Environment Variable Prefixes** - Use custom prefixes for multi-tenant applications
3. **Instance vs Global** - Use `.Connect()` for isolated instances, `.Init()` for global state
4. **Close Resources** - Always close Database instances and SQL rows/statements
5. **Error Handling** - Check for specific errors like `ErrGORMNotEnabled`
6. **Health Monitoring** - Use `db.Ping()` for health checks
7. **Testing Isolation** - Create separate instances for tests, avoid global state

## Migration Patterns

### Using with GORM AutoMigrate

```go
// Initialize with GORM
db, err := database.WithGORM().Connect()
if err != nil {
    log.Fatal(err)
}

// Get GORM instance for migrations
gormDB, err := db.GORM()
if err != nil {
    log.Fatal(err)
}

// Auto-migrate your models
type User struct {
    ID   uint
    Name string
    Email string `gorm:"uniqueIndex"`
}

if err := gormDB.AutoMigrate(&User{}); err != nil {
    log.Fatal(err)
}
```

### Integration with Migration Tools

```go
// Get raw SQL connection for migration tools
db, err := database.New().Connect()
if err != nil {
    log.Fatal(err)
}

sqlDB := db.SQL()

// Use with golang-migrate
import "github.com/golang-migrate/migrate/v4"
import "github.com/golang-migrate/migrate/v4/database/postgres"

driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
m, err := migrate.NewWithDatabaseInstance(
    "file://migrations",
    "postgres", driver)

if err := m.Up(); err != nil {
    log.Fatal(err)
}
```

## License

See the main Beaver Kit license.