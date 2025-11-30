package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	"github.com/gobeaver/beaver-kit/oauth"
)

// MockOAuthServer simulates an OAuth 2.0 provider for testing
type MockOAuthServer struct {
	server *httptest.Server
	mu     sync.RWMutex
	config MockServerConfig

	// State tracking
	authorizedCodes map[string]*AuthorizedCode
	issuedTokens    map[string]*IssuedToken
	revokedTokens   map[string]time.Time
	userInfo        map[string]*oauth.UserInfo

	// Behavior control
	failureScenarios map[string]bool
	latencies        map[string]time.Duration
	errorRates       map[string]float64
}

// MockServerConfig configures the mock OAuth server
type MockServerConfig struct {
	ProviderName    string
	ClientID        string
	ClientSecret    string
	SupportsPKCE    bool
	SupportsRefresh bool
	TokenExpiry     time.Duration
	RefreshExpiry   time.Duration
	RequireHTTPS    bool
}

// AuthorizedCode represents an authorized code
type AuthorizedCode struct {
	Code         string
	ClientID     string
	RedirectURI  string
	State        string
	PKCEVerifier string
	UserID       string
	Scopes       []string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// IssuedToken represents an issued token
type IssuedToken struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
	UserID       string
	Scopes       []string
	IssuedAt     time.Time
}

// NewMockOAuthServer creates a new mock OAuth server
func NewMockOAuthServer(config MockServerConfig) *MockOAuthServer {
	if config.TokenExpiry <= 0 {
		config.TokenExpiry = 1 * time.Hour
	}
	if config.RefreshExpiry <= 0 {
		config.RefreshExpiry = 30 * 24 * time.Hour
	}

	mock := &MockOAuthServer{
		config:           config,
		authorizedCodes:  make(map[string]*AuthorizedCode),
		issuedTokens:     make(map[string]*IssuedToken),
		revokedTokens:    make(map[string]time.Time),
		userInfo:         make(map[string]*oauth.UserInfo),
		failureScenarios: make(map[string]bool),
		latencies:        make(map[string]time.Duration),
		errorRates:       make(map[string]float64),
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", mock.handleAuthorize)
	mux.HandleFunc("/token", mock.handleToken)
	mux.HandleFunc("/userinfo", mock.handleUserInfo)
	mux.HandleFunc("/revoke", mock.handleRevoke)
	mux.HandleFunc("/.well-known/openid-configuration", mock.handleDiscovery)

	mock.server = httptest.NewServer(mux)

	return mock
}

// GetURL returns the mock server URL
func (m *MockOAuthServer) GetURL() string {
	return m.server.URL
}

// GetAuthURL returns the authorization endpoint URL
func (m *MockOAuthServer) GetAuthURL() string {
	return m.server.URL + "/authorize"
}

// GetTokenURL returns the token endpoint URL
func (m *MockOAuthServer) GetTokenURL() string {
	return m.server.URL + "/token"
}

// GetUserInfoURL returns the user info endpoint URL
func (m *MockOAuthServer) GetUserInfoURL() string {
	return m.server.URL + "/userinfo"
}

// GetRevokeURL returns the revoke endpoint URL
func (m *MockOAuthServer) GetRevokeURL() string {
	return m.server.URL + "/revoke"
}

// Close shuts down the mock server
func (m *MockOAuthServer) Close() {
	m.server.Close()
}

// SetUserInfo sets user information for a specific user ID
func (m *MockOAuthServer) SetUserInfo(userID string, info *oauth.UserInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userInfo[userID] = info
}

// SetFailureScenario enables a specific failure scenario
func (m *MockOAuthServer) SetFailureScenario(scenario string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureScenarios[scenario] = enabled
}

// SetLatency sets artificial latency for an endpoint
func (m *MockOAuthServer) SetLatency(endpoint string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencies[endpoint] = latency
}

// SetErrorRate sets the error rate for an endpoint (0.0 to 1.0)
func (m *MockOAuthServer) SetErrorRate(endpoint string, rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRates[endpoint] = rate
}

// IssueAuthorizationCode issues a new authorization code
func (m *MockOAuthServer) IssueAuthorizationCode(userID, state, redirectURI string, pkceVerifier string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	code := fmt.Sprintf("mock_code_%d", time.Now().UnixNano())
	m.authorizedCodes[code] = &AuthorizedCode{
		Code:         code,
		ClientID:     m.config.ClientID,
		RedirectURI:  redirectURI,
		State:        state,
		PKCEVerifier: pkceVerifier,
		UserID:       userID,
		Scopes:       []string{"openid", "email", "profile"},
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}

	return code
}

// HTTP Handlers

func (m *MockOAuthServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Apply latency if configured
	if latency := m.getLatency("authorize"); latency > 0 {
		time.Sleep(latency)
	}

	// Check for failure scenarios
	if m.shouldFail("authorize") {
		http.Error(w, "Authorization server error", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	responseType := r.URL.Query().Get("response_type")

	// Validate request
	if clientID != m.config.ClientID {
		http.Error(w, "Invalid client_id", http.StatusBadRequest)
		return
	}

	if responseType != "code" {
		http.Error(w, "Unsupported response_type", http.StatusBadRequest)
		return
	}

	// Generate authorization code
	code := m.IssueAuthorizationCode("test_user", state, redirectURI, "")

	// Redirect back with code
	http.Redirect(w, r, fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state), http.StatusFound)
}

func (m *MockOAuthServer) handleToken(w http.ResponseWriter, r *http.Request) {
	// Apply latency if configured
	if latency := m.getLatency("token"); latency > 0 {
		time.Sleep(latency)
	}

	// Check for failure scenarios
	if m.shouldFail("token") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "server_error",
			"error_description": "Token server error",
		})
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		m.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		m.handleRefreshTokenGrant(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "unsupported_grant_type",
		})
	}
}

