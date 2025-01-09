package krypto

import (
	"crypto/rand"
	"encoding/base64"
)

// generateHS256Key generates a random key suitable for use with HS256 algorithm.
// It returns the generated key as a base64-encoded string and any error encountered.
func generateHS256Key() (string, error) {
	// Generate 32 bytes of random data
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	// Encode the random bytes to base64
	encodedKey := base64.URLEncoding.EncodeToString(key)
	return encodedKey, nil
}
