---
title: "Database Package API Reference"
tags: ["database", "sql", "gorm", "postgresql", "mysql", "sqlite", "turso"]
prerequisites:
  - "getting-started"
  - "config"
relatedDocs:
  - "cache"
  - "integration-patterns"
---

# Database Package

## Overview

The database package provides a flexible, SQL-first database abstraction for Go applications with optional GORM support and zero CGO dependencies. It supports multiple database drivers and follows Beaver Kit's environment-first configuration approach.

```json
{
  "@context": "http://schema.org",
  "@type": "TechArticle",
  "name": "Database Package API Reference",
  "about": "SQL-first database abstraction with multiple driver support",
  "programmingLanguage": "Go",
  "codeRepository": "https://github.com/gobeaver/beaver-kit",
  "keywords": ["database", "sql", "orm", "postgresql", "mysql", "sqlite"]
}
```

## Key Features

- **Pure Go Implementation** - No CGO required for any database driver
- **SQL-First Design** - Direct access to `*sql.DB` for maximum control
- **Optional GORM Support** - Enable ORM functionality when needed
- **Multi-Database Support** - PostgreSQL, MySQL, SQLite, and Turso/LibSQL
- **Environment Configuration** - Zero-config initialization via environment variables
- **Connection Pooling** - Built-in connection pool management with health monitoring
- **Transaction Helpers** - Simplified transaction handling with automatic rollback
- **Health Checks** - Monitor database connectivity and performance

## Quick Start

### Environment Configuration

```bash
# Purpose: Configure database connection via environment variables
# Prerequisites: Database server running and accessible
# Expected outcome: Database package ready for initialization

# Basic configuration
BEAVER_DB_DRIVER=postgres
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=user
BEAVER_DB_PASSWORD=secret

# Or use connection URL (overrides individual settings)
BEAVER_DB_URL=postgres://user:secret@localhost:5432/myapp

# Connection pool settings
BEAVER_DB_MAX_OPEN_CONNS=25
BEAVER_DB_MAX_IDLE_CONNS=5
BEAVER_DB_CONN_MAX_LIFETIME=300  # seconds
BEAVER_DB_CONN_MAX_IDLE_TIME=60  # seconds

# Optional GORM support
BEAVER_DB_ORM=gorm
BEAVER_DB_DEBUG=true
```

### Basic Usage

```go
// Purpose: Initialize and use database with SQL-first approach
// Prerequisites: Environment variables configured
// Expected outcome: Database connection established and query executed

package main

import (
    "context"
    "log"
    
    "github.com/gobeaver/beaver-kit/database"
)

func main() {
    // Initialize from environment
    if err := database.Init(); err != nil {
        log.Fatal("Database initialization failed:", err)
    }
    defer database.Shutdown(context.Background())
    
    // Get the database instance
    db := database.DB()
    
    // Execute queries
    rows, err := db.Query("SELECT id, name, email FROM users WHERE active = ?", true)
    if err != nil {
        log.Fatal("Query failed:", err)
    }
    defer rows.Close()
    
    // Process results
    for rows.Next() {
        var id int64
        var name, email string
        if err := rows.Scan(&id, &name, &email); err != nil {
            log.Printf("Scan error: %v", err)
            continue
        }
        log.Printf("User: %d - %s (%s)", id, name, email)
    }
}
```

## Configuration

### Config Struct

