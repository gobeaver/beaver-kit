# Beaver Kit - Krypto

A comprehensive and secure cryptographic utilities package for Go applications. The `krypto` package provides a collection of commonly needed cryptographic operations with a stable API designed to be reliable and future-proof.

## Installation

```bash
go get github.com/gobeaver/beaver-kit/krypto
```

## Features

- **Password Hashing**
  - Argon2id implementation with secure salt generation
  - Bcrypt implementation with configurable work factor
  
- **Token Management**
  - JWT token generation and validation (HS256)
  - Secure random token generation
  - OTP (One-Time Password) generation
  
- **Encryption**
  - AES-GCM encryption and decryption
  - RSA key pair generation and validation
  
- **Hashing**
  - SHA-256 hashing with verification
  
- **Security Utilities**
  - Random delay function to prevent timing attacks

## Configuration

The package can be initialized with environment variables:

```go
import (
    "github.com/gobeaver/beaver-kit/config"
    "github.com/gobeaver/beaver-kit/krypto"
)

// Initialize configuration
loader := config.NewLoader()
krypto.InitConfig(loader)
```

Environment variables:
- `KRYPTO_JWT_KEY`: Secret key for JWT signing
- `KRYPTO_RSA_PATH`: Path to RSA key files
- `KRYPTO_TOKEN_SECRET`: Secret for token generation

## Examples

### Password Hashing (Argon2id)

```go
// Hash a password
hashedPassword, err := krypto.Argon2idHashPassword("secure_password")
if err != nil {
    // Handle error
}

// Verify a password
match, err := krypto.Argon2idVerifyPassword("secure_password", hashedPassword)
if err != nil {
    // Handle error
}
```

### JWT Token Handling

```go
// Create user claims
claims := krypto.UserClaims{
    First: "John",
    Last: "Doe",
    Token: "user-identifier",
    StandardClaims: jwt.StandardClaims{
        ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
    },
}

// Generate access token
token, err := krypto.NewHs256AccessToken(claims)
if err != nil {
    // Handle error
}

// Parse and validate token
parsedClaims, err := krypto.ParseHs256AccessToken(token)
if err != nil {
    // Handle error
}
```

### Secure Token Generation

```go
// Generate a secure random token
token, err := krypto.GenerateSecureToken(32)
if err != nil {
    // Handle error
}

// Generate OTP
otp := krypto.GenerateOTP(6)
```

### AES Encryption

```go
// Create encryption service
aesService := krypto.NewAESGCMService("your-encryption-key")

// Encrypt data
encrypted, err := aesService.Encrypt([]byte("sensitive data"))
if err != nil {
    // Handle error
}

// Decrypt data
decrypted, err := aesService.Decrypt(encrypted)
if err != nil {
    // Handle error
}
```

## Security Considerations

This package follows best practices for cryptographic implementations:

- Uses cryptographically secure random number generation
- Implements proper error handling for all cryptographic operations
- Uses modern algorithms with appropriate parameters
- Avoids timing attacks with random delays where appropriate

For detailed API documentation, see the [doc.md](./doc.md) file.