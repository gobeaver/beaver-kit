package oauth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
)

func TestAdvancedTokenManager_CacheAndRetrieve(t *testing.T) {
	// Create token manager
	config := oauth.TokenManagerConfig{
		Store:            oauth.NewMemoryTokenStore(1 * time.Hour),
		AutoRefresh:      false,
		RefreshThreshold: 5 * time.Minute,
		MaxTokensPerUser: 5,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()
	userID := "user123"
	provider := "google"

	// Create a test token
	token := &oauth.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	// Cache the token
	if err := tm.CacheToken(ctx, userID, provider, token); err != nil {
		t.Fatalf("Failed to cache token: %v", err)
	}

	// Retrieve the token
	retrieved, err := tm.GetCachedToken(ctx, userID, provider)
	if err != nil {
		t.Fatalf("Failed to retrieve token: %v", err)
	}

	// Verify token fields
	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: got %s, want %s", retrieved.AccessToken, token.AccessToken)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %s, want %s", retrieved.RefreshToken, token.RefreshToken)
	}
}

func TestAdvancedTokenManager_Encryption(t *testing.T) {
	// Create encryption key
	encryptionKey := []byte("test-encryption-key-32-bytes-long!")

	// Create encryptor
	encryptor, err := oauth.NewAESGCMEncryptor(encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Create token manager with encryption
	config := oauth.TokenManagerConfig{
		Store:            oauth.NewMemoryTokenStore(1 * time.Hour),
		Encryptor:        encryptor,
		AutoRefresh:      false,
		RefreshThreshold: 5 * time.Minute,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()
	userID := "user456"
	provider := "github"

	// Create a test token with sensitive data
	token := &oauth.Token{
		AccessToken:  "super-secret-access-token",
		RefreshToken: "super-secret-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	// Cache the token (should be encrypted)
	if err := tm.CacheToken(ctx, userID, provider, token); err != nil {
		t.Fatalf("Failed to cache token: %v", err)
	}

	// Retrieve the token (should be decrypted)
	retrieved, err := tm.GetCachedToken(ctx, userID, provider)
	if err != nil {
		t.Fatalf("Failed to retrieve token: %v", err)
	}

	// Verify decrypted token
	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch after decryption: got %s, want %s",
			retrieved.AccessToken, token.AccessToken)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch after decryption: got %s, want %s",
			retrieved.RefreshToken, token.RefreshToken)
	}
}

