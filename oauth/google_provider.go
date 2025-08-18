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

// GoogleProvider implements OAuth provider for Google
type GoogleProvider struct {
	config     ProviderConfig
	httpClient HTTPClient
}

// GoogleUser represents the user data returned by Google
type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	HD            string `json:"hd"` // Hosted domain for G Suite accounts
}

// NewGoogle creates a new Google OAuth provider
func NewGoogle(config ProviderConfig) *GoogleProvider {
	// Set default endpoints if not provided
	if config.AuthURL == "" {
		config.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}

	// Set default scopes if not provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}

	return &GoogleProvider{
		config:     config,
		httpClient: http.DefaultClient,
	}
}

// SetHTTPClient sets a custom HTTP client
func (g *GoogleProvider) SetHTTPClient(client HTTPClient) {
	g.httpClient = client
}

// GetAuthURL returns the authorization URL with PKCE parameters if enabled
func (g *GoogleProvider) GetAuthURL(state string, pkce *PKCEChallenge) string {
	params := url.Values{
		"client_id":     {g.config.ClientID},
		"redirect_uri":  {g.config.RedirectURL},
		"scope":         {strings.Join(g.config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
		"access_type":   {"offline"}, // Request refresh token
		"prompt":        {"consent"},  // Force consent to get refresh token
	}

	// Add PKCE parameters if provided
	if pkce != nil {
		params.Set("code_challenge", pkce.Challenge)
		params.Set("code_challenge_method", pkce.ChallengeMethod)
	}

	return g.config.AuthURL + "?" + params.Encode()
}

// Exchange exchanges an authorization code for tokens
func (g *GoogleProvider) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
	data := url.Values{
		"client_id":     {g.config.ClientID},
		"client_secret": {g.config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {g.config.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	// Add PKCE verifier if provided
	if pkce != nil {
		data.Set("code_verifier", pkce.Verifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
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
			Provider:    "google",
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
		Error        string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &OAuthError{
			Provider:    "google",
			Code:        tokenResp.Error,
			Description: tokenResp.ErrorDescription,
		}
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
func (g *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{
		"client_id":     {g.config.ClientID},
		"client_secret": {g.config.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
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
			Provider:    "google",
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &OAuthError{
			Provider:    "google",
			Code:        tokenResp.Error,
			Description: tokenResp.ErrorDescription,
		}
	}

	// Calculate expiry time
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: refreshToken, // Keep the original refresh token
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		IDToken:      tokenResp.IDToken,
		Scope:        tokenResp.Scope,
	}, nil
}

// GetUserInfo retrieves user information using the access token
func (g *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", g.config.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Map Google user to generic UserInfo
	userInfo := &UserInfo{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.VerifiedEmail,
		Name:          user.Name,
		FirstName:     user.GivenName,
		LastName:      user.FamilyName,
		Picture:       user.Picture,
		Locale:        user.Locale,
		Provider:      "google",
		Raw:           make(map[string]interface{}),
	}

	// Add additional fields to raw data
	userInfo.Raw["hd"] = user.HD // Hosted domain for G Suite accounts

	return userInfo, nil
}

// RevokeToken revokes the access token
func (g *GoogleProvider) RevokeToken(ctx context.Context, token string) error {
	// Google's revoke endpoint
	revokeURL := "https://oauth2.googleapis.com/revoke"
	
	data := url.Values{
		"token": {token},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token: status code %d", resp.StatusCode)
	}

	return nil
}

// ValidateConfig validates the provider configuration
func (g *GoogleProvider) ValidateConfig() error {
	if g.config.ClientID == "" {
		return fmt.Errorf("missing client ID")
	}
	if g.config.ClientSecret == "" {
		return fmt.Errorf("missing client secret")
	}
	if g.config.RedirectURL == "" {
		return fmt.Errorf("missing redirect URL")
	}
	return nil
}

// Name returns the provider name
func (g *GoogleProvider) Name() string {
	return "google"
}

// SupportsRefresh indicates if the provider supports token refresh
func (g *GoogleProvider) SupportsRefresh() bool {
	return true
}

// SupportsPKCE indicates if the provider supports PKCE
func (g *GoogleProvider) SupportsPKCE() bool {
	return true
}

// ParseIDToken parses and validates Google ID Token (JWT)
// This is a basic implementation - for production use, consider using a proper JWT library
func (g *GoogleProvider) ParseIDToken(idToken string) (map[string]interface{}, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid ID token format")
	}

	// Decode the payload (second part)
	payload := parts[1]
	
	// Add padding if needed for base64 decoding
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	decoded, err := base64URLDecode(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ID token payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	return claims, nil
}

