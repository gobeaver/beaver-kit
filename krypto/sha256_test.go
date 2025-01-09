package krypto

import "testing"

func TestHashSHA256(t *testing.T) {
	input := "password123"
	expectedHash := "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f" // Precomputed hash for "password123"

	hashedValue := HashSHA256(input)

	if hashedValue != expectedHash {
		t.Errorf("HashSHA256(%s) = %s; want %s", input, hashedValue, expectedHash)
	}
}

func TestVerifySHA256(t *testing.T) {
	input := "password123"
	hashedValue := HashSHA256(input)

	// Verify with correct input
	if !VerifySHA256(input, hashedValue) {
		t.Errorf("VerifySHA256(%s, %s) returned false; want true", input, hashedValue)
	}

	// Verify with incorrect input
	if VerifySHA256("wrongpassword", hashedValue) {
		t.Errorf("VerifySHA256(%s, %s) returned true; want false", "wrongpassword", hashedValue)
	}
}
