---
title: "Integration Patterns and Best Practices"
tags: ["integration", "patterns", "best-practices", "architecture"]
prerequisites:
  - "getting-started"
  - "framework-overview"
relatedDocs:
  - "packages/database"
  - "packages/cache"
  - "learning-paths/common-patterns"
---

# Integration Patterns and Best Practices

## Overview

This guide demonstrates common integration patterns for combining multiple Beaver Kit packages to build robust, production-ready applications. Each pattern includes complete code examples with error handling, testing strategies, and performance considerations.

```json
{
  "@context": "http://schema.org",
  "@type": "TechArticle",
  "name": "Beaver Kit Integration Patterns",
  "about": "Common patterns for integrating multiple Beaver Kit packages",
  "programmingLanguage": "Go",
  "codeRepository": "https://github.com/gobeaver/beaver-kit",
  "keywords": ["integration", "patterns", "architecture", "microservices", "web-applications"]
}
```

## Web Application Stack

### Complete Web Application with All Packages

```go
// Purpose: Build a complete web application using all Beaver Kit packages
// Prerequisites: All packages properly configured via environment variables
// Expected outcome: Production-ready web application with authentication, caching, and monitoring

package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"
    
    "github.com/gorilla/mux"
    
    "github.com/gobeaver/beaver-kit/database"
    "github.com/gobeaver/beaver-kit/cache"
    "github.com/gobeaver/beaver-kit/captcha"
    "github.com/gobeaver/beaver-kit/slack"
    "github.com/gobeaver/beaver-kit/krypto"
    "github.com/gobeaver/beaver-kit/urlsigner"
    "github.com/gobeaver/beaver-kit/filekit"
    "github.com/gobeaver/beaver-kit/filevalidator"
)

type Application struct {
    router     *mux.Router
    fileUpload filekit.FileSystem
    validator  *filevalidator.Validator
}

func main() {
    app, err := initializeApplication()
    if err != nil {
        log.Fatal("Application initialization failed:", err)
    }
    
    // Start server
    server := &http.Server{
        Addr:         ":8080",
        Handler:      app.router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }
    
    // Graceful shutdown
    go func() {
        log.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server start failed:", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }
    
    // Cleanup resources
    database.Shutdown(ctx)
    cache.Close()
    
    log.Println("Server exited")
}

func initializeApplication() (*Application, error) {
    // Initialize all services
    if err := initializeServices(); err != nil {
        return nil, fmt.Errorf("service initialization failed: %w", err)
    }
    
    // Setup file handling
    fileUpload, err := setupFileHandling()
    if err != nil {
        return nil, fmt.Errorf("file handling setup failed: %w", err)
    }
    
    validator := filevalidator.New(filevalidator.ImageOnlyConstraints())
    
    app := &Application{
        router:     mux.NewRouter(),
        fileUpload: fileUpload,
        validator:  validator,
    }
    
    // Setup routes
    app.setupRoutes()
    
    // Send startup notification
    slack.Slack().SendInfo("Application started successfully")
    
    return app, nil
}

func initializeServices() error {
    // Purpose: Initialize all Beaver Kit services from environment
    // Prerequisites: Environment variables properly configured
    // Expected outcome: All services ready for use
    
    services := []struct {
        name string
        init func() error
    }{
        {"database", database.Init},
        {"cache", cache.Init},
        {"captcha", captcha.Init},
        {"slack", slack.Init},
        {"urlsigner", urlsigner.Init},
    }
    
    for _, service := range services {
        if err := service.init(); err != nil {
            return fmt.Errorf("%s initialization failed: %w", service.name, err)
        }
        log.Printf("%s service initialized", service.name)
    }
    
    return nil
}

func setupFileHandling() (filekit.FileSystem, error) {
    // Purpose: Setup file handling based on environment configuration
    // Prerequisites: File storage configuration available
    // Expected outcome: File system ready for uploads
    
    if err := filekit.InitFromEnv(); err != nil {
        return nil, err
    }
    
    return filekit.FS(), nil
}

func (app *Application) setupRoutes() {
    // Purpose: Setup all application routes with middleware
    // Prerequisites: Application initialized
    // Expected outcome: Complete routing configuration
    
    // Middleware
    app.router.Use(app.loggingMiddleware)
    app.router.Use(app.corsMiddleware)
    
    // Public routes
    app.router.HandleFunc("/health", app.healthHandler).Methods("GET")
    app.router.HandleFunc("/register", app.registerHandler).Methods("POST")
    app.router.HandleFunc("/login", app.loginHandler).Methods("POST")
    
    // Protected routes
    protected := app.router.PathPrefix("/api").Subrouter()
    protected.Use(app.authMiddleware)
    
    protected.HandleFunc("/profile", app.profileHandler).Methods("GET")
    protected.HandleFunc("/upload", app.uploadHandler).Methods("POST")
    protected.HandleFunc("/download/{id}", app.downloadHandler).Methods("GET")
    
    // Admin routes
    admin := protected.PathPrefix("/admin").Subrouter()
    admin.Use(app.adminMiddleware)
    admin.HandleFunc("/users", app.listUsersHandler).Methods("GET")
}
```

