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
// RFC 7636 specifies: 43-128 characters from unreserved characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
func generateCodeVerifier() (string, error) {
	// Generate between 43 and 128 characters (we'll use 64 bytes = 86 chars when base64url encoded)
	// This gives us a good balance between security and URL length
	data := make([]byte, 64)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	// Base64url encode without padding (uses only unreserved characters)
	verifier := base64.RawURLEncoding.EncodeToString(data)

	// Ensure it meets the RFC 7636 length requirements (43-128 characters)
	if len(verifier) < 43 || len(verifier) > 128 {
		return "", fmt.Errorf("generated verifier invalid length: %d chars (must be 43-128)", len(verifier))
	}

	// Validate that only unreserved characters are used (base64url guarantees this)
	for _, c := range verifier {
		if !isUnreservedChar(c) {
			return "", fmt.Errorf("invalid character in verifier: %c", c)
		}
	}

	return verifier, nil
}

// isUnreservedChar checks if a character is an unreserved character per RFC 7636
func isUnreservedChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
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

// GeneratePKCEChallengeWithLength generates a PKCE challenge with specified verifier length
func GeneratePKCEChallengeWithLength(method string, length int) (*PKCEChallenge, error) {
	// Validate length per RFC 7636
	if length < 43 || length > 128 {
		return nil, fmt.Errorf("invalid verifier length %d: must be between 43 and 128", length)
	}

	// Generate code verifier with specific length
	verifier, err := generateCodeVerifierWithLength(length)
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
		return nil, fmt.Errorf("unsupported PKCE method: %s (must be 'S256' or 'plain')", method)
	}

	return &PKCEChallenge{
		Verifier:        verifier,
		Challenge:       challenge,
		ChallengeMethod: method,
	}, nil
}

// generateCodeVerifierWithLength generates a code verifier with specific length
func generateCodeVerifierWithLength(length int) (string, error) {
	// Calculate how many bytes we need (roughly 4/3 ratio for base64)
	byteLength := (length * 3) / 4
	if byteLength < 32 {
		byteLength = 32 // Minimum for security
	}

	data := make([]byte, byteLength)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	// Base64url encode without padding
	verifier := base64.RawURLEncoding.EncodeToString(data)

	// Truncate to exact length if needed
	if len(verifier) > length {
		verifier = verifier[:length]
	}

	// Pad with random characters if too short (shouldn't happen)
	for len(verifier) < length {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		char := base64.RawURLEncoding.EncodeToString(b)
		if len(char) > 0 {
			verifier += string(char[0])
		}
	}

	// Final validation
	for _, c := range verifier {
		if !isUnreservedChar(c) {
			return "", fmt.Errorf("invalid character in verifier: %c", c)
		}
	}

	return verifier, nil
}

// ValidateVerifier validates a PKCE verifier per RFC 7636
func ValidateVerifier(verifier string) error {
	// Check length
	if len(verifier) < 43 || len(verifier) > 128 {
		return fmt.Errorf("invalid verifier length %d: must be between 43 and 128", len(verifier))
	}

	// Check characters
	for _, c := range verifier {
		if !isUnreservedChar(c) {
			return fmt.Errorf("invalid character in verifier: %c", c)
		}
	}

	return nil
}

// GetRecommendedPKCEMethod returns the recommended PKCE method
func GetRecommendedPKCEMethod() string {
	// RFC 7636 recommends S256 if the client can support it
	return "S256"
}
