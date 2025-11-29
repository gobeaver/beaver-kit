---
title: "Getting Started with Beaver Kit"
tags: ["getting-started", "installation", "configuration", "quickstart"]
prerequisites: []
relatedDocs:
  - "framework-overview"
  - "core-concepts"
  - "integration-patterns"
---

# Getting Started with Beaver Kit

## Installation

### Prerequisites

- Go 1.21 or later
- Git for version control

### Install Beaver Kit

```bash
# Purpose: Install Beaver Kit framework
# Prerequisites: Go 1.21+ installed and configured
# Expected outcome: Beaver Kit available for import in Go projects

go get github.com/gobeaver/beaver-kit
```

### Verify Installation

```go
// Purpose: Verify Beaver Kit installation
// Prerequisites: Beaver Kit installed via go get
// Expected outcome: Successful compilation and execution

package main

import (
    "fmt"
    "github.com/gobeaver/beaver-kit/config"
)

func main() {
    fmt.Println("Beaver Kit installed successfully!")
}
```

## Quick Start (5 Minutes)

### Step 1: Environment Configuration

Create a `.env` file in your project root:

```bash
# Purpose: Configure Beaver Kit packages via environment variables
# Prerequisites: Project directory created
# Expected outcome: All packages configured for immediate use

# Database configuration
BEAVER_DB_DRIVER=sqlite
BEAVER_DB_DATABASE=./app.db

# Cache configuration
BEAVER_CACHE_DRIVER=memory

# Optional: Slack notifications
BEAVER_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Optional: Enable debug logging
BEAVER_CONFIG_DEBUG=true
```

### Step 2: Basic Application

```go
// Purpose: Create a basic application using multiple Beaver Kit packages
// Prerequisites: Environment variables configured
// Expected outcome: Working application with database, cache, and notifications

package main

import (
    "context"
    "log"
    "time"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/cache"
    "github.com/gobeaver/beaver-kit/slack"
)

func main() {
    // Initialize all services from environment
    if err := initializeServices(); err != nil {
        log.Fatal("Failed to initialize services:", err)
    }
    
    // Use the services
    if err := runApplication(); err != nil {
        log.Fatal("Application error:", err)
    }
}

func initializeServices() error {
    // Purpose: Initialize all Beaver Kit services
    // Prerequisites: Environment variables set
    // Expected outcome: All services ready for use
    
    // Initialize database
    if err := database.Init(); err != nil {
        return fmt.Errorf("database init failed: %w", err)
    }
    
    // Initialize cache
    if err := cache.Init(); err != nil {
        return fmt.Errorf("cache init failed: %w", err)
    }
    
    // Initialize Slack (optional, will succeed even if not configured)
    slack.Init() // Ignoring error for optional service
    
    return nil
}

func runApplication() error {
    // Purpose: Demonstrate basic usage of initialized services
    // Prerequisites: Services must be initialized
    // Expected outcome: Database query, cache operation, and notification sent
    
    ctx := context.Background()
    
    // Get service instances
    db := database.DB()
    
    // Create a simple table
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return fmt.Errorf("table creation failed: %w", err)
    }
    
    // Insert a user
    result, err := db.Exec(
        "INSERT INTO users (name, email) VALUES (?, ?)",
        "John Doe", "john@example.com",
    )
    if err != nil {
        return fmt.Errorf("user insertion failed: %w", err)
    }
    
    userID, _ := result.LastInsertId()
    log.Printf("Created user with ID: %d", userID)
    
    // Cache the user data
    userData := fmt.Sprintf(`{"id": %d, "name": "John Doe", "email": "john@example.com"}`, userID)
    err = cache.Set(ctx, fmt.Sprintf("user:%d", userID), []byte(userData), 5*time.Minute)
    if err != nil {
        return fmt.Errorf("cache set failed: %w", err)
    }
    
    // Retrieve from cache
    cachedData, err := cache.Get(ctx, fmt.Sprintf("user:%d", userID))
    if err != nil {
        return fmt.Errorf("cache get failed: %w", err)
    }
    
    log.Printf("Retrieved from cache: %s", string(cachedData))
    
    // Send notification (if configured)
    if slackService := slack.Slack(); slackService != nil {
        slackService.SendInfo(fmt.Sprintf("New user created: %s", "John Doe"))
    }
    
    return nil
}
```

### Step 3: Run the Application

