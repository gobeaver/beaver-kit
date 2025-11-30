// Package krypto provides cryptographic utilities for common security operations
// including password hashing, token generation, digital signatures, and secure random generation.
//
// This package offers production-ready cryptographic functions with secure defaults
// and easy-to-use APIs. It emphasizes security best practices and provides utilities
// for common authentication and authorization scenarios.
//
// # Password Hashing
//
// Secure password hashing using bcrypt with automatic salt generation:
//
//	import "github.com/gobeaver/beaver-kit/krypto"
//
//	// Hash a password
//	hashedPassword, err := krypto.BcryptHashPassword("userPassword123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify a password
//	isValid := krypto.BcryptCheckPasswordHash("userPassword123", hashedPassword)
//	if isValid {
//	    // Password is correct
//	    fmt.Println("Authentication successful")
//	}
//
// # Secure Token Generation
//
// Generate cryptographically secure random tokens for sessions, API keys, etc.:
//
//	// Generate a secure random token (32 bytes, hex-encoded)
//	token, err := krypto.GenerateSecureToken(32)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Secure token: %s\n", token) // e.g., "a1b2c3d4e5f6..."
//
//	// Generate shorter token
//	shortToken, err := krypto.GenerateSecureToken(16)
//
// # One-Time Passwords (OTP)
//
// Generate secure OTP codes for two-factor authentication:
//
//	// Generate 6-digit OTP
//	otp, err := krypto.GenerateOTP(6)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Your OTP: %s\n", otp) // e.g., "123456"
//
//	// Generate 8-digit OTP
//	longOTP, err := krypto.GenerateOTP(8)
//
// # SHA-256 Hashing
//
// Hash data using SHA-256 with optional verification:
//
//	data := "sensitive data to hash"
//	hash := krypto.HashSHA256(data)
//	fmt.Printf("SHA-256: %s\n", hash)
//
//	// Verify hash
//	isValid := krypto.VerifySHA256(data, hash)
//	if isValid {
//	    fmt.Println("Hash verification successful")
//	}
//
// # RSA Key Generation
//
// Generate RSA key pairs for asymmetric cryptography:
//
//	// Generate 2048-bit RSA key pair
//	keyPair, err := krypto.GenerateRSAKeyPair(2048)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Private Key: %s\n", keyPair.PrivateKey)
//	fmt.Printf("Public Key: %s\n", keyPair.PublicKey)
//
//	// Validate the generated key pair
//	isValid, err := krypto.ValidateRSAKeyPair(keyPair.PrivateKey, keyPair.PublicKey)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if isValid {
//	    fmt.Println("RSA key pair is valid")
//	}
//
// # JWT Token Operations
//
// Create and validate JWT tokens with RSA or HMAC signing:
//
//	// Create user claims
//	claims := &krypto.UserClaims{
//	    UserID:   "user123",
//	    Username: "john.doe",
//	    Email:    "john@example.com",
//	    Roles:    []string{"user", "admin"},
//	}
//	claims.ExpiresAt = time.Now().Add(time.Hour * 24).Unix() // 24 hours
//	claims.IssuedAt = time.Now().Unix()
//
//	// Sign JWT with RSA private key
//	token, err := krypto.CreateJWT(claims, keyPair.PrivateKey, krypto.RSAAlgorithm)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify JWT with RSA public key
//	verifiedClaims, err := krypto.VerifyJWT(token, keyPair.PublicKey, krypto.RSAAlgorithm)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # HMAC Operations
//
// Generate HMAC signatures for message authentication:
//
//	message := "important message"
//	secret := "shared-secret-key"
//
//	// Generate HMAC-SHA256 signature
//	signature := krypto.GenerateHMAC(message, secret)
//	fmt.Printf("HMAC: %s\n", signature)
//
//	// Verify HMAC signature
//	isValid := krypto.VerifyHMAC(message, secret, signature)
//	if isValid {
//	    fmt.Println("HMAC verification successful")
//	}
//
// # AES Encryption
//
// Symmetric encryption using AES-GCM for data protection:
//
//	// Generate AES key
//	key, err := krypto.GenerateAESKey(32) // 256-bit key
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	plaintext := "sensitive data to encrypt"
//
//	// Encrypt data
//	ciphertext, err := krypto.AESGCMEncrypt([]byte(plaintext), key)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Decrypt data
//	decrypted, err := krypto.AESGCMDecrypt(ciphertext, key)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Decrypted: %s\n", string(decrypted))
//
// # Secure Random Generation
//
// Generate cryptographically secure random data:
//
//	// Generate random bytes
//	randomBytes, err := krypto.GenerateRandomBytes(32)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Generate random string (base64 encoded)
//	randomString, err := krypto.GenerateRandomString(16)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Types and Constants
//
// Common types used throughout the package:
//
//	// RSA key pair structure
//	type PublicPrivatePair struct {
//	    PrivateKey string `json:"private_key"`
//	    PublicKey  string `json:"public_key"`
//	}
//
//	// JWT user claims
//	type UserClaims struct {
//	    UserID   string   `json:"user_id"`
//	    Username string   `json:"username"`
//	    Email    string   `json:"email"`
//	    Roles    []string `json:"roles"`
//	    jwt.StandardClaims
//	}
//
//	// Algorithm constants
//	const (
//	    RSAAlgorithm  = "RS256"
//	    HMACAlgorithm = "HS256"
//	)
//
// # Security Best Practices
//
// The package follows cryptographic best practices:
//   - Uses cryptographically secure random number generation
//   - Implements proper key derivation for password hashing (bcrypt)
//   - Supports modern algorithms (AES-GCM, RSA with OAEP, HMAC-SHA256)
//   - Provides secure defaults for key sizes and parameters
//   - Includes proper error handling for all operations
//   - Uses timing-safe comparison for hash verification
//
// # Error Handling
//
// All cryptographic operations return detailed errors:
//
//	token, err := krypto.GenerateSecureToken(32)
//	if err != nil {
//	    // Handle specific error types
//	    switch err {
//	    case krypto.ErrInsufficientEntropy:
//	        log.Fatal("System entropy too low")
//	    case krypto.ErrInvalidKeySize:
//	        log.Fatal("Invalid key size specified")
//	    default:
//	        log.Fatalf("Crypto error: %v", err)
//	    }
//	}
//
// # Integration Examples
//
// Common integration patterns:
//
//	// User registration with password hashing
//	func RegisterUser(username, password string) error {
//	    hashedPassword, err := krypto.BcryptHashPassword(password)
//	    if err != nil {
//	        return fmt.Errorf("failed to hash password: %w", err)
//	    }
//
//	    user := &User{
//	        Username: username,
//	        Password: hashedPassword,
//	        APIKey:   generateAPIKey(),
//	    }
//
//	    return saveUser(user)
//	}
//
//	func generateAPIKey() string {
//	    apiKey, _ := krypto.GenerateSecureToken(32)
//	    return apiKey
//	}
//
//	// Session token validation
//	func ValidateSession(tokenString string) (*UserClaims, error) {
//	    publicKey := getJWTPublicKey() // Your key management
//
//	    claims, err := krypto.VerifyJWT(tokenString, publicKey, krypto.RSAAlgorithm)
//	    if err != nil {
//	        return nil, fmt.Errorf("invalid session token: %w", err)
//	    }
//
//	    return claims, nil
//	}
//
// # Performance Considerations
//
//   - Password hashing (bcrypt) is intentionally slow - cache results when possible
//   - RSA operations are expensive - consider ECDSA for high-throughput scenarios
//   - AES-GCM is fast and suitable for real-time encryption/decryption
//   - Token generation is fast and suitable for high-frequency operations
//   - Consider key rotation policies for long-lived applications
//
// # Thread Safety
//
// All functions in this package are thread-safe and can be called concurrently
// from multiple goroutines without additional synchronization.
//
// For complete examples and advanced usage patterns, see the project documentation
// and security examples.
package krypto