```go
type Config struct {
    // Database driver: postgres, mysql, sqlite, turso, libsql
    Driver string `env:"BEAVER_DB_DRIVER,default:sqlite"`
    
    // Connection details
    Host     string `env:"BEAVER_DB_HOST,default:localhost"`
    Port     string `env:"BEAVER_DB_PORT"`                    // Driver default if empty
    Database string `env:"BEAVER_DB_DATABASE,default:app.db"` // SQLite file path for sqlite
    Username string `env:"BEAVER_DB_USERNAME"`
    Password string `env:"BEAVER_DB_PASSWORD"`
    
    // Connection URL (overrides individual settings)
    URL string `env:"BEAVER_DB_URL"`
    
    // SSL/TLS settings
    SSLMode    string `env:"BEAVER_DB_SSL_MODE"`        // PostgreSQL: disable, require, verify-ca, verify-full
    TLSConfig  string `env:"BEAVER_DB_TLS_CONFIG"`      // MySQL: true, false, skip-verify, preferred
    CertFile   string `env:"BEAVER_DB_CERT_FILE"`       // Client certificate file
    KeyFile    string `env:"BEAVER_DB_KEY_FILE"`        // Client key file
    CAFile     string `env:"BEAVER_DB_CA_FILE"`         // CA certificate file
    
    // Connection pool settings
    MaxOpenConns    int           `env:"BEAVER_DB_MAX_OPEN_CONNS,default:25"`
    MaxIdleConns    int           `env:"BEAVER_DB_MAX_IDLE_CONNS,default:5"`
    ConnMaxLifetime time.Duration `env:"BEAVER_DB_CONN_MAX_LIFETIME,default:300s"`
    ConnMaxIdleTime time.Duration `env:"BEAVER_DB_CONN_MAX_IDLE_TIME,default:60s"`
    
    // Additional connection parameters
    Params map[string]string `env:"BEAVER_DB_PARAMS"` // JSON string: {"param1":"value1"}
    
    // ORM settings
    ORM              string `env:"BEAVER_DB_ORM"`                     // "gorm" to enable
    DisableORMLog    bool   `env:"BEAVER_DB_DISABLE_ORM_LOG,default:false"`
    AutoMigrate      bool   `env:"BEAVER_DB_AUTO_MIGRATE,default:false"`
    MigrationsPath   string `env:"BEAVER_DB_MIGRATIONS_PATH,default:migrations"`
    
    // Debug and monitoring
    Debug           bool          `env:"BEAVER_DB_DEBUG,default:false"`
    SlowQueryTime   time.Duration `env:"BEAVER_DB_SLOW_QUERY_TIME,default:200ms"`
    HealthCheckTime time.Duration `env:"BEAVER_DB_HEALTH_CHECK_TIME,default:30s"`
    
    // Turso/LibSQL specific
    AuthToken string `env:"BEAVER_DB_AUTH_TOKEN"` // Required for Turso
}
```

### Environment Variables Reference

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| **Connection** |
| `BEAVER_DB_DRIVER` | Database driver | `sqlite` | `postgres`, `mysql`, `sqlite`, `turso` |
| `BEAVER_DB_HOST` | Database host | `localhost` | `prod-db.example.com` |
| `BEAVER_DB_PORT` | Database port | Driver default | `5432`, `3306` |
| `BEAVER_DB_DATABASE` | Database name/file | `app.db` | `myapp`, `/path/to/db.sqlite` |
| `BEAVER_DB_USERNAME` | Database username | - | `dbuser` |
| `BEAVER_DB_PASSWORD` | Database password | - | `secure_password` |
| `BEAVER_DB_URL` | Full connection URL | - | `postgres://user:pass@host:5432/db` |
| **Pool Settings** |
| `BEAVER_DB_MAX_OPEN_CONNS` | Maximum open connections | `25` | `50` |
| `BEAVER_DB_MAX_IDLE_CONNS` | Maximum idle connections | `5` | `10` |
| `BEAVER_DB_CONN_MAX_LIFETIME` | Connection max lifetime | `300s` | `1h` |
| `BEAVER_DB_CONN_MAX_IDLE_TIME` | Connection max idle time | `60s` | `10m` |
| **Security** |
| `BEAVER_DB_SSL_MODE` | PostgreSQL SSL mode | - | `require`, `verify-full` |
| `BEAVER_DB_TLS_CONFIG` | MySQL TLS config | - | `true`, `skip-verify` |
| **ORM** |
| `BEAVER_DB_ORM` | Enable ORM support | - | `gorm` |
| `BEAVER_DB_AUTO_MIGRATE` | Auto-migrate schemas | `false` | `true` |
| **Monitoring** |
| `BEAVER_DB_DEBUG` | Enable debug logging | `false` | `true` |
| `BEAVER_DB_SLOW_QUERY_TIME` | Slow query threshold | `200ms` | `500ms` |

## API Reference

### Initialization Functions

```go
// Purpose: Initialize database with environment configuration
// Prerequisites: Environment variables set or explicit config provided
// Expected outcome: Global database instance ready for use

// Initialize from environment variables
func Init() error

// Initialize with explicit configuration
func Init(config Config) error

// Must variant (panics on error)
func MustInit()

// Create new instance without global state
func New(config Config) (*sql.DB, error)

// Create from environment without global state
func NewFromEnv() (*sql.DB, error)
```

