package krypto

import (
	"testing"
)

func TestGenerateAndValidateRSAKeyPair(t *testing.T) {
	// Generate RSA key pair
	keyPair, err := GenerateRSAKeyPair()
	if err != nil {
		t.Errorf("Error generating RSA key pair: %v", err)
	}

	// Validate RSA key pair
	valid, err := ValidateRSAKeyPair(keyPair)
	if err != nil {
		t.Errorf("Error validating RSA key pair: %v", err)
	}

	// Check if the key pair is valid
	if !valid {
		t.Error("Generated RSA key pair is not valid")
	}
}

func TestValidateRSAKeyPairInvalidKey(t *testing.T) {
	// Create an invalid RSA key pair
	invalidKeyPair := PublicPrivatePair{
		PrivateKey: "Invalid private key",
		PublicKey:  "Invalid public key",
	}

	// Validate RSA key pair
	valid, err := ValidateRSAKeyPair(invalidKeyPair)
	if err == nil {
		t.Error("Expected error while validating invalid RSA key pair, got nil")
	}

	// Check if the key pair is invalid
	if valid {
		t.Error("Invalid RSA key pair was considered valid")
	}
}

func TestValidateRSAKeyPairMismatchedKeys(t *testing.T) {
	// Generate RSA key pair
	keyPair, err := GenerateRSAKeyPair()
	if err != nil {
		t.Errorf("Error generating RSA key pair: %v", err)
	}

	// Manipulate public key to make it mismatched
	keyPair.PublicKey = "Manipulated public key"

	// Validate RSA key pair
	valid, err := ValidateRSAKeyPair(keyPair)
	if err == nil {
		t.Error("Expected error due to mismatched keys, got nil")
	}

	// Check if the key pair is invalid
	if valid {
		t.Error("RSA key pair with mismatched keys was considered valid")
	}
}
