package krypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewAESGCMService(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key (32 bytes)",
			key:     strings.Repeat("a", 32),
			wantErr: false,
		},
		{
			name:    "invalid key size",
			key:     "too-short",
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewAESGCMService(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAESGCMService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && svc == nil {
				t.Error("NewAESGCMService() returned nil service with no error")
			}
		})
	}
}

func TestAESGCMService_Encrypt_Decrypt(t *testing.T) {
	// Create a service with a test key
	svc, err := NewAESGCMService(strings.Repeat("a", 32))
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "encrypt/decrypt normal text",
			data:    []byte("hello world"),
			wantErr: false,
		},
		{
			name:    "encrypt/decrypt empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "encrypt/decrypt binary data",
			data:    []byte{0xFF, 0x00, 0xFE, 0x01},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			ciphertext, nonce, err := svc.Encrypt(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify ciphertext is different from plaintext
				if bytes.Equal(ciphertext, tt.data) {
					t.Error("Encrypt() ciphertext equals plaintext")
				}

				// Test decryption
				plaintext, err := svc.Decrypt(ciphertext, nonce)
				if err != nil {
					t.Errorf("Decrypt() error = %v", err)
					return
				}

				// Verify decrypted data matches original
				if !bytes.Equal(plaintext, tt.data) {
					t.Errorf("Decrypt() got = %v, want %v", plaintext, tt.data)
				}
			}
		})
	}
}

func TestAESGCMService_EncryptString_DecryptString(t *testing.T) {
	svc, err := NewAESGCMService(strings.Repeat("a", 32))
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "normal string",
			plaintext: "hello world",
			wantErr:   false,
		},
		{
			name:      "empty string",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "unicode string",
			plaintext: "Hello, 世界",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			ciphertextB64, nonceB64, err := svc.EncryptString(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the results are valid base64
				if !isValidBase64(ciphertextB64) || !isValidBase64(nonceB64) {
					t.Error("EncryptString() returned invalid base64")
				}

				// Test decryption
				got, err := svc.DecryptString(ciphertextB64, nonceB64)
				if err != nil {
					t.Errorf("DecryptString() error = %v", err)
					return
				}

				// Verify decrypted string matches original
				if got != tt.plaintext {
					t.Errorf("DecryptString() got = %v, want %v", got, tt.plaintext)
				}
			}
		})
	}
}

// Test helper function to verify base64 strings
func isValidBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func TestAESGCMService_DecryptString_Invalid(t *testing.T) {
	svc, err := NewAESGCMService(strings.Repeat("a", 32))
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Get valid inputs for comparison
	validCiphertext, validNonce, err := svc.Encrypt([]byte("test"))
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}
	validCiphertextB64 := base64.StdEncoding.EncodeToString(validCiphertext)
	validNonceB64 := base64.StdEncoding.EncodeToString(validNonce)

	tests := []struct {
		name          string
		ciphertextB64 string
		nonceB64      string
		wantErrMsg    string
	}{
		{
			name:          "empty strings",
			ciphertextB64: "",
			nonceB64:      "",
			wantErrMsg:    "empty input",
		},
		{
			name:          "empty ciphertext",
			ciphertextB64: "",
			nonceB64:      validNonceB64,
			wantErrMsg:    "empty input",
		},
		{
			name:          "empty nonce",
			ciphertextB64: validCiphertextB64,
			nonceB64:      "",
			wantErrMsg:    "empty input",
		},
		{
			name:          "invalid base64 ciphertext",
			ciphertextB64: "invalid-base64",
			nonceB64:      validNonceB64,
			wantErrMsg:    "failed to decode ciphertext",
		},
		{
			name:          "invalid base64 nonce",
			ciphertextB64: validCiphertextB64,
			nonceB64:      "invalid-base64",
			wantErrMsg:    "failed to decode nonce",
		},
		{
			name:          "wrong nonce size",
			ciphertextB64: validCiphertextB64,
			nonceB64:      base64.StdEncoding.EncodeToString([]byte("wrongsize")),
			wantErrMsg:    "invalid nonce size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.DecryptString(tt.ciphertextB64, tt.nonceB64)
			if err == nil {
				t.Errorf("DecryptString() expected error for invalid input")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("DecryptString() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}
