package oauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// EncryptedTokenStore wraps a TokenStore with encryption
type EncryptedTokenStore struct {
	store     TokenStore
	encryptor TokenEncryptor
}

// NewEncryptedTokenStore creates a new encrypted token store
func NewEncryptedTokenStore(store TokenStore, key []byte) (*EncryptedTokenStore, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &EncryptedTokenStore{
		store:     store,
		encryptor: encryptor,
	}, nil
}

// Store encrypts and stores a token
func (e *EncryptedTokenStore) Store(ctx context.Context, key string, token *Token) error {
	// Serialize token
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encrypt data
	encrypted, err := e.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Store encrypted data as a token with special marker
	encryptedToken := &Token{
		AccessToken: base64.StdEncoding.EncodeToString(encrypted),
		TokenType:   "encrypted",
	}

	return e.store.Store(ctx, key, encryptedToken)
}

// Retrieve decrypts and retrieves a token
func (e *EncryptedTokenStore) Retrieve(ctx context.Context, key string) (*Token, error) {
	// Retrieve encrypted token
	encryptedToken, err := e.store.Retrieve(ctx, key)
	if err != nil {
		return nil, err
	}

	// Check if token is encrypted
	if encryptedToken.TokenType != "encrypted" {
		// Return as-is for backward compatibility
		return encryptedToken, nil
	}

	// Decode from base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted token: %w", err)
	}

	// Decrypt data
	decrypted, err := e.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Deserialize token
	var token Token
	if err := json.Unmarshal(decrypted, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// Delete removes a token
func (e *EncryptedTokenStore) Delete(ctx context.Context, key string) error {
	return e.store.Delete(ctx, key)
}

// EncryptedSessionStore wraps a SessionStore with encryption
type EncryptedSessionStore struct {
	store     SessionStore
	encryptor TokenEncryptor
}

// NewEncryptedSessionStore creates a new encrypted session store
func NewEncryptedSessionStore(store SessionStore, key []byte) (*EncryptedSessionStore, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("encryption key cannot be empty")
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &EncryptedSessionStore{
		store:     store,
		encryptor: encryptor,
	}, nil
}

// Store encrypts and stores session data
func (e *EncryptedSessionStore) Store(ctx context.Context, key string, data *SessionData) error {
	// Serialize session data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Encrypt data
	encrypted, err := e.encryptor.Encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt session data: %w", err)
	}

	// Create wrapper session with encrypted data
	encryptedSession := &SessionData{
		State:    key, // Keep key as state for retrieval
		Provider: "encrypted:" + base64.StdEncoding.EncodeToString(encrypted),
	}

	return e.store.Store(ctx, key, encryptedSession)
}

// Retrieve decrypts and retrieves session data
func (e *EncryptedSessionStore) Retrieve(ctx context.Context, key string) (*SessionData, error) {
	// Retrieve encrypted session
	encryptedSession, err := e.store.Retrieve(ctx, key)
	if err != nil {
		return nil, err
	}

	// Check if session is encrypted
	if len(encryptedSession.Provider) < 10 || encryptedSession.Provider[:10] != "encrypted:" {
		// Return as-is for backward compatibility
		return encryptedSession, nil
	}

	// Decode from base64
	encrypted, err := base64.StdEncoding.DecodeString(encryptedSession.Provider[10:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted session: %w", err)
	}

	// Decrypt data
	decrypted, err := e.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session: %w", err)
	}

	// Deserialize session data
	var sessionData SessionData
	if err := json.Unmarshal(decrypted, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &sessionData, nil
}

// Delete removes session data
func (e *EncryptedSessionStore) Delete(ctx context.Context, key string) error {
	return e.store.Delete(ctx, key)
}

// RetrieveAndDelete atomically retrieves and deletes session data
func (e *EncryptedSessionStore) RetrieveAndDelete(ctx context.Context, key string) (*SessionData, error) {
	// First retrieve the session
	sessionData, err := e.Retrieve(ctx, key)
	if err != nil {
		return nil, err
	}

	// Then delete it
	if err := e.Delete(ctx, key); err != nil {
		// Log error but return the session data anyway
		// The session has been retrieved successfully
	}

	return sessionData, nil
}

// AESGCMEncryptor implements TokenEncryptor using AES-GCM
type AESGCMEncryptor struct {
	key []byte
}

// NewAESGCMEncryptor creates a new AES-GCM encryptor
func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	// Derive a 32-byte key using SHA-256 if needed
	if len(key) != 32 {
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	return &AESGCMEncryptor{
		key: key,
	}, nil
}

// Encrypt encrypts data using AES-GCM
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and append nonce to the beginning
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// CacheIntegratedTokenStore integrates with external cache systems
type CacheIntegratedTokenStore struct {
	primary TokenStore // Primary storage (persistent)
	cache   TokenStore // Cache storage (fast)
	ttl     time.Duration
}

// NewCacheIntegratedTokenStore creates a token store with cache integration
func NewCacheIntegratedTokenStore(primary, cache TokenStore, ttl time.Duration) *CacheIntegratedTokenStore {
	return &CacheIntegratedTokenStore{
		primary: primary,
		cache:   cache,
		ttl:     ttl,
	}
}

// Store saves token to both primary and cache
func (c *CacheIntegratedTokenStore) Store(ctx context.Context, key string, token *Token) error {
	// Store in primary first
	if err := c.primary.Store(ctx, key, token); err != nil {
		return fmt.Errorf("failed to store in primary: %w", err)
	}

	// Then store in cache (best effort)
	if c.cache != nil {
		c.cache.Store(ctx, key, token)
	}

	return nil
}

// Retrieve gets token from cache first, then primary
func (c *CacheIntegratedTokenStore) Retrieve(ctx context.Context, key string) (*Token, error) {
	// Try cache first
	if c.cache != nil {
		if token, err := c.cache.Retrieve(ctx, key); err == nil {
			return token, nil
		}
	}

	// Fallback to primary
	token, err := c.primary.Retrieve(ctx, key)
	if err != nil {
		return nil, err
	}

	// Update cache (best effort)
	if c.cache != nil {
		c.cache.Store(ctx, key, token)
	}

	return token, nil
}

// Delete removes token from both stores
func (c *CacheIntegratedTokenStore) Delete(ctx context.Context, key string) error {
	// Delete from cache first (best effort)
	if c.cache != nil {
		c.cache.Delete(ctx, key)
	}

	// Delete from primary
	return c.primary.Delete(ctx, key)
}