func TestAdvancedTokenManager_UserTokenLimit(t *testing.T) {
	config := oauth.TokenManagerConfig{
		Store:            oauth.NewMemoryTokenStore(1 * time.Hour),
		AutoRefresh:      false,
		MaxTokensPerUser: 3,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()
	userID := "user789"

	// Cache tokens up to the limit
	for i := 0; i < 3; i++ {
		provider := fmt.Sprintf("provider%d", i)
		token := &oauth.Token{
			AccessToken: fmt.Sprintf("token-%d", i),
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}

		if err := tm.CacheToken(ctx, userID, provider, token); err != nil {
			t.Fatalf("Failed to cache token %d: %v", i, err)
		}
	}

	// Try to cache one more token (should fail)
	extraToken := &oauth.Token{
		AccessToken: "extra-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	err := tm.CacheToken(ctx, userID, "extra-provider", extraToken)
	if err == nil {
		t.Error("Expected error when exceeding user token limit")
	}
}

func TestAdvancedTokenManager_GetAllUserTokens(t *testing.T) {
	config := oauth.TokenManagerConfig{
		Store:       oauth.NewMemoryTokenStore(1 * time.Hour),
		AutoRefresh: false,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()
	userID := "user-multi"

	// Cache multiple tokens for the user
	providers := []string{"google", "github", "twitter"}
	for _, provider := range providers {
		token := &oauth.Token{
			AccessToken: fmt.Sprintf("%s-token", provider),
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}

		if err := tm.CacheToken(ctx, userID, provider, token); err != nil {
			t.Fatalf("Failed to cache token for %s: %v", provider, err)
		}
	}

	// Get all user tokens
	tokens, err := tm.GetAllUserTokens(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get all user tokens: %v", err)
	}

	// Verify we got all tokens
	if len(tokens) != len(providers) {
		t.Errorf("Expected %d tokens, got %d", len(providers), len(tokens))
	}

	// Verify each token
	for _, provider := range providers {
		token, exists := tokens[provider]
		if !exists {
			t.Errorf("Token for provider %s not found", provider)
		}
		if token.AccessToken != fmt.Sprintf("%s-token", provider) {
			t.Errorf("Wrong token for provider %s", provider)
		}
	}
}

func TestAdvancedTokenManager_DeleteToken(t *testing.T) {
	config := oauth.TokenManagerConfig{
		Store:       oauth.NewMemoryTokenStore(1 * time.Hour),
		AutoRefresh: false,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()
	userID := "user-delete"
	provider := "google"

	// Cache a token
	token := &oauth.Token{
		AccessToken: "token-to-delete",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	if err := tm.CacheToken(ctx, userID, provider, token); err != nil {
		t.Fatalf("Failed to cache token: %v", err)
	}

	// Delete the token
	if err := tm.DeleteToken(ctx, userID, provider); err != nil {
		t.Fatalf("Failed to delete token: %v", err)
	}

	// Try to retrieve the deleted token
	_, err := tm.GetCachedToken(ctx, userID, provider)
	if err == nil {
		t.Error("Expected error when retrieving deleted token")
	}
}

func TestAdvancedTokenManager_Stats(t *testing.T) {
	config := oauth.TokenManagerConfig{
		Store:       oauth.NewMemoryTokenStore(1 * time.Hour),
		AutoRefresh: false,
	}

	tm := oauth.NewAdvancedTokenManager(config)
	defer tm.Stop()

	ctx := context.Background()

	// Cache tokens for different users and providers
	testData := []struct {
		userID   string
		provider string
		expired  bool
	}{
		{"user1", "google", false},
		{"user1", "github", false},
		{"user2", "google", true},
		{"user2", "twitter", false},
	}

	for _, td := range testData {
		token := &oauth.Token{
			AccessToken:  fmt.Sprintf("%s-%s-token", td.userID, td.provider),
			RefreshToken: "refresh-token",
		}

		if td.expired {
			token.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired
		} else {
			token.ExpiresAt = time.Now().Add(1 * time.Hour) // Active
		}

		if err := tm.CacheToken(ctx, td.userID, td.provider, token); err != nil {
			t.Fatalf("Failed to cache token: %v", err)
		}
	}

	// Get stats
	stats := tm.GetTokenStats()

	// Verify stats
	if stats.TotalTokens != 4 {
		t.Errorf("Expected 4 total tokens, got %d", stats.TotalTokens)
	}

	if len(stats.ProviderCounts) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(stats.ProviderCounts))
	}

	if stats.ProviderCounts["google"] != 2 {
		t.Errorf("Expected 2 google tokens, got %d", stats.ProviderCounts["google"])
	}
}

func TestEncryptedTokenStore(t *testing.T) {
	// Create base store
	baseStore := oauth.NewMemoryTokenStore(1 * time.Hour)

	// Create encrypted store
	encryptionKey := []byte("test-encryption-key-for-tokens!")
	encryptedStore, err := oauth.NewEncryptedTokenStore(baseStore, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create encrypted store: %v", err)
	}

	ctx := context.Background()
	key := "test-key"

	// Store a token
	token := &oauth.Token{
		AccessToken:  "sensitive-access-token",
		RefreshToken: "sensitive-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := encryptedStore.Store(ctx, key, token); err != nil {
		t.Fatalf("Failed to store encrypted token: %v", err)
	}

	// Retrieve the token
	retrieved, err := encryptedStore.Retrieve(ctx, key)
	if err != nil {
		t.Fatalf("Failed to retrieve encrypted token: %v", err)
	}

	// Verify the token
	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: got %s, want %s", retrieved.AccessToken, token.AccessToken)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %s, want %s", retrieved.RefreshToken, token.RefreshToken)
	}

	// Verify that the base store contains encrypted data
	rawToken, err := baseStore.Retrieve(ctx, key)
	if err != nil {
		t.Fatalf("Failed to retrieve raw token: %v", err)
	}

	// The raw token should be encrypted (not the original values)
	if rawToken.AccessToken == token.AccessToken {
		t.Error("Token was not encrypted in base store")
	}
	if rawToken.TokenType != "encrypted" {
		t.Errorf("Expected token type 'encrypted', got %s", rawToken.TokenType)
	}
}

func TestAESGCMEncryptor(t *testing.T) {
	key := []byte("test-key-for-aes-gcm-encryption")
	encryptor, err := oauth.NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Test data
	plaintext := []byte("This is sensitive data that needs encryption")

	// Encrypt
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Verify ciphertext is different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Error("Ciphertext should be different from plaintext")
	}

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Verify decrypted data matches original
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data mismatch: got %s, want %s", string(decrypted), string(plaintext))
	}

	// Test that different encryptions produce different ciphertexts (due to nonce)
	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt second time: %v", err)
	}

	if string(ciphertext) == string(ciphertext2) {
		t.Error("Multiple encryptions should produce different ciphertexts due to random nonce")
	}
}

func TestEncryptedSessionStore(t *testing.T) {
	// Create base store
	baseStore := oauth.NewMemorySessionStore(5 * time.Minute)

	// Create encrypted store
	encryptionKey := []byte("test-encryption-key-for-sessions")
	encryptedStore, err := oauth.NewEncryptedSessionStore(baseStore, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create encrypted session store: %v", err)
	}

	ctx := context.Background()
	key := "session-key"

	// Create session data
	sessionData := &oauth.SessionData{
		State:    "test-state",
		Provider: "google",
		Metadata: map[string]interface{}{
			"user_id": "123",
			"origin":  "web",
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	// Store session
	if err := encryptedStore.Store(ctx, key, sessionData); err != nil {
		t.Fatalf("Failed to store encrypted session: %v", err)
	}

	// Retrieve session
	retrieved, err := encryptedStore.Retrieve(ctx, key)
	if err != nil {
		t.Fatalf("Failed to retrieve encrypted session: %v", err)
	}

	// Verify session data
	if retrieved.State != sessionData.State {
		t.Errorf("State mismatch: got %s, want %s", retrieved.State, sessionData.State)
	}
	if retrieved.Provider != sessionData.Provider {
		t.Errorf("Provider mismatch: got %s, want %s", retrieved.Provider, sessionData.Provider)
	}

	// Test RetrieveAndDelete
	retrieved2, err := encryptedStore.RetrieveAndDelete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to retrieve and delete: %v", err)
	}

	if retrieved2.State != sessionData.State {
		t.Error("RetrieveAndDelete returned wrong session")
	}

	// Verify session was deleted
	_, err = encryptedStore.Retrieve(ctx, key)
	if err == nil {
		t.Error("Session should have been deleted")
	}
}
