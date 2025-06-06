// database/service.go - SQL-first implementation
// This package uses pure Go database drivers to ensure CGO-free builds,
// enabling easy cross-compilation and deployment across different platforms.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"

	// Database drivers - pure Go implementations for CGO-free builds
	_ "github.com/go-sql-driver/mysql"      // MySQL - already pure Go
	_ "github.com/jackc/pgx/v5/stdlib"      // PostgreSQL - pure Go, performant
	_ "github.com/tursodatabase/libsql-client-go/libsql" // LibSQL/Turso - pure Go
	_ "modernc.org/sqlite"                   // SQLite - pure Go alternative to go-sqlite3

	// GORM (only loaded when needed)
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Global instances
var (
	defaultDB     *sql.DB  // Primary instance - always *sql.DB
	defaultGORM   *gorm.DB // Optional GORM instance
	defaultConfig *Config  // Stored config
	defaultOnce   sync.Once
	defaultErr    error
	gormOnce      sync.Once
	gormErr       error
)

// Common errors
var (
	ErrNotInitialized = errors.New("database not initialized")
	ErrInvalidDriver  = errors.New("invalid database driver")
	ErrInvalidConfig  = errors.New("invalid database configuration")
	ErrGORMNotEnabled = errors.New("GORM not enabled - set BEAVER_DB_ORM=gorm or use WithGORM()")
)

// Builder provides a way to create database instances with custom prefixes
type Builder struct {
	prefix string
}

// WithPrefix creates a new Builder with the specified prefix
func WithPrefix(prefix string) *Builder {
	return &Builder{prefix: prefix}
}

// Init initializes the global database instance using the builder's prefix
func (b *Builder) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return err
	}
	return Init(*cfg)
}

// New creates a new database connection using the builder's prefix
func (b *Builder) New() (*sql.DB, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	return New(*cfg)
}

// NewGORM creates a new GORM instance using the builder's prefix
func (b *Builder) NewGORM() (*gorm.DB, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: b.prefix}); err != nil {
		return nil, err
	}
	cfg.UseORM = "gorm"
	
	sqlDB, err := New(*cfg)
	if err != nil {
		return nil, err
	}
	
	return NewGORM(*cfg, sqlDB)
}

// Init initializes the global SQL database instance with optional config
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

		defaultConfig = cfg
		defaultDB, defaultErr = New(*cfg)

		// Check if GORM is requested via config or environment
		if defaultErr == nil && shouldInitGORM(cfg) {
			gormOnce.Do(func() {
				defaultGORM, gormErr = NewGORM(*cfg, defaultDB)
			})
			if gormErr != nil {
				defaultErr = fmt.Errorf("failed to initialize GORM: %w", gormErr)
			}
		}
	})

	return defaultErr
}

// New creates a new SQL database connection with given config
func New(cfg Config) (*sql.DB, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	var dsn string
	var driverName string

	switch cfg.Driver {
	case "mysql":
		driverName = "mysql"
		dsn = buildMySQLDSN(cfg)

	case "postgres", "postgresql":
		driverName = "pgx" // Using pgx for better performance
		dsn = buildPostgresDSN(cfg)

	case "sqlite", "sqlite3":
		driverName = "sqlite"
		dsn = cfg.Database
		if dsn == "" {
			dsn = "file:sqlite.db?cache=shared&mode=rwc"
		}

	case "libsql", "turso":
		driverName = "libsql"
		dsn = cfg.URL
		if cfg.AuthToken != "" {
			dsn = fmt.Sprintf("%s?authToken=%s", cfg.URL, cfg.AuthToken)
		}

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidDriver, cfg.Driver)
	}

	// Open connection
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// NewGORM creates a GORM instance from an existing SQL connection
func NewGORM(cfg Config, sqlDB *sql.DB) (*gorm.DB, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("sql.DB instance is required for GORM")
	}

	var dialector gorm.Dialector

	switch cfg.Driver {
	case "mysql":
		dialector = mysql.New(mysql.Config{
			Conn: sqlDB,
		})

	case "postgres", "postgresql":
		dialector = postgres.New(postgres.Config{
			Conn: sqlDB,
		})

	case "sqlite", "sqlite3", "libsql", "turso":
		dialector = sqlite.Dialector{
			Conn: sqlDB,
		}

	default:
		return nil, fmt.Errorf("unsupported driver for GORM: %s", cfg.Driver)
	}

	// Configure GORM
	gormCfg := &gorm.Config{}

	// Configure logging
	if cfg.Debug && !cfg.DisableORMLog {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormCfg.Logger = logger.Default.LogMode(logger.Silent)
	}

	gormDB, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, err
	}

	// Run auto-migrations if configured
	if cfg.AutoMigrate && defaultGORM != nil {
		// This would need to be implemented with a migration registry
		// For now, users should handle migrations manually
	}

	return gormDB, nil
}

// DB returns the global SQL database instance
func DB() *sql.DB {
	if defaultDB == nil {
		Init()
	}
	return defaultDB
}

