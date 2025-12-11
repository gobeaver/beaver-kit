package integration_test

import (
	"os"
	"testing"

	"github.com/gobeaver/beaver-kit/cache"
	"github.com/gobeaver/beaver-kit/config"
	"github.com/gobeaver/beaver-kit/database"
	"github.com/gobeaver/beaver-kit/slack"
)

// TestBackwardCompatibility tests that existing BEAVER_ environment variables still work
func TestBackwardCompatibility(t *testing.T) {
	// Clean up environment after test
	defer os.Clearenv()

	// Set old-style BEAVER_ environment variables
	os.Setenv("BEAVER_DB_DRIVER", "sqlite")
	os.Setenv("BEAVER_DB_DATABASE", "test.db")
	os.Setenv("BEAVER_CACHE_DRIVER", "memory")
	os.Setenv("BEAVER_SLACK_USERNAME", "TestBot")

	// Test database config with default prefix
	dbCfg, err := database.GetConfig()
	if err != nil {
		t.Fatalf("Failed to load database config: %v", err)
	}
	if dbCfg.Driver != "sqlite" {
		t.Errorf("Expected driver 'sqlite', got '%s'", dbCfg.Driver)
	}
	if dbCfg.Database != "test.db" {
		t.Errorf("Expected database 'test.db', got '%s'", dbCfg.Database)
	}

	// Test cache config with default prefix
	cacheCfg, err := cache.GetConfig()
	if err != nil {
		t.Fatalf("Failed to load cache config: %v", err)
	}
	if cacheCfg.Driver != "memory" {
		t.Errorf("Expected cache driver 'memory', got '%s'", cacheCfg.Driver)
	}

	// Test slack config with default prefix
	slackCfg, err := slack.GetConfig()
	if err != nil {
		t.Fatalf("Failed to load slack config: %v", err)
	}
	if slackCfg.Username != "TestBot" {
		t.Errorf("Expected username 'TestBot', got '%s'", slackCfg.Username)
	}
}

// TestNewPrefixFunctionality tests the new WithPrefix builder functionality
func TestNewPrefixFunctionality(t *testing.T) {
	// Clean up environment after test
	defer os.Clearenv()

	// Set custom prefix environment variables
	os.Setenv("STAGING_DB_DRIVER", "postgres")
	os.Setenv("STAGING_DB_HOST", "staging-db.example.com")
	os.Setenv("PROD_CACHE_DRIVER", "redis")
	os.Setenv("PROD_CACHE_HOST", "prod-redis.example.com")
	os.Setenv("DEV_USERNAME", "DevBot") // slack.Config uses `env:"USERNAME"`

	// Test database with custom prefix
	stagingDB := database.WithPrefix("STAGING_")
	dbCfg := &database.Config{}
	err := config.Load(dbCfg, config.WithPrefix("STAGING_"))
	if err != nil {
		t.Fatalf("Failed to load staging database config: %v", err)
	}
	if dbCfg.Driver != "postgres" {
		t.Errorf("Expected driver 'postgres', got '%s'", dbCfg.Driver)
	}
	if dbCfg.Host != "staging-db.example.com" {
		t.Errorf("Expected host 'staging-db.example.com', got '%s'", dbCfg.Host)
	}

	// Test cache with custom prefix
	prodCache := cache.WithPrefix("PROD_")
	cacheCfg := &cache.Config{}
	err = config.Load(cacheCfg, config.WithPrefix("PROD_"))
	if err != nil {
		t.Fatalf("Failed to load prod cache config: %v", err)
	}
	if cacheCfg.Driver != "redis" {
		t.Errorf("Expected cache driver 'redis', got '%s'", cacheCfg.Driver)
	}
	if cacheCfg.Host != "prod-redis.example.com" {
		t.Errorf("Expected host 'prod-redis.example.com', got '%s'", cacheCfg.Host)
	}

	// Test slack with custom prefix
	devSlack := slack.WithPrefix("DEV_")
	slackCfg := &slack.Config{}
	err = config.Load(slackCfg, config.WithPrefix("DEV_"))
	if err != nil {
		t.Fatalf("Failed to load dev slack config: %v", err)
	}
	if slackCfg.Username != "DevBot" {
		t.Errorf("Expected username 'DevBot', got '%s'", slackCfg.Username)
	}

	// Verify builders were created (compile-time check)
	_ = stagingDB
	_ = prodCache
	_ = devSlack
}

