package krypto

import (
	"regexp"
	"testing"
)

func TestGenerateSecureToken(t *testing.T) {
	length := 16 // Example length

	// Generate secure token
	token, err := GenerateSecureToken(length)

	// Check for errors
	if err != nil {
		t.Errorf("Error generating secure token: %v", err)
	}

	// Check if the length of generated token matches the expected length
	if len(token) != length*2 { // Since each byte is represented by 2 hexadecimal characters
		t.Errorf("Generated secure token length is %d, expected %d", len(token), length*2)
	}

	// Check if the generated token contains only hexadecimal characters
	match, _ := regexp.MatchString("^[0-9a-fA-F]+$", token)
	if !match {
		t.Errorf("Generated secure token contains invalid characters")
	}
}

func TestGenerateRandomString(t *testing.T) {
	length := 10 // Example length

	// Generate random string
	randomString := GenerateRandomString(length)

	// Check if the length of generated string matches the expected length
	if len(randomString) != length {
		t.Errorf("Generated random string length is %d, expected %d", len(randomString), length)
	}

	// Check if the generated string contains only alphanumeric characters
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", randomString)
	if !match {
		t.Errorf("Generated random string contains invalid characters")
	}
}