### Global Instance Access

```go
// Purpose: Access initialized database instances
// Prerequisites: Database must be initialized first
// Expected outcome: Database instance for operations

// Get global SQL database instance
func DB() *sql.DB

// Get global GORM instance (if enabled)
func GORM() (*gorm.DB, error)

// Must variant for GORM (panics on error)
func MustGORM() *gorm.DB

// Check if database is initialized
func IsInitialized() bool
```

### GORM Integration

```go
// Purpose: Enable GORM ORM functionality
// Prerequisites: Database initialized
// Expected outcome: GORM instance for ORM operations

// Enable GORM on existing database
func WithGORM() (*gorm.DB, error)

// Create GORM instance with config
func NewGORM(config Config, db *sql.DB) (*gorm.DB, error)

// Example usage
func useGORM() error {
    // Initialize database first
    if err := database.Init(); err != nil {
        return err
    }
    
    // Enable GORM
    gormDB, err := database.WithGORM()
    if err != nil {
        return err
    }
    
    // Use GORM for ORM operations
    var users []User
    result := gormDB.Find(&users)
    return result.Error
}
```

### Query Helper Functions

```go
// Purpose: Convenient query operations with context support
// Prerequisites: Database initialized
// Expected outcome: Query results or appropriate error

// Execute query without results (INSERT, UPDATE, DELETE)
func Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

// Query multiple rows
func Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Query single row
func QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

// Prepare statement
func Prepare(ctx context.Context, query string) (*sql.Stmt, error)

// Example usage
func queryHelperExample() error {
    ctx := context.Background()
    
    // Insert with helper
    result, err := database.Exec(ctx, 
        "INSERT INTO users (name, email) VALUES (?, ?)", 
        "John Doe", "john@example.com")
    if err != nil {
        return err
    }
    
    userID, _ := result.LastInsertId()
    log.Printf("Created user ID: %d", userID)
    
    // Query with helper
    rows, err := database.Query(ctx, "SELECT * FROM users WHERE active = ?", true)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    return nil
}
```

### Transaction Support

```go
// Purpose: Execute operations within database transactions
// Prerequisites: Database initialized
// Expected outcome: All operations committed or rolled back atomically

// Execute function within transaction
func Transaction(ctx context.Context, fn func(*sql.Tx) error) error

// Example usage
func transactionExample() error {
    ctx := context.Background()
    
    return database.Transaction(ctx, func(tx *sql.Tx) error {
        // All operations within this function are in a transaction
        
        // Debit account
        _, err := tx.Exec(
            "UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?",
            100.00, 1, 100.00)
        if err != nil {
            return err // Will automatically rollback
        }
        
        // Credit account
        _, err = tx.Exec(
            "UPDATE accounts SET balance = balance + ? WHERE id = ?",
            100.00, 2)
        if err != nil {
            return err // Will automatically rollback
        }
        
        // Log transaction
        _, err = tx.Exec(
            "INSERT INTO transaction_log (from_account, to_account, amount) VALUES (?, ?, ?)",
            1, 2, 100.00)
        
        return err // Will commit if nil, rollback if error
    })
}
```

### Health Monitoring

```go
// Purpose: Monitor database health and performance
// Prerequisites: Database initialized
// Expected outcome: Health status and connection statistics

// Check database health with context timeout
func Health(ctx context.Context) error

// Quick health check
func IsHealthy() bool

// Get connection pool statistics
func Stats() sql.DBStats

// Example health monitoring
func healthCheckExample() {
    // Quick check
    if !database.IsHealthy() {
        log.Println("Database is unhealthy!")
        return
    }
    
    // Detailed health check with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := database.Health(ctx); err != nil {
        log.Printf("Database health check failed: %v", err)
        return
    }
    
    // Get detailed statistics
    stats := database.Stats()
    log.Printf("Open connections: %d", stats.OpenConnections)
    log.Printf("In use: %d", stats.InUse)
    log.Printf("Idle: %d", stats.Idle)
    log.Printf("Max open connections: %d", stats.MaxOpenConnections)
    log.Printf("Total connections: %d", stats.MaxIdleClosed + stats.MaxLifetimeClosed)
}
```

### Lifecycle Management

