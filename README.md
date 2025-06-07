# Beaver Kit 🦫

A comprehensive, modular Go framework providing production-ready components for modern applications. Beaver Kit offers a collection of well-designed packages that follow consistent patterns, making it easy to build secure, scalable, and maintainable Go applications.

┌─────────────────────────────────────────────────┐
│                 BEAVER CLI                      │
│  Code generation, scaffolding, migrations, etc. │
└─────────────────────┬───────────────────────────┘
                      │ generates/manages
┌─────────────────────▼───────────────────────────┐
│             BEAVER FRAMEWORK                    │
│  Opinionated structure, conventions, patterns   │
└─────────────────────┬───────────────────────────┘
                      │ built entirely on
┌─────────────────────▼───────────────────────────┐
│               BEAVER KIT                        │
│  Modular, driver-agnostic, flexible components  │
└─────────────────────────────────────────────────┘

## 🌟 Features

- **🔧 Modular Design**: Use only what you need - each package is independent
- **🌍 Environment-First Configuration**: All packages support environment variables with the `BEAVER_` prefix
- **🎯 Configurable Prefixes**: Create multiple instances with custom environment variable prefixes
- **🏗️ Builder Pattern**: Use `WithPrefix()` for multi-tenant and custom configurations
- **🔄 Multi-Instance Support**: Run multiple instances with different configurations simultaneously
- **⚡ CLI Code Generation**: Optional Beaver CLI with API mode and template system
- **🔒 Secure by Default**: Built-in security features across all components
- **📦 Minimal Dependencies**: Lightweight core with optional integrations
- **🧪 Testing-Friendly**: Built-in support for testing with Reset() functions
- **🏭 Production-Ready**: Battle-tested components with comprehensive error handling
- **📖 Well-Documented**: Extensive documentation and examples for every package

## 📦 Available Packages

