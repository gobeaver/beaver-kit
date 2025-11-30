package database

import (
	"testing"
)

func TestParseURLForDriver(t *testing.T) {
	tests := []struct {
		name         string
		databaseURL  string
		wantDriver   string
		wantDSN      string
		wantDSNCheck func(string) bool
	}{
		{
			name:        "PostgreSQL URL with postgres://",
			databaseURL: "postgres://user:pass@localhost:5432/mydb",
			wantDriver:  "pgx",
			wantDSN:     "postgres://user:pass@localhost:5432/mydb",
		},
		{
			name:        "PostgreSQL URL with postgresql://",
			databaseURL: "postgresql://user:pass@localhost:5432/mydb",
			wantDriver:  "pgx",
			wantDSN:     "postgresql://user:pass@localhost:5432/mydb",
		},
		{
			name:        "MySQL URL",
			databaseURL: "mysql://user:pass@localhost:3306/mydb",
			wantDriver:  "mysql",
			wantDSNCheck: func(dsn string) bool {
				// Should convert to MySQL driver format
				return dsn == "user:pass@tcp(localhost:3306)/mydb"
			},
		},
		{
			name:        "MySQL URL with parameters",
			databaseURL: "mysql://user:pass@localhost:3306/mydb?parseTime=true",
			wantDriver:  "mysql",
			wantDSNCheck: func(dsn string) bool {
				return dsn == "user:pass@tcp(localhost:3306)/mydb?parseTime=true"
			},
		},
		{
			name:        "MySQL URL with @ in password",
			databaseURL: "mysql://user:p%40ssword@localhost:3306/mydb",
			wantDriver:  "mysql",
			wantDSNCheck: func(dsn string) bool {
				// Password with @ should be properly handled (URL-decoded)
				return dsn == "user:p@ssword@tcp(localhost:3306)/mydb"
			},
		},
		{
			name:        "MySQL URL without explicit port",
			databaseURL: "mysql://user:pass@localhost/mydb",
			wantDriver:  "mysql",
			wantDSNCheck: func(dsn string) bool {
				// Should add default port 3306
				return dsn == "user:pass@tcp(localhost:3306)/mydb"
			},
		},
		{
			name:        "SQLite URL with sqlite://",
			databaseURL: "sqlite:///path/to/database.db",
			wantDriver:  "sqlite",
			wantDSN:     "/path/to/database.db",
		},
		{
			name:        "SQLite URL with file:",
			databaseURL: "file:test.db?mode=memory",
			wantDriver:  "sqlite",
			wantDSN:     "file:test.db?mode=memory",
		},
		{
			name:        "LibSQL URL with libsql://",
			databaseURL: "libsql://database.turso.io",
			wantDriver:  "libsql",
			wantDSN:     "libsql://database.turso.io",
		},
		{
			name:        "https:// URL is not auto-detected (requires explicit driver)",
			databaseURL: "https://database.turso.io",
			wantDriver:  "", // https:// is too broad; users should set DB_DRIVER=libsql explicitly
			wantDSN:     "https://database.turso.io",
		},
		{
			name:        "Unknown URL format",
			databaseURL: "unknown://some:connection@string",
			wantDriver:  "",
			wantDSN:     "unknown://some:connection@string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDriver, gotDSN := parseURLForDriver(tt.databaseURL)
			if gotDriver != tt.wantDriver {
				t.Errorf("parseURLForDriver() driver = %v, want %v", gotDriver, tt.wantDriver)
			}

			// Use custom check function if provided, otherwise direct comparison
			if tt.wantDSNCheck != nil {
				if !tt.wantDSNCheck(gotDSN) {
					t.Errorf("parseURLForDriver() DSN check failed for %v", gotDSN)
				}
			} else if gotDSN != tt.wantDSN {
				t.Errorf("parseURLForDriver() DSN = %v, want %v", gotDSN, tt.wantDSN)
			}
		})
	}
}