```go
// Purpose: Manage database lifecycle and cleanup
// Prerequisites: Database initialized
// Expected outcome: Clean shutdown or reset state

// Graceful shutdown with context timeout
func Shutdown(ctx context.Context) error

// Reset global instance (for testing)
func Reset()

// Example graceful shutdown
func gracefulShutdownExample() {
    // In main function or signal handler
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        log.Println("Shutting down database...")
        
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        if err := database.Shutdown(ctx); err != nil {
            log.Printf("Database shutdown error: %v", err)
        }
        
        os.Exit(0)
    }()
}
```

## Database Drivers

### PostgreSQL

```bash
# Purpose: Configure PostgreSQL connection
# Prerequisites: PostgreSQL server running
# Expected outcome: Secure PostgreSQL connection

BEAVER_DB_DRIVER=postgres
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=postgres
BEAVER_DB_PASSWORD=secret
BEAVER_DB_SSL_MODE=require

# Or use connection URL
BEAVER_DB_URL=postgres://postgres:secret@localhost:5432/myapp?sslmode=require
```

**Features:**
- Uses `jackc/pgx/v5` driver (high-performance, pure Go)
- Full PostgreSQL feature support
- SSL/TLS configuration
- Connection pooling
- Prepared statements

### MySQL

```bash
# Purpose: Configure MySQL connection
# Prerequisites: MySQL server running
# Expected outcome: Optimized MySQL connection

BEAVER_DB_DRIVER=mysql
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=3306
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=root
BEAVER_DB_PASSWORD=secret
BEAVER_DB_TLS_CONFIG=true

# Connection parameters
BEAVER_DB_PARAMS={"parseTime":"true","loc":"UTC"}
```

**Features:**
- Uses `go-sql-driver/mysql` (pure Go)
- TLS support
- Parameter parsing
- Timezone handling
- Character set configuration

### SQLite

```bash
# Purpose: Configure SQLite for development or lightweight production
# Prerequisites: Write permissions to database file location
# Expected outcome: File-based SQLite database

BEAVER_DB_DRIVER=sqlite
BEAVER_DB_DATABASE=./app.db

# For in-memory database (testing)
BEAVER_DB_DATABASE=:memory:
```

**Features:**
- Uses `modernc.org/sqlite` (pure Go, no CGO)
- File-based or in-memory
- ACID transactions
- Concurrent read access
- Write-ahead logging (WAL) mode

### Turso (LibSQL)

```bash
# Purpose: Configure Turso cloud database
# Prerequisites: Turso account and database created
# Expected outcome: Connection to Turso edge database

BEAVER_DB_DRIVER=turso
BEAVER_DB_URL=libsql://your-database.turso.io
BEAVER_DB_AUTH_TOKEN=your-auth-token

# Local development with LibSQL
BEAVER_DB_DRIVER=libsql
BEAVER_DB_DATABASE=file:local.db
```

**Features:**
- Uses `tursodatabase/libsql-client-go` (pure Go)
- Edge database replication
- SQLite compatibility
- HTTP/WebSocket protocols
- Multi-region deployment

## Advanced Usage Examples

### Custom Configuration

```go
// Purpose: Create database instance with custom configuration
// Prerequisites: Understanding of specific database requirements
// Expected outcome: Customized database connection

func customConfigExample() error {
    config := database.Config{
        Driver:   "postgres",
        Host:     "prod-db.example.com",
        Port:     "5432",
        Database: "myapp_prod",
        Username: "app_user",
        Password: os.Getenv("DB_PASSWORD"),
        SSLMode:  "require",
        
        // Production pool settings
        MaxOpenConns:    50,
        MaxIdleConns:    10,
        ConnMaxLifetime: 1 * time.Hour,
        ConnMaxIdleTime: 10 * time.Minute,
        
        // Monitoring
        Debug:         false,
        SlowQueryTime: 500 * time.Millisecond,
    }
    
    return database.Init(config)
}
```

### Connection with Custom Parameters

```go
// Purpose: Connect with database-specific parameters
// Prerequisites: Understanding of driver-specific options
// Expected outcome: Optimized connection for specific use case

func customParametersExample() error {
    // PostgreSQL with custom parameters
    postgresConfig := database.Config{
        Driver:   "postgres",
        URL:      "postgres://user:pass@host:5432/db?application_name=myapp&search_path=public",
    }
    
    // MySQL with parsing and timezone settings
    mysqlConfig := database.Config{
        Driver: "mysql",
        URL:    "user:pass@tcp(host:3306)/db?parseTime=true&loc=UTC&charset=utf8mb4",
    }
    
    // SQLite with WAL mode and cache settings
    sqliteConfig := database.Config{
        Driver:   "sqlite",
        Database: "./app.db?cache=shared&mode=rwc&_journal_mode=WAL",
    }
    
    return database.Init(postgresConfig)
}
```

