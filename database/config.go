// database/config.go
package database

import (
	"github.com/gobeaver/beaver-kit/config"
)

// Config holds database configuration
type Config struct {
	// Driver: postgres, mysql, sqlite, turso, libsql
	Driver string `env:"BEAVER_DB_DRIVER,default:sqlite"`

	// Connection details (for traditional databases)
	Host     string `env:"BEAVER_DB_HOST,default:localhost"`
	Port     string `env:"BEAVER_DB_PORT"`
	Database string `env:"BEAVER_DB_DATABASE,default:beaver.db"`
	Username string `env:"BEAVER_DB_USERNAME"`
	Password string `env:"BEAVER_DB_PASSWORD"`

	// URL for direct connection string (overrides individual settings)
	URL string `env:"BEAVER_DB_URL"`

	// Auth token for Turso/LibSQL
	AuthToken string `env:"BEAVER_DB_AUTH_TOKEN"`

	// SSL/TLS Configuration
	SSLMode   string `env:"BEAVER_DB_SSL_MODE,default:disable"` // For PostgreSQL
	TLSConfig string `env:"BEAVER_DB_TLS_CONFIG"`               // For MySQL

	// Connection Pool Settings
	MaxOpenConns    int `env:"BEAVER_DB_MAX_OPEN_CONNS,default:25"`
	MaxIdleConns    int `env:"BEAVER_DB_MAX_IDLE_CONNS,default:5"`
	ConnMaxLifetime int `env:"BEAVER_DB_CONN_MAX_LIFETIME,default:300"` // seconds
	ConnMaxIdleTime int `env:"BEAVER_DB_CONN_MAX_IDLE_TIME,default:60"` // seconds

	// Additional driver-specific parameters
	Params string `env:"BEAVER_DB_PARAMS"`

	// Debug mode
	Debug bool `env:"BEAVER_DB_DEBUG,default:false"`

	// ORM Support (optional)
	UseORM        string `env:"BEAVER_DB_ORM"`                          // "gorm" or empty
	DisableORMLog bool   `env:"BEAVER_DB_DISABLE_ORM_LOG,default:true"` // Only applies when UseORM is set

	// Migrations (optional, mainly for GORM users)
	AutoMigrate     bool   `env:"BEAVER_DB_AUTO_MIGRATE,default:false"`
	MigrationsPath  string `env:"BEAVER_DB_MIGRATIONS_PATH,default:migrations"`
	MigrationsTable string `env:"BEAVER_DB_MIGRATIONS_TABLE,default:schema_migrations"`
}

// GetConfig loads configuration from environment variables
func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
