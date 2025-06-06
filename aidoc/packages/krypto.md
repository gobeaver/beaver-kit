---
title: "Krypto Package API Reference"
tags: ["cryptography", "jwt", "encryption", "hashing", "security"]
prerequisites:
  - "getting-started"
  - "config"
relatedDocs:
  - "urlsigner"
  - "security-best-practices"
---

# Krypto Package

## Overview

The krypto package provides comprehensive cryptographic utilities for Go applications with a stable API designed to be reliable and future-proof. It includes password hashing, JWT token management, encryption, and various security utilities.

```json
{
  "@context": "http://schema.org",
  "@type": "TechArticle",
  "name": "Krypto Package API Reference",
  "about": "Comprehensive cryptographic utilities for secure Go applications",
  "programmingLanguage": "Go",
  "codeRepository": "https://github.com/gobeaver/beaver-kit",
  "keywords": ["cryptography", "jwt", "encryption", "hashing", "security", "argon2", "bcrypt", "aes"]
}
```

## Key Features

- **Password Hashing** - Argon2id and Bcrypt implementations with secure defaults
- **JWT Token Management** - HS256 token generation and validation with custom claims
- **Encryption** - AES-GCM encryption/decryption with authenticated encryption
- **RSA Operations** - Key pair generation and validation
- **Secure Token Generation** - Cryptographically secure random tokens
- **OTP Generation** - One-time password generation for 2FA
- **SHA-256 Hashing** - Fast hashing with verification
- **Security Utilities** - Random delay functions to prevent timing attacks

## Quick Start

### Basic Password Hashing

```go
// Purpose: Securely hash and verify passwords using Argon2id
// Prerequisites: User password input
// Expected outcome: Secure password hash and verification

package main

import (
    "fmt"
    "log"
    
    "github.com/gobeaver/beaver-kit/krypto"
)

func main() {
    password := "secure_password123"
    
    // Hash password
    hash, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        log.Fatal("Password hashing failed:", err)
    }
    
    fmt.Printf("Password hash: %s\n", hash)
    
    // Verify password
    valid, err := krypto.Argon2idVerifyPassword(password, hash)
    if err != nil {
        log.Fatal("Password verification failed:", err)
    }
    
    fmt.Printf("Password valid: %t\n", valid)
}
```

### JWT Token Operations

```go
// Purpose: Generate and validate JWT tokens with custom claims
// Prerequisites: User authentication data
// Expected outcome: Secure JWT token for session management

import (
    "time"
    "github.com/dgrijalva/jwt-go"
    "github.com/gobeaver/beaver-kit/krypto"
)

func jwtExample() error {
    // Create user claims
    claims := krypto.UserClaims{
        First: "John",
        Last:  "Doe",
        Token: "user-123",
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
            Issuer:    "myapp",
        },
    }
    
    // Generate access token
    token, err := krypto.NewHs256AccessToken(claims)
    if err != nil {
        return fmt.Errorf("token generation failed: %w", err)
    }
    
    fmt.Printf("JWT Token: %s\n", token)
    
    // Parse and validate token
    parsedClaims, err := krypto.ParseHs256AccessToken(token)
    if err != nil {
        return fmt.Errorf("token parsing failed: %w", err)
    }
    
    fmt.Printf("User: %s %s (ID: %s)\n", parsedClaims.First, parsedClaims.Last, parsedClaims.Token)
    
    return nil
}
```

## API Reference

### Password Hashing

#### Argon2id (Recommended)

```go
// Purpose: Hash passwords using Argon2id algorithm with secure parameters
// Prerequisites: Plain text password
// Expected outcome: Secure password hash resistant to attacks

// Hash password with secure defaults
func Argon2idHashPassword(password string) (string, error)

// Verify password against hash
func Argon2idVerifyPassword(password, hash string) (bool, error)

// Example usage
func argon2idExample() error {
    password := "user_password123"
    
    // Hash with automatic salt generation
    hash, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        return fmt.Errorf("hashing failed: %w", err)
    }
    
    // Store hash in database
    if err := storeUserPassword(userID, hash); err != nil {
        return err
    }
    
    // Later, during login
    storedHash, err := getUserPassword(userID)
    if err != nil {
        return err
    }
    
    valid, err := krypto.Argon2idVerifyPassword(password, storedHash)
    if err != nil {
        return fmt.Errorf("verification failed: %w", err)
    }
    
    if !valid {
        return errors.New("invalid password")
    }
    
    return nil
}
```

