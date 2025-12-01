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

// TwitterProvider implements OAuth provider for Twitter (X)
type TwitterProvider struct {
	config     ProviderConfig
	httpClient HTTPClient
}

// TwitterUser represents the user data returned by Twitter API v2
type TwitterUser struct {
	ID              string `json:"id"`
	Username        string `json:"username"`
	Name            string `json:"name"`
	ProfileImageURL string `json:"profile_image_url"`
	Description     string `json:"description"`
	Location        string `json:"location"`
	URL             string `json:"url"`
	Verified        bool   `json:"verified"`
	Protected       bool   `json:"protected"`
	PublicMetrics   struct {
		FollowersCount int `json:"followers_count"`
		FollowingCount int `json:"following_count"`
		TweetCount     int `json:"tweet_count"`
		ListedCount    int `json:"listed_count"`
	} `json:"public_metrics"`
}

// TwitterUserResponse represents the response from Twitter user endpoint
type TwitterUserResponse struct {
	Data   TwitterUser `json:"data"`
	Errors []struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
		Type   string `json:"type"`
	} `json:"errors,omitempty"`
}

// NewTwitter creates a new Twitter OAuth provider
func NewTwitter(config ProviderConfig) *TwitterProvider {
	// Set default endpoints if not provided
	if config.AuthURL == "" {
		config.AuthURL = "https://twitter.com/i/oauth2/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://api.twitter.com/2/oauth2/token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://api.twitter.com/2/users/me"
	}
	if config.APIVersion == "" {
		config.APIVersion = "2" // Default to API v2
	}

	// Set default scopes if not provided (Twitter OAuth 2.0 scopes)
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"tweet.read", "users.read"}
	}

	return &TwitterProvider{
		config:     config,
		httpClient: http.DefaultClient,
	}
}

// SetHTTPClient sets a custom HTTP client
func (t *TwitterProvider) SetHTTPClient(client HTTPClient) {
	t.httpClient = client
}

// GetAuthURL returns the authorization URL
// Note: Twitter OAuth 2.0 with PKCE doesn't support the traditional PKCE interface
// but uses PKCE internally for public clients
func (t *TwitterProvider) GetAuthURL(state string, pkce *PKCEChallenge) string {
	params := url.Values{
		"client_id":             {t.config.ClientID},
		"redirect_uri":          {t.config.RedirectURL},
		"scope":                 {strings.Join(t.config.Scopes, " ")},
		"state":                 {state},
		"response_type":         {"code"},
		"code_challenge_method": {"S256"}, // Twitter requires PKCE for public clients
	}

	// Twitter OAuth 2.0 always uses PKCE for public clients
	if pkce != nil {
		params.Set("code_challenge", pkce.Challenge)
	}

	return t.config.AuthURL + "?" + params.Encode()
}

// Exchange exchanges an authorization code for tokens
func (t *TwitterProvider) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
	data := url.Values{
		"client_id":    {t.config.ClientID},
		"code":         {code},
		"redirect_uri": {t.config.RedirectURL},
		"grant_type":   {"authorization_code"},
	}

	// Twitter OAuth 2.0 requires PKCE verifier for public clients
	if pkce != nil {
		data.Set("code_verifier", pkce.Verifier)
	}

	// For confidential clients, include client secret
	if t.config.ClientSecret != "" {
		data.Set("client_secret", t.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
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
		return nil, &Error{
			Provider:    "twitter",
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &Error{
			Provider:    "twitter",
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
		Scope:        tokenResp.Scope,
	}, nil
}

// RefreshToken refreshes the access token using a refresh token
func (t *TwitterProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{
		"client_id":     {t.config.ClientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	// For confidential clients, include client secret
	if t.config.ClientSecret != "" {
		data.Set("client_secret", t.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
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
		return nil, &Error{
			Provider:    "twitter",
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		ExpiresIn        int    `json:"expires_in"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &Error{
			Provider:    "twitter",
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
		Scope:        tokenResp.Scope,
	}, nil
}

// GetUserInfo retrieves user information using the access token
func (t *TwitterProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Build user info URL with expanded fields
	userInfoURL := t.config.UserInfoURL + "?user.fields=id,username,name,profile_image_url,description,location,url,verified,protected,public_metrics"

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var userResp TwitterUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Check for API errors
	if len(userResp.Errors) > 0 {
		return nil, fmt.Errorf("twitter API error: %s", userResp.Errors[0].Detail)
	}

	user := userResp.Data

	// Map Twitter user to generic UserInfo
	userInfo := &UserInfo{
		ID:            user.ID,
		Email:         "", // Twitter API v2 doesn't provide email in user endpoint
		EmailVerified: false,
		Name:          user.Name,
		Picture:       user.ProfileImageURL,
		Provider:      "twitter",
		Raw:           make(map[string]interface{}),
	}

	// Add Twitter-specific fields to raw data
	userInfo.Raw["username"] = user.Username
	userInfo.Raw["description"] = user.Description
	userInfo.Raw["location"] = user.Location
	userInfo.Raw["url"] = user.URL
	userInfo.Raw["verified"] = user.Verified
	userInfo.Raw["protected"] = user.Protected
	userInfo.Raw["followers_count"] = user.PublicMetrics.FollowersCount
	userInfo.Raw["following_count"] = user.PublicMetrics.FollowingCount
	userInfo.Raw["tweet_count"] = user.PublicMetrics.TweetCount
	userInfo.Raw["listed_count"] = user.PublicMetrics.ListedCount

	return userInfo, nil
}

// RevokeToken revokes the access token
func (t *TwitterProvider) RevokeToken(ctx context.Context, token string) error {
	// Twitter's revoke endpoint
	revokeURL := "https://api.twitter.com/2/oauth2/revoke"

	data := url.Values{
		"client_id": {t.config.ClientID},
		"token":     {token},
	}

	// For confidential clients, include client secret
	if t.config.ClientSecret != "" {
		data.Set("client_secret", t.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
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
func (t *TwitterProvider) ValidateConfig() error {
	if t.config.ClientID == "" {
		return fmt.Errorf("missing client ID")
	}
	if t.config.RedirectURL == "" {
		return fmt.Errorf("missing redirect URL")
	}
	// Note: Client secret is optional for public clients using PKCE
	return nil
}

// Name returns the provider name
func (t *TwitterProvider) Name() string {
	return "twitter"
}

// SupportsRefresh indicates if the provider supports token refresh
func (t *TwitterProvider) SupportsRefresh() bool {
	return true
}

// SupportsPKCE indicates if the provider supports PKCE
func (t *TwitterProvider) SupportsPKCE() bool {
	return true // Twitter OAuth 2.0 requires PKCE for public clients
}