### Authentication and Session Management

```go
// Purpose: Implement comprehensive authentication with JWT and session caching
// Prerequisites: Database, cache, krypto, and slack packages initialized
// Expected outcome: Secure authentication system with session management

type User struct {
    ID        int64     `json:"id" db:"id"`
    Username  string    `json:"username" db:"username"`
    Email     string    `json:"email" db:"email"`
    Password  string    `json:"-" db:"password_hash"`
    FirstName string    `json:"first_name" db:"first_name"`
    LastName  string    `json:"last_name" db:"last_name"`
    Role      string    `json:"role" db:"role"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type AuthResponse struct {
    Token string `json:"token"`
    User  User   `json:"user"`
}

func (app *Application) registerHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle user registration with CAPTCHA validation
    // Prerequisites: CAPTCHA service configured
    // Expected outcome: New user created or appropriate error response
    
    var req struct {
        Username     string `json:"username"`
        Email        string `json:"email"`
        Password     string `json:"password"`
        FirstName    string `json:"first_name"`
        LastName     string `json:"last_name"`
        CaptchaToken string `json:"captcha_token"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validate CAPTCHA
    valid, err := captcha.Service().Validate(r.Context(), req.CaptchaToken, r.RemoteAddr)
    if err != nil || !valid {
        http.Error(w, "Invalid CAPTCHA", http.StatusBadRequest)
        slack.Slack().SendWarning(fmt.Sprintf("Invalid CAPTCHA attempt: %s", r.RemoteAddr))
        return
    }
    
    // Validate input
    if err := validateRegistrationInput(req.Username, req.Email, req.Password); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Hash password
    passwordHash, err := krypto.Argon2idHashPassword(req.Password)
    if err != nil {
        log.Printf("Password hashing failed: %v", err)
        http.Error(w, "Registration failed", http.StatusInternalServerError)
        return
    }
    
    // Create user
    user := User{
        Username:  req.Username,
        Email:     req.Email,
        Password:  passwordHash,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Role:      "user",
        CreatedAt: time.Now(),
    }
    
    // Insert into database
    db := database.DB()
    result, err := db.ExecContext(r.Context(), `
        INSERT INTO users (username, email, password_hash, first_name, last_name, role, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)`,
        user.Username, user.Email, user.Password, user.FirstName, user.LastName, user.Role, user.CreatedAt)
    
    if err != nil {
        log.Printf("User creation failed: %v", err)
        http.Error(w, "Username or email already exists", http.StatusConflict)
        return
    }
    
    userID, _ := result.LastInsertId()
    user.ID = userID
    
    // Generate JWT token
    token, err := app.generateUserToken(user)
    if err != nil {
        log.Printf("Token generation failed: %v", err)
        http.Error(w, "Registration failed", http.StatusInternalServerError)
        return
    }
    
    // Cache user session
    if err := app.cacheUserSession(r.Context(), token, user); err != nil {
        log.Printf("Session caching failed: %v", err)
        // Don't fail registration for cache error
    }
    
    // Send notification
    slack.Slack().SendInfo(fmt.Sprintf("New user registered: %s (%s)", user.Username, user.Email))
    
    // Return response
    response := AuthResponse{
        Token: token,
        User:  user,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (app *Application) loginHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle user login with rate limiting and session management
    // Prerequisites: User exists in database
    // Expected outcome: JWT token and user data or authentication error
    
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Rate limiting using cache
    clientIP := r.RemoteAddr
    if blocked, err := app.checkRateLimit(r.Context(), clientIP); err != nil {
        log.Printf("Rate limit check failed: %v", err)
    } else if blocked {
        http.Error(w, "Too many login attempts", http.StatusTooManyRequests)
        return
    }
    
    // Add timing attack protection
    defer krypto.RandomDelay(100, 300)
    
    // Get user from database
    user, err := app.getUserByUsername(r.Context(), req.Username)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        app.recordFailedLogin(r.Context(), clientIP)
        return
    }
    
    // Verify password
    valid, err := krypto.Argon2idVerifyPassword(req.Password, user.Password)
    if err != nil || !valid {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        app.recordFailedLogin(r.Context(), clientIP)
        return
    }
    
    // Generate JWT token
    token, err := app.generateUserToken(*user)
    if err != nil {
        log.Printf("Token generation failed: %v", err)
        http.Error(w, "Login failed", http.StatusInternalServerError)
        return
    }
    
    // Cache user session
    if err := app.cacheUserSession(r.Context(), token, *user); err != nil {
        log.Printf("Session caching failed: %v", err)
        // Don't fail login for cache error
    }
    
    // Clear rate limit on successful login
    app.clearRateLimit(r.Context(), clientIP)
    
    // Return response
    response := AuthResponse{
        Token: token,
        User:  *user,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (app *Application) generateUserToken(user User) (string, error) {
    // Purpose: Generate JWT token with user claims
    // Prerequisites: User data available
    // Expected outcome: Valid JWT token for authentication
    
    claims := krypto.UserClaims{
        First: user.FirstName,
        Last:  user.LastName,
        Token: fmt.Sprintf("%d", user.ID),
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
            Issuer:    "beaver-app",
            Subject:   fmt.Sprintf("user:%d", user.ID),
        },
    }
    
    return krypto.NewHs256AccessToken(claims)
}

func (app *Application) cacheUserSession(ctx context.Context, token string, user User) error {
    // Purpose: Cache user session data for fast access
    // Prerequisites: Cache service initialized
    // Expected outcome: User session cached with TTL
    
    sessionData := map[string]interface{}{
        "user_id":    user.ID,
        "username":   user.Username,
        "email":      user.Email,
        "first_name": user.FirstName,
        "last_name":  user.LastName,
        "role":       user.Role,
        "cached_at":  time.Now().Unix(),
    }
    
    data, err := json.Marshal(sessionData)
    if err != nil {
        return err
    }
    
    cacheKey := fmt.Sprintf("session:%s", token)
    return cache.Set(ctx, cacheKey, data, 24*time.Hour)
}

func (app *Application) checkRateLimit(ctx context.Context, clientIP string) (bool, error) {
    // Purpose: Check if client has exceeded login rate limit
    // Prerequisites: Cache service initialized
    // Expected outcome: Rate limit status
    
    key := fmt.Sprintf("rate_limit:login:%s", clientIP)
    
    data, err := cache.Get(ctx, key)
    if err != nil {
        if errors.Is(err, cache.ErrKeyNotFound) {
            return false, nil // No rate limit record
        }
        return false, err
    }
    
    var attempts int
    if err := json.Unmarshal(data, &attempts); err != nil {
        return false, err
    }
    
    return attempts >= 5, nil // Block after 5 attempts
}

func (app *Application) recordFailedLogin(ctx context.Context, clientIP string) {
    // Purpose: Record failed login attempt for rate limiting
    // Prerequisites: Cache service initialized
    // Expected outcome: Failed attempt counter incremented
    
    key := fmt.Sprintf("rate_limit:login:%s", clientIP)
    
    var attempts int
    if data, err := cache.Get(ctx, key); err == nil {
        json.Unmarshal(data, &attempts)
    }
    
    attempts++
    data, _ := json.Marshal(attempts)
    cache.Set(ctx, key, data, 15*time.Minute) // 15 minute lockout
}
```

### File Upload and Management

```go
// Purpose: Secure file upload with validation, processing, and signed download URLs
// Prerequisites: FileKit, FileValidator, URLSigner packages initialized
// Expected outcome: Complete file management system

func (app *Application) uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle file upload with validation and secure storage
    // Prerequisites: User authenticated, file upload configured
    // Expected outcome: File uploaded and download URL generated
    
    // Parse multipart form (10MB max)
    if err := r.ParseMultipartForm(10 << 20); err != nil {
        http.Error(w, "Could not parse form", http.StatusBadRequest)
        return
    }
    
    // Get uploaded file
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error retrieving file", http.StatusBadRequest)
        return
    }
    defer file.Close()
    
    // Validate file
    if err := app.validator.Validate(header); err != nil {
        http.Error(w, fmt.Sprintf("File validation failed: %v", err), http.StatusBadRequest)
        return
    }
    
    // Get user from context (set by auth middleware)
    user := r.Context().Value("user").(User)
    
    // Generate unique filename
    fileID := generateFileID()
    ext := filepath.Ext(header.Filename)
    filename := fmt.Sprintf("%s%s", fileID, ext)
    filePath := fmt.Sprintf("uploads/%d/%s", user.ID, filename)
    
    // Upload file
    err = app.fileUpload.Upload(r.Context(), filePath, file,
        filekit.WithContentType(header.Header.Get("Content-Type")),
        filekit.WithMetadata(map[string]string{
            "original_filename": header.Filename,
            "uploaded_by":       fmt.Sprintf("%d", user.ID),
            "upload_time":       time.Now().UTC().Format(time.RFC3339),
        }),
    )
    if err != nil {
        log.Printf("File upload failed: %v", err)
        http.Error(w, "Upload failed", http.StatusInternalServerError)
        return
    }
    
    // Store file metadata in database
    fileRecord := FileRecord{
        ID:           fileID,
        UserID:       user.ID,
        OriginalName: header.Filename,
        FilePath:     filePath,
        ContentType:  header.Header.Get("Content-Type"),
        Size:         header.Size,
        UploadedAt:   time.Now(),
    }
    
    if err := app.saveFileRecord(r.Context(), fileRecord); err != nil {
        log.Printf("File record save failed: %v", err)
        // Try to clean up uploaded file
        app.fileUpload.Delete(r.Context(), filePath)
        http.Error(w, "Upload failed", http.StatusInternalServerError)
        return
    }
    
    // Generate signed download URL (valid for 7 days)
    downloadURL := fmt.Sprintf("https://%s/api/download/%s", r.Host, fileID)
    signedURL, err := urlsigner.Service().SignURL(downloadURL, 7*24*time.Hour, "")
    if err != nil {
        log.Printf("URL signing failed: %v", err)
        http.Error(w, "Upload succeeded but download URL generation failed", http.StatusInternalServerError)
        return
    }
    
    // Send notification for large files
    if header.Size > 50*1024*1024 { // 50MB
        slack.Slack().SendInfo(fmt.Sprintf("Large file uploaded: %s (%s) by %s", 
            header.Filename, formatFileSize(header.Size), user.Username))
    }
    
    // Return response
    response := map[string]interface{}{
        "file_id":      fileID,
        "filename":     header.Filename,
        "size":         header.Size,
        "content_type": header.Header.Get("Content-Type"),
        "download_url": signedURL,
        "uploaded_at":  fileRecord.UploadedAt,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (app *Application) downloadHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Handle secure file downloads with URL signature verification
    // Prerequisites: File exists and URL is properly signed
    // Expected outcome: File download or appropriate error
    
    vars := mux.Vars(r)
    fileID := vars["id"]
    
    // Verify signed URL
    valid, _, err := urlsigner.Service().VerifyURL(r.URL.String())
    if err != nil || !valid {
        http.Error(w, "Invalid or expired download link", http.StatusForbidden)
        return
    }
    
    // Get file record from database
    fileRecord, err := app.getFileRecord(r.Context(), fileID)
    if err != nil {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }
    
    // Check if user has access (owner or admin)
    user := r.Context().Value("user").(User)
    if fileRecord.UserID != user.ID && user.Role != "admin" {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }
    
    // Get file from storage
    reader, err := app.fileUpload.Download(r.Context(), fileRecord.FilePath)
    if err != nil {
        log.Printf("File download failed: %v", err)
        http.Error(w, "Download failed", http.StatusInternalServerError)
        return
    }
    defer reader.Close()
    
    // Set headers
    w.Header().Set("Content-Type", fileRecord.ContentType)
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileRecord.OriginalName))
    w.Header().Set("Content-Length", fmt.Sprintf("%d", fileRecord.Size))
    
    // Stream file to client
    if _, err := io.Copy(w, reader); err != nil {
        log.Printf("File streaming failed: %v", err)
    }
    
    // Update download statistics
    app.recordFileDownload(r.Context(), fileID, user.ID)
}

type FileRecord struct {
    ID           string    `json:"id" db:"id"`
    UserID       int64     `json:"user_id" db:"user_id"`
    OriginalName string    `json:"original_name" db:"original_name"`
    FilePath     string    `json:"file_path" db:"file_path"`
    ContentType  string    `json:"content_type" db:"content_type"`
    Size         int64     `json:"size" db:"size"`
    UploadedAt   time.Time `json:"uploaded_at" db:"uploaded_at"`
}

func generateFileID() string {
    token, _ := krypto.GenerateSecureToken(16)
    return token
}

func formatFileSize(size int64) string {
    const unit = 1024
    if size < unit {
        return fmt.Sprintf("%d B", size)
    }
    div, exp := int64(unit), 0
    for n := size / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
```

### Middleware Chain

```go
// Purpose: Implement comprehensive middleware chain for security and monitoring
// Prerequisites: All packages initialized
// Expected outcome: Secure, monitored request handling

func (app *Application) loggingMiddleware(next http.Handler) http.Handler {
    // Purpose: Log all HTTP requests with timing and status
    // Prerequisites: Logging system configured
    // Expected outcome: Comprehensive request logging
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap response writer to capture status code
        lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(lrw, r)
        
        duration := time.Since(start)
        
        log.Printf("%s %s %d %s %s",
            r.Method,
            r.RequestURI,
            lrw.statusCode,
            duration,
            r.RemoteAddr,
        )
        
        // Alert on errors
        if lrw.statusCode >= 500 {
            slack.Slack().SendAlert(fmt.Sprintf("Server error: %s %s returned %d", 
                r.Method, r.RequestURI, lrw.statusCode))
        }
    })
}

type loggingResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
    lrw.statusCode = code
    lrw.ResponseWriter.WriteHeader(code)
}

func (app *Application) authMiddleware(next http.Handler) http.Handler {
    // Purpose: Authenticate requests using JWT tokens with session caching
    // Prerequisites: Krypto and cache packages initialized
    // Expected outcome: User context added to request or 401 error
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if !strings.HasPrefix(authHeader, "Bearer ") {
            http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
            return
        }
        
        token := authHeader[7:] // Remove "Bearer " prefix
        
        // Try to get user from cache first
        cacheKey := fmt.Sprintf("session:%s", token)
        if data, err := cache.Get(r.Context(), cacheKey); err == nil {
            var sessionData map[string]interface{}
            if err := json.Unmarshal(data, &sessionData); err == nil {
                user := User{
                    ID:        int64(sessionData["user_id"].(float64)),
                    Username:  sessionData["username"].(string),
                    Email:     sessionData["email"].(string),
                    FirstName: sessionData["first_name"].(string),
                    LastName:  sessionData["last_name"].(string),
                    Role:      sessionData["role"].(string),
                }
                
                // Add user to request context
                ctx := context.WithValue(r.Context(), "user", user)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
        }
        
        // Cache miss - validate JWT token
        claims, err := krypto.ParseHs256AccessToken(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // Check token expiration
        if time.Now().Unix() > claims.ExpiresAt {
            http.Error(w, "Token expired", http.StatusUnauthorized)
            return
        }
        
        // Get user from database
        userID, _ := strconv.ParseInt(claims.Token, 10, 64)
        user, err := app.getUserByID(r.Context(), userID)
        if err != nil {
            http.Error(w, "User not found", http.StatusUnauthorized)
            return
        }
        
        // Cache user session
        app.cacheUserSession(r.Context(), token, *user)
        
        // Add user to request context
        ctx := context.WithValue(r.Context(), "user", *user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (app *Application) adminMiddleware(next http.Handler) http.Handler {
    // Purpose: Ensure user has admin role
    // Prerequisites: User authenticated via authMiddleware
    // Expected outcome: Admin access granted or 403 error
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(User)
        
        if user.Role != "admin" {
            http.Error(w, "Admin access required", http.StatusForbidden)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func (app *Application) corsMiddleware(next http.Handler) http.Handler {
    // Purpose: Handle CORS for web applications
    // Prerequisites: Understanding of CORS requirements
    // Expected outcome: Proper CORS headers set
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // Allow specific origins in production
        allowedOrigins := []string{
            "http://localhost:3000",
            "https://myapp.com",
            "https://www.myapp.com",
        }
        
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }
        
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Health Monitoring and Observability

```go
// Purpose: Comprehensive health monitoring across all services
// Prerequisites: All packages initialized
// Expected outcome: Complete system health visibility

func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
    // Purpose: Provide comprehensive health check for all services
    // Prerequisites: All services initialized
    // Expected outcome: Detailed health status report
    
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().UTC(),
        "services":  make(map[string]interface{}),
    }
    
    overallHealthy := true
    
    // Check database health
    dbHealth := checkDatabaseHealth(ctx)
    health["services"].(map[string]interface{})["database"] = dbHealth
    if !dbHealth["healthy"].(bool) {
        overallHealthy = false
    }
    
    // Check cache health
    cacheHealth := checkCacheHealth(ctx)
    health["services"].(map[string]interface{})["cache"] = cacheHealth
    if !cacheHealth["healthy"].(bool) {
        overallHealthy = false
    }
    
    // Check file storage health
    fileHealth := checkFileStorageHealth(ctx)
    health["services"].(map[string]interface{})["file_storage"] = fileHealth
    if !fileHealth["healthy"].(bool) {
        overallHealthy = false
    }
    
    // Check external services
    externalHealth := checkExternalServices(ctx)
    health["services"].(map[string]interface{})["external"] = externalHealth
    
    if !overallHealthy {
        health["status"] = "unhealthy"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func checkDatabaseHealth(ctx context.Context) map[string]interface{} {
    health := map[string]interface{}{
        "healthy": false,
        "latency": 0,
        "stats":   nil,
        "error":   nil,
    }
    
    start := time.Now()
    err := database.Health(ctx)
    latency := time.Since(start)
    
    health["latency"] = latency.Milliseconds()
    
    if err != nil {
        health["error"] = err.Error()
        return health
    }
    
    health["healthy"] = true
    health["stats"] = database.Stats()
    
    return health
}

func checkCacheHealth(ctx context.Context) map[string]interface{} {
    health := map[string]interface{}{
        "healthy": false,
        "latency": 0,
        "stats":   nil,
        "error":   nil,
    }
    
    start := time.Now()
    err := cache.Health(ctx)
    latency := time.Since(start)
    
    health["latency"] = latency.Milliseconds()
    
    if err != nil {
        health["error"] = err.Error()
        return health
    }
    
    health["healthy"] = true
    health["stats"] = cache.Stats()
    
    return health
}

func checkFileStorageHealth(ctx context.Context) map[string]interface{} {
    health := map[string]interface{}{
        "healthy": false,
        "latency": 0,
        "error":   nil,
    }
    
    start := time.Now()
    
    // Test file operations
    testKey := "health-check-" + time.Now().Format("20060102150405")
    testData := strings.NewReader("health check")
    
    fs := filekit.FS()
    err := fs.Upload(ctx, testKey, testData)
    if err != nil {
        health["error"] = err.Error()
        health["latency"] = time.Since(start).Milliseconds()
        return health
    }
    
    // Clean up test file
    fs.Delete(ctx, testKey)
    
    health["healthy"] = true
    health["latency"] = time.Since(start).Milliseconds()
    
    return health
}

func checkExternalServices(ctx context.Context) map[string]interface{} {
    services := map[string]interface{}{
        "captcha": checkCaptchaHealth(ctx),
        "slack":   checkSlackHealth(ctx),
    }
    
    return services
}

func checkCaptchaHealth(ctx context.Context) map[string]interface{} {
    // For CAPTCHA, we just check if service is configured
    service := captcha.Service()
    
    return map[string]interface{}{
        "healthy":     service != nil,
        "configured": service != nil,
    }
}

func checkSlackHealth(ctx context.Context) map[string]interface{} {
    // For Slack, we could send a test message or just check configuration
    service := slack.Slack()
    
    return map[string]interface{}{
        "healthy":     service != nil,
        "configured": service != nil,
    }
}
```

### Background Job Processing

```go
// Purpose: Implement background job processing with database and cache integration
// Prerequisites: Database and cache services initialized
// Expected outcome: Reliable background job processing system

type JobProcessor struct {
    workers int
    jobCh   chan Job
    quit    chan bool
}

type Job struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    Attempts  int                    `json:"attempts"`
    MaxRetries int                   `json:"max_retries"`
    CreatedAt time.Time              `json:"created_at"`
    RunAt     time.Time              `json:"run_at"`
}