**Argon2id Parameters:**
- **Memory**: 64 MB
- **Time**: 1 iteration
- **Threads**: 4
- **Salt**: 16 bytes (automatically generated)
- **Key Length**: 32 bytes

#### Bcrypt (Legacy Support)

```go
// Purpose: Hash passwords using Bcrypt algorithm for compatibility
// Prerequisites: Plain text password
// Expected outcome: Bcrypt password hash

// Hash password with configurable cost
func BcryptHashPassword(password string, cost int) (string, error)

// Verify password against Bcrypt hash
func BcryptVerifyPassword(password, hash string) (bool, error)

// Example usage
func bcryptExample() error {
    password := "user_password123"
    cost := 12 // Higher cost = more secure but slower
    
    hash, err := krypto.BcryptHashPassword(password, cost)
    if err != nil {
        return err
    }
    
    valid, err := krypto.BcryptVerifyPassword(password, hash)
    if err != nil {
        return err
    }
    
    fmt.Printf("Password valid: %t\n", valid)
    return nil
}
```

### JWT Token Management

#### User Claims Structure

```go
// Purpose: Define custom JWT claims for user authentication
// Prerequisites: Understanding of JWT claims
// Expected outcome: Structured user claims for tokens

type UserClaims struct {
    First string `json:"first,omitempty"`
    Last  string `json:"last,omitempty"`
    Token string `json:"token,omitempty"` // User identifier
    jwt.StandardClaims
}

// Example claims creation
func createUserClaims(userID int64, firstName, lastName string) krypto.UserClaims {
    return krypto.UserClaims{
        First: firstName,
        Last:  lastName,
        Token: fmt.Sprintf("%d", userID),
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
            NotBefore: time.Now().Unix(),
            Issuer:    "myapp",
            Subject:   fmt.Sprintf("user:%d", userID),
        },
    }
}
```

#### Token Operations

```go
// Purpose: Generate and validate JWT tokens with HS256 algorithm
// Prerequisites: JWT secret key configured
// Expected outcome: Secure token operations

// Generate access token
func NewHs256AccessToken(claims UserClaims) (string, error)

// Parse and validate token
func ParseHs256AccessToken(tokenString string) (*UserClaims, error)

// Example authentication flow
func authenticationFlow() error {
    // After successful login
    userClaims := createUserClaims(123, "John", "Doe")
    
    token, err := krypto.NewHs256AccessToken(userClaims)
    if err != nil {
        return fmt.Errorf("token generation failed: %w", err)
    }
    
    // Send token to client
    fmt.Printf("Access token: %s\n", token)
    
    // Later, validate incoming token
    claims, err := krypto.ParseHs256AccessToken(token)
    if err != nil {
        return fmt.Errorf("token validation failed: %w", err)
    }
    
    // Check if token is expired
    if time.Now().Unix() > claims.ExpiresAt {
        return errors.New("token expired")
    }
    
    // Use claims for authorization
    fmt.Printf("Authenticated user: %s %s (ID: %s)\n", 
        claims.First, claims.Last, claims.Token)
    
    return nil
}
```

### AES Encryption

```go
// Purpose: Encrypt and decrypt data using AES-GCM authenticated encryption
// Prerequisites: 32-byte encryption key
// Expected outcome: Secure encryption with integrity protection

type AESGCMService struct {
    key []byte
}

// Create new AES-GCM service
func NewAESGCMService(key string) *AESGCMService

// Encrypt data
func (a *AESGCMService) Encrypt(plaintext []byte) ([]byte, []byte, error)

// Decrypt data
func (a *AESGCMService) Decrypt(ciphertext, nonce []byte) ([]byte, error)

// Example encryption workflow
func encryptionExample() error {
    // Use 32-byte key for AES-256
    key := "32-byte-encryption-key-here!!!!"
    aesService := krypto.NewAESGCMService(key)
    
    // Encrypt sensitive data
    sensitiveData := []byte("user credit card: 4111-1111-1111-1111")
    encrypted, nonce, err := aesService.Encrypt(sensitiveData)
    if err != nil {
        return fmt.Errorf("encryption failed: %w", err)
    }
    
    fmt.Printf("Encrypted data length: %d bytes\n", len(encrypted))
    fmt.Printf("Nonce length: %d bytes\n", len(nonce))
    
    // Store encrypted data and nonce separately
    if err := storeEncryptedData(encrypted, nonce); err != nil {
        return err
    }
    
    // Later, decrypt the data
    retrievedEncrypted, retrievedNonce, err := getEncryptedData()
    if err != nil {
        return err
    }
    
    decrypted, err := aesService.Decrypt(retrievedEncrypted, retrievedNonce)
    if err != nil {
        return fmt.Errorf("decryption failed: %w", err)
    }
    
    fmt.Printf("Decrypted data: %s\n", string(decrypted))
    
    return nil
}
```

