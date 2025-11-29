package database

import (
	"os"
	"testing"
)

func TestDatabaseURLIntegration(t *testing.T) {
	// Skip if in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test 1: DATABASE_URL takes precedence over individual fields
	t.Run("DATABASE_URL precedence", func(t *testing.T) {
		// Set environment variables
		os.Setenv("BEAVER_DATABASE_URL", "postgres://urluser:urlpass@urlhost:5432/urldb")
		os.Setenv("BEAVER_DB_DRIVER", "mysql")
		os.Setenv("BEAVER_DB_HOST", "otherhost")
		os.Setenv("BEAVER_DB_DATABASE", "otherdb")
		defer func() {
			os.Unsetenv("BEAVER_DATABASE_URL")
			os.Unsetenv("BEAVER_DB_DRIVER")
			os.Unsetenv("BEAVER_DB_HOST")
			os.Unsetenv("BEAVER_DB_DATABASE")
		}()

		cfg, err := GetConfig()
		if err != nil {
			t.Fatalf("Failed to get config: %v", err)
		}

		// DATABASE_URL should be populated
		if cfg.URL != "postgres://urluser:urlpass@urlhost:5432/urldb" {
			t.Errorf("Expected DATABASE_URL to be used, got %s", cfg.URL)
		}

		// The separate DB_DRIVER should still be loaded
		if cfg.Driver != "mysql" {
			t.Errorf("Expected Driver to be mysql, got %s", cfg.Driver)
		}
	})

	// Test 2: Different URL formats
	t.Run("Different URL formats", func(t *testing.T) {
		testCases := []struct {
			name string
			url  string
		}{
			{"PostgreSQL", "postgres://user:pass@localhost:5432/testdb"},
			{"MySQL", "mysql://user:pass@localhost:3306/testdb"},
			{"SQLite", "sqlite:///tmp/test.db"},
			{"LibSQL", "libsql://test.turso.io"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				os.Setenv("BEAVER_DATABASE_URL", tc.url)
				defer os.Unsetenv("BEAVER_DATABASE_URL")

				cfg, err := GetConfig()
				if err != nil {
					t.Fatalf("Failed to get config: %v", err)
				}

				if cfg.URL != tc.url {
					t.Errorf("Expected URL %s, got %s", tc.url, cfg.URL)
				}
			})
		}
	})

	// Test 3: Backward compatibility - DB_URL still works (deprecated)
	t.Run("Backward compatibility", func(t *testing.T) {
		// Make sure DATABASE_URL is not set
		os.Unsetenv("BEAVER_DATABASE_URL")

		// Test that if someone still uses DB_URL in their env, it doesn't work
		// since we changed the struct tag to DATABASE_URL
		os.Setenv("BEAVER_DB_URL", "postgres://old:format@localhost:5432/olddb")
		defer os.Unsetenv("BEAVER_DB_URL")

		cfg, err := GetConfig()
		if err != nil {
			t.Fatalf("Failed to get config: %v", err)
		}

		// DB_URL should NOT work anymore since we changed the struct tag
		if cfg.URL != "" {
			t.Errorf("DB_URL should not be loaded since we changed to DATABASE_URL, got %s", cfg.URL)
		}
	})
}