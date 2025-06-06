---
title: "Cache Package API Reference"
tags: ["cache", "redis", "memory", "performance", "distributed"]
prerequisites:
  - "getting-started"
  - "config"
relatedDocs:
  - "database"
  - "integration-patterns"
---

# Cache Package

## Overview

The cache package provides a flexible caching solution for Go applications that supports both in-memory and Redis backends. It follows Beaver Kit's driver-agnostic design, allowing you to switch between implementations with just an environment variable change.

```json
{
  "@context": "http://schema.org",
  "@type": "TechArticle",
  "name": "Cache Package API Reference",
  "about": "Flexible caching solution with multiple backend drivers",
  "programmingLanguage": "Go",
  "codeRepository": "https://github.com/gobeaver/beaver-kit",
  "keywords": ["cache", "redis", "memory", "performance", "distributed", "ttl"]
}
```

## Key Features

- **Multiple Drivers** - Built-in memory cache and Redis support
- **Zero Code Changes** - Switch drivers via environment variables
- **Environment-First Configuration** - Follows Beaver Kit's `BEAVER_` prefix convention
- **Connection Pooling** - Optimized Redis connection management
- **TTL Support** - Set expiration times for cached values
- **Namespace Isolation** - Separate cache spaces with prefixes
- **Health Checks** - Built-in health monitoring
- **Thread-Safe** - Safe for concurrent use across goroutines

## Quick Start

### Environment Configuration

```bash
# Purpose: Configure cache driver and connection settings
# Prerequisites: Redis server running (if using Redis driver)
# Expected outcome: Cache package ready for initialization

# For in-memory cache (default)
BEAVER_CACHE_DRIVER=memory
BEAVER_CACHE_MAX_SIZE=104857600  # 100MB
BEAVER_CACHE_MAX_KEYS=10000
BEAVER_CACHE_DEFAULT_TTL=5m

# For Redis cache
BEAVER_CACHE_DRIVER=redis
BEAVER_CACHE_HOST=localhost
BEAVER_CACHE_PORT=6379
BEAVER_CACHE_PASSWORD=your_redis_password
BEAVER_CACHE_DATABASE=0

# Connection pooling (Redis only)
BEAVER_CACHE_POOL_SIZE=10
BEAVER_CACHE_MIN_IDLE_CONNS=2
BEAVER_CACHE_MAX_IDLE_CONNS=5

# Namespace and prefixes
BEAVER_CACHE_KEY_PREFIX=myapp:
BEAVER_CACHE_NAMESPACE=prod
```

### Basic Usage

```go
// Purpose: Initialize and use cache with environment configuration
// Prerequisites: Environment variables configured
// Expected outcome: Cache operations working with configured driver

package main

import (
    "context"
    "log"
    "time"
    
    "github.com/gobeaver/beaver-kit/cache"
)

func main() {
    // Initialize from environment
    if err := cache.Init(); err != nil {
        log.Fatal("Cache initialization failed:", err)
    }
    
    ctx := context.Background()
    
    // Store a value with TTL
    err := cache.Set(ctx, "user:123", []byte("John Doe"), 5*time.Minute)
    if err != nil {
        log.Fatal("Cache set failed:", err)
    }
    
    // Retrieve the value
    data, err := cache.Get(ctx, "user:123")
    if err != nil {
        if errors.Is(err, cache.ErrKeyNotFound) {
            log.Println("Key not found in cache")
        } else {
            log.Fatal("Cache get failed:", err)
        }
    } else {
        log.Printf("Retrieved from cache: %s", string(data))
    }
    
    // Check if key exists
    exists, err := cache.Exists(ctx, "user:123")
    if err != nil {
        log.Fatal("Cache exists check failed:", err)
    }
    log.Printf("Key exists: %t", exists)
    
    // Delete the key
    err = cache.Delete(ctx, "user:123")
    if err != nil {
        log.Fatal("Cache delete failed:", err)
    }
}
```

## API Reference

### Global Operations

```go
// Purpose: Perform cache operations using global instance
// Prerequisites: Cache must be initialized first
// Expected outcome: Cache operations with configured driver

// Store value with TTL (0 = no expiry)
func Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

// Retrieve value by key
func Get(ctx context.Context, key string) ([]byte, error)

// Check if key exists
func Exists(ctx context.Context, key string) (bool, error)

// Delete key
func Delete(ctx context.Context, key string) error

// Clear all keys (respects prefix/namespace)
func Clear(ctx context.Context) error

// Check cache health
func IsHealthy() bool

// Reset global instance (for testing)
func Reset()
```

## Driver Comparison

### Memory Driver
- **Best for**: Development, single-instance applications
- **Features**: Zero dependencies, fast access, memory limits
- **Limitations**: No persistence, single process only

### Redis Driver  
- **Best for**: Production, distributed applications
- **Features**: Persistence, clustering, advanced data types
- **Limitations**: Network latency, requires Redis server

## Error Handling

```go
// Package-specific errors
var (
    ErrKeyNotFound    = errors.New("key not found")
    ErrNotInitialized = errors.New("cache not initialized")
    ErrInvalidDriver  = errors.New("invalid cache driver")
)

// Example error handling
data, err := cache.Get(ctx, "key")
if err != nil {
    if errors.Is(err, cache.ErrKeyNotFound) {
        // Handle cache miss
        return fetchFromDatabase(key)
    }
    return fmt.Errorf("cache error: %w", err)
}
```

## Testing

```go
func TestCacheOperations(t *testing.T) {
    defer cache.Reset() // Clean up after test
    
    config := cache.Config{
        Driver: "memory",
        MaxSize: 10 * 1024 * 1024, // 10MB for tests
    }
    
    if err := cache.Init(config); err != nil {
        t.Fatal("Failed to initialize test cache:", err)
    }
    
    // Test operations...
}
```