### Secure Token Generation

```go
// Purpose: Generate cryptographically secure random tokens
// Prerequisites: Entropy available from OS
// Expected outcome: Secure random tokens for various purposes

// Generate secure random token
func GenerateSecureToken(length int) (string, error)

// Generate numeric OTP
func GenerateOTP(length int) string

// Example token generation
func tokenGenerationExample() error {
    // Generate API key
    apiKey, err := krypto.GenerateSecureToken(32)
    if err != nil {
        return fmt.Errorf("API key generation failed: %w", err)
    }
    fmt.Printf("API Key: %s\n", apiKey)
    
    // Generate session token
    sessionToken, err := krypto.GenerateSecureToken(64)
    if err != nil {
        return fmt.Errorf("session token generation failed: %w", err)
    }
    fmt.Printf("Session Token: %s\n", sessionToken)
    
    // Generate 6-digit OTP for 2FA
    otp := krypto.GenerateOTP(6)
    fmt.Printf("2FA Code: %s\n", otp)
    
    // Generate 8-digit OTP for verification
    verificationCode := krypto.GenerateOTP(8)
    fmt.Printf("Verification Code: %s\n", verificationCode)
    
    return nil
}
```

### RSA Operations

```go
// Purpose: Generate and use RSA key pairs for asymmetric cryptography
// Prerequisites: Understanding of RSA cryptography
// Expected outcome: RSA key pair generation and validation

// Generate RSA key pair
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error)

// Example RSA usage
func rsaExample() error {
    // Generate 2048-bit RSA key pair
    privateKey, publicKey, err := krypto.GenerateRSAKeyPair(2048)
    if err != nil {
        return fmt.Errorf("RSA key generation failed: %w", err)
    }
    
    // Convert keys to PEM format for storage
    privateKeyPEM := krypto.PrivateKeyToPEM(privateKey)
    publicKeyPEM := krypto.PublicKeyToPEM(publicKey)
    
    fmt.Printf("Private Key PEM length: %d\n", len(privateKeyPEM))
    fmt.Printf("Public Key PEM length: %d\n", len(publicKeyPEM))
    
    // Store keys securely
    if err := storeRSAKeys(privateKeyPEM, publicKeyPEM); err != nil {
        return err
    }
    
    return nil
}
```

### SHA-256 Hashing

```go
// Purpose: Fast hashing for non-password data
// Prerequisites: Data to hash
// Expected outcome: SHA-256 hash for integrity verification

// Hash data with SHA-256
func SHA256Hash(data []byte) string

// Verify data against hash
func SHA256Verify(data []byte, hash string) bool

// Example SHA-256 usage
func sha256Example() error {
    // Hash file content for integrity checking
    fileContent := []byte("important file content")
    hash := krypto.SHA256Hash(fileContent)
    
    fmt.Printf("SHA-256 Hash: %s\n", hash)
    
    // Store hash for later verification
    if err := storeFileHash(filename, hash); err != nil {
        return err
    }
    
    // Later, verify file integrity
    currentContent, err := readFileContent(filename)
    if err != nil {
        return err
    }
    
    storedHash, err := getFileHash(filename)
    if err != nil {
        return err
    }
    
    if !krypto.SHA256Verify(currentContent, storedHash) {
        return errors.New("file integrity check failed")
    }
    
    fmt.Println("File integrity verified")
    return nil
}
```

### Security Utilities

