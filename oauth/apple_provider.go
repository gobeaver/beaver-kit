package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AppleProvider implements OAuth provider for Apple Sign-In
type AppleProvider struct {
	config       ProviderConfig
	httpClient   HTTPClient
	privateKey   *ecdsa.PrivateKey
	jwtValidator *AppleJWTValidator
}

// AppleUser represents the user data returned by Apple
type AppleUser struct {
	ID    string `json:"sub"`
	Email string `json:"email"`
	Name  struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
	EmailVerified string `json:"email_verified"` // Apple returns "true"/"false" as string
}

// NewApple creates a new Apple OAuth provider
func NewApple(config ProviderConfig) (*AppleProvider, error) {
	// Set default endpoints if not provided
	if config.AuthURL == "" {
		config.AuthURL = "https://appleid.apple.com/auth/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://appleid.apple.com/auth/token"
	}

	// Set default scopes if not provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"name", "email"}
	}

	provider := &AppleProvider{
		config:       config,
		httpClient:   http.DefaultClient,
		jwtValidator: NewAppleJWTValidator(config.ClientID, http.DefaultClient),
	}

	// Parse private key if provided
	if config.PrivateKey != "" {
		privateKey, err := parseApplePrivateKey(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Apple private key: %w", err)
		}
		provider.privateKey = privateKey
	}

	return provider, nil
}

// SetHTTPClient sets a custom HTTP client
func (a *AppleProvider) SetHTTPClient(client HTTPClient) {
	a.httpClient = client
	a.jwtValidator = NewAppleJWTValidator(a.config.ClientID, client)
}

// GetAuthURL returns the authorization URL with PKCE parameters if enabled
func (a *AppleProvider) GetAuthURL(state string, pkce *PKCEChallenge) string {
	params := url.Values{
		"client_id":     {a.config.ClientID},
		"redirect_uri":  {a.config.RedirectURL},
		"scope":         {strings.Join(a.config.Scopes, " ")},
		"state":         {state},
		"response_type": {"code"},
		"response_mode": {"form_post"}, // Apple recommends form_post
	}

	// Add PKCE parameters if provided
	if pkce != nil {
		params.Set("code_challenge", pkce.Challenge)
		params.Set("code_challenge_method", pkce.ChallengeMethod)
	}

	return a.config.AuthURL + "?" + params.Encode()
}