### Multi-Database Setup

```go
// Purpose: Use multiple database connections
// Prerequisites: Multiple database servers configured
// Expected outcome: Separate connections for different purposes

func multiDatabaseExample() error {
    // Main application database
    mainDB, err := database.New(database.Config{
        Driver:   "postgres",
        Host:     "main-db.example.com",
        Database: "app_main",
        Username: "app_user",
        Password: os.Getenv("MAIN_DB_PASSWORD"),
    })
    if err != nil {
        return err
    }
    
    // Analytics database (read-only)
    analyticsDB, err := database.New(database.Config{
        Driver:   "postgres",
        Host:     "analytics-db.example.com",
        Database: "app_analytics",
        Username: "analytics_reader",
        Password: os.Getenv("ANALYTICS_DB_PASSWORD"),
        MaxOpenConns: 10, // Fewer connections for analytics
    })
    if err != nil {
        return err
    }
    
    // Use databases for different purposes
    _ = mainDB     // For transactional operations
    _ = analyticsDB // For reporting queries
    
    return nil
}
```

### GORM with Auto-Migration

```go
// Purpose: Use GORM with automatic schema migration
// Prerequisites: GORM enabled and models defined
// Expected outcome: Database schema automatically updated

type User struct {
    ID        uint      `gorm:"primarykey"`
    Name      string    `gorm:"not null"`
    Email     string    `gorm:"uniqueIndex"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Post struct {
    ID      uint   `gorm:"primarykey"`
    Title   string `gorm:"not null"`
    Content string `gorm:"type:text"`
    UserID  uint   `gorm:"not null"`
    User    User   `gorm:"foreignKey:UserID"`
}

func gormMigrationExample() error {
    // Initialize database with GORM
    if err := database.Init(); err != nil {
        return err
    }
    
    gormDB, err := database.WithGORM()
    if err != nil {
        return err
    }
    
    // Auto-migrate schemas
    err = gormDB.AutoMigrate(&User{}, &Post{})
    if err != nil {
        return err
    }
    
    // Use GORM for operations
    user := User{Name: "John Doe", Email: "john@example.com"}
    result := gormDB.Create(&user)
    if result.Error != nil {
        return result.Error
    }
    
    log.Printf("Created user with ID: %d", user.ID)
    return nil
}
```

## Error Handling

### Error Types

```go
// Package-specific errors
var (
    ErrNotInitialized = errors.New("database not initialized")
    ErrInvalidDriver  = errors.New("invalid database driver")
    ErrInvalidConfig  = errors.New("invalid database configuration")
    ErrGORMNotEnabled = errors.New("GORM not enabled")
    ErrConnectionFailed = errors.New("database connection failed")
)
```

### Error Handling Patterns

```go
// Purpose: Handle database errors appropriately
// Prerequisites: Understanding of error types and context
// Expected outcome: Robust error handling and recovery

func errorHandlingExample() error {
    // Initialization errors
    if err := database.Init(); err != nil {
        if errors.Is(err, database.ErrInvalidDriver) {
            return fmt.Errorf("unsupported database driver: %w", err)
        } else if errors.Is(err, database.ErrInvalidConfig) {
            return fmt.Errorf("database configuration error: %w", err)
        } else {
            return fmt.Errorf("database initialization failed: %w", err)
        }
    }
    
    // Query errors
    db := database.DB()
    var user User
    err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", 1).
        Scan(&user.ID, &user.Name, &user.Email)
    
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return fmt.Errorf("user not found")
        } else {
            return fmt.Errorf("database query failed: %w", err)
        }
    }
    
    return nil
}

// Connection recovery pattern
func connectionRecoveryExample() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        if !database.IsHealthy() {
            log.Println("Database connection lost, attempting recovery...")
            
            // Try to reconnect
            ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
            err := database.Health(ctx)
            cancel()
            
            if err != nil {
                log.Printf("Database recovery failed: %v", err)
                // Could trigger alert here
            } else {
                log.Println("Database connection recovered")
            }
        }
    }
}
```

## Performance Optimization

### Connection Pool Tuning

```go
// Purpose: Optimize connection pool for specific workload
// Prerequisites: Understanding of application's database usage patterns
// Expected outcome: Optimal database performance

