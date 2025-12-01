package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GitHubProvider implements OAuth provider for GitHub
type GitHubProvider struct {
	config     ProviderConfig
	httpClient HTTPClient
}

// GitHubUser represents the user data returned by GitHub
type GitHubUser struct {
	ID              int64  `json:"id"`
	Login           string `json:"login"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	AvatarURL       string `json:"avatar_url"`
	Bio             string `json:"bio"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	Blog            string `json:"blog"`
	TwitterUsername string `json:"twitter_username"`
}

// NewGitHub creates a new GitHub OAuth provider
func NewGitHub(config ProviderConfig) *GitHubProvider {
	// Set default endpoints if not provided
	if config.AuthURL == "" {
		config.AuthURL = "https://github.com/login/oauth/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://github.com/login/oauth/access_token"
	}
	if config.UserInfoURL == "" {
		config.UserInfoURL = "https://api.github.com/user"
	}

	// Set default scopes if not provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"read:user", "user:email"}
	}

	return &GitHubProvider{
		config:     config,
		httpClient: http.DefaultClient,
	}
}

// SetHTTPClient sets a custom HTTP client
func (g *GitHubProvider) SetHTTPClient(client HTTPClient) {
	g.httpClient = client
}

// GetAuthURL returns the authorization URL with PKCE parameters if enabled
func (g *GitHubProvider) GetAuthURL(state string, pkce *PKCEChallenge) string {
	params := url.Values{
		"client_id":     {g.config.ClientID},
		"redirect_uri":  {g.config.RedirectURL},
		"scope":         {strings.Join(g.config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
	}

	// Add PKCE parameters if provided
	if pkce != nil {
		params.Set("code_challenge", pkce.Challenge)
		params.Set("code_challenge_method", pkce.ChallengeMethod)
	}

	return g.config.AuthURL + "?" + params.Encode()
}

// Exchange exchanges an authorization code for tokens
func (g *GitHubProvider) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
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
		return nil, &Error{
			Provider:    "github",
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &Error{
			Provider:    "github",
			Code:        tokenResp.Error,
			Description: tokenResp.ErrorDescription,
		}
	}

	// GitHub doesn't return refresh tokens or expiry by default
	return &Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: "", // GitHub doesn't provide refresh tokens
		ExpiresIn:    0,  // GitHub tokens don't expire
		Scope:        tokenResp.Scope,
	}, nil
}

// RefreshToken refreshes the access token
func (g *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	// GitHub doesn't support refresh tokens
	return nil, ErrNoRefreshToken
}

// GetUserInfo retrieves user information using the access token
func (g *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
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

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is not public, fetch it separately
	if user.Email == "" {
		email, err := g.getPrimaryEmail(ctx, accessToken)
		if err == nil {
			user.Email = email
		}
	}

	// Map GitHub user to generic UserInfo
	userInfo := &UserInfo{
		ID:            fmt.Sprintf("%d", user.ID),
		Email:         user.Email,
		Name:          user.Name,
		Picture:       user.AvatarURL,
		EmailVerified: user.Email != "", // GitHub verifies emails
		Provider:      "github",
		Raw:           make(map[string]interface{}),
	}

	// Add additional fields to raw data
	userInfo.Raw["login"] = user.Login
	userInfo.Raw["bio"] = user.Bio
	userInfo.Raw["company"] = user.Company
	userInfo.Raw["location"] = user.Location
	userInfo.Raw["blog"] = user.Blog
	userInfo.Raw["twitter_username"] = user.TwitterUsername

	return userInfo, nil
}

// RevokeToken revokes the access token
func (g *GitHubProvider) RevokeToken(ctx context.Context, token string) error {
	// GitHub requires using their API to revoke tokens
	revokeURL := fmt.Sprintf("https://api.github.com/applications/%s/token", g.config.ClientID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub uses Basic Auth for token revocation
	req.SetBasicAuth(g.config.ClientID, g.config.ClientSecret)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// The token to revoke goes in the Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token: status code %d", resp.StatusCode)
	}

	return nil
}

// ValidateConfig validates the provider configuration
func (g *GitHubProvider) ValidateConfig() error {
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
func (g *GitHubProvider) Name() string {
	return "github"
}

// SupportsRefresh indicates if the provider supports token refresh
func (g *GitHubProvider) SupportsRefresh() bool {
	return false
}

// SupportsPKCE indicates if the provider supports PKCE
func (g *GitHubProvider) SupportsPKCE() bool {
	return true // GitHub supports PKCE
}

// getPrimaryEmail fetches the primary verified email from GitHub
func (g *GitHubProvider) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get emails: status code %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find the primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// Fallback to first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found")
}
