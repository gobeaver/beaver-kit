// Package database database/service.go - SQL-first implementation
// This package uses pure Go database drivers to ensure CGO-free builds,
// enabling easy cross-compilation and deployment across different platforms.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/config"

	// Database drivers - pure Go implementations for CGO-free builds
	_ "github.com/go-sql-driver/mysql"                   // MySQL - already pure Go
	_ "github.com/jackc/pgx/v5/stdlib"                   // PostgreSQL - pure Go, performant
	_ "github.com/tursodatabase/libsql-client-go/libsql" // LibSQL/Turso - pure Go
	_ "modernc.org/sqlite"                               // SQLite - pure Go alternative to go-sqlite3

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
	ErrGORMNotEnabled = errors.New("GORM not enabled - set BEAVER_DB_ORM=gorm or use InitWithGORM()")
)

// Database wraps both sql.DB and gorm.DB providing unified access
type Database struct {
	sqlDB   *sql.DB
	gormDB  *gorm.DB
	prefix  string
	useGORM bool
}

// New creates a new Database with default settings
func New() *Database {
	return &Database{prefix: "BEAVER_"}
}

// WithPrefix creates a new Database with the specified prefix
func WithPrefix(prefix string) *Database {
	return &Database{prefix: prefix}
}

// WithGORM creates a new Database with GORM enabled
func WithGORM() *Database {
	return &Database{prefix: "BEAVER_", useGORM: true}
}

// WithPrefix sets a custom environment variable prefix and returns the database for chaining
func (db *Database) WithPrefix(prefix string) *Database {
	db.prefix = prefix
	return db
}

// WithGORM enables GORM support and returns the database for chaining
func (db *Database) WithGORM() *Database {
	db.useGORM = true
	return db
}

// Init initializes the global database instance with the configured settings
func (db *Database) Init() error {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: db.prefix}); err != nil {
		return err
	}
	if db.useGORM {
		cfg.UseORM = "gorm"
	}
	return Init(*cfg)
}

// Connect creates a new database connection with the configured settings
func (db *Database) Connect() (*Database, error) {
	cfg := &Config{}
	if err := config.Load(cfg, config.LoadOptions{Prefix: db.prefix}); err != nil {
		return nil, err
	}

	if db.useGORM {
		cfg.UseORM = "gorm"
		sqlDB, err := NewSQL(*cfg)
		if err != nil {
			return nil, err
		}
		gormDB, err := NewGORM(*cfg, sqlDB)
		if err != nil {
			return nil, err
		}
		return &Database{
			gormDB:  gormDB,
			prefix:  db.prefix,
			useGORM: db.useGORM,
		}, nil
	}

	sqlDB, err := NewSQL(*cfg)
	if err != nil {
		return nil, err
	}
	return &Database{
		sqlDB:   sqlDB,
		prefix:  db.prefix,
		useGORM: db.useGORM,
	}, nil
}

// SQL returns the underlying sql.DB instance
func (db *Database) SQL() *sql.DB {
	if db.gormDB != nil {
		sqlDB, _ := db.gormDB.DB()
		return sqlDB
	}
	return db.sqlDB
}

// GORM returns the GORM instance or error if not enabled
func (db *Database) GORM() (*gorm.DB, error) {
	if db.gormDB == nil {
		return nil, ErrGORMNotEnabled
	}
	return db.gormDB, nil
}

// MustGORM returns the GORM instance or panics if not enabled
func (db *Database) MustGORM() *gorm.DB {
	gormDB, err := db.GORM()
	if err != nil {
		panic(fmt.Sprintf("failed to get GORM instance: %v", err))
	}
	return gormDB
}

// Close closes the database connection
func (db *Database) Close() error {
	if db.gormDB != nil {
		if sqlDB, err := db.gormDB.DB(); err == nil {
			return sqlDB.Close()
		}
	}
	if db.sqlDB != nil {
		return db.sqlDB.Close()
	}
	return nil
}