### Core Infrastructure
- **[config](#config-package)** - Environment variable configuration loader with struct tag support
- **[database](#database-package)** - Database abstraction supporting PostgreSQL, MySQL, SQLite, and Turso
- **[krypto](#krypto-package)** - Comprehensive cryptographic utilities (JWT, hashing, encryption)
- **[cache](#cache-package)** - Flexible caching with in-memory and Redis drivers

### Service Integrations
- **[captcha](#captcha-package)** - Multi-provider CAPTCHA service (Google reCAPTCHA, hCaptcha, Cloudflare Turnstile)
- **[slack](#slack-package)** - Slack webhook notifications with formatted messages
- **[urlsigner](#urlsigner-package)** - Secure URL signing for temporary access and file downloads

### File Handling
- **[filekit](#filekit-package)** - File system abstraction with local and cloud storage support
- **[filevalidator](#filevalidator-package)** - Comprehensive file validation with security features

## 🚀 Installation

```bash
go get github.com/gobeaver/beaver-kit
```

## 🎯 Quick Start

### Zero-Config Usage

Set environment variables and start using immediately:

```bash
# Set required environment variables
export BEAVER_DB_DRIVER=postgres
export BEAVER_DB_HOST=localhost
export BEAVER_DB_DATABASE=myapp
export BEAVER_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
export BEAVER_CAPTCHA_SITE_KEY=your-site-key
export BEAVER_CAPTCHA_SECRET_KEY=your-secret-key
```

```go
package main

import (
    "log"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/slack"
    "github.com/gobeaver/beaver-kit/captcha"
)

func main() {
    // All packages initialize automatically from environment
    db := database.DB()
    
    // Send a Slack notification
    slack.Slack().SendInfo("Application started successfully")
    
    // Validate a CAPTCHA token
    valid, err := captcha.Service().Validate(ctx, token, clientIP)
    if err != nil || !valid {
        slack.Slack().SendWarning("Invalid CAPTCHA attempt")
    }
}
```

### Multi-Instance Architecture

Build multiple instances with different configurations for advanced use cases:

```go
// Default instance with BEAVER_ prefix
if err := database.Init(); err != nil {
    log.Fatal(err)
}
defaultDB := database.DB()

// AWS-compatible instance (no prefix)
if err := cache.WithPrefix("").Init(); err != nil {
    log.Fatal(err)
}
// Uses: AWS_REGION, AWS_ACCESS_KEY_ID, etc.

// Multi-tenant database connections
primaryDB, err := database.WithPrefix("PRIMARY_").New()
if err != nil {
    log.Fatal(err)
}

replicaDB, err := database.WithPrefix("REPLICA_").New() 
if err != nil {
    log.Fatal(err)
}

// Separate cache instances per service
userCache, err := cache.WithPrefix("USER_").New()
sessionCache, err := cache.WithPrefix("SESSION_").New()

// Environment configuration:
// PRIMARY_DB_HOST=primary.db.example.com
// REPLICA_DB_HOST=replica.db.example.com
// USER_CACHE_DRIVER=redis
// SESSION_CACHE_DRIVER=memory
```

### Custom Prefix Examples

```bash
# Multi-tenant SaaS application
TENANT1_DB_HOST=tenant1.db.example.com
TENANT1_CACHE_DRIVER=redis
TENANT1_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T1/...

TENANT2_DB_HOST=tenant2.db.example.com  
TENANT2_CACHE_DRIVER=memory
TENANT2_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T2/...

# Microservices with service-specific configs
AUTH_DB_HOST=auth.db.example.com
USER_DB_HOST=users.db.example.com
BILLING_DB_HOST=billing.db.example.com

# Environment-specific instances
DEV_DB_HOST=dev.db.example.com
STAGING_DB_HOST=staging.db.example.com
PROD_DB_HOST=prod.db.example.com
```

## ⚡ Beaver CLI (Optional)

The Beaver CLI provides optional code generation and scaffolding with two flexible modes:

### CLI Installation

```bash
go install github.com/gobeaver/beaver-kit/cmd/beaver@latest
```

### API Mode (Recommended)

Pure programmatic generation without templates - type-safe and flexible:

```bash
# Initialize new project
beaver init my-api

# Edit beaver.yml configuration
# Then generate code
beaver generate
```

**beaver.yml example:**
```yaml
version: "1.0"
mode: "api"

project:
  name: "my-api"
  module: "github.com/user/my-api"

environment:
  prefix: "MYAPI_"

api:
  generators:
    - name: "database"
      config:
        driver: "postgres"
        migrations: true
    - name: "cache"
      config:
        driver: "redis"
        namespace: "myapi"
    - name: "auth"
      config:
        provider: "jwt"
        middleware: true

packages:
  database:
    prefix: "DB_"
  cache:
    prefix: "CACHE_"
```

### Template Mode (Optional)

Template-based generation for visual project structure:

```yaml
version: "1.0"
mode: "template"

project:
  name: "my-service"
  module: "github.com/user/my-service"

templates:
  preset: "microservice"
  variables:
    service_name: "user-service"
    database_driver: "postgres"
    docker: true
```

### CLI Benefits

**API Mode:**
- Type-safe configuration validation
- Programmatic control over generation
- No template syntax to learn
- Better error handling and debugging
- Fast execution

**Template Mode:**
- Visual project structure
- Community template sharing
- No Go knowledge required for templates
- Rapid prototyping from existing patterns

## 📚 Package Documentation

### Config Package

The foundation for all Beaver Kit packages - loads configuration from environment variables into Go structs.

```go
import "github.com/gobeaver/beaver-kit/config"

type AppConfig struct {
    DatabaseURL string `env:"DATABASE_URL,default:postgres://localhost/myapp"`
    Port        int    `env:"PORT,default:8080"`
    Debug       bool   `env:"DEBUG,default:false"`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        panic(err)
    }
    
    fmt.Printf("Starting on port %d\n", cfg.Port)
}
```

### Database Package

A flexible, SQL-first database package with optional GORM support. All drivers are pure Go implementations with zero CGO dependencies, ensuring easy cross-compilation and deployment.

#### Key Features

- **Pure Go Drivers** - No CGO required for any database
- **SQL-First Design** - Direct access to `*sql.DB` for maximum control
- **Optional GORM** - Enable ORM functionality when needed
- **Multi-Database** - PostgreSQL, MySQL, SQLite, and Turso/LibSQL support

#### Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_DB_DRIVER` | Database driver (postgres, mysql, sqlite, turso) | `sqlite` |
| `BEAVER_DB_HOST` | Database host | `localhost` |
| `BEAVER_DB_PORT` | Database port | Driver default |
| `BEAVER_DB_DATABASE` | Database name | `beaver.db` |
| `BEAVER_DB_USERNAME` | Database username | - |
| `BEAVER_DB_PASSWORD` | Database password | - |
| `BEAVER_DB_URL` | Full connection URL (overrides other settings) | - |
| `BEAVER_DB_MAX_OPEN_CONNS` | Maximum open connections | `25` |
| `BEAVER_DB_MAX_IDLE_CONNS` | Maximum idle connections | `5` |
| `BEAVER_DB_ORM` | Enable GORM support (`gorm` to enable) | - |
| `BEAVER_DB_DEBUG` | Enable debug logging | `false` |

#### Usage

```go
import "github.com/gobeaver/beaver-kit/database"

// SQL-first approach (default)
db := database.DB()
rows, err := db.Query("SELECT * FROM users WHERE active = ?", true)

// With GORM support
gormDB, err := database.WithGORM()
gormDB.Find(&users)

// Transaction helper
err := database.Transaction(ctx, func(tx *sql.Tx) error {
    _, err := tx.Exec("INSERT INTO users ...")
    return err
})

// Health checks
if database.IsHealthy() {
    stats := database.Stats()
    fmt.Printf("Open connections: %d\n", stats.OpenConnections)
}

// Shutdown gracefully
defer database.Shutdown(context.Background())
```

[Read full documentation →](database/README.md)

### Cache Package

A flexible caching solution supporting both in-memory and Redis backends. Switch drivers with just an environment variable - no code changes required.

#### Key Features

- **Multiple Drivers** - Built-in memory cache and Redis support
- **Zero Code Changes** - Switch drivers via environment variables  
- **Connection Pooling** - Optimized Redis connection management
- **TTL Support** - Set expiration times for cached values
- **Namespace Isolation** - Separate cache spaces with prefixes
- **Thread-Safe** - Safe for concurrent use

#### Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_CACHE_DRIVER` | Cache driver (`memory` or `redis`) | `memory` |
| `BEAVER_CACHE_HOST` | Redis host | `localhost` |
| `BEAVER_CACHE_PORT` | Redis port | `6379` |
| `BEAVER_CACHE_PASSWORD` | Redis password | - |
| `BEAVER_CACHE_DATABASE` | Redis database number | `0` |
| `BEAVER_CACHE_URL` | Redis URL (overrides host/port) | - |
| `BEAVER_CACHE_KEY_PREFIX` | Prefix for all keys | - |
| `BEAVER_CACHE_NAMESPACE` | Namespace for isolation | - |
| `BEAVER_CACHE_MAX_SIZE` | Max memory in bytes (memory driver) | `0` (unlimited) |
| `BEAVER_CACHE_MAX_KEYS` | Max number of keys (memory driver) | `0` (unlimited) |
| `BEAVER_CACHE_DEFAULT_TTL` | Default TTL (e.g., "5m", "1h") | `0` (no expiry) |

#### Usage

```go
import "github.com/gobeaver/beaver-kit/cache"

// Initialize from environment (BEAVER_CACHE_DRIVER=memory or redis)
if err := cache.Init(); err != nil {
    log.Fatal(err)
}

ctx := context.Background()

// Store a value with TTL
err := cache.Set(ctx, "user:123", []byte("John Doe"), 5*time.Minute)

// Retrieve a value
data, err := cache.Get(ctx, "user:123")
if err == nil {
    fmt.Printf("User: %s\n", string(data))
}

// Check if key exists
exists, err := cache.Exists(ctx, "user:123")

// Delete a key
cache.Delete(ctx, "user:123")

// Clear all keys (with prefix if configured)
cache.Clear(ctx)

// Health check
if cache.IsHealthy() {
    fmt.Println("Cache is operational")
}

// Switch drivers without code changes:
// Development: BEAVER_CACHE_DRIVER=memory
// Production:  BEAVER_CACHE_DRIVER=redis
```

#### Driver-Specific Configuration

```go
// In-memory cache with limits
memCache, err := cache.New(cache.Config{
    Driver:    "memory",
    MaxKeys:   10000,
    MaxSize:   100 * 1024 * 1024, // 100MB
    DefaultTTL: 10 * time.Minute,
})

// Redis cache with connection pooling
redisCache, err := cache.New(cache.Config{
    Driver:     "redis",
    Host:       "localhost",
    Port:       "6379", 
    Database:   0,
    PoolSize:   20,
    KeyPrefix:  "myapp:",
})
```

[Read full documentation →](cache/README.md)

### Krypto Package

Comprehensive cryptographic utilities for secure applications.

#### Features

- **Password Hashing**: Argon2id and Bcrypt implementations
- **JWT Tokens**: HS256 token generation and validation
- **Encryption**: AES-GCM encryption/decryption
- **RSA**: Key pair generation and validation
- **Utilities**: SHA-256 hashing, secure token generation, OTP generation

#### Usage

```go
import "github.com/gobeaver/beaver-kit/krypto"

// Password hashing
hash, err := krypto.Argon2idHashPassword("secure_password")
valid, err := krypto.Argon2idVerifyPassword("secure_password", hash)

// JWT tokens
claims := krypto.UserClaims{
    First: "John",
    Last:  "Doe",
    Token: "user-123",
}
token, err := krypto.NewHs256AccessToken(claims)

// AES encryption
aes := krypto.NewAESGCMService("32-byte-encryption-key-here!!!!!")
encrypted, nonce, err := aes.Encrypt([]byte("sensitive data"))
decrypted, err := aes.Decrypt(encrypted, nonce)

// Secure tokens
token, err := krypto.GenerateSecureToken(32)
otp := krypto.GenerateOTP(6)
```

### Captcha Package

Unified interface for multiple CAPTCHA providers with zero-config support.

#### Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_CAPTCHA_PROVIDER` | Provider (recaptcha, hcaptcha, turnstile) | `recaptcha` |
| `BEAVER_CAPTCHA_SITE_KEY` | Public site key | - |
| `BEAVER_CAPTCHA_SECRET_KEY` | Private secret key | - |
| `BEAVER_CAPTCHA_VERSION` | Version (only for recaptcha: 2 or 3) | `2` |
| `BEAVER_CAPTCHA_ENABLED` | Enable/disable validation | `false` |

#### Usage

```go
import "github.com/gobeaver/beaver-kit/captcha"

// Initialize from environment
if err := captcha.Init(); err != nil {
    log.Fatal(err)
}

service := captcha.Service()

// Generate HTML for forms
html := service.GenerateHTML()

// Validate token
valid, err := service.Validate(ctx, token, clientIP)
if err != nil || !valid {
    // Handle invalid captcha
}

// Direct provider usage
googleCaptcha := captcha.NewGoogleCaptcha(siteKey, secretKey, 2)
hcaptcha := captcha.NewHCaptcha(siteKey, secretKey)
turnstile := captcha.NewTurnstile(siteKey, secretKey)
```

### Slack Package

Send formatted notifications to Slack channels via webhooks.

#### Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_SLACK_WEBHOOK_URL` | Slack webhook URL | Required |
| `BEAVER_SLACK_CHANNEL` | Default channel | - |
| `BEAVER_SLACK_USERNAME` | Default username | `Beaver` |
| `BEAVER_SLACK_ICON_EMOJI` | Default emoji icon | - |
| `BEAVER_SLACK_TIMEOUT` | Request timeout | `10s` |

#### Usage

```go
import "github.com/gobeaver/beaver-kit/slack"

// Initialize from environment
if err := slack.Init(); err != nil {
    log.Fatal(err)
}

service := slack.Slack()

// Send formatted messages
service.SendInfo("Deployment completed successfully")
service.SendWarning("High memory usage detected")
service.SendAlert("Database connection lost!")

// Custom options
opts := &slack.MessageOptions{
    Channel:   "#critical-alerts",
    Username:  "AlertBot",
    IconEmoji: ":rotating_light:",
}
service.SendAlertWithOptions("Production issue detected", opts)
```

### URLSigner Package

Create secure, expiring URLs for temporary access to resources.

#### Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BEAVER_URLSIGNER_SECRET_KEY` | HMAC secret key | Required |
| `BEAVER_URLSIGNER_DEFAULT_EXPIRY` | Default expiration | `30m` |
| `BEAVER_URLSIGNER_SIGNATURE_PARAM` | Signature parameter | `sig` |
| `BEAVER_URLSIGNER_EXPIRES_PARAM` | Expiration parameter | `expires` |

#### Usage

```go
import "github.com/gobeaver/beaver-kit/urlsigner"

// Initialize from environment
if err := urlsigner.Init(); err != nil {
    log.Fatal(err)
}

signer := urlsigner.Service()

// Sign URLs
signedURL, err := signer.SignURL(
    "https://example.com/download/file.pdf",
    30*time.Minute,
    `{"user_id": 123}`, // optional payload
)

// Verify signed URLs
valid, payload, err := signer.VerifyURL(signedURL)
if valid {
    fmt.Printf("Valid URL with payload: %s\n", payload)
}

// Check expiration
expired, err := signer.IsExpired(signedURL)
remaining, err := signer.RemainingValidity(signedURL)
```

### FileKit Package

Comprehensive file system abstraction supporting local and cloud storage.

#### Features

- Local filesystem support
- S3-compatible storage (AWS S3, MinIO, etc.)
- Streaming operations for large files
- Built-in encryption layer
- File validation integration

#### Usage

```go
import (
    "github.com/gobeaver/beaver-kit/filekit"
    "github.com/gobeaver/beaver-kit/filekit/driver/local"
    "github.com/gobeaver/beaver-kit/filekit/driver/s3"
)

// Local filesystem
localFS, err := local.New("/var/uploads")

// S3 storage
s3FS := s3.New(s3Client, "my-bucket", s3.WithPrefix("uploads/"))

// Upload files
content := strings.NewReader("Hello, World!")
err = fs.Upload(ctx, "hello.txt", content, 
    filekit.WithContentType("text/plain"),
    filekit.WithMetadata(map[string]string{
        "uploaded_by": "user123",
    }),
)

// Download files
reader, err := fs.Download(ctx, "hello.txt")
defer reader.Close()

// List files
files, err := fs.List(ctx, "documents/")
for _, file := range files {
    fmt.Printf("%s (%d bytes)\n", file.Name, file.Size)
}

// Encrypted storage
encryptedFS := filekit.NewEncryptedFS(fs, encryptionKey)
```

### FileValidator Package

Comprehensive file validation with security features to prevent malicious uploads.

#### Features

- File size, MIME type, and extension validation
- Content-based validation for security
- Protection against zip bombs, malicious images, and dangerous PDFs
- Streaming validation for large files
- Detailed error types

#### Usage

```go
import "github.com/gobeaver/beaver-kit/filevalidator"

// Create validator with constraints
validator := filevalidator.New(filevalidator.Constraints{
    MaxFileSize:   10 * filevalidator.MB,
    AcceptedTypes: []string{"image/jpeg", "image/png", "application/pdf"},
    AllowedExts:   []string{".jpg", ".jpeg", ".png", ".pdf"},
    ContentValidationEnabled: true,
})

// Validate uploaded files
err := validator.Validate(fileHeader)
if err != nil {
    if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize) {
        // Handle size error
    }
}

// Use predefined constraints
imageValidator := filevalidator.New(filevalidator.ImageOnlyConstraints())
docValidator := filevalidator.New(filevalidator.DocumentOnlyConstraints())

// Stream validation for large files
err = filevalidator.StreamValidate(reader, filename, validator, 8192)
```

## 🔧 Common Patterns

### Global Instance Management

All service packages follow a consistent pattern:

```go
// Initialize with environment variables
if err := package.Init(); err != nil {
    log.Fatal(err)
}

// Or with direct configuration
err := package.Init(package.Config{
    // ... configuration
})

// Get the global instance
service := package.Service() // or package.DB(), package.Slack(), etc.

// Reset for testing
defer package.Reset()
```

### Environment Variables

All packages use the `BEAVER_` prefix by default, but this is configurable:

```bash
# Default prefix (backward compatible)
BEAVER_DB_DRIVER=postgres
BEAVER_CACHE_DRIVER=redis
BEAVER_SLACK_WEBHOOK_URL=https://...

# Custom prefix example
MYAPP_DB_DRIVER=postgres
MYAPP_CACHE_DRIVER=redis

# No prefix (AWS-style)
DB_DRIVER=postgres
CACHE_DRIVER=redis
```

Configure custom prefixes using the Builder pattern:

```go
// Use custom prefix
if err := database.WithPrefix("MYAPP_").Init(); err != nil {
    log.Fatal(err)
}

// Use no prefix
if err := cache.WithPrefix("").Init(); err != nil {
    log.Fatal(err)
}
```

Enable debug mode to see loaded configuration:

```bash
BEAVER_CONFIG_DEBUG=true ./myapp
```

### Error Handling

All packages provide detailed error types:

```go
// Check specific error types
if errors.Is(err, database.ErrNotInitialized) {
    // Handle not initialized
}

// Package-specific error checking
if filevalidator.IsErrorOfType(err, filevalidator.ErrorTypeSize) {
    // Handle size validation error
}
```

## 🏗️ Building a Complete Application

Here's an example combining multiple Beaver Kit packages:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/cache"
    "github.com/gobeaver/beaver-kit/captcha"
    "github.com/gobeaver/beaver-kit/slack"
    "github.com/gobeaver/beaver-kit/filekit"
    "github.com/gobeaver/beaver-kit/filekit/driver/s3"
    "github.com/gobeaver/beaver-kit/filevalidator"
    "github.com/gobeaver/beaver-kit/urlsigner"
    "github.com/gobeaver/beaver-kit/krypto"
)

type User struct {
    ID       uint   `gorm:"primarykey"`
    Email    string `gorm:"uniqueIndex"`
    Password string
}

func main() {
    // Initialize all services from environment
    if err := database.Init(); err != nil {
        log.Fatal(err)
    }
    if err := cache.Init(); err != nil {
        log.Fatal(err)
    }
    if err := captcha.Init(); err != nil {
        log.Fatal(err)
    }
    if err := slack.Init(); err != nil {
        log.Fatal(err)
    }
    if err := urlsigner.Init(); err != nil {
        log.Fatal(err)
    }
    
    // Get service instances
    db := database.DB()
    captchaService := captcha.Service()
    slackService := slack.Slack()
    urlSigner := urlsigner.Service()
    
    // If using GORM for migrations
    if gormDB, err := database.GORM(); err == nil {
        gormDB.AutoMigrate(&User{})
    }
    
    // Notify ops team
    slackService.SendInfo("Application started successfully")
    
    // Setup HTTP handlers
    http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
        // Verify CAPTCHA
        token := r.FormValue("captcha_token")
        valid, err := captchaService.Validate(r.Context(), token, r.RemoteAddr)
        if err != nil || !valid {
            http.Error(w, "Invalid CAPTCHA", http.StatusBadRequest)
            return
        }
        
        // Hash password
        password := r.FormValue("password")
        hashedPassword, err := krypto.Argon2idHashPassword(password)
        if err != nil {
            http.Error(w, "Error processing request", http.StatusInternalServerError)
            return
        }
        
        // Create user
        user := User{
            Email:    r.FormValue("email"),
            Password: hashedPassword,
        }
        
        // Using raw SQL
        _, err = db.Exec(`
            INSERT INTO users (email, password) 
            VALUES (?, ?)`,
            user.Email, user.Password)
        if err != nil {
            http.Error(w, "Email already exists", http.StatusConflict)
            return
        }
        
        // Generate JWT token
        claims := krypto.UserClaims{
            Token: fmt.Sprintf("%d", user.ID),
        }
        token, err := krypto.NewHs256AccessToken(claims)
        if err != nil {
            http.Error(w, "Error generating token", http.StatusInternalServerError)
            return
        }
        
        // Cache user session
        sessionKey := fmt.Sprintf("session:%s", token)
        userJSON := fmt.Sprintf(`{"id": %d, "email": "%s"}`, user.ID, user.Email)
        cache.Set(r.Context(), sessionKey, []byte(userJSON), 24*time.Hour)
        
        // Notify team
        slackService.SendInfo(fmt.Sprintf("New user registered: %s", user.Email))
        
        // Return token
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"token": "%s"}`, token)
    })
    
    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        // Rate limiting with cache
        clientIP := r.RemoteAddr
        rateLimitKey := fmt.Sprintf("rate_limit:%s", clientIP)
        
        // Check if rate limit exceeded
        if data, err := cache.Get(r.Context(), rateLimitKey); err == nil {
            // Client already made a request within the window
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        // Set rate limit for 1 minute
        cache.Set(r.Context(), rateLimitKey, []byte("1"), 1*time.Minute)
        
        // Parse multipart form
        r.ParseMultipartForm(10 << 20)
        
        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "Error retrieving file", http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        // Validate file
        validator := filevalidator.New(filevalidator.ImageOnlyConstraints())
        if err := validator.Validate(header); err != nil {
            http.Error(w, fmt.Sprintf("Invalid file: %v", err), http.StatusBadRequest)
            return
        }
        
        // Upload to S3
        s3Client := s3.New(awsS3Client, "uploads-bucket")
        path := fmt.Sprintf("images/%s", header.Filename)
        
        err = s3Client.Upload(r.Context(), path, file,
            filekit.WithContentType(header.Header.Get("Content-Type")),
        )
        if err != nil {
            http.Error(w, "Upload failed", http.StatusInternalServerError)
            return
        }
        
        // Generate signed URL for download
        downloadURL := fmt.Sprintf("https://example.com/download/%s", path)
        signedURL, err := urlSigner.SignURL(downloadURL, 24*time.Hour, "")
        if err != nil {
            http.Error(w, "Error generating download URL", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"url": "%s"}`, signedURL)
    })
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 🧪 Testing

All packages include testing utilities:

```go
func TestMyFeature(t *testing.T) {
    // Reset global state after test
    defer database.Reset()
    defer cache.Reset()
    defer slack.Reset()
    defer captcha.Reset()
    
    // Initialize with test configuration
    testDBConfig := database.Config{
        Driver:   "sqlite",
        Database: ":memory:",
    }
    
    testCacheConfig := cache.Config{
        Driver: "memory",
        MaxKeys: 1000,
    }
    
    if err := database.Init(testDBConfig); err != nil {
        t.Fatal(err)
    }
    
    if err := cache.Init(testCacheConfig); err != nil {
        t.Fatal(err)
    }
    
    // Your test code here
}
```

## 🤝 Contributing

We welcome contributions! When adding new packages or features:

1. **Follow the conventions**: Use the patterns established by existing packages
2. **Add tests**: Maintain high test coverage
3. **Document thoroughly**: Include README.md and examples
4. **Use environment variables**: Follow the `BEAVER_` prefix convention
5. **Keep it simple**: Favor clarity over cleverness

### Adding a New Package

1. Create the package directory
2. Implement the core pattern (Init, Service/Instance, Reset)
3. Add comprehensive tests
4. Create a README.md with examples
5. Update this main README

## 📝 License

Licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with ❤️ by the Beaver team, focused on providing high-performance, production-ready tools for the Go community.

---

## 📚 Additional Resources

- [Security Best Practices](docs/security.md) - Security considerations for production use
- [Performance Tuning](docs/performance.md) - Optimizing Beaver Kit applications
- [Migration Guide](docs/migration.md) - Upgrading between versions

## 🐛 Troubleshooting

### Common Issues

**Q: "service not initialized" error**  
A: Make sure to call `Init()` before using `Service()`:
```go
if err := package.Init(); err != nil {
    log.Fatal(err)
}
service := package.Service()
```

**Q: Environment variables not loading**  
A: Check that variables use the `BEAVER_` prefix and enable debug mode:
```bash
BEAVER_CONFIG_DEBUG=true ./myapp
```

**Q: Database connection errors**  
A: Verify your connection settings and that the database server is running:
```bash
BEAVER_DB_DRIVER=postgres
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp
BEAVER_DB_USERNAME=user
BEAVER_DB_PASSWORD=pass
```