func optimizeConnectionPool() {
    config := database.Config{
        Driver: "postgres",
        URL:    os.Getenv("DATABASE_URL"),
        
        // High-throughput settings
        MaxOpenConns:    100,                // Max concurrent connections
        MaxIdleConns:    20,                 // Keep connections warm
        ConnMaxLifetime: 30 * time.Minute,   // Rotate connections
        ConnMaxIdleTime: 5 * time.Minute,    // Close idle connections
        
        // Monitoring
        SlowQueryTime: 200 * time.Millisecond,
        Debug:         false, // Disable in production
    }
    
    database.Init(config)
}
```

### Query Performance

```go
// Purpose: Optimize database queries for performance
// Prerequisites: Database with proper indexes
// Expected outcome: Efficient query execution

func queryOptimizationExample() error {
    db := database.DB()
    ctx := context.Background()
    
    // Use prepared statements for repeated queries
    stmt, err := database.Prepare(ctx, "SELECT id, name, email FROM users WHERE created_at > ?")
    if err != nil {
        return err
    }
    defer stmt.Close()
    
    // Execute with different parameters
    since := time.Now().AddDate(0, -1, 0) // Last month
    rows, err := stmt.QueryContext(ctx, since)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Process results efficiently
    users := make([]User, 0, 100) // Pre-allocate slice
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
            log.Printf("Scan error: %v", err)
            continue
        }
        users = append(users, user)
    }
    
    return rows.Err()
}
```

## Testing Patterns

### Test Setup

```go
// Purpose: Set up database for testing
// Prerequisites: Test environment configured
// Expected outcome: Clean database state for each test

func TestDatabaseOperations(t *testing.T) {
    // Clean up after test
    defer database.Reset()
    
    // Use in-memory SQLite for fast tests
    config := database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
        Debug:    true, // Enable debug logging for tests
    }
    
    if err := database.Init(config); err != nil {
        t.Fatal("Failed to initialize test database:", err)
    }
    
    // Setup test schema
    db := database.DB()
    _, err := db.Exec(`
        CREATE TABLE users (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL
        )
    `)
    if err != nil {
        t.Fatal("Failed to create test table:", err)
    }
    
    // Run test operations
    testUserOperations(t)
}

func testUserOperations(t *testing.T) {
    ctx := context.Background()
    
    // Test user creation
    result, err := database.Exec(ctx,
        "INSERT INTO users (name, email) VALUES (?, ?)",
        "Test User", "test@example.com")
    if err != nil {
        t.Fatal("Failed to create user:", err)
    }
    
    userID, _ := result.LastInsertId()
    if userID == 0 {
        t.Fatal("Expected user ID > 0")
    }
    
    // Test user retrieval
    var name, email string
    err = database.QueryRow(ctx, "SELECT name, email FROM users WHERE id = ?", userID).
        Scan(&name, &email)
    if err != nil {
        t.Fatal("Failed to retrieve user:", err)
    }
    
    if name != "Test User" || email != "test@example.com" {
        t.Fatalf("Expected 'Test User' and 'test@example.com', got '%s' and '%s'", name, email)
    }
}
```

### Integration Testing

```go
// Purpose: Test database operations in realistic scenarios
// Prerequisites: Test database server available
// Expected outcome: Validated database integration

func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    defer database.Reset()
    
    // Use real database for integration tests
    config := database.Config{
        Driver:   "postgres",
        Host:     "localhost",
        Port:     "5432",
        Database: "test_db",
        Username: "test_user",
        Password: "test_password",
    }
    
    if err := database.Init(config); err != nil {
        t.Skip("PostgreSQL not available for integration test:", err)
    }
    
    // Test health check
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := database.Health(ctx); err != nil {
        t.Fatal("Database health check failed:", err)
    }
    
    // Test transaction handling
    err := database.Transaction(ctx, func(tx *sql.Tx) error {
        _, err := tx.Exec("INSERT INTO test_table (value) VALUES (?)", "test")
        return err
    })
    if err != nil {
        t.Fatal("Transaction failed:", err)
    }
}
```

This comprehensive documentation provides AI assistants with complete context for understanding and using the database package effectively, including all configuration options, usage patterns, error handling, and best practices.