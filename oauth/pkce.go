package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// GeneratePKCEChallenge generates a PKCE challenge with verifier and challenge
func GeneratePKCEChallenge(method string) (*PKCEChallenge, error) {
	// Generate code verifier (43-128 characters)
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Generate code challenge based on method
	var challenge string
	switch method {
	case "S256":
		challenge = generateS256Challenge(verifier)
	case "plain":
		challenge = verifier
	default:
		return nil, fmt.Errorf("unsupported PKCE method: %s", method)
	}

	return &PKCEChallenge{
		Verifier:        verifier,
		Challenge:       challenge,
		ChallengeMethod: method,
	}, nil
}

// generateCodeVerifier generates a cryptographically random code verifier
func generateCodeVerifier() (string, error) {
	// Generate 32 bytes of random data (will be 43 chars when base64url encoded)
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	// Base64url encode without padding
	verifier := base64.RawURLEncoding.EncodeToString(data)
	
	// Ensure it meets the length requirements (43-128 characters)
	if len(verifier) < 43 {
		// This shouldn't happen with 32 bytes, but just in case
		return "", fmt.Errorf("generated verifier too short: %d chars", len(verifier))
	}
	
	return verifier, nil
}

// generateS256Challenge generates SHA256 challenge from verifier
func generateS256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ValidatePKCEChallenge validates that a verifier matches a challenge
func ValidatePKCEChallenge(verifier, challenge, method string) bool {
	switch method {
	case "S256":
		expectedChallenge := generateS256Challenge(verifier)
		return challenge == expectedChallenge
	case "plain":
		return verifier == challenge
	default:
		return false
	}
}

// IsPKCESupported checks if a provider supports PKCE based on provider name
func IsPKCESupported(provider string) bool {
	// Most modern providers support PKCE
	supportedProviders := map[string]bool{
		"google":  true,
		"github":  true,
		"apple":   true,
		"twitter": true, // Twitter OAuth 2.0 supports PKCE
		"custom":  true, // Assume custom providers support it
	}
	
	return supportedProviders[strings.ToLower(provider)]
}

// PKCEParams returns URL parameters for PKCE
func PKCEParams(pkce *PKCEChallenge) map[string]string {
	if pkce == nil {
		return nil
	}
	
	return map[string]string{
		"code_challenge":        pkce.Challenge,
		"code_challenge_method": pkce.ChallengeMethod,
	}
}

// PKCETokenParams returns token exchange parameters for PKCE
func PKCETokenParams(pkce *PKCEChallenge) map[string]string {
	if pkce == nil {
		return nil
	}
	
	return map[string]string{
		"code_verifier": pkce.Verifier,
	}
}