// Ping verifies the database connection is alive
func (db *Database) Ping() error {
	return db.SQL().Ping()
}

// PingContext verifies the database connection is alive with context
func (db *Database) PingContext(ctx context.Context) error {
	return db.SQL().PingContext(ctx)
}

// Stats returns connection pool statistics
func (db *Database) Stats() sql.DBStats {
	return db.SQL().Stats()
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
		defaultDB, defaultErr = NewSQL(*cfg)

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

// parseURLForDriver extracts the driver type from a database URL
// Returns the appropriate driver name and DSN for sql.Open
func parseURLForDriver(databaseURL string) (driver, dsn string) {
	// PostgreSQL URLs
	if strings.HasPrefix(databaseURL, "postgres://") ||
		strings.HasPrefix(databaseURL, "postgresql://") {
		return "pgx", databaseURL
	}

	// MySQL URLs - need to convert to Go MySQL driver format
	if strings.HasPrefix(databaseURL, "mysql://") {
		// MySQL URLs need special handling for go-sql-driver/mysql
		// Format: mysql://user:pass@host:port/dbname?params
		// Convert to: user:pass@tcp(host:port)/dbname?params
		// Use url.Parse to properly handle special characters in password
		parsed, err := url.Parse(databaseURL)
		if err != nil {
			// Fall back to raw URL if parsing fails
			return "mysql", strings.TrimPrefix(databaseURL, "mysql://")
		}

		var userPass string
		if parsed.User != nil {
			password, hasPassword := parsed.User.Password()
			if hasPassword {
				userPass = fmt.Sprintf("%s:%s", parsed.User.Username(), password)
			} else {
				userPass = parsed.User.Username()
			}
		}

		host := parsed.Host
		if host == "" {
			host = "localhost:3306"
		} else if !strings.Contains(host, ":") {
			host = host + ":3306" // Add default MySQL port
		}

		path := parsed.Path // includes leading /
		query := parsed.RawQuery
		if query != "" {
			dsn = fmt.Sprintf("%s@tcp(%s)%s?%s", userPass, host, path, query)
		} else {
			dsn = fmt.Sprintf("%s@tcp(%s)%s", userPass, host, path)
		}
		return "mysql", dsn
	}

	// SQLite URLs
	if strings.HasPrefix(databaseURL, "sqlite://") ||
		strings.HasPrefix(databaseURL, "file:") {
		if strings.HasPrefix(databaseURL, "sqlite://") {
			dsn = strings.TrimPrefix(databaseURL, "sqlite://")
		} else {
			dsn = databaseURL
		}
		return "sqlite", dsn
	}

	// LibSQL/Turso URLs - only match libsql:// scheme
	// For https:// Turso URLs, users should set DB_DRIVER=libsql explicitly
	if strings.HasPrefix(databaseURL, "libsql://") {
		return "libsql", databaseURL
	}

	// Default: assume it's a direct DSN for the configured driver
	return "", databaseURL
}

// NewSQL creates a new SQL database connection with given config
func NewSQL(cfg Config) (*sql.DB, error) {
	// Validation
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	var dsn string
	var driverName string

	// Resolve URL with fallback: DATABASE_URL > DB_URL (legacy)
	effectiveURL := cfg.URL
	if effectiveURL == "" && cfg.LegacyURL != "" {
		effectiveURL = cfg.LegacyURL
	}

	// Priority 1: Use URL if provided (DATABASE_URL or DB_URL)
	if effectiveURL != "" {
		parsedDriver, parsedDSN := parseURLForDriver(effectiveURL)
		if parsedDriver != "" {
			driverName = parsedDriver
			dsn = parsedDSN
			// For LibSQL/Turso, append auth token if provided
			if driverName == "libsql" && cfg.AuthToken != "" && !strings.Contains(dsn, "authToken=") {
				if strings.Contains(dsn, "?") {
					dsn = fmt.Sprintf("%s&authToken=%s", dsn, cfg.AuthToken)
				} else {
					dsn = fmt.Sprintf("%s?authToken=%s", dsn, cfg.AuthToken)
				}
			}
		} else {
			// If we couldn't parse the driver, fall back to configured driver
			switch cfg.Driver {
			case "mysql":
				driverName = "mysql"
				dsn = effectiveURL
			case "postgres", "postgresql":
				driverName = "pgx"
				dsn = effectiveURL
			case "sqlite", "sqlite3":
				driverName = "sqlite"
				dsn = effectiveURL
			case "libsql", "turso":
				driverName = "libsql"
				dsn = effectiveURL
			default:
				return nil, fmt.Errorf("%w: unable to determine driver from URL", ErrInvalidDriver)
			}
		}
	} else {
		// Priority 2: Build DSN from separate fields
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
			// LibSQL/Turso requires a URL - this should be caught by validation,
			// but we handle it explicitly here for safety
			if cfg.Host != "" {
				// Build URL from host if provided
				driverName = "libsql"
				dsn = fmt.Sprintf("libsql://%s", cfg.Host)
				if cfg.AuthToken != "" {
					dsn = fmt.Sprintf("%s?authToken=%s", dsn, cfg.AuthToken)
				}
			} else {
				return nil, fmt.Errorf("%w: libsql/turso requires DATABASE_URL or DB_HOST to be set", ErrInvalidConfig)
			}

		default:
			return nil, fmt.Errorf("%w: %s", ErrInvalidDriver, cfg.Driver)
		}
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

	// Note: Auto-migrations would need a migration registry
	// For now, users should handle migrations manually via GORM's AutoMigrate
	_ = cfg.AutoMigrate // Acknowledge the config field exists but is not implemented

	return gormDB, nil
}

// DB returns the global SQL database instance
func DB() *sql.DB {
	if defaultDB == nil {
		_ = Init()
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

// InitWithGORM initializes the global database instance with GORM support enabled
func InitWithGORM(configs ...Config) error {
	var cfg Config
	if len(configs) > 0 {
		cfg = configs[0]
	} else {
		c, err := GetConfig()
		if err != nil {
			return err
		}
		cfg = *c
	}

	cfg.UseORM = "gorm"
	return Init(cfg)
}

// NewWithGORM creates a new GORM instance without affecting global state
func NewWithGORM(configs ...Config) (*gorm.DB, error) {
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

	sqlDB, err := NewSQL(cfg)
	if err != nil {
		return nil, err
	}

	return NewGORM(cfg, sqlDB)
}

// MustInitWithGORM initializes with GORM and panics on error
func MustInitWithGORM(configs ...Config) {
	if err := InitWithGORM(configs...); err != nil {
		panic(fmt.Sprintf("failed to initialize database with GORM: %v", err))
	}
}

// DEPRECATED: WithGORMDeprecated is deprecated. Use InitWithGORM for global initialization
// or NewWithGORM for creating new instances.
// This function will be removed in v2.0.0
func WithGORMDeprecated(configs ...Config) (*gorm.DB, error) {
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

	// Check if any URL is provided (DATABASE_URL or legacy DB_URL)
	hasURL := cfg.URL != "" || cfg.LegacyURL != ""

	// For turso/libsql, URL or Host is required
	if (cfg.Driver == "libsql" || cfg.Driver == "turso") && !hasURL && cfg.Host == "" {
		return errors.New("libsql/turso requires DATABASE_URL, DB_URL, or DB_HOST to be set")
	}

	// For other drivers, validate connection details
	if cfg.Driver != "sqlite3" && cfg.Driver != "libsql" && cfg.Driver != "turso" && !hasURL {
		if cfg.Host == "" || cfg.Database == "" {
			return errors.New("database connection details required")
		}
	}

	return nil
}

func buildMySQLDSN(cfg Config) string {
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