func NewJobProcessor(workers int) *JobProcessor {
    return &JobProcessor{
        workers: workers,
        jobCh:   make(chan Job, 100),
        quit:    make(chan bool),
    }
}

func (jp *JobProcessor) Start() {
    // Purpose: Start background job processing workers
    // Prerequisites: Job processor configured
    // Expected outcome: Workers processing jobs from queue
    
    for i := 0; i < jp.workers; i++ {
        go jp.worker(i)
    }
    
    // Job scheduler
    go jp.scheduler()
    
    log.Printf("Started %d job processing workers", jp.workers)
}

func (jp *JobProcessor) Stop() {
    close(jp.quit)
}

func (jp *JobProcessor) EnqueueJob(jobType string, payload map[string]interface{}, runAt time.Time) error {
    // Purpose: Add job to processing queue
    // Prerequisites: Job processor running
    // Expected outcome: Job queued for processing
    
    job := Job{
        ID:         generateJobID(),
        Type:       jobType,
        Payload:    payload,
        Attempts:   0,
        MaxRetries: 3,
        CreatedAt:  time.Now(),
        RunAt:      runAt,
    }
    
    // Store job in database
    return jp.storeJob(job)
}

func (jp *JobProcessor) worker(id int) {
    // Purpose: Process jobs from queue
    // Prerequisites: Job queue initialized
    // Expected outcome: Jobs processed according to their type
    
    log.Printf("Worker %d started", id)
    
    for {
        select {
        case job := <-jp.jobCh:
            if err := jp.processJob(job); err != nil {
                log.Printf("Worker %d: Job %s failed: %v", id, job.ID, err)
                jp.handleJobFailure(job, err)
            } else {
                log.Printf("Worker %d: Job %s completed", id, job.ID)
                jp.markJobCompleted(job.ID)
            }
        case <-jp.quit:
            log.Printf("Worker %d stopped", id)
            return
        }
    }
}

