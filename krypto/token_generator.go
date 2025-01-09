package krypto

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/google/uuid"
	rand2 "math/rand"
	"strings"
	"time"
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
// It utilizes a pseudo-random number generator seeded with the current time
// to ensure randomness in the generated string.
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

	// Seed the random number generator with the current time
	rand2.Seed(time.Now().UnixNano())

	// Generate the random string
	randomString := make([]byte, length)
	for i := range randomString {
		randomString[i] = charset[rand2.Intn(len(charset))]
	}

	return string(randomString)
}

func GenerateToken64() string {
	return strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")
}
