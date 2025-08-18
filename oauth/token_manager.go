package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TokenManager provides advanced token lifecycle management
type TokenManager interface {
	// Basic operations
	CacheToken(ctx context.Context, userID, provider string, token *Token) error
	GetCachedToken(ctx context.Context, userID, provider string) (*Token, error)
	DeleteToken(ctx context.Context, userID, provider string) error
	
	// Advanced operations
	RefreshIfNeeded(ctx context.Context, userID, provider string) (*Token, error)
	RevokeToken(ctx context.Context, userID, provider string) error
	GetAllUserTokens(ctx context.Context, userID string) (map[string]*Token, error)
	
	// Bulk operations
	RefreshExpiredTokens(ctx context.Context) error
	CleanupExpiredTokens(ctx context.Context) error
	
	// Statistics
	GetTokenStats() *TokenStats
}

// TokenStats provides statistics about cached tokens
type TokenStats struct {
	TotalTokens      int
	ActiveTokens     int
	ExpiredTokens    int
	RefreshableTokens int
	ProviderCounts   map[string]int
	LastCleanup      time.Time
	LastRefresh      time.Time
}

// AdvancedTokenManager implements TokenManager with advanced features
type AdvancedTokenManager struct {
	store           TokenStore
	providerService *MultiProviderService
	encryptor       TokenEncryptor
	mu              sync.RWMutex
	
	// Configuration
	autoRefresh        bool
	refreshThreshold   time.Duration
	cleanupInterval    time.Duration
	maxTokensPerUser   int
	
	// Internal state
	tokenMetadata map[string]*TokenMetadata
	stats         *TokenStats
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// TokenMetadata stores additional information about a token
type TokenMetadata struct {
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"`
	CachedAt     time.Time `json:"cached_at"`
	LastAccessed time.Time `json:"last_accessed"`
	AccessCount  int       `json:"access_count"`
	RefreshCount int       `json:"refresh_count"`
	LastRefresh  time.Time `json:"last_refresh,omitempty"`
}

// TokenEncryptor interface for token encryption
type TokenEncryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

// TokenManagerConfig configures the advanced token manager
type TokenManagerConfig struct {
	Store              TokenStore
	ProviderService    *MultiProviderService
	Encryptor          TokenEncryptor
	AutoRefresh        bool
	RefreshThreshold   time.Duration
	CleanupInterval    time.Duration
	MaxTokensPerUser   int
}

// NewAdvancedTokenManager creates a new advanced token manager
func NewAdvancedTokenManager(config TokenManagerConfig) *AdvancedTokenManager {
	if config.RefreshThreshold == 0 {
		config.RefreshThreshold = 5 * time.Minute // Refresh tokens 5 minutes before expiry
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour // Cleanup every hour
	}
	if config.MaxTokensPerUser == 0 {
		config.MaxTokensPerUser = 10 // Maximum 10 providers per user
	}
	
	tm := &AdvancedTokenManager{
		store:            config.Store,
		providerService:  config.ProviderService,
		encryptor:        config.Encryptor,
		autoRefresh:      config.AutoRefresh,
		refreshThreshold: config.RefreshThreshold,
		cleanupInterval:  config.CleanupInterval,
		maxTokensPerUser: config.MaxTokensPerUser,
		tokenMetadata:    make(map[string]*TokenMetadata),
		stats: &TokenStats{
			ProviderCounts: make(map[string]int),
			LastCleanup:    time.Now(),
		},
		stopCh: make(chan struct{}),
	}
	
	// Start background tasks if auto-refresh is enabled
	if config.AutoRefresh {
		tm.startBackgroundTasks()
	}
	
	return tm
}

// CacheToken stores a token with encryption if configured
func (tm *AdvancedTokenManager) CacheToken(ctx context.Context, userID, provider string, token *Token) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	// Check user token limit
	userTokens := 0
	for key := range tm.tokenMetadata {
		if md := tm.tokenMetadata[key]; md != nil && md.UserID == userID {
			userTokens++
		}
	}
	if userTokens >= tm.maxTokensPerUser {
		return fmt.Errorf("user %s has reached maximum token limit (%d)", userID, tm.maxTokensPerUser)
	}
	
	key := tm.tokenKey(userID, provider)
	
	// Encrypt token if encryptor is configured
	var dataToStore []byte
	tokenData, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	
	if tm.encryptor != nil {
		dataToStore, err = tm.encryptor.Encrypt(tokenData)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %w", err)
		}
	} else {
		dataToStore = tokenData
	}
	
	// Create wrapper for encrypted storage
	wrapper := &encryptedTokenWrapper{
		Data:      dataToStore,
		Encrypted: tm.encryptor != nil,
	}
	
	// Store the token
	if err := tm.store.Store(ctx, key, &Token{
		AccessToken: string(wrapper.Data), // Store as string for compatibility
		TokenType:   fmt.Sprintf("encrypted:%v", wrapper.Encrypted),
	}); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}
	
	// Update metadata
	tm.tokenMetadata[key] = &TokenMetadata{
		UserID:       userID,
		Provider:     provider,
		CachedAt:     time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  0,
		RefreshCount: 0,
	}
	
	// Update stats
	tm.updateStats()
	
	return nil
}

