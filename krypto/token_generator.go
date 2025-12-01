package krypto

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

// GenerateSecureToken generates a secure token of the specified length.
// It utilizes the cryptographic randomness provided by the rand package
// to ensure the security and unpredictability of the generated token.
//
// Parameters:
//
//	length: The length of the secure token to be generated.
//
// Returns:
//
//	string: The randomly generated secure token in hexadecimal format.
//	error: An error, if any, encountered during the token generation process.
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	return token, nil
}

func RetryGenerateSecureToken(length int, retries int) (string, error) {
	var err error
	for i := 0; i < retries; i++ {
		b := make([]byte, length)
		_, err = rand.Read(b)
		if err == nil {
			token := hex.EncodeToString(b)
			return token, nil
		}
	}
	return "", err // All retries failed, return the error
}

// GenerateRandomString generates a random string of the specified length.
// It utilizes a cryptographically secure random number generator
// to ensure randomness and security in the generated string.
//
// Parameters:
//
//	length: The length of the random string to be generated.
//
// Returns:
//
//	string: The randomly generated string.
func GenerateRandomString(length int) string {
	// Define the character set from which the random string will be generated
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLen := big.NewInt(int64(len(charset)))

	// Generate the random string using crypto/rand
	randomString := make([]byte, length)
	for i := range randomString {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback should never happen with crypto/rand
			panic("crypto/rand failed: " + err.Error())
		}
		randomString[i] = charset[n.Int64()]
	}

	return string(randomString)
}

func GenerateToken64() string {
	return strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")
}