```go
// Purpose: Additional security utilities to prevent attacks
// Prerequisites: Security-conscious application design
// Expected outcome: Enhanced security against timing attacks

// Add random delay to prevent timing attacks
func RandomDelay(minMs, maxMs int)

// Example timing attack prevention
func secureLoginExample(username, password string) error {
    // Always add random delay regardless of outcome
    defer krypto.RandomDelay(100, 500) // 100-500ms delay
    
    // Get user from database
    user, err := getUserByUsername(username)
    if err != nil {
        // Don't reveal whether user exists
        return errors.New("invalid credentials")
    }
    
    // Verify password (always takes significant time due to Argon2id)
    valid, err := krypto.Argon2idVerifyPassword(password, user.PasswordHash)
    if err != nil || !valid {
        return errors.New("invalid credentials")
    }
    
    return nil
}
```

## Real-World Integration Examples

### User Registration and Authentication

```go
// Purpose: Complete user registration and authentication system
// Prerequisites: Database setup for user storage
// Expected outcome: Secure user management system

type User struct {
    ID           int64     `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"` // Never serialize password
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

func registerUser(username, email, password, firstName, lastName string) (*User, error) {
    // Validate password strength
    if len(password) < 8 {
        return nil, errors.New("password must be at least 8 characters")
    }
    
    // Hash password
    passwordHash, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        return nil, fmt.Errorf("password hashing failed: %w", err)
    }
    
    // Create user
    user := &User{
        Username:     username,
        Email:        email,
        PasswordHash: passwordHash,
        FirstName:    firstName,
        LastName:     lastName,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    // Save to database
    if err := saveUser(user); err != nil {
        return nil, fmt.Errorf("user creation failed: %w", err)
    }
    
    return user, nil
}

func authenticateUser(username, password string) (string, error) {
    // Add timing attack protection
    defer krypto.RandomDelay(100, 300)
    
    // Get user from database
    user, err := getUserByUsername(username)
    if err != nil {
        return "", errors.New("invalid credentials")
    }
    
    // Verify password
    valid, err := krypto.Argon2idVerifyPassword(password, user.PasswordHash)
    if err != nil || !valid {
        return "", errors.New("invalid credentials")
    }
    
    // Generate JWT token
    claims := krypto.UserClaims{
        First: user.FirstName,
        Last:  user.LastName,
        Token: fmt.Sprintf("%d", user.ID),
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
            Issuer:    "myapp",
            Subject:   fmt.Sprintf("user:%d", user.ID),
        },
    }
    
    token, err := krypto.NewHs256AccessToken(claims)
    if err != nil {
        return "", fmt.Errorf("token generation failed: %w", err)
    }
    
    return token, nil
}
```

### API Key Management

```go
// Purpose: Secure API key generation and validation system
// Prerequisites: Database for API key storage
// Expected outcome: Secure API key management