// GORM returns the global GORM instance (returns error if not enabled)
func GORM() (*gorm.DB, error) {
	if defaultDB == nil {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	// Check if GORM should be initialized
	if defaultGORM == nil && defaultConfig != nil && shouldInitGORM(defaultConfig) {
		gormOnce.Do(func() {
			defaultGORM, gormErr = NewGORM(*defaultConfig, defaultDB)
		})
	}

	if gormErr != nil {
		return nil, gormErr
	}

	if defaultGORM == nil {
		return nil, ErrGORMNotEnabled
	}

	return defaultGORM, nil
}

// MustGORM returns the global GORM instance or panics
func MustGORM() *gorm.DB {
	gormDB, err := GORM()
	if err != nil {
		panic(fmt.Sprintf("failed to get GORM instance: %v", err))
	}
	return gormDB
}

// WithGORM initializes the database with GORM support enabled
func WithGORM(configs ...Config) (*gorm.DB, error) {
	var cfg Config
	if len(configs) > 0 {
		cfg = configs[0]
	} else {
		c, err := GetConfig()
		if err != nil {
			return nil, err
		}
		cfg = *c
	}

	cfg.UseORM = "gorm"

	if err := Init(cfg); err != nil {
		return nil, err
	}

	return GORM()
}

// Health checks database connectivity
func Health(ctx context.Context) error {
	if defaultDB == nil {
		return ErrNotInitialized
	}

	return defaultDB.PingContext(ctx)
}

// Stats returns database statistics
func Stats() sql.DBStats {
	if defaultDB == nil {
		return sql.DBStats{}
	}
	return defaultDB.Stats()
}

// IsHealthy returns true if the database is reachable
func IsHealthy() bool {
	return Health(context.Background()) == nil
}

// Reset clears the global instances (for testing)
func Reset() {
	if defaultDB != nil {
		defaultDB.Close()
	}
	defaultDB = nil
	defaultGORM = nil
	defaultConfig = nil
	defaultOnce = sync.Once{}
	gormOnce = sync.Once{}
	defaultErr = nil
	gormErr = nil
}

// Shutdown gracefully closes database connections
func Shutdown(ctx context.Context) error {
	if defaultDB == nil {
		return nil
	}

	// Close with context timeout
	done := make(chan struct{})
	go func() {
		defaultDB.Close()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Transaction executes a function within a SQL transaction
func Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	if defaultDB == nil {
		return ErrNotInitialized
	}

	tx, err := defaultDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Helper functions

func shouldInitGORM(cfg *Config) bool {
	return cfg.UseORM == "gorm" || strings.ToLower(cfg.UseORM) == "true"
}

func validateConfig(cfg Config) error {
	if cfg.Driver == "" {
		return errors.New("database driver required")
	}

	// Normalize driver names
	switch cfg.Driver {
	case "postgres":
		cfg.Driver = "postgresql"
	case "sqlite":
		cfg.Driver = "sqlite3"
	}

	// For turso/libsql, URL is required
	if (cfg.Driver == "libsql" || cfg.Driver == "turso") && cfg.URL == "" {
		return errors.New("turso requires URL to be set")
	}

	// For other drivers, validate connection details
	if cfg.Driver != "sqlite3" && cfg.Driver != "libsql" && cfg.Driver != "turso" && cfg.URL == "" {
		if cfg.Host == "" || cfg.Database == "" {
			return errors.New("database connection details required")
		}
	}

	return nil
}

func buildMySQLDSN(cfg Config) string {
	if cfg.URL != "" {
		return cfg.URL
	}

	port := cfg.Port
	if port == "" {
		port = "3306"
	}

	// MySQL DSN format
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		cfg.Username, cfg.Password, cfg.Host, port, cfg.Database)

	// Add parameters
	params := []string{
		"charset=utf8mb4",
		"parseTime=True",
		"loc=Local",
	}

	if cfg.Params != "" {
		params = append(params, cfg.Params)
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}

func buildPostgresDSN(cfg Config) string {
	if cfg.URL != "" {
		return cfg.URL
	}

	port := cfg.Port
	if port == "" {
		port = "5432"
	}

	// PostgreSQL DSN format
	parts := []string{
		fmt.Sprintf("host=%s", cfg.Host),
		fmt.Sprintf("port=%s", port),
		fmt.Sprintf("user=%s", cfg.Username),
		fmt.Sprintf("password=%s", cfg.Password),
		fmt.Sprintf("dbname=%s", cfg.Database),
		fmt.Sprintf("sslmode=%s", cfg.SSLMode),
	}

	if cfg.Params != "" {
		parts = append(parts, cfg.Params)
	}

	return strings.Join(parts, " ")
}

// Convenience functions for common operations

// Exec executes a query without returning any rows
func Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if defaultDB == nil {
		return nil, ErrNotInitialized
	}
	return defaultDB.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows
func Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if defaultDB == nil {
		return nil, ErrNotInitialized
	}
	return defaultDB.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if defaultDB == nil {
		return nil
	}
	return defaultDB.QueryRowContext(ctx, query, args...)
}

// Prepare creates a prepared statement
func Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if defaultDB == nil {
		return nil, ErrNotInitialized
	}
	return defaultDB.PrepareContext(ctx, query)
}

// MustInit initializes the database and panics on error
func MustInit(configs ...Config) {
	if err := Init(configs...); err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}
}

// Default returns the global SQL instance with error handling
func Default() (*sql.DB, error) {
	if defaultDB == nil {
		if err := Init(); err != nil {
			return nil, err
		}
	}
	return defaultDB, nil
}

// NewFromEnv creates SQL instance from environment variables
func NewFromEnv() (*sql.DB, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return New(*cfg)
}

// InitFromEnv is an alias for Init with no arguments
func InitFromEnv() error {
	return Init()
}
