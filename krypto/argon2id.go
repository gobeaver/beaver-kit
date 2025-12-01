package krypto

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory      = 4096
	iterations  = 3
	parallelism = 6
	saltLength  = 16
	keyLength   = 32
)

// Argon2idHashPassword generates an Argon2id hash from the provided password.
// It uses a cryptographically secure random salt and configurable parameters
// for iterations, memory, parallelism, and key length to generate the hash.
// The generated hash is encoded in a string format containing information about
// the hash parameters and the encoded hash itself.
//
// Parameters:
//
//	password: The plaintext password to be hashed.
//
// Returns:
//
//	string: The hashed password encoded in a string format containing hash parameters.
//	error: An error, if any, encountered during the hashing process.
func Argon2idHashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	encodedHash := base64.StdEncoding.EncodeToString(hash)
	return fmt.Sprintf("d%d$%d$%d$%s$%s", memory, iterations, parallelism, base64.StdEncoding.EncodeToString(salt), encodedHash), nil
}

// Argon2idVerifyPassword verifies a password against an Argon2id hash.
// It parses the encoded hash string to extract the parameters and salt used to create
// the original hash, then applies the same parameters to the input password to verify
// if it matches the stored hash. This function uses constant-time comparison to prevent
// timing attacks.
//
// Parameters:
//
//	password: The plaintext password to verify.
//	encodedHash: The encoded hash string to verify against, as returned by Argon2idHashPassword.
//
// Returns:
//
//	bool: True if the password matches the hash, false otherwise.
//	error: An error if the hash format is invalid or parameters couldn't be parsed.
func Argon2idVerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash to extract parameters and salt
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 5 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Parse parameters from the hash
	var m uint32
	_, err := fmt.Sscanf(parts[0], "d%d", &m)
	if err != nil {
		return false, fmt.Errorf("failed to parse memory parameter: %w", err)
	}

	var i uint32
	_, err = fmt.Sscanf(parts[1], "%d", &i)
	if err != nil {
		return false, fmt.Errorf("failed to parse iterations parameter: %w", err)
	}

	var p uint32
	_, err = fmt.Sscanf(parts[2], "%d", &p)
	if err != nil {
		return false, fmt.Errorf("failed to parse parallelism parameter: %w", err)
	}
	// Validate parallelism fits in uint8 (argon2 API requirement)
	if p > 255 {
		return false, fmt.Errorf("parallelism parameter too large: %d (max 255)", p)
	}

	// Decode the salt
	salt, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decode the hash
	decodedHash, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute hash of the password with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, i, m, uint8(p), uint32(len(decodedHash))) //nolint:gosec // p validated above

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1, nil
}