type APIKey struct {
    ID        int64     `json:"id"`
    UserID    int64     `json:"user_id"`
    Name      string    `json:"name"`
    KeyHash   string    `json:"-"` // Store hash, not actual key
    Prefix    string    `json:"prefix"` // First 8 chars for identification
    CreatedAt time.Time `json:"created_at"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    LastUsed  *time.Time `json:"last_used,omitempty"`
}

func generateAPIKey(userID int64, name string, expiresAt *time.Time) (*APIKey, string, error) {
    // Generate secure API key
    keyValue, err := krypto.GenerateSecureToken(32)
    if err != nil {
        return nil, "", fmt.Errorf("key generation failed: %w", err)
    }
    
    // Create key with prefix for identification
    prefix := keyValue[:8]
    fullKey := fmt.Sprintf("bk_%s_%s", prefix, keyValue[8:])
    
    // Hash the key for storage (never store plaintext)
    keyHash := krypto.SHA256Hash([]byte(fullKey))
    
    apiKey := &APIKey{
        UserID:    userID,
        Name:      name,
        KeyHash:   keyHash,
        Prefix:    prefix,
        CreatedAt: time.Now(),
        ExpiresAt: expiresAt,
    }
    
    // Save to database
    if err := saveAPIKey(apiKey); err != nil {
        return nil, "", fmt.Errorf("API key storage failed: %w", err)
    }
    
    return apiKey, fullKey, nil
}

func validateAPIKey(keyValue string) (*APIKey, error) {
    // Extract prefix from key
    if !strings.HasPrefix(keyValue, "bk_") || len(keyValue) < 11 {
        return nil, errors.New("invalid API key format")
    }
    
    prefix := keyValue[3:11] // Skip "bk_" prefix
    
    // Find API key by prefix
    apiKey, err := getAPIKeyByPrefix(prefix)
    if err != nil {
        return nil, errors.New("API key not found")
    }
    
    // Verify key hash
    keyHash := krypto.SHA256Hash([]byte(keyValue))
    if !krypto.SHA256Verify([]byte(keyValue), apiKey.KeyHash) {
        return nil, errors.New("invalid API key")
    }
    
    // Check expiration
    if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
        return nil, errors.New("API key expired")
    }
    
    // Update last used timestamp
    now := time.Now()
    apiKey.LastUsed = &now
    updateAPIKeyLastUsed(apiKey.ID, now)
    
    return apiKey, nil
}
```

### Two-Factor Authentication

```go
// Purpose: Implement TOTP-based two-factor authentication
// Prerequisites: User with 2FA enabled
// Expected outcome: Secure 2FA implementation

type TwoFactorAuth struct {
    UserID    int64     `json:"user_id"`
    Secret    string    `json:"-"` // Encrypted secret
    Enabled   bool      `json:"enabled"`
    BackupCodes []string `json:"-"` // Encrypted backup codes
    CreatedAt time.Time `json:"created_at"`
}

func setupTwoFactorAuth(userID int64) (*TwoFactorAuth, []string, error) {
    // Generate secret for TOTP
    secret, err := krypto.GenerateSecureToken(32)
    if err != nil {
        return nil, nil, fmt.Errorf("secret generation failed: %w", err)
    }
    
    // Generate backup codes
    backupCodes := make([]string, 8)
    for i := range backupCodes {
        code := krypto.GenerateOTP(8)
        backupCodes[i] = code
    }
    
    // Encrypt secret and backup codes for storage
    encryptionKey := get2FAEncryptionKey() // Application-specific key
    aesService := krypto.NewAESGCMService(encryptionKey)
    
    encryptedSecret, secretNonce, err := aesService.Encrypt([]byte(secret))
    if err != nil {
        return nil, nil, fmt.Errorf("secret encryption failed: %w", err)
    }
    
    // Encrypt backup codes
    encryptedBackupCodes := make([]string, len(backupCodes))
    for i, code := range backupCodes {
        encrypted, nonce, err := aesService.Encrypt([]byte(code))
        if err != nil {
            return nil, nil, fmt.Errorf("backup code encryption failed: %w", err)
        }
        // Store encrypted code with nonce
        encryptedBackupCodes[i] = base64.StdEncoding.EncodeToString(append(nonce, encrypted...))
    }
    
    tfa := &TwoFactorAuth{
        UserID:      userID,
        Secret:      base64.StdEncoding.EncodeToString(append(secretNonce, encryptedSecret...)),
        Enabled:     false, // Enable after user confirms setup
        BackupCodes: encryptedBackupCodes,
        CreatedAt:   time.Now(),
    }
    
    // Save to database
    if err := saveTwoFactorAuth(tfa); err != nil {
        return nil, nil, fmt.Errorf("2FA storage failed: %w", err)
    }
    
    return tfa, backupCodes, nil
}

func verifyTOTP(userID int64, code string) (bool, error) {
    // Get 2FA settings
    tfa, err := getTwoFactorAuth(userID)
    if err != nil || !tfa.Enabled {
        return false, errors.New("2FA not enabled")
    }
    
    // Decrypt secret
    encryptionKey := get2FAEncryptionKey()
    aesService := krypto.NewAESGCMService(encryptionKey)
    
    secretData, err := base64.StdEncoding.DecodeString(tfa.Secret)
    if err != nil {
        return false, err
    }
    
    nonce := secretData[:12] // AES-GCM nonce is 12 bytes
    encrypted := secretData[12:]
    
    secretBytes, err := aesService.Decrypt(encrypted, nonce)
    if err != nil {
        return false, fmt.Errorf("secret decryption failed: %w", err)
    }
    
    secret := string(secretBytes)
    
    // Verify TOTP code (implementation would use a TOTP library)
    valid := verifyTOTPCode(secret, code)
    
    return valid, nil
}
```

## Security Best Practices

### Key Management

```go
// Purpose: Secure key management practices
// Prerequisites: Understanding of cryptographic key security
// Expected outcome: Secure key handling throughout application

// Never hardcode keys in source code
// ❌ Bad
const jwtSecret = "hardcoded-secret-key"

// ✅ Good - Load from environment
func getJWTSecret() string {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }
    return secret
}

// Generate secure keys programmatically
func generateApplicationKeys() error {
    // Generate JWT secret
    jwtSecret, err := krypto.GenerateSecureToken(64)
    if err != nil {
        return err
    }
    
    // Generate encryption key (32 bytes for AES-256)
    encryptionKey, err := krypto.GenerateSecureToken(32)
    if err != nil {
        return err
    }
    
    fmt.Printf("JWT_SECRET=%s\n", jwtSecret)
    fmt.Printf("ENCRYPTION_KEY=%s\n", encryptionKey)
    
    return nil
}
```

### Secure Password Policies

```go
// Purpose: Implement secure password policies
// Prerequisites: User registration system
// Expected outcome: Strong password requirements

func validatePasswordStrength(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters long")
    }
    
    var (
        hasUpper   = false
        hasLower   = false
        hasNumber  = false
        hasSpecial = false
    )
    
    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsNumber(char):
            hasNumber = true
        case unicode.IsPunct(char) || unicode.IsSymbol(char):
            hasSpecial = true
        }
    }
    
    if !hasUpper {
        return errors.New("password must contain at least one uppercase letter")
    }
    if !hasLower {
        return errors.New("password must contain at least one lowercase letter")
    }
    if !hasNumber {
        return errors.New("password must contain at least one number")
    }
    if !hasSpecial {
        return errors.New("password must contain at least one special character")
    }
    
    return nil
}
```

## Error Handling

```go
// Purpose: Handle cryptographic operation errors securely
// Prerequisites: Understanding of security implications
// Expected outcome: Secure error handling without information leakage

func secureErrorHandling() {
    // ❌ Bad - Reveals internal details
    if err := krypto.Argon2idVerifyPassword(password, hash); err != nil {
        log.Printf("Argon2id verification failed: %v", err)
        return errors.New("argon2id verification failed")
    }
    
    // ✅ Good - Generic error message
    if err := krypto.Argon2idVerifyPassword(password, hash); err != nil {
        log.Printf("Password verification failed for user") // Log internally
        return errors.New("invalid credentials") // Generic user message
    }
}
```

## Testing Patterns

```go
// Purpose: Test cryptographic functions securely
// Prerequisites: Test environment setup
// Expected outcome: Comprehensive security testing

func TestPasswordHashing(t *testing.T) {
    password := "test_password_123"
    
    // Test hashing
    hash, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        t.Fatal("Password hashing failed:", err)
    }
    
    // Hash should be different each time (due to random salt)
    hash2, err := krypto.Argon2idHashPassword(password)
    if err != nil {
        t.Fatal("Second password hashing failed:", err)
    }
    
    if hash == hash2 {
        t.Error("Password hashes should be different due to random salt")
    }
    
    // Test verification
    valid, err := krypto.Argon2idVerifyPassword(password, hash)
    if err != nil {
        t.Fatal("Password verification failed:", err)
    }
    
    if !valid {
        t.Error("Password verification should succeed")
    }
    
    // Test with wrong password
    valid, err = krypto.Argon2idVerifyPassword("wrong_password", hash)
    if err != nil {
        t.Fatal("Password verification error:", err)
    }
    
    if valid {
        t.Error("Password verification should fail with wrong password")
    }
}

func TestJWTTokens(t *testing.T) {
    // Set test JWT secret
    os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
    defer os.Unsetenv("JWT_SECRET")
    
    claims := krypto.UserClaims{
        First: "Test",
        Last:  "User",
        Token: "123",
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(time.Hour).Unix(),
            IssuedAt:  time.Now().Unix(),
        },
    }
    
    // Test token generation
    token, err := krypto.NewHs256AccessToken(claims)
    if err != nil {
        t.Fatal("Token generation failed:", err)
    }
    
    if token == "" {
        t.Error("Token should not be empty")
    }
    
    // Test token parsing
    parsedClaims, err := krypto.ParseHs256AccessToken(token)
    if err != nil {
        t.Fatal("Token parsing failed:", err)
    }
    
    if parsedClaims.First != claims.First {
        t.Errorf("Expected first name %s, got %s", claims.First, parsedClaims.First)
    }
    
    if parsedClaims.Token != claims.Token {
        t.Errorf("Expected token %s, got %s", claims.Token, parsedClaims.Token)
    }
}
```

The krypto package provides a comprehensive suite of cryptographic utilities designed with security best practices in mind, making it easy to implement secure authentication, encryption, and token management in your applications.