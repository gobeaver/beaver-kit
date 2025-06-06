package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/cache"
)

func TestCacheService(t *testing.T) {
	// Test with memory driver
	t.Run("MemoryDriver", func(t *testing.T) {
		cfg := cache.Config{
			Driver:    "memory",
			MaxKeys:   100,
			MaxSize:   1024 * 1024, // 1MB
			DefaultTTL: "5m",
		}

		c, err := cache.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create memory cache: %v", err)
		}
		defer c.Close()

		testCacheOperations(t, c)
	})

	// Test with Redis driver (skip if Redis not available)
	t.Run("RedisDriver", func(t *testing.T) {
		cfg := cache.Config{
			Driver:   "redis",
			Host:     "localhost",
			Port:     "6379",
			Database: 1,
			KeyPrefix: "test:",
		}

		c, err := cache.New(cfg)
		if err != nil {
			t.Skipf("Redis not available: %v", err)
		}
		defer c.Close()

		testCacheOperations(t, c)
	})
}

func testCacheOperations(t *testing.T, c cache.Cache) {
	ctx := context.Background()

	// Test Set and Get
	key := "test-key"
	value := []byte("test-value")

	err := c.Set(ctx, key, value, 1*time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	got, err := c.Get(ctx, key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get returned wrong value: got %s, want %s", got, value)
	}

	// Test Exists
	exists, err := c.Exists(ctx, key)
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist")
	}

	// Test Delete
	err = c.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	exists, err = c.Exists(ctx, key)
	if err != nil {
		t.Errorf("Exists after delete failed: %v", err)
	}
	if exists {
		t.Error("Key should not exist after delete")
	}

	// Test TTL expiration
	err = c.Set(ctx, "ttl-key", []byte("ttl-value"), 100*time.Millisecond)
	if err != nil {
		t.Errorf("Set with TTL failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = c.Get(ctx, "ttl-key")
	if err == nil {
		t.Error("Key should have expired")
	}

	// Test Clear
	c.Set(ctx, "key1", []byte("value1"), 0)
	c.Set(ctx, "key2", []byte("value2"), 0)

	err = c.Clear(ctx)
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	exists, _ = c.Exists(ctx, "key1")
	if exists {
		t.Error("Key1 should not exist after clear")
	}

	exists, _ = c.Exists(ctx, "key2")
	if exists {
		t.Error("Key2 should not exist after clear")
	}

	// Test Ping
	err = c.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestGlobalInstance(t *testing.T) {
	// Reset to ensure clean state
	cache.Reset()

	// Test initialization from environment
	t.Setenv("BEAVER_CACHE_DRIVER", "memory")
	t.Setenv("BEAVER_CACHE_MAX_KEYS", "50")

	err := cache.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	ctx := context.Background()

	// Test global functions
	err = cache.Set(ctx, "global-key", []byte("global-value"), 1*time.Minute)
	if err != nil {
		t.Errorf("Global Set failed: %v", err)
	}

	value, err := cache.Get(ctx, "global-key")
	if err != nil {
		t.Errorf("Global Get failed: %v", err)
	}

	if string(value) != "global-value" {
		t.Errorf("Wrong value: got %s, want global-value", value)
	}

	// Test health check
	if !cache.IsHealthy() {
		t.Error("Cache should be healthy")
	}

	// Cleanup
	cache.Reset()
}