func (jp *JobProcessor) scheduler() {
    // Purpose: Schedule jobs from database to workers
    // Prerequisites: Database service available
    // Expected outcome: Due jobs sent to workers
    
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            jobs, err := jp.getDueJobs()
            if err != nil {
                log.Printf("Failed to get due jobs: %v", err)
                continue
            }
            
            for _, job := range jobs {
                select {
                case jp.jobCh <- job:
                    jp.markJobProcessing(job.ID)
                default:
                    log.Printf("Job queue full, skipping job %s", job.ID)
                }
            }
        case <-jp.quit:
            return
        }
    }
}

func (jp *JobProcessor) processJob(job Job) error {
    // Purpose: Process individual job based on type
    // Prerequisites: Job handler registered for job type
    // Expected outcome: Job processing completed or error returned
    
    ctx := context.Background()
    
    switch job.Type {
    case "send_email":
        return jp.processSendEmailJob(ctx, job)
    case "cleanup_files":
        return jp.processCleanupFilesJob(ctx, job)
    case "generate_report":
        return jp.processGenerateReportJob(ctx, job)
    case "backup_data":
        return jp.processBackupDataJob(ctx, job)
    default:
        return fmt.Errorf("unknown job type: %s", job.Type)
    }
}

func (jp *JobProcessor) processSendEmailJob(ctx context.Context, job Job) error {
    // Purpose: Process email sending job
    // Prerequisites: Email service configured
    // Expected outcome: Email sent to recipient
    
    to := job.Payload["to"].(string)
    subject := job.Payload["subject"].(string)
    body := job.Payload["body"].(string)
    
    // Send email (implementation depends on email service)
    if err := sendEmail(to, subject, body); err != nil {
        return fmt.Errorf("email sending failed: %w", err)
    }
    
    // Cache sent email info
    emailKey := fmt.Sprintf("sent_email:%s", job.ID)
    emailData := map[string]interface{}{
        "to":      to,
        "subject": subject,
        "sent_at": time.Now(),
    }
    data, _ := json.Marshal(emailData)
    cache.Set(ctx, emailKey, data, 24*time.Hour)
    
    return nil
}

