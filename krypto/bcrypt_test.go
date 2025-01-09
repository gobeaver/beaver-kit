package krypto

import "testing"

func TestBcryptHashPasswordAndCheckPasswordHash(t *testing.T) {
	// Define a test password
	testPassword := "password123"

	// Hash the test password
	hashedPassword, err := BcryptHashPassword(testPassword)
	if err != nil {
		t.Errorf("Error hashing password: %v", err)
	}

	// Check if the hashed password matches the original password
	match := BcryptCheckPasswordHash(testPassword, hashedPassword)
	if !match {
		t.Errorf("Password hash check failed")
	}

	// Check a wrong password
	wrongPassword := "wrongpassword"
	wrongMatch := BcryptCheckPasswordHash(wrongPassword, hashedPassword)
	if wrongMatch {
		t.Errorf("Incorrect password should not match")
	}
}
