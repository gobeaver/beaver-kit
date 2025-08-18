package oauth_test

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestGeneratePKCEChallenge(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "S256 method",
			method:  "S256",
			wantErr: false,
		},
		{
			name:    "plain method",
			method:  "plain",
			wantErr: false,
		},
		{
			name:    "invalid method",
			method:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := oauth.GeneratePKCEChallenge(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePKCEChallenge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Validate verifier length (RFC 7636: 43-128 characters)
				if len(challenge.Verifier) < 43 || len(challenge.Verifier) > 128 {
					t.Errorf("Verifier length = %d, want 43-128", len(challenge.Verifier))
				}

				// Validate verifier characters (unreserved characters only)
				for _, c := range challenge.Verifier {
					if !isValidPKCEChar(c) {
						t.Errorf("Invalid character in verifier: %c", c)
					}
				}

				// Validate challenge method
				if challenge.ChallengeMethod != tt.method {
					t.Errorf("ChallengeMethod = %s, want %s", challenge.ChallengeMethod, tt.method)
				}

				// Validate challenge generation
				if tt.method == "S256" {
					// Verify S256 challenge is correct
					expectedChallenge := generateS256Challenge(challenge.Verifier)
					if challenge.Challenge != expectedChallenge {
						t.Errorf("S256 challenge mismatch")
					}
				} else if tt.method == "plain" {
					// For plain method, challenge should equal verifier
					if challenge.Challenge != challenge.Verifier {
						t.Errorf("Plain challenge should equal verifier")
					}
				}
			}
		})
	}
}

func TestGeneratePKCEChallengeWithLength(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		length  int
		wantErr bool
	}{
		{
			name:    "minimum length (43)",
			method:  "S256",
			length:  43,
			wantErr: false,
		},
		{
			name:    "maximum length (128)",
			method:  "S256",
			length:  128,
			wantErr: false,
		},
		{
			name:    "typical length (86)",
			method:  "S256",
			length:  86,
			wantErr: false,
		},
		{
			name:    "too short (42)",
			method:  "S256",
			length:  42,
			wantErr: true,
		},
		{
			name:    "too long (129)",
			method:  "S256",
			length:  129,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge, err := oauth.GeneratePKCEChallengeWithLength(tt.method, tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePKCEChallengeWithLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Validate exact verifier length
				if len(challenge.Verifier) != tt.length {
					t.Errorf("Verifier length = %d, want %d", len(challenge.Verifier), tt.length)
				}

				// Validate verifier characters
				for _, c := range challenge.Verifier {
					if !isValidPKCEChar(c) {
						t.Errorf("Invalid character in verifier: %c", c)
					}
				}
			}
		})
	}
}

func TestValidatePKCEChallenge(t *testing.T) {
	// Generate a valid challenge
	validChallenge, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("Failed to generate valid challenge: %v", err)
	}

	tests := []struct {
		name      string
		verifier  string
		challenge string
		method    string
		want      bool
	}{
		{
			name:      "valid S256 challenge",
			verifier:  validChallenge.Verifier,
			challenge: validChallenge.Challenge,
			method:    "S256",
			want:      true,
		},
		{
			name:      "invalid S256 challenge",
			verifier:  validChallenge.Verifier,
			challenge: "invalid_challenge",
			method:    "S256",
			want:      false,
		},
		{
			name:      "valid plain challenge",
			verifier:  "test_verifier_with_43_characters_minimum_length",
			challenge: "test_verifier_with_43_characters_minimum_length",
			method:    "plain",
			want:      true,
		},
		{
			name:      "invalid plain challenge",
			verifier:  "test_verifier_with_43_characters_minimum_length",
			challenge: "different_challenge_with_43_characters_minimum_",
			method:    "plain",
			want:      false,
		},
		{
			name:      "unsupported method",
			verifier:  validChallenge.Verifier,
			challenge: validChallenge.Challenge,
			method:    "unsupported",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oauth.ValidatePKCEChallenge(tt.verifier, tt.challenge, tt.method)
			if got != tt.want {
				t.Errorf("ValidatePKCEChallenge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateVerifier(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		wantErr  bool
	}{
		{
			name:     "valid minimum length verifier",
			verifier: strings.Repeat("a", 43),
			wantErr:  false,
		},
		{
			name:     "valid maximum length verifier",
			verifier: strings.Repeat("A", 128),
			wantErr:  false,
		},
		{
			name:     "too short verifier",
			verifier: strings.Repeat("a", 42),
			wantErr:  true,
		},
		{
			name:     "too long verifier",
			verifier: strings.Repeat("a", 129),
			wantErr:  true,
		},
		{
			name:     "verifier with invalid characters",
			verifier: strings.Repeat("a", 43) + "!@#",
			wantErr:  true,
		},
		{
			name:     "valid verifier with all allowed characters",
			verifier: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := oauth.ValidateVerifier(tt.verifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVerifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRecommendedPKCEMethod(t *testing.T) {
	method := oauth.GetRecommendedPKCEMethod()
	if method != "S256" {
		t.Errorf("GetRecommendedPKCEMethod() = %s, want S256", method)
	}
}

func TestPKCEParams(t *testing.T) {
	challenge, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("Failed to generate challenge: %v", err)
	}

	params := oauth.PKCEParams(challenge)
	if params == nil {
		t.Fatal("PKCEParams() returned nil")
	}

	if params["code_challenge"] != challenge.Challenge {
		t.Errorf("code_challenge = %s, want %s", params["code_challenge"], challenge.Challenge)
	}

	if params["code_challenge_method"] != challenge.ChallengeMethod {
		t.Errorf("code_challenge_method = %s, want %s", params["code_challenge_method"], challenge.ChallengeMethod)
	}

	// Test with nil challenge
	nilParams := oauth.PKCEParams(nil)
	if nilParams != nil {
		t.Error("PKCEParams(nil) should return nil")
	}
}

func TestPKCETokenParams(t *testing.T) {
	challenge, err := oauth.GeneratePKCEChallenge("S256")
	if err != nil {
		t.Fatalf("Failed to generate challenge: %v", err)
	}

	params := oauth.PKCETokenParams(challenge)
	if params == nil {
		t.Fatal("PKCETokenParams() returned nil")
	}

	if params["code_verifier"] != challenge.Verifier {
		t.Errorf("code_verifier = %s, want %s", params["code_verifier"], challenge.Verifier)
	}

	// Test with nil challenge
	nilParams := oauth.PKCETokenParams(nil)
	if nilParams != nil {
		t.Error("PKCETokenParams(nil) should return nil")
	}
}

func TestPKCEUniqueness(t *testing.T) {
	// Generate multiple challenges and ensure they're unique
	challenges := make(map[string]bool)
	for i := 0; i < 100; i++ {
		challenge, err := oauth.GeneratePKCEChallenge("S256")
		if err != nil {
			t.Fatalf("Failed to generate challenge: %v", err)
		}

		if challenges[challenge.Verifier] {
			t.Error("Generated duplicate verifier")
		}
		challenges[challenge.Verifier] = true
	}
}

// Helper functions

func isValidPKCEChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

func generateS256Challenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}