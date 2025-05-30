# Cache Package

A flexible caching solution for Go applications that supports both in-memory and Redis backends. Switch between drivers with just an environment variable - no code changes required.

## Features

- **Multiple Drivers**: Built-in memory cache and Redis support
- **Zero Code Changes**: Switch drivers via environment variables
- **Environment-First**: Follows Beaver Kit's `BEAVER_` prefix convention
- **Connection Pooling**: Optimized Redis connection management
- **TTL Support**: Set expiration times for cached values
- **Namespace Isolation**: Separate cache spaces with prefixes
- **Health Checks**: Built-in health monitoring
- **Thread-Safe**: Safe for concurrent use

## Installation

```bash
go get github.com/gobeaver/beaver-kit/cache
```

## Quick Start

### Using Environment Variables

```bash
# For in-memory cache (default)
export BEAVER_CACHE_DRIVER=memory

# For Redis
export BEAVER_CACHE_DRIVER=redis
export BEAVER_CACHE_HOST=localhost
export BEAVER_CACHE_PORT=6379
```

```go
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
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Store a value
    err := cache.Set(ctx, "user:123", []byte("John Doe"), 5*time.Minute)
    if err != nil {
        log.Fatal(err)
    }
    
    // Retrieve a value
    data, err := cache.Get(ctx, "user:123")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User: %s", string(data))
}
```

### Using Specific Configuration

```go
// In-memory cache
memCache, err := cache.New(cache.Config{
    Driver:    "memory",
    MaxKeys:   10000,
    MaxSize:   100 * 1024 * 1024, // 100MB
    DefaultTTL: 10 * time.Minute,
})

// Redis cache
redisCache, err := cache.New(cache.Config{
    Driver:   "redis",
    Host:     "localhost",
    Port:     "6379",
    Database: 0,
    KeyPrefix: "myapp:",
})
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BEAVER_CACHE_DRIVER` | Cache driver: `memory` or `redis` | `memory` |
| `BEAVER_CACHE_HOST` | Redis host | `localhost` |
| `BEAVER_CACHE_PORT` | Redis port | `6379` |
| `BEAVER_CACHE_PASSWORD` | Redis password | - |
| `BEAVER_CACHE_DATABASE` | Redis database number | `0` |
| `BEAVER_CACHE_URL` | Redis URL (overrides host/port) | - |
| `BEAVER_CACHE_KEY_PREFIX` | Prefix for all keys | - |
| `BEAVER_CACHE_NAMESPACE` | Namespace for isolation | - |
| **Memory Cache Settings** | | |
| `BEAVER_CACHE_MAX_SIZE` | Max memory in bytes | `0` (unlimited) |
| `BEAVER_CACHE_MAX_KEYS` | Max number of keys | `0` (unlimited) |
| `BEAVER_CACHE_DEFAULT_TTL` | Default TTL (e.g., "5m", "1h") | `0` (no expiry) |
| `BEAVER_CACHE_CLEANUP_INTERVAL` | Cleanup interval | `1m` |
| **Redis Connection Pool** | | |
| `BEAVER_CACHE_POOL_SIZE` | Connection pool size | `10` |
| `BEAVER_CACHE_MIN_IDLE_CONNS` | Min idle connections | `2` |
| `BEAVER_CACHE_MAX_IDLE_CONNS` | Max idle connections | `5` |
| `BEAVER_CACHE_MAX_RETRIES` | Max retry attempts | `3` |
| **TLS Settings** | | |
| `BEAVER_CACHE_USE_TLS` | Enable TLS | `false` |
| `BEAVER_CACHE_CERT_FILE` | TLS certificate file | - |
| `BEAVER_CACHE_KEY_FILE` | TLS key file | - |
| `BEAVER_CACHE_CA_FILE` | TLS CA file | - |

## Driver Features

### Memory Driver

- Zero dependencies
- Fast for small datasets
- Automatic cleanup of expired items
- Memory and key count limits
- Best for: Development, small apps, temporary data

### Redis Driver

- Distributed caching across servers
- Persistence options
- Pub/sub capabilities
- Clustering support
- Best for: Production, microservices, shared cache

## API Reference

### Core Operations

```go
// Get retrieves a value by key
Get(ctx context.Context, key string) ([]byte, error)

// Set stores a value with optional TTL
Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

// Delete removes a key
Delete(ctx context.Context, key string) error

// Exists checks if a key exists
Exists(ctx context.Context, key string) (bool, error)

// Clear removes all keys (with prefix if configured)
Clear(ctx context.Context) error

// Ping checks if cache is reachable
Ping(ctx context.Context) error
```

### Global Functions

All operations are available as package-level functions after initialization:

```go
cache.Init()
cache.Set(ctx, "key", data, ttl)
cache.Get(ctx, "key")
cache.Delete(ctx, "key")
cache.Exists(ctx, "key")
cache.Clear(ctx)
cache.IsHealthy()
```

## Examples

### Switching Drivers Without Code Changes

```go
// Your code remains the same
func saveUserSession(userID string, session []byte) error {
    return cache.Set(context.Background(), 
        fmt.Sprintf("session:%s", userID), 
        session, 
        30*time.Minute)
}

// Switch driver via environment
// Development: BEAVER_CACHE_DRIVER=memory
// Production:  BEAVER_CACHE_DRIVER=redis
```

### Using Namespaces

```go
// Separate cache spaces for different services
userCache, _ := cache.New(cache.Config{
    Driver:    "redis",
    Namespace: "users",
})

orderCache, _ := cache.New(cache.Config{
    Driver:    "redis",
    Namespace: "orders",
})

// Keys are automatically prefixed: "users:123", "orders:456"
```

### Connection URL

```go
// Use Redis URL for cloud providers
cache.Init(cache.Config{
    Driver: "redis",
    URL:    "redis://user:pass@redis.example.com:6379/0",
})
```

### Health Monitoring

```go
// Check cache health
if !cache.IsHealthy() {
    log.Println("Cache is down!")
}

// Or with context
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

if err := cache.Health(ctx); err != nil {
    log.Printf("Cache unhealthy: %v", err)
}
```

## Testing

The package includes comprehensive tests for both drivers:

```bash
# Run all tests
go test ./cache/...

# Test with Redis (requires running Redis)
docker run -d -p 6379:6379 redis:alpine
go test ./cache/...
```

## Performance Considerations

### Memory Driver
- O(1) operations for get/set/delete
- Cleanup runs periodically (configurable)
- Best for <100MB of data
- Consider MaxKeys to prevent unbounded growth

### Redis Driver
- Network latency considerations
- Use connection pooling settings
- Enable pipelining for batch operations
- Consider Redis persistence settings

## Error Handling

```go
data, err := cache.Get(ctx, "key")
if err != nil {
    if errors.Is(err, cache.ErrKeyNotFound) {
        // Key doesn't exist
    } else {
        // Other error (network, etc.)
    }
}
```

## Thread Safety

Both drivers are safe for concurrent use. The memory driver uses fine-grained locking, while Redis handles concurrency at the server level.