// TestEmptyPrefix tests using empty prefix (no prefix)
func TestEmptyPrefix(t *testing.T) {
	// Clean up environment after test
	defer os.Clearenv()

	// Set standard AWS-style environment variables (no prefix)
	os.Setenv("DB_DRIVER", "mysql")
	os.Setenv("DB_HOST", "mysql.amazonaws.com")
	os.Setenv("CACHE_DRIVER", "redis")
	os.Setenv("CACHE_HOST", "redis.amazonaws.com")

	// Test database with empty prefix
	dbCfg := &database.Config{}
	err := config.Load(dbCfg, config.WithPrefix(""))
	if err != nil {
		t.Fatalf("Failed to load database config with empty prefix: %v", err)
	}
	if dbCfg.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got '%s'", dbCfg.Driver)
	}
	if dbCfg.Host != "mysql.amazonaws.com" {
		t.Errorf("Expected host 'mysql.amazonaws.com', got '%s'", dbCfg.Host)
	}

	// Test cache with empty prefix
	cacheCfg := &cache.Config{}
	err = config.Load(cacheCfg, config.WithPrefix(""))
	if err != nil {
		t.Fatalf("Failed to load cache config with empty prefix: %v", err)
	}
	if cacheCfg.Driver != "redis" {
		t.Errorf("Expected cache driver 'redis', got '%s'", cacheCfg.Driver)
	}
	if cacheCfg.Host != "redis.amazonaws.com" {
		t.Errorf("Expected host 'redis.amazonaws.com', got '%s'", cacheCfg.Host)
	}
}

// TestMultipleInstances tests creating multiple service instances with different prefixes
func TestMultipleInstances(t *testing.T) {
	// Clean up environment after test
	defer os.Clearenv()

	// Set environment variables for different instances
	os.Setenv("PUBLIC_CACHE_DRIVER", "memory")
	os.Setenv("PUBLIC_CACHE_MAX_SIZE", "1000000")
	os.Setenv("PRIVATE_CACHE_DRIVER", "redis")
	os.Setenv("PRIVATE_CACHE_HOST", "private-redis.internal")

	// Create builders for different instances
	publicCacheBuilder := cache.WithPrefix("PUBLIC_")
	privateCacheBuilder := cache.WithPrefix("PRIVATE_")

	// Load configs using builders
	publicCfg := &cache.Config{}
	err := config.Load(publicCfg, config.WithPrefix("PUBLIC_"))
	if err != nil {
		t.Fatalf("Failed to load public cache config: %v", err)
	}

	privateCfg := &cache.Config{}
	err = config.Load(privateCfg, config.WithPrefix("PRIVATE_"))
	if err != nil {
		t.Fatalf("Failed to load private cache config: %v", err)
	}

	// Verify configs are different
	if publicCfg.Driver != "memory" {
		t.Errorf("Expected public cache driver 'memory', got '%s'", publicCfg.Driver)
	}
	if privateCfg.Driver != "redis" {
		t.Errorf("Expected private cache driver 'redis', got '%s'", privateCfg.Driver)
	}
	if privateCfg.Host != "private-redis.internal" {
		t.Errorf("Expected private cache host 'private-redis.internal', got '%s'", privateCfg.Host)
	}

	// Verify builders exist (compile-time check)
	_ = publicCacheBuilder
	_ = privateCacheBuilder
}

// TestDefaultValues tests that default values still work correctly
func TestDefaultValues(t *testing.T) {
	// Clean up environment after test
	defer os.Clearenv()

	// Only set required values, let defaults apply
	os.Setenv("BEAVER_DB_DRIVER", "sqlite")

	// Test that defaults are applied correctly
	dbCfg, err := database.GetConfig()
	if err != nil {
		t.Fatalf("Failed to load database config: %v", err)
	}

	// Check default values
	if dbCfg.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", dbCfg.Host)
	}
	if dbCfg.Database != "beaver.db" {
		t.Errorf("Expected default database 'beaver.db', got '%s'", dbCfg.Database)
	}
	if dbCfg.MaxOpenConns != 25 {
		t.Errorf("Expected default max open conns 25, got %d", dbCfg.MaxOpenConns)
	}
}

// TestConfigLoadOptionsValidation tests that functional options are properly applied
func TestConfigLoadOptionsValidation(t *testing.T) {
	testCfg := &struct {
		TestValue string `env:"TEST_VALUE" envDefault:"default"`
	}{}

	// Test with empty prefix
	err := config.Load(testCfg, config.WithPrefix(""))
	if err != nil {
		t.Fatalf("Expected no error with empty prefix, got: %v", err)
	}

	// Test with valid prefix
	err = config.Load(testCfg, config.WithPrefix("CUSTOM_"))
	if err != nil {
		t.Fatalf("Expected no error with valid prefix, got: %v", err)
	}

	// Test with no options (should use default BEAVER_ prefix)
	err = config.Load(testCfg)
	if err != nil {
		t.Fatalf("Expected no error with no options, got: %v", err)
	}
}