func (jp *JobProcessor) processCleanupFilesJob(ctx context.Context, job Job) error {
    // Purpose: Clean up old files from storage
    // Prerequisites: File storage service available
    // Expected outcome: Old files removed from storage
    
    olderThan := job.Payload["older_than_days"].(float64)
    cutoff := time.Now().AddDate(0, 0, -int(olderThan))
    
    // Get old files from database
    files, err := jp.getOldFiles(ctx, cutoff)
    if err != nil {
        return fmt.Errorf("failed to get old files: %w", err)
    }
    
    fs := filekit.FS()
    deletedCount := 0
    
    for _, file := range files {
        if err := fs.Delete(ctx, file.FilePath); err != nil {
            log.Printf("Failed to delete file %s: %v", file.FilePath, err)
            continue
        }
        
        // Remove from database
        if err := jp.deleteFileRecord(ctx, file.ID); err != nil {
            log.Printf("Failed to delete file record %s: %v", file.ID, err)
        }
        
        deletedCount++
    }
    
    // Send notification about cleanup
    slack.Slack().SendInfo(fmt.Sprintf("File cleanup completed: %d files deleted", deletedCount))
    
    return nil
}

// Helper functions for job processing
func generateJobID() string {
    token, _ := krypto.GenerateSecureToken(16)
    return token
}

func sendEmail(to, subject, body string) error {
    // Implementation depends on chosen email service
    // Could use SendGrid, AWS SES, etc.
    log.Printf("Sending email to %s: %s", to, subject)
    return nil
}
```

This comprehensive integration guide shows how to combine all Beaver Kit packages into real-world applications with proper error handling, security, monitoring, and scalability considerations.