// GetCachedToken retrieves and decrypts a cached token
func (tm *AdvancedTokenManager) GetCachedToken(ctx context.Context, userID, provider string) (*Token, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	key := tm.tokenKey(userID, provider)
	
	// Retrieve the token
	storedToken, err := tm.store.Retrieve(ctx, key)
	if err != nil {
		return nil, err
	}
	
	// Decrypt if necessary
	var token Token
	if storedToken.TokenType == "encrypted:true" && tm.encryptor != nil {
		decrypted, err := tm.encryptor.Decrypt([]byte(storedToken.AccessToken))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}
		if err := json.Unmarshal(decrypted, &token); err != nil {
			return nil, fmt.Errorf("failed to unmarshal token: %w", err)
		}
	} else {
		// Try to unmarshal directly (for backward compatibility)
		if err := json.Unmarshal([]byte(storedToken.AccessToken), &token); err != nil {
			// If unmarshal fails, return the token as-is (backward compatibility)
			token = *storedToken
		}
	}
	
	// Update metadata
	if metadata, exists := tm.tokenMetadata[key]; exists {
		metadata.LastAccessed = time.Now()
		metadata.AccessCount++
	}
	
	return &token, nil
}

// DeleteToken removes a token from cache
func (tm *AdvancedTokenManager) DeleteToken(ctx context.Context, userID, provider string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	key := tm.tokenKey(userID, provider)
	
	// Delete from store
	if err := tm.store.Delete(ctx, key); err != nil {
		return err
	}
	
	// Delete metadata
	delete(tm.tokenMetadata, key)
	
	// Update stats
	tm.updateStats()
	
	return nil
}

// RefreshIfNeeded checks if a token needs refresh and refreshes it
func (tm *AdvancedTokenManager) RefreshIfNeeded(ctx context.Context, userID, provider string) (*Token, error) {
	// Get the cached token
	token, err := tm.GetCachedToken(ctx, userID, provider)
	if err != nil {
		return nil, err
	}
	
	// Check if token needs refresh
	if !tm.needsRefresh(token) {
		return token, nil
	}
	
	// Get the provider
	if tm.providerService == nil {
		return nil, fmt.Errorf("provider service not configured for automatic refresh")
	}
	
	prov, err := tm.providerService.GetProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	
	if !prov.SupportsRefresh() {
		return token, nil // Return existing token if refresh not supported
	}
	
	// Refresh the token
	newToken, err := prov.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	
	// Cache the new token
	if err := tm.CacheToken(ctx, userID, provider, newToken); err != nil {
		return nil, fmt.Errorf("failed to cache refreshed token: %w", err)
	}
	
	// Update metadata
	tm.mu.Lock()
	key := tm.tokenKey(userID, provider)
	if metadata, exists := tm.tokenMetadata[key]; exists {
		metadata.RefreshCount++
		metadata.LastRefresh = time.Now()
	}
	tm.mu.Unlock()
	
	return newToken, nil
}

// RevokeToken revokes a token with the provider and removes it from cache
func (tm *AdvancedTokenManager) RevokeToken(ctx context.Context, userID, provider string) error {
	// Get the token
	token, err := tm.GetCachedToken(ctx, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to get token for revocation: %w", err)
	}
	
	// Revoke with provider if service is configured
	if tm.providerService != nil {
		if err := tm.providerService.RevokeToken(ctx, provider, token.AccessToken); err != nil {
			// Log error but continue to remove from cache
			// Some providers may not support revocation
		}
	}
	
	// Remove from cache
	return tm.DeleteToken(ctx, userID, provider)
}

// GetAllUserTokens retrieves all tokens for a user
func (tm *AdvancedTokenManager) GetAllUserTokens(ctx context.Context, userID string) (map[string]*Token, error) {
	// Collect providers for the user first
	tm.mu.RLock()
	providers := []string{}
	for _, metadata := range tm.tokenMetadata {
		if metadata.UserID == userID {
			providers = append(providers, metadata.Provider)
		}
	}
	tm.mu.RUnlock()
	
	// Now get tokens without holding the lock
	tokens := make(map[string]*Token)
	for _, provider := range providers {
		token, err := tm.GetCachedToken(ctx, userID, provider)
		if err == nil {
			tokens[provider] = token
		}
	}
	
	return tokens, nil
}

