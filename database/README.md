# Database Package

A flexible, SQL-first database package for Go applications with optional GORM support and zero CGO dependencies.

## Features

- **Pure Go Implementation** - No CGO required, all drivers are pure Go
- **SQL-First Design** - Direct access to `*sql.DB` for maximum control
- **Optional GORM Support** - Enable ORM functionality when needed
- **Multi-Database Support** - PostgreSQL, MySQL, SQLite, and Turso/LibSQL
- **Environment Configuration** - Easy setup via environment variables
- **Connection Pooling** - Built-in connection pool management
- **Health Checks** - Monitor database connectivity
- **Transaction Helpers** - Simplified transaction handling

## Installation

```bash
go get github.com/gobeaver/beaver-kit/database
```

All database drivers are pure Go implementations, ensuring easy cross-compilation and deployment without CGO dependencies.

## Quick Start

### Basic Usage (SQL)

```go
package main

import (
    "context"
    "log"
    
    "github.com/gobeaver/beaver-kit/database"
)

func main() {
    // Initialize with environment variables
    if err := database.Init(); err != nil {
        log.Fatal(err)
    }
    defer database.Shutdown(context.Background())
    
    // Get the global DB instance
    db := database.DB()
    
    // Execute queries
    rows, err := db.Query("SELECT * FROM users WHERE active = ?", true)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    
    // Use convenience functions
    result, err := database.Exec(context.Background(), 
        "INSERT INTO users (name, email) VALUES (?, ?)", 
        "John Doe", "john@example.com")
}
```

### With GORM Support

```go
// Enable GORM during initialization
gormDB, err := database.WithGORM()
if err != nil {
    log.Fatal(err)
}

// Or enable via environment variable
// BEAVER_DB_ORM=gorm

// Then use GORM
var users []User
gormDB.Find(&users)
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

### Initialization Functions

```go
// Initialize with environment variables
err := database.Init()

// Initialize with custom config
err := database.Init(config)

// Initialize with GORM support
gormDB, err := database.WithGORM()

// Must variants (panic on error)
database.MustInit()
```

### Getting Database Instances

```go
// Get global SQL database
db := database.DB()

// Get global GORM instance (error if not enabled)
gormDB, err := database.GORM()

// Must variant for GORM
gormDB := database.MustGORM()
```

### Creating New Instances

```go
// Create new SQL connection
db, err := database.New(config)

// Create from environment
db, err := database.NewFromEnv()

// Create GORM from existing SQL connection
gormDB, err := database.NewGORM(config, sqlDB)
```

### Query Helpers

```go
// Execute query without results
result, err := database.Exec(ctx, "DELETE FROM users WHERE id = ?", id)

// Query multiple rows
rows, err := database.Query(ctx, "SELECT * FROM users")

// Query single row
row := database.QueryRow(ctx, "SELECT * FROM users WHERE id = ?", id)

// Prepare statement
stmt, err := database.Prepare(ctx, "SELECT * FROM users WHERE email = ?")
```

### Transaction Support

```go
err := database.Transaction(ctx, func(tx *sql.Tx) error {
    // Execute queries within transaction
    _, err := tx.Exec("INSERT INTO users ...")
    if err != nil {
        return err // Will rollback
    }
    
    _, err = tx.Exec("UPDATE accounts ...")
    return err // Will commit if nil, rollback if error
})
```

### Health & Monitoring

```go
// Check database health
err := database.Health(ctx)

// Check health status
if database.IsHealthy() {
    // Database is reachable
}

// Get connection statistics
stats := database.Stats()
fmt.Printf("Open connections: %d\n", stats.OpenConnections)
```

### Cleanup

```go
// Graceful shutdown
err := database.Shutdown(ctx)

// Reset global instance (for testing)
database.Reset()
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

### Using Transactions

```go
err := database.Transaction(ctx, func(tx *sql.Tx) error {
    // Debit account
    _, err := tx.Exec(`
        UPDATE accounts 
        SET balance = balance - ? 
        WHERE id = ? AND balance >= ?`,
        amount, fromID, amount)
    if err != nil {
        return err
    }
    
    // Credit account
    _, err = tx.Exec(`
        UPDATE accounts 
        SET balance = balance + ? 
        WHERE id = ?`,
        amount, toID)
    
    return err
})
```

### Health Check Endpoint

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    if err := database.Health(ctx); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error":  err.Error(),
        })
        return
    }
    
    stats := database.Stats()
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "stats":  stats,
    })
}
```

## Testing

For testing, you can use the Reset function to clear the global instance:

```go
func TestMyFunction(t *testing.T) {
    defer database.Reset()
    
    // Initialize with test configuration
    err := database.Init(database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
    })
    if err != nil {
        t.Fatal(err)
    }
    
    // Run tests...
}
```

## Migration Support

While this package focuses on database connectivity, migrations should be handled by dedicated tools:

- [golang-migrate/migrate](https://github.com/golang-migrate/migrate) - Database agnostic migration tool
- [pressly/goose](https://github.com/pressly/goose) - SQL migration tool
- GORM AutoMigrate - For simple schema updates when using GORM

## Best Practices

1. **Initialize Early** - Call `database.Init()` in your main function or init
2. **Use Context** - Always pass context for cancellation and timeouts
3. **Close Resources** - Always close rows and statements when done
4. **Handle Errors** - Check all errors, especially in transactions
5. **Monitor Health** - Use health checks in production
6. **Graceful Shutdown** - Call `database.Shutdown()` before exiting

## Error Handling

The package defines several error types:

```go
var (
    ErrNotInitialized = errors.New("database not initialized")
    ErrInvalidDriver  = errors.New("invalid database driver")
    ErrInvalidConfig  = errors.New("invalid database configuration")
    ErrGORMNotEnabled = errors.New("GORM not enabled")
)
```

Always check for these errors when initializing or using the database.

## License

See the main Beaver Kit license.