// Exchange exchanges an authorization code for tokens
func (a *AppleProvider) Exchange(ctx context.Context, code string, pkce *PKCEChallenge) (*Token, error) {
	// Generate client secret JWT
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{
		"client_id":     {a.config.ClientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {a.config.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	// Add PKCE verifier if provided
	if pkce != nil {
		data.Set("code_verifier", pkce.Verifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
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
			Provider:    "apple",
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
		Error        string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &OAuthError{
			Provider:    "apple",
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
	}, nil
}

// RefreshToken refreshes the access token using a refresh token
func (a *AppleProvider) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	// Generate client secret JWT
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	data := url.Values{
		"client_id":     {a.config.ClientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
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
			Provider:    "apple",
			Code:        errResp.Error,
			Description: errResp.ErrorDescription,
		}
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		Error        string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, &OAuthError{
			Provider:    "apple",
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
	}, nil
}

// GetUserInfo retrieves user information from the ID token (Apple doesn't provide a userinfo endpoint)
func (a *AppleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Apple doesn't provide a traditional userinfo endpoint
	// User information must be extracted from the ID token during the initial authentication
	// This method should be called with the ID token, not the access token
	return nil, fmt.Errorf("apple provider requires user info to be extracted from ID token during initial authentication")
}

// GetUserInfoFromIDToken extracts user information from Apple ID token
func (a *AppleProvider) GetUserInfoFromIDToken(idToken string) (*UserInfo, error) {
	return a.GetUserInfoFromIDTokenWithNonce(idToken, "")
}

// GetUserInfoFromIDTokenWithNonce extracts user information from Apple ID token with nonce verification
func (a *AppleProvider) GetUserInfoFromIDTokenWithNonce(idToken string, nonce string) (*UserInfo, error) {
	// Validate and parse ID token
	claims, err := a.jwtValidator.ValidateIDToken(context.Background(), idToken, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}

	userInfo := &UserInfo{
		Provider:      "apple",
		ID:            claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.IsEmailVerified(),
	}

	// Add raw claims for backward compatibility
	rawClaims := make(map[string]interface{})
	rawClaims["sub"] = claims.Subject
	rawClaims["iss"] = claims.Issuer
	rawClaims["aud"] = claims.Audience
	rawClaims["exp"] = claims.ExpirationTime
	rawClaims["iat"] = claims.IssuedAt
	
	if claims.Email != "" {
		rawClaims["email"] = claims.Email
	}
	if claims.EmailVerified != nil {
		rawClaims["email_verified"] = claims.EmailVerified
	}
	if claims.IsPrivateEmail != nil {
		rawClaims["is_private_email"] = claims.IsPrivateEmail
	}
	if claims.RealUserStatus > 0 {
		rawClaims["real_user_status"] = claims.RealUserStatus
	}
	
	// Apple sometimes includes name in extra claims
	if name, ok := claims.Extra["name"].(map[string]interface{}); ok {
		if firstName, ok := name["firstName"].(string); ok {
			userInfo.FirstName = firstName
		}
		if lastName, ok := name["lastName"].(string); ok {
			userInfo.LastName = lastName
		}
		if userInfo.FirstName != "" || userInfo.LastName != "" {
			userInfo.Name = strings.TrimSpace(userInfo.FirstName + " " + userInfo.LastName)
		}
	}
	
	userInfo.Raw = rawClaims
	
	return userInfo, nil
}

// RevokeToken revokes the access token
func (a *AppleProvider) RevokeToken(ctx context.Context, token string) error {
	// Generate client secret JWT
	clientSecret, err := a.generateClientSecret()
	if err != nil {
		return fmt.Errorf("failed to generate client secret: %w", err)
	}

	// Apple's revoke endpoint
	revokeURL := "https://appleid.apple.com/auth/revoke"
	
	data := url.Values{
		"client_id":     {a.config.ClientID},
		"client_secret": {clientSecret},
		"token":         {token},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
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
func (a *AppleProvider) ValidateConfig() error {
	if a.config.ClientID == "" {
		return fmt.Errorf("missing client ID")
	}
	if a.config.RedirectURL == "" {
		return fmt.Errorf("missing redirect URL")
	}
	if a.config.TeamID == "" {
		return fmt.Errorf("missing Apple Team ID")
	}
	if a.config.KeyID == "" {
		return fmt.Errorf("missing Apple Key ID")
	}
	if a.config.PrivateKey == "" {
		return fmt.Errorf("missing Apple private key")
	}
	return nil
}

// Name returns the provider name
func (a *AppleProvider) Name() string {
	return "apple"
}

// SupportsRefresh indicates if the provider supports token refresh
func (a *AppleProvider) SupportsRefresh() bool {
	return true
}

// SupportsPKCE indicates if the provider supports PKCE
func (a *AppleProvider) SupportsPKCE() bool {
	return true
}

// EnableTestMode enables test mode which skips JWT signature verification (TESTING ONLY)
func (a *AppleProvider) EnableTestMode() {
	a.jwtValidator.EnableTestMode()
}

// generateClientSecret creates a JWT client secret for Apple
func (a *AppleProvider) generateClientSecret() (string, error) {
	if a.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	// Create JWT header
	header := map[string]interface{}{
		"alg": "ES256",
		"typ": "JWT",
		"kid": a.config.KeyID,
	}

	// Create JWT claims
	now := time.Now()
	claims := map[string]interface{}{
		"iss": a.config.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(6 * 30 * 24 * time.Hour).Unix(), // Apple allows up to 6 months
		"aud": "https://appleid.apple.com",
		"sub": a.config.ClientID,
	}

	// Generate JWT
	token, err := a.generateJWT(header, claims)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return token, nil
}

// generateJWT creates a JWT token with ES256 signing
func (a *AppleProvider) generateJWT(header, claims map[string]interface{}) (string, error) {
	// Encode header
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)

	// Encode claims
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsBytes)

	// Create signing input
	signingInput := headerEncoded + "." + claimsEncoded

	// Sign with ECDSA
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, a.privateKey, hash[:])
	if err != nil {
		return "", err
	}

	// Convert signature to bytes (32 bytes each for r and s)
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	// Encode signature
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureEncoded, nil
}

// ParseIDToken parses and validates Apple ID Token (JWT)
func (a *AppleProvider) ParseIDToken(idToken string) (map[string]interface{}, error) {
	return a.ParseIDTokenWithNonce(idToken, "")
}

// ParseIDTokenWithNonce parses and validates Apple ID Token with nonce verification
func (a *AppleProvider) ParseIDTokenWithNonce(idToken string, nonce string) (map[string]interface{}, error) {
	claims, err := a.jwtValidator.ValidateIDToken(context.Background(), idToken, nonce)
	if err != nil {
		return nil, err
	}
	
	// Convert claims to map[string]interface{} for backward compatibility
	result := make(map[string]interface{})
	
	// Standard claims
	result["iss"] = claims.Issuer
	result["sub"] = claims.Subject
	result["aud"] = claims.Audience
	result["exp"] = claims.ExpirationTime
	result["iat"] = claims.IssuedAt
	
	if claims.AuthTime > 0 {
		result["auth_time"] = claims.AuthTime
	}
	if claims.Nonce != "" {
		result["nonce"] = claims.Nonce
	}
	
	// Apple-specific claims
	if claims.Email != "" {
		result["email"] = claims.Email
	}
	if claims.EmailVerified != nil {
		result["email_verified"] = claims.EmailVerified
	}
	if claims.IsPrivateEmail != nil {
		result["is_private_email"] = claims.IsPrivateEmail
	}
	if claims.RealUserStatus > 0 {
		result["real_user_status"] = claims.RealUserStatus
	}
	if claims.TransferSub != "" {
		result["transfer_sub"] = claims.TransferSub
	}
	if claims.AtHash != "" {
		result["at_hash"] = claims.AtHash
	}
	
	// Add extra claims
	for k, v := range claims.Extra {
		result[k] = v
	}
	
	return result, nil
}

// parseApplePrivateKey parses the Apple private key from PEM format
func parseApplePrivateKey(pemKey string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}

	return ecdsaKey, nil
}