// RefreshExpiredTokens refreshes all tokens that are about to expire
func (tm *AdvancedTokenManager) RefreshExpiredTokens(ctx context.Context) error {
	tm.mu.RLock()
	tokenList := make([]struct{ userID, provider string }, 0)
	for _, metadata := range tm.tokenMetadata {
		tokenList = append(tokenList, struct{ userID, provider string }{
			userID:   metadata.UserID,
			provider: metadata.Provider,
		})
	}
	tm.mu.RUnlock()
	
	var errors []error
	for _, item := range tokenList {
		token, err := tm.GetCachedToken(ctx, item.userID, item.provider)
		if err != nil {
			continue
		}
		
		if tm.needsRefresh(token) {
			if _, err := tm.RefreshIfNeeded(ctx, item.userID, item.provider); err != nil {
				errors = append(errors, fmt.Errorf("failed to refresh token for %s/%s: %w", 
					item.userID, item.provider, err))
			}
		}
	}
	
	tm.mu.Lock()
	tm.stats.LastRefresh = time.Now()
	tm.mu.Unlock()
	
	if len(errors) > 0 {
		return fmt.Errorf("refresh errors: %v", errors)
	}
	
	return nil
}

// CleanupExpiredTokens removes expired tokens from cache
func (tm *AdvancedTokenManager) CleanupExpiredTokens(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	toDelete := []string{}
	
	for key, metadata := range tm.tokenMetadata {
		token, err := tm.store.Retrieve(ctx, key)
		if err != nil {
			toDelete = append(toDelete, key)
			continue
		}
		
		// Check if token is expired and can't be refreshed
		if token.IsExpired() && token.RefreshToken == "" {
			toDelete = append(toDelete, key)
		}
		
		// Remove tokens not accessed in a long time (30 days)
		if time.Since(metadata.LastAccessed) > 30*24*time.Hour {
			toDelete = append(toDelete, key)
		}
	}
	
	// Delete expired tokens
	for _, key := range toDelete {
		tm.store.Delete(ctx, key)
		delete(tm.tokenMetadata, key)
	}
	
	tm.stats.LastCleanup = time.Now()
	tm.updateStats()
	
	return nil
}

// GetTokenStats returns current token statistics
func (tm *AdvancedTokenManager) GetTokenStats() *TokenStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	// Create a copy of stats
	statsCopy := &TokenStats{
		TotalTokens:       tm.stats.TotalTokens,
		ActiveTokens:      tm.stats.ActiveTokens,
		ExpiredTokens:     tm.stats.ExpiredTokens,
		RefreshableTokens: tm.stats.RefreshableTokens,
		LastCleanup:       tm.stats.LastCleanup,
		LastRefresh:       tm.stats.LastRefresh,
		ProviderCounts:    make(map[string]int),
	}
	
	for k, v := range tm.stats.ProviderCounts {
		statsCopy.ProviderCounts[k] = v
	}
	
	return statsCopy
}

// Stop stops background tasks
func (tm *AdvancedTokenManager) Stop() {
	close(tm.stopCh)
	tm.wg.Wait()
}

// Helper methods

func (tm *AdvancedTokenManager) tokenKey(userID, provider string) string {
	return fmt.Sprintf("token:%s:%s", userID, provider)
}

func (tm *AdvancedTokenManager) needsRefresh(token *Token) bool {
	if token.RefreshToken == "" {
		return false // Can't refresh without refresh token
	}
	
	if token.ExpiresAt.IsZero() {
		return false // No expiration set
	}
	
	// Refresh if token expires within threshold
	return time.Until(token.ExpiresAt) < tm.refreshThreshold
}

func (tm *AdvancedTokenManager) updateStats() {
	tm.stats.TotalTokens = len(tm.tokenMetadata)
	
	activeCount := 0
	expiredCount := 0
	refreshableCount := 0
	providerCounts := make(map[string]int)
	
	ctx := context.Background()
	for key, metadata := range tm.tokenMetadata {
		providerCounts[metadata.Provider]++
		
		if token, err := tm.store.Retrieve(ctx, key); err == nil {
			if !token.IsExpired() {
				activeCount++
			} else {
				expiredCount++
			}
			
			if token.RefreshToken != "" {
				refreshableCount++
			}
		}
	}
	
	tm.stats.ActiveTokens = activeCount
	tm.stats.ExpiredTokens = expiredCount
	tm.stats.RefreshableTokens = refreshableCount
	tm.stats.ProviderCounts = providerCounts
}

func (tm *AdvancedTokenManager) startBackgroundTasks() {
	// Token refresh task
	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				tm.RefreshExpiredTokens(ctx)
			case <-tm.stopCh:
				return
			}
		}
	}()
	
	// Cleanup task
	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		ticker := time.NewTicker(tm.cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				tm.CleanupExpiredTokens(ctx)
			case <-tm.stopCh:
				return
			}
		}
	}()
}

// encryptedTokenWrapper wraps encrypted token data
type encryptedTokenWrapper struct {
	Data      []byte `json:"data"`
	Encrypted bool   `json:"encrypted"`
}