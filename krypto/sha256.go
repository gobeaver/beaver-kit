package krypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSHA256 hashes the input string using SHA-256 algorithm.
//
// Parameters:
//
//	input (string): The input string to be hashed.
//
// Returns:
//
//	string: The hexadecimal representation of the hashed value.
func HashSHA256(input string) string {
	// Convert the input string to bytes
	inputBytes := []byte(input)

	// Create a new SHA-256 hash object
	hash := sha256.New()

	// Write the input bytes to the hash object
	hash.Write(inputBytes)

	// Get the hashed bytes
	hashedBytes := hash.Sum(nil)

	// Convert the hashed bytes to a hexadecimal string
	hashedString := hex.EncodeToString(hashedBytes)

	return hashedString
}

// VerifySHA256 verifies if the input matches the hashed value.
//
// Parameters:
//
//	input (string): The input string to be verified.
//	hashedValue (string): The hashed value to be compared with the hashed input.
//
// Returns:
//
//	bool: Returns true if the input matches the hashed value; otherwise, returns false.
func VerifySHA256(input, hashedValue string) bool {
	// Hash the input using SHA-256
	hashedInput := HashSHA256(input)

	// Compare the hashed input with the provided hashed value
	return hashedInput == hashedValue
}