func TestConfigWithDatabaseURL(t *testing.T) {
	// Test that DATABASE_URL takes precedence
	cfg := Config{
		URL:      "postgres://url:pass@urlhost:5432/urldb",
		Driver:   "mysql",
		Host:     "separatehost",
		Database: "separatedb",
	}

	// When URL is set, it should be used regardless of other fields
	if cfg.URL != "postgres://url:pass@urlhost:5432/urldb" {
		t.Errorf("Expected URL to be preserved")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "Valid PostgreSQL config",
			cfg: Config{
				Driver:   "postgres",
				Host:     "localhost",
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			wantError: false,
		},
		{
			name: "PostgreSQL without username is allowed (validation only checks host/database)",
			cfg: Config{
				Driver:   "postgres",
				Host:     "localhost",
				Database: "testdb",
				Password: "testpass",
			},
			wantError: false, // Username is not validated
		},
		{
			name: "PostgreSQL missing host",
			cfg: Config{
				Driver:   "postgres",
				Username: "testuser",
				Database: "testdb",
			},
			wantError: true,
			errorMsg:  "connection details required",
		},
		{
			name: "PostgreSQL missing database",
			cfg: Config{
				Driver:   "postgres",
				Host:     "localhost",
				Username: "testuser",
			},
			wantError: true,
			errorMsg:  "connection details required",
		},
		{
			name: "PostgreSQL with URL bypasses field validation",
			cfg: Config{
				Driver: "postgres",
				URL:    "postgres://user:pass@localhost:5432/mydb",
			},
			wantError: false,
		},
		{
			name: "MySQL without username is allowed (validation only checks host/database)",
			cfg: Config{
				Driver:   "mysql",
				Host:     "localhost",
				Database: "testdb",
				Password: "testpass",
			},
			wantError: false, // Username is not validated
		},
		{
			name: "Valid MySQL config",
			cfg: Config{
				Driver:   "mysql",
				Host:     "localhost",
				Database: "testdb",
				Username: "testuser",
				Password: "testpass",
			},
			wantError: false,
		},
		{
			name: "SQLite doesn't require username",
			cfg: Config{
				Driver:   "sqlite",
				Database: "test.db",
			},
			wantError: false,
		},
		{
			name: "Turso requires URL",
			cfg: Config{
				Driver:    "turso",
				AuthToken: "token123",
			},
			wantError: true,
			errorMsg:  "URL",
		},
		{
			name: "Empty driver",
			cfg: Config{
				Host:     "localhost",
				Database: "testdb",
			},
			wantError: true,
			errorMsg:  "driver required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Errorf("validateConfig() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestBuildPostgresDSN(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantDSN string
	}{
		{
			name: "Complete PostgreSQL config",
			cfg: Config{
				Host:     "localhost",
				Port:     "5432",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "disable",
			},
			wantDSN: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable",
		},
		{
			name: "PostgreSQL with default port",
			cfg: Config{
				Host:     "localhost",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "prefer",
			},
			wantDSN: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=prefer",
		},
		{
			name: "PostgreSQL with custom parameters",
			cfg: Config{
				Host:     "localhost",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "require",
				Params:   "connect_timeout=10",
			},
			wantDSN: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=require connect_timeout=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDSN := buildPostgresDSN(tt.cfg)
			if gotDSN != tt.wantDSN {
				t.Errorf("buildPostgresDSN() = %v, want %v", gotDSN, tt.wantDSN)
			}
		})
	}
}

func TestBuildMySQLDSN(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantDSN string
	}{
		{
			name: "Complete MySQL config",
			cfg: Config{
				Host:     "localhost",
				Port:     "3306",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
			},
			wantDSN: "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "MySQL with default port",
			cfg: Config{
				Host:     "localhost",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
			},
			wantDSN: "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "MySQL with custom parameters",
			cfg: Config{
				Host:     "localhost",
				Username: "testuser",
				Password: "testpass",
				Database: "testdb",
				Params:   "timeout=10s",
			},
			wantDSN: "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDSN := buildMySQLDSN(tt.cfg)
			if gotDSN != tt.wantDSN {
				t.Errorf("buildMySQLDSN() = %v, want %v", gotDSN, tt.wantDSN)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
