package krypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Service defines the interface for encryption operations
type Service interface {
	Encrypt(data []byte) (ciphertext, nonce []byte, err error)
	Decrypt(ciphertext, nonce []byte) ([]byte, error)
	EncryptString(plaintext string) (ciphertextB64, nonceB64 string, err error)
	DecryptString(ciphertextB64, nonceB64 string) (string, error)
}

// aesGCMService implements the Service interface using AES-GCM
type aesGCMService struct {
	gcm cipher.AEAD // This is all we need
}

// NewAESGCMService creates a new AES-GCM encryption service
func NewAESGCMService(key string) (Service, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &aesGCMService{gcm: gcm}, nil
}

// Encrypt encrypts byte data using AES-GCM
func (s *aesGCMService) Encrypt(data []byte) ([]byte, []byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := s.gcm.Seal(nil, nonce, data, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts byte data using AES-GCM
func (s *aesGCMService) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded results
func (s *aesGCMService) EncryptString(plaintext string) (string, string, error) {
	ciphertext, nonce, err := s.Encrypt([]byte(plaintext))
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

// DecryptString decrypts base64 encoded strings
func (s *aesGCMService) DecryptString(ciphertextB64, nonceB64 string) (string, error) {
	// Handle empty input cases first
	if ciphertextB64 == "" || nonceB64 == "" {
		return "", fmt.Errorf("empty input: ciphertext and nonce cannot be empty")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Validate nonce size before attempting decryption
	if len(nonce) != s.gcm.NonceSize() {
		return "", fmt.Errorf("invalid nonce size: got %d, want %d", len(nonce), s.gcm.NonceSize())
	}

	plaintext, err := s.Decrypt(ciphertext, nonce)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func GenerateAESKey(keySize int) (string, error) {
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return "", fmt.Errorf("invalid key size: must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256")
	}

	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Encode to base64 for storage/transmission
	return base64.StdEncoding.EncodeToString(key), nil
}