```bash
# Purpose: Execute the basic application
# Prerequisites: Go code saved to main.go, environment configured
# Expected outcome: Application runs successfully with output

go run main.go
```

Expected output:
```
[BEAVER] BEAVER_DB_DRIVER=sqlite
[BEAVER] BEAVER_DB_DATABASE=./app.db
[BEAVER] BEAVER_CACHE_DRIVER=memory
2024/01/01 12:00:00 Created user with ID: 1
2024/01/01 12:00:00 Retrieved from cache: {"id": 1, "name": "John Doe", "email": "john@example.com"}
```

## Progressive Learning Path

### Level 1: Single Package Usage

Start with individual packages to understand the patterns:

```go
// Purpose: Learn basic config package usage
// Prerequisites: Basic Go knowledge
// Expected outcome: Understanding of environment variable loading

package main

import (
    "fmt"
    "github.com/gobeaver/beaver-kit/config"
)

type AppConfig struct {
    Port        int    `env:"PORT,default:8080"`
    DatabaseURL string `env:"DATABASE_URL,default:sqlite://app.db"`
    Debug       bool   `env:"DEBUG,default:false"`
}

func main() {
    cfg := &AppConfig{}
    if err := config.Load(cfg); err != nil {
        panic(err)
    }
    
    fmt.Printf("Port: %d\n", cfg.Port)
    fmt.Printf("Database: %s\n", cfg.DatabaseURL)
    fmt.Printf("Debug: %t\n", cfg.Debug)
}
```

### Level 2: Multi-Package Integration

Combine packages to see how they work together:

```go
// Purpose: Integrate database and cache packages
// Prerequisites: Understanding of individual packages
// Expected outcome: Data persistence with caching layer

package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/cache"
)

type User struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // Initialize services
    database.Init()
    cache.Init()
    
    // Create and cache user
    user, err := createUser("Jane Doe", "jane@example.com")
    if err != nil {
        panic(err)
    }
    
    // Retrieve user (will use cache)
    retrieved, err := getUser(user.ID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Retrieved user: %+v\n", retrieved)
}

func createUser(name, email string) (*User, error) {
    // Purpose: Create user with database persistence and caching
    // Prerequisites: Database and cache initialized
    // Expected outcome: User created and cached
    
    db := database.DB()
    ctx := context.Background()
    
    // Insert to database
    result, err := db.Exec(
        "INSERT INTO users (name, email) VALUES (?, ?)",
        name, email,
    )
    if err != nil {
        return nil, err
    }
    
    id, _ := result.LastInsertId()
    user := &User{ID: id, Name: name, Email: email}
    
    // Cache the user
    userData, _ := json.Marshal(user)
    cache.Set(ctx, fmt.Sprintf("user:%d", id), userData, 10*time.Minute)
    
    return user, nil
}

func getUser(id int64) (*User, error) {
    // Purpose: Retrieve user with cache-first strategy
    // Prerequisites: User exists in database
    // Expected outcome: User data from cache or database
    
    ctx := context.Background()
    cacheKey := fmt.Sprintf("user:%d", id)
    
    // Try cache first
    if data, err := cache.Get(ctx, cacheKey); err == nil {
        var user User
        if err := json.Unmarshal(data, &user); err == nil {
            fmt.Println("Retrieved from cache")
            return &user, nil
        }
    }
    
    // Fallback to database
    db := database.DB()
    var user User
    err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).
        Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    userData, _ := json.Marshal(user)
    cache.Set(ctx, cacheKey, userData, 10*time.Minute)
    
    fmt.Println("Retrieved from database")
    return &user, nil
}
```

### Level 3: Production Application

Build a complete application with all packages:

```go
// Purpose: Create production-ready application using multiple Beaver Kit packages
// Prerequisites: Understanding of all individual packages
// Expected outcome: Complete web application with all features

package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/cache"
    "github.com/gobeaver/beaver-kit/captcha"
    "github.com/gobeaver/beaver-kit/slack"
    "github.com/gobeaver/beaver-kit/krypto"
)

type User struct {
    ID       int64  `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"-"` // Never serialize password
}

func main() {
    // Initialize all services
    initServices()
    
    // Setup routes
    http.HandleFunc("/register", registerHandler)
    http.HandleFunc("/login", loginHandler)
    http.HandleFunc("/user/", userHandler)
    
    fmt.Println("Server starting on :8080")
    http.ListenAndServe(":8080", nil)
}

func initServices() {
    // Purpose: Initialize all Beaver Kit services for production use
    // Prerequisites: Environment variables configured
    // Expected outcome: All services ready for production traffic
    
    database.Init()
    cache.Init()
    captcha.Init()
    slack.Init()
    
    // Setup database schema
    setupDatabase()
    
    // Notify startup
    slack.Slack().SendInfo("Application started successfully")
}

func setupDatabase() {
    // Purpose: Create necessary database tables
    // Prerequisites: Database service initialized
    // Expected outcome: Database schema ready
    
    db := database.DB()
    db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle user registration with CAPTCHA validation
    // Prerequisites: CAPTCHA service configured
    // Expected outcome: User registered or appropriate error response
    
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse form data
    name := r.FormValue("name")
    email := r.FormValue("email")
    password := r.FormValue("password")
    captchaToken := r.FormValue("captcha_token")
    
    // Validate CAPTCHA
    valid, err := captcha.Service().Validate(r.Context(), captchaToken, r.RemoteAddr)
    if err != nil || !valid {
        http.Error(w, "Invalid CAPTCHA", http.StatusBadRequest)
        slack.Slack().SendWarning(fmt.Sprintf("Invalid CAPTCHA attempt from %s", r.RemoteAddr))
        return
    }
    
    // Hash password
    hashedPassword, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        http.Error(w, "Registration failed", http.StatusInternalServerError)
        return
    }
    
    // Create user
    db := database.DB()
    result, err := db.Exec(
        "INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
        name, email, hashedPassword,
    )
    if err != nil {
        http.Error(w, "Email already exists", http.StatusConflict)
        return
    }
    
    userID, _ := result.LastInsertId()
    
    // Cache user data (without password)
    user := User{ID: userID, Name: name, Email: email}
    userData, _ := json.Marshal(user)
    cache.Set(r.Context(), fmt.Sprintf("user:%d", userID), userData, time.Hour)
    
    // Notify new registration
    slack.Slack().SendInfo(fmt.Sprintf("New user registered: %s (%s)", name, email))
    
    // Return success
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "user_id": userID,
    })
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle user login with password verification
    // Prerequisites: User exists in database
    // Expected outcome: JWT token or authentication error
    
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    email := r.FormValue("email")
    password := r.FormValue("password")
    
    // Get user from database
    var user User
    db := database.DB()
    err := db.QueryRow("SELECT id, name, email, password FROM users WHERE email = ?", email).
        Scan(&user.ID, &user.Name, &user.Email, &user.Password)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    // Verify password
    valid, err := krypto.Argon2idVerifyPassword(password, user.Password)
    if err != nil || !valid {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    // Generate JWT token
    claims := krypto.UserClaims{
        First: user.Name,
        Token: fmt.Sprintf("%d", user.ID),
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
        },
    }
    
    token, err := krypto.NewHs256AccessToken(claims)
    if err != nil {
        http.Error(w, "Token generation failed", http.StatusInternalServerError)
        return
    }
    
    // Cache user session
    sessionData, _ := json.Marshal(map[string]interface{}{
        "user_id": user.ID,
        "name":    user.Name,
        "email":   user.Email,
    })
    cache.Set(r.Context(), fmt.Sprintf("session:%s", token), sessionData, 24*time.Hour)
    
    // Return token
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "token": token,
    })
}

func userHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle user profile requests with caching
    // Prerequisites: Valid user ID in URL path
    // Expected outcome: User profile data or appropriate error
    
    // Extract user ID from URL
    userIDStr := r.URL.Path[len("/user/"):]
    userID, err := strconv.ParseInt(userIDStr, 10, 64)
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }
    
    // Try cache first
    cacheKey := fmt.Sprintf("user:%d", userID)
    if data, err := cache.Get(r.Context(), cacheKey); err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-Cache", "HIT")
        w.Write(data)
        return
    }
    
    // Fallback to database
    var user User
    db := database.DB()
    err = db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", userID).
        Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }
    
    // Cache and return
    userData, _ := json.Marshal(user)
    cache.Set(r.Context(), cacheKey, userData, time.Hour)
    
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Cache", "MISS")
    w.Write(userData)
}
```

## Environment Configuration Guide

### Development Environment