func (m *MockOAuthServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	redirectURI := r.FormValue("redirect_uri")
	codeVerifier := r.FormValue("code_verifier")

	// Validate client credentials
	if clientID != m.config.ClientID || clientSecret != m.config.ClientSecret {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_client",
		})
		return
	}

	// Validate authorization code
	m.mu.Lock()
	authCode, exists := m.authorizedCodes[code]
	if !exists || time.Now().After(authCode.ExpiresAt) {
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant",
		})
		return
	}

	// Validate redirect URI
	if authCode.RedirectURI != redirectURI {
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "redirect_uri mismatch",
		})
		return
	}

	// Validate PKCE if required
	if m.config.SupportsPKCE && authCode.PKCEVerifier != "" {
		if !m.validatePKCE(authCode.PKCEVerifier, codeVerifier) {
			m.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_grant",
				"error_description": "PKCE verification failed",
			})
			return
		}
	}

	// Remove used authorization code
	delete(m.authorizedCodes, code)

	// Issue tokens
	accessToken := fmt.Sprintf("mock_access_%d", time.Now().UnixNano())
	refreshToken := ""
	if m.config.SupportsRefresh {
		refreshToken = fmt.Sprintf("mock_refresh_%d", time.Now().UnixNano())
	}

	expiresAt := time.Now().Add(m.config.TokenExpiry)

	m.issuedTokens[accessToken] = &IssuedToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		UserID:       authCode.UserID,
		Scopes:       authCode.Scopes,
		IssuedAt:     time.Now(),
	}

	if refreshToken != "" {
		m.issuedTokens[refreshToken] = &IssuedToken{
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(m.config.RefreshExpiry),
			UserID:       authCode.UserID,
			Scopes:       authCode.Scopes,
			IssuedAt:     time.Now(),
		}
	}
	m.mu.Unlock()

	// Return token response
	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(m.config.TokenExpiry.Seconds()),
		"scope":        "openid email profile",
	}

	if refreshToken != "" {
		response["refresh_token"] = refreshToken
	}

	// Add ID token for OpenID Connect
	if contains(authCode.Scopes, "openid") {
		response["id_token"] = m.generateMockIDToken(authCode.UserID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockOAuthServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	if !m.config.SupportsRefresh {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "unsupported_grant_type",
		})
		return
	}

	refreshToken := r.FormValue("refresh_token")

	m.mu.Lock()
	token, exists := m.issuedTokens[refreshToken]
	if !exists || time.Now().After(token.ExpiresAt) {
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant",
		})
		return
	}

	// Check if token is revoked
	if _, revoked := m.revokedTokens[refreshToken]; revoked {
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Token has been revoked",
		})
		return
	}

	// Issue new access token
	newAccessToken := fmt.Sprintf("mock_access_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(m.config.TokenExpiry)

	m.issuedTokens[newAccessToken] = &IssuedToken{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		UserID:       token.UserID,
		Scopes:       token.Scopes,
		IssuedAt:     time.Now(),
	}
	m.mu.Unlock()

	// Return token response
	response := map[string]interface{}{
		"access_token": newAccessToken,
		"token_type":   "Bearer",
		"expires_in":   int(m.config.TokenExpiry.Seconds()),
		"scope":        "openid email profile",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockOAuthServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	// Apply latency if configured
	if latency := m.getLatency("userinfo"); latency > 0 {
		time.Sleep(latency)
	}

	// Check for failure scenarios
	if m.shouldFail("userinfo") {
		http.Error(w, "User info server error", http.StatusInternalServerError)
		return
	}

	// Extract access token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_token",
		})
		return
	}

	accessToken := authHeader[7:]

	// Validate access token
	m.mu.RLock()
	token, exists := m.issuedTokens[accessToken]
	if !exists || time.Now().After(token.ExpiresAt) {
		m.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_token",
		})
		return
	}

	// Get user info
	userInfo, exists := m.userInfo[token.UserID]
	if !exists {
		// Return default user info
		userInfo = &oauth.UserInfo{
			ID:            token.UserID,
			Email:         token.UserID + "@example.com",
			EmailVerified: true,
			Name:          "Test User",
			FirstName:     "Test",
			LastName:      "User",
			Picture:       "https://example.com/avatar.jpg",
			Provider:      m.config.ProviderName,
		}
	}
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func (m *MockOAuthServer) handleRevoke(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")

	m.mu.Lock()
	m.revokedTokens[token] = time.Now()
	m.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (m *MockOAuthServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery := map[string]interface{}{
		"issuer":                                m.server.URL,
		"authorization_endpoint":                m.GetAuthURL(),
		"token_endpoint":                        m.GetTokenURL(),
		"userinfo_endpoint":                     m.GetUserInfoURL(),
		"revocation_endpoint":                   m.GetRevokeURL(),
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"code_challenge_methods_supported":      []string{"S256"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discovery)
}

// Helper methods

func (m *MockOAuthServer) getLatency(endpoint string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latencies[endpoint]
}

func (m *MockOAuthServer) shouldFail(endpoint string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check specific failure scenario
	if m.failureScenarios[endpoint] {
		return true
	}

	// Check error rate
	if rate, exists := m.errorRates[endpoint]; exists && rate > 0 {
		// Simple random failure based on rate
		return time.Now().UnixNano()%100 < int64(rate*100)
	}

	return false
}

func (m *MockOAuthServer) validatePKCE(stored, provided string) bool {
	// Simple validation for testing
	return stored == provided || stored == generateS256Challenge(provided)
}

func (m *MockOAuthServer) generateMockIDToken(userID string) string {
	// Generate a simple mock ID token (not a real JWT)
	claims := map[string]interface{}{
		"iss":   m.server.URL,
		"sub":   userID,
		"aud":   m.config.ClientID,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"email": userID + "@example.com",
		"name":  "Test User",
	}

	// For testing, just return a base64 encoded JSON
	data, _ := json.Marshal(claims)
	return "mock." + base64URLEncode(data) + ".signature"
}

// CreateMockProvider creates an OAuth provider configured for the mock server
func (m *MockOAuthServer) CreateMockProvider() oauth.Provider {
	return &MockProvider{
		server: m,
		config: oauth.ProviderConfig{
			ClientID:     m.config.ClientID,
			ClientSecret: m.config.ClientSecret,
			RedirectURL:  "http://localhost:8080/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
	}
}

// MockProvider implements oauth.Provider for testing
type MockProvider struct {
	server *MockOAuthServer
	config oauth.ProviderConfig
}

func (p *MockProvider) Name() string {
	return p.server.config.ProviderName
}

func (p *MockProvider) GetAuthURL(state string, pkce *oauth.PKCEChallenge) string {
	url := p.server.GetAuthURL() + "?"
	url += "client_id=" + p.config.ClientID
	url += "&redirect_uri=" + p.config.RedirectURL
	url += "&response_type=code"
	url += "&state=" + state
	url += "&scope=openid+email+profile"

	if pkce != nil && p.server.config.SupportsPKCE {
		url += "&code_challenge=" + pkce.Challenge
		url += "&code_challenge_method=" + pkce.ChallengeMethod
	}

	return url
}

func (p *MockProvider) Exchange(ctx context.Context, code string, pkce *oauth.PKCEChallenge) (*oauth.Token, error) {
	// Simulate token exchange
	client := &http.Client{Timeout: 10 * time.Second}

	data := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     p.config.ClientID,
		"client_secret": p.config.ClientSecret,
		"redirect_uri":  p.config.RedirectURL,
	}

	if pkce != nil {
		data["code_verifier"] = pkce.Verifier
	}

	// Make request to mock server
	resp, err := client.PostForm(p.server.GetTokenURL(), formValues(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token exchange failed: %s", errResp["error"])
	}

	var token oauth.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// Calculate expiry time
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

func (p *MockProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth.Token, error) {
	if !p.server.config.SupportsRefresh {
		return nil, fmt.Errorf("refresh not supported")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	data := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     p.config.ClientID,
		"client_secret": p.config.ClientSecret,
	}

	resp, err := client.PostForm(p.server.GetTokenURL(), formValues(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token refresh failed: %s", errResp["error"])
	}

	var token oauth.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// Calculate expiry time
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

func (p *MockProvider) GetUserInfo(ctx context.Context, accessToken string) (*oauth.UserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", p.server.GetUserInfoURL(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo oauth.UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (p *MockProvider) RevokeToken(ctx context.Context, token string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	data := map[string]string{
		"token": token,
	}

	resp, err := client.PostForm(p.server.GetRevokeURL(), formValues(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token: status %d", resp.StatusCode)
	}

	return nil
}

func (p *MockProvider) ValidateConfig() error {
	if p.config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if p.config.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}
	if p.config.RedirectURL == "" {
		return fmt.Errorf("redirect URL is required")
	}
	return nil
}

func (p *MockProvider) SupportsPKCE() bool {
	return p.server.config.SupportsPKCE
}

func (p *MockProvider) SupportsRefresh() bool {
	return p.server.config.SupportsRefresh
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func base64URLEncode(data []byte) string {
	// Simple base64 URL encoding for testing
	encoded := make([]byte, (len(data)+2)/3*4)
	n := len(encoded)
	for i := 0; i < len(data); i += 3 {
		// Simplified encoding
		encoded[i/3*4] = 'A' + byte(i%26)
		encoded[i/3*4+1] = 'A' + byte((i+1)%26)
		encoded[i/3*4+2] = 'A' + byte((i+2)%26)
		encoded[i/3*4+3] = 'A' + byte((i+3)%26)
	}
	return string(encoded[:n])
}

func generateS256Challenge(verifier string) string {
	// Simplified S256 challenge generation for testing
	return "mock_challenge_" + verifier
}

func formValues(data map[string]string) url.Values {
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}
	return values
}
