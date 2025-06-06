// database/config.go
package database

import (
	"github.com/gobeaver/beaver-kit/config"
)

// Config holds database configuration
type Config struct {
	// Driver: postgres, mysql, sqlite, turso, libsql
	Driver string `env:"DB_DRIVER,default:sqlite"`

	// Connection details (for traditional databases)
	Host     string `env:"DB_HOST,default:localhost"`
	Port     string `env:"DB_PORT"`
	Database string `env:"DB_DATABASE,default:beaver.db"`
	Username string `env:"DB_USERNAME"`
	Password string `env:"DB_PASSWORD"`

	// URL for direct connection string (overrides individual settings)
	URL string `env:"DB_URL"`

	// Auth token for Turso/LibSQL
	AuthToken string `env:"DB_AUTH_TOKEN"`

	// SSL/TLS Configuration
	SSLMode   string `env:"DB_SSL_MODE,default:disable"` // For PostgreSQL
	TLSConfig string `env:"DB_TLS_CONFIG"`               // For MySQL

	// Connection Pool Settings
	MaxOpenConns    int `env:"DB_MAX_OPEN_CONNS,default:25"`
	MaxIdleConns    int `env:"DB_MAX_IDLE_CONNS,default:5"`
	ConnMaxLifetime int `env:"DB_CONN_MAX_LIFETIME,default:300"` // seconds
	ConnMaxIdleTime int `env:"DB_CONN_MAX_IDLE_TIME,default:60"` // seconds

	// Additional driver-specific parameters
	Params string `env:"DB_PARAMS"`

	// Debug mode
	Debug bool `env:"DB_DEBUG,default:false"`

	// ORM Support (optional)
	UseORM        string `env:"DB_ORM"`                          // "gorm" or empty
	DisableORMLog bool   `env:"DB_DISABLE_ORM_LOG,default:true"` // Only applies when UseORM is set

	// Migrations (optional, mainly for GORM users)
	AutoMigrate     bool   `env:"DB_AUTO_MIGRATE,default:false"`
	MigrationsPath  string `env:"DB_MIGRATIONS_PATH,default:migrations"`
	MigrationsTable string `env:"DB_MIGRATIONS_TABLE,default:schema_migrations"`
}

// GetConfig loads configuration from environment variables
func GetConfig() (*Config, error) {
	cfg := &Config{}
	if err := config.Load(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