```bash
# Purpose: Configure Beaver Kit for local development
# Prerequisites: Local development environment
# Expected outcome: All services configured for development

# Database - Use SQLite for simplicity
BEAVER_DB_DRIVER=sqlite
BEAVER_DB_DATABASE=./dev.db

# Cache - Use memory for development
BEAVER_CACHE_DRIVER=memory

# Debug logging
BEAVER_CONFIG_DEBUG=true
BEAVER_DB_DEBUG=true

# Optional services (set if needed)
BEAVER_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/DEV/WEBHOOK
BEAVER_CAPTCHA_ENABLED=false  # Disable CAPTCHA in development
```

### Production Environment

```bash
# Purpose: Configure Beaver Kit for production deployment
# Prerequisites: Production infrastructure provisioned
# Expected outcome: All services configured for production scale

# Database - Use PostgreSQL for production
BEAVER_DB_DRIVER=postgres
BEAVER_DB_HOST=prod-db.example.com
BEAVER_DB_PORT=5432
BEAVER_DB_DATABASE=myapp_prod
BEAVER_DB_USERNAME=myapp_user
BEAVER_DB_PASSWORD=secure_password
BEAVER_DB_SSL_MODE=require
BEAVER_DB_MAX_OPEN_CONNS=50
BEAVER_DB_MAX_IDLE_CONNS=10

# Cache - Use Redis for production
BEAVER_CACHE_DRIVER=redis
BEAVER_CACHE_HOST=prod-redis.example.com
BEAVER_CACHE_PORT=6379
BEAVER_CACHE_PASSWORD=redis_password
BEAVER_CACHE_POOL_SIZE=20

# Production services
BEAVER_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/PROD/WEBHOOK
BEAVER_CAPTCHA_ENABLED=true
BEAVER_CAPTCHA_PROVIDER=recaptcha
BEAVER_CAPTCHA_SITE_KEY=prod_site_key
BEAVER_CAPTCHA_SECRET_KEY=prod_secret_key

# Security
BEAVER_KRYPTO_JWT_KEY=production_jwt_secret_key_32_chars
BEAVER_URLSIGNER_SECRET_KEY=production_url_signer_secret

# Disable debug logging in production
BEAVER_CONFIG_DEBUG=false
BEAVER_DB_DEBUG=false
```

## Common Patterns and Best Practices

### Graceful Shutdown

```go
// Purpose: Implement graceful shutdown for production applications
// Prerequisites: Services initialized
// Expected outcome: Clean shutdown with resource cleanup

package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/slack"
)

func main() {
    // Initialize services
    database.Init()
    slack.Init()
    
    // Start application
    server := startServer()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    // Graceful shutdown
    gracefulShutdown(server)
}

func gracefulShutdown(server *http.Server) {
    // Purpose: Shutdown services gracefully
    // Prerequisites: Services initialized and running
    // Expected outcome: All connections closed cleanly
    
    slack.Slack().SendWarning("Application shutting down...")
    
    // Create shutdown context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Shutdown HTTP server
    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    
    // Shutdown database connections
    database.Shutdown(ctx)
    
    slack.Slack().SendInfo("Application shutdown complete")
}
```

### Error Handling

```go
// Purpose: Implement comprehensive error handling
// Prerequisites: Understanding of Beaver Kit error types
// Expected outcome: Robust error handling throughout application

func handleDatabaseError(err error) {
    if errors.Is(err, database.ErrNotInitialized) {
        log.Fatal("Database not initialized")
    } else if errors.Is(err, sql.ErrNoRows) {
        // Handle not found
        return
    } else {
        // Log and handle other database errors
        log.Printf("Database error: %v", err)
        slack.Slack().SendAlert(fmt.Sprintf("Database error: %v", err))
    }
}

func handleCacheError(err error) {
    if errors.Is(err, cache.ErrKeyNotFound) {
        // Cache miss is expected behavior
        return
    } else {
        // Log cache errors but don't fail the request
        log.Printf("Cache error: %v", err)
    }
}
```

## Next Steps

1. **Explore Individual Packages** - Dive deep into each package's documentation
2. **Review Integration Patterns** - Learn common integration scenarios
3. **Study Example Applications** - Examine complete example projects
4. **Performance Optimization** - Learn about tuning for production workloads
5. **Security Best Practices** - Understand security considerations for each package

Continue with:
- [Core Concepts](learning-paths/core-concepts.md) - Deep dive into framework principles
- [Package Documentation](packages/) - Detailed API references
- [Integration Patterns](integration-patterns.md) - Real-world usage scenarios