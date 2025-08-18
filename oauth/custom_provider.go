package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CustomProvider implements a generic OAuth 2.0 provider
type CustomProvider struct {
	config     ProviderConfig
	httpClient HTTPClient
	name       string
}

// NewCustom creates a new custom OAuth provider
func NewCustom(config ProviderConfig) (*CustomProvider, error) {
	// Validate required fields
	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if config.RedirectURL == "" {
		return nil, fmt.Errorf("redirect URL is required")
	}
	if config.AuthURL == "" {
		return nil, fmt.Errorf("auth URL is required for custom provider")
	}
	if config.TokenURL == "" {
		return nil, fmt.Errorf("token URL is required for custom provider")
	}
	
	// Set default scopes if not provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}
	
	provider := &CustomProvider{
		config:     config,
		httpClient: http.DefaultClient,
		name:       "custom",
	}
	
	if config.HTTPClient != nil {
		provider.httpClient = config.HTTPClient
	}
	
	if config.Type != "" {
		provider.name = config.Type
	}
	
	return provider, nil
}

// SetHTTPClient sets a custom HTTP client
func (c *CustomProvider) SetHTTPClient(client HTTPClient) {
	c.httpClient = client
}

// GetAuthURL returns the authorization URL with optional PKCE parameters
func (c *CustomProvider) GetAuthURL(state string, pkce *PKCEChallenge) string {
	params := url.Values{
		"client_id":     {c.config.ClientID},
		"redirect_uri":  {c.config.RedirectURL},
		"response_type": {"code"},
		"state":         {state},
	}
	
	// Add scopes
	if len(c.config.Scopes) > 0 {
		params.Set("scope", strings.Join(c.config.Scopes, " "))
	}
	
	// Add PKCE parameters if provided
	if pkce != nil {
		params.Set("code_challenge", pkce.Challenge)
		params.Set("code_challenge_method", pkce.ChallengeMethod)
	}
	
	return c.config.AuthURL + "?" + params.Encode()
}

// Exchange exchanges an authorization code for tokens
func (c *CustomProvider) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.config.RedirectURL},
		"client_id":    {c.config.ClientID},
	}
	
	// Add client secret if configured
	if c.config.ClientSecret != "" {
		data.Set("client_secret", c.config.ClientSecret)
	}
	
	// Add PKCE verifier if provided
	if pkce != nil {
		data.Set("code_verifier", pkce.Verifier)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil, &OAuthError{
			Provider:    c.name,
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}
	
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		Scope        string `json:"scope"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Calculate expiry time
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	
	return &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		IDToken:      tokenResp.IDToken,
		Scope:        tokenResp.Scope,
	}, nil
}

// RefreshToken refreshes the access token using a refresh token
func (c *CustomProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.config.ClientID},
	}
	
	// Add client secret if configured
	if c.config.ClientSecret != "" {
		data.Set("client_secret", c.config.ClientSecret)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil, &OAuthError{
			Provider:    c.name,
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}
	
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		Scope        string `json:"scope"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Calculate expiry time
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	
	// Keep the original refresh token if not provided in response
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = refreshToken
	}
	
	return &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		IDToken:      tokenResp.IDToken,
		Scope:        tokenResp.Scope,
	}, nil
}

// GetUserInfo retrieves user information from the userinfo endpoint
func (c *CustomProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if c.config.UserInfoURL == "" {
		return nil, fmt.Errorf("userinfo URL not configured for custom provider")
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status code %d", resp.StatusCode)
	}
	
	var rawData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	
	userInfo := &UserInfo{
		Provider: c.name,
		Raw:      rawData,
	}
	
	// Try to extract standard fields
	if id, ok := getString(rawData, "sub", "id", "user_id"); ok {
		userInfo.ID = id
	}
	
	if email, ok := getString(rawData, "email", "mail", "e-mail"); ok {
		userInfo.Email = email
	}
	
	if emailVerified, ok := getBool(rawData, "email_verified", "verified_email"); ok {
		userInfo.EmailVerified = emailVerified
	}
	
	if name, ok := getString(rawData, "name", "display_name", "full_name"); ok {
		userInfo.Name = name
	}
	
	if firstName, ok := getString(rawData, "given_name", "first_name", "firstname"); ok {
		userInfo.FirstName = firstName
	}
	
	if lastName, ok := getString(rawData, "family_name", "last_name", "lastname"); ok {
		userInfo.LastName = lastName
	}
	
	if picture, ok := getString(rawData, "picture", "avatar_url", "profile_image_url"); ok {
		userInfo.Picture = picture
	}
	
	if locale, ok := getString(rawData, "locale", "lang", "language"); ok {
		userInfo.Locale = locale
	}
	
	return userInfo, nil
}

// RevokeToken revokes the access token
func (c *CustomProvider) RevokeToken(ctx context.Context, token string) error {
	if c.config.RevokeURL == "" {
		// If no revoke URL is configured, we can't revoke the token
		// This is not an error for providers that don't support revocation
		return nil
	}
	
	data := url.Values{
		"token":     {token},
		"client_id": {c.config.ClientID},
	}
	
	// Add client secret if configured
	if c.config.ClientSecret != "" {
		data.Set("client_secret", c.config.ClientSecret)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.RevokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to revoke token: status code %d", resp.StatusCode)
	}
	
	return nil
}

// ValidateConfig validates the provider configuration
func (c *CustomProvider) ValidateConfig() error {
	if c.config.ClientID == "" {
		return fmt.Errorf("missing client ID")
	}
	if c.config.RedirectURL == "" {
		return fmt.Errorf("missing redirect URL")
	}
	if c.config.AuthURL == "" {
		return fmt.Errorf("missing auth URL")
	}
	if c.config.TokenURL == "" {
		return fmt.Errorf("missing token URL")
	}
	return nil
}

// Name returns the provider name
func (c *CustomProvider) Name() string {
	return c.name
}

// SupportsRefresh indicates if the provider supports token refresh
func (c *CustomProvider) SupportsRefresh() bool {
	// Assume it supports refresh if we have a refresh token endpoint
	return true
}

// SupportsPKCE indicates if the provider supports PKCE
func (c *CustomProvider) SupportsPKCE() bool {
	// Most modern OAuth 2.0 providers support PKCE
	return true
}

// Helper functions to extract values from raw data

func getString(data map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok {
				return str, true
			}
		}
	}
	return "", false
}

func getBool(data map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			switch v := val.(type) {
			case bool:
				return v, true
			case string:
				return v == "true" || v == "1", true
			}
		}
	}
	return false, false
}