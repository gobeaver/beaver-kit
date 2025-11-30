package oauth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	appleKeysURL = "https://appleid.apple.com/auth/keys"
	appleIssuer  = "https://appleid.apple.com"
)

// ApplePublicKey represents Apple's public key
type ApplePublicKey struct {
	KTY string `json:"kty"` // Key Type
	KID string `json:"kid"` // Key ID
	Use string `json:"use"` // Key Use
	Alg string `json:"alg"` // Algorithm
	N   string `json:"n"`   // Modulus (for RSA)
	E   string `json:"e"`   // Exponent (for RSA)
	X   string `json:"x"`   // X coordinate (for ECDSA)
	Y   string `json:"y"`   // Y coordinate (for ECDSA)
	CRV string `json:"crv"` // Curve (for ECDSA)
}

// AppleKeysResponse represents the response from Apple's keys endpoint
type AppleKeysResponse struct {
	Keys []ApplePublicKey `json:"keys"`
}

// AppleJWTValidator handles Apple ID token validation
type AppleJWTValidator struct {
	httpClient         HTTPClient
	keysCache          *appleKeysCache
	clientID           string
	skipSignatureCheck bool // For testing only
}

// appleKeysCache caches Apple's public keys
type appleKeysCache struct {
	mu         sync.RWMutex
	keys       map[string]interface{} // kid -> public key (rsa.PublicKey or ecdsa.PublicKey)
	lastUpdate time.Time
	ttl        time.Duration
}

// AppleIDTokenClaims represents the claims in an Apple ID token
type AppleIDTokenClaims struct {
	// Standard JWT claims
	Issuer         string        `json:"iss"`
	Subject        string        `json:"sub"`
	Audience       StringOrArray `json:"aud"`
	ExpirationTime int64         `json:"exp"`
	IssuedAt       int64         `json:"iat"`
	AuthTime       int64         `json:"auth_time,omitempty"`
	Nonce          string        `json:"nonce,omitempty"`
	NonceSupported bool          `json:"nonce_supported,omitempty"`

	// Apple-specific claims
	Email          string      `json:"email,omitempty"`
	EmailVerified  interface{} `json:"email_verified,omitempty"`   // Can be bool or string
	IsPrivateEmail interface{} `json:"is_private_email,omitempty"` // Can be bool or string
	RealUserStatus int         `json:"real_user_status,omitempty"`
	TransferSub    string      `json:"transfer_sub,omitempty"`
	AtHash         string      `json:"at_hash,omitempty"`

	// Additional claims
	Extra map[string]interface{} `json:"-"`
}

// StringOrArray handles fields that can be either string or []string
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*s = arr
	return nil
}

// NewAppleJWTValidator creates a new Apple JWT validator
func NewAppleJWTValidator(clientID string, httpClient HTTPClient) *AppleJWTValidator {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &AppleJWTValidator{
		httpClient: httpClient,
		clientID:   clientID,
		keysCache: &appleKeysCache{
			keys: make(map[string]interface{}),
			ttl:  24 * time.Hour, // Cache keys for 24 hours
		},
	}
}

// EnableTestMode enables test mode which skips signature verification (TESTING ONLY)
func (v *AppleJWTValidator) EnableTestMode() {
	v.skipSignatureCheck = true
}

// ValidateIDToken validates an Apple ID token
func (v *AppleJWTValidator) ValidateIDToken(ctx context.Context, idToken string, nonce string) (*AppleIDTokenClaims, error) {
	// Parse the JWT parts
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid ID token format: expected 3 parts, got %d", len(parts))
	}

	// Decode and parse the header
	headerData, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerData, &header); err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Validate header
	if header.Type != "JWT" && header.Type != "" {
		return nil, fmt.Errorf("invalid token type: %s", header.Type)
	}

	// Skip signature verification in test mode
	if !v.skipSignatureCheck {
		// Get the public key for verification
		publicKey, err := v.getPublicKey(ctx, header.KeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		// Verify the signature
		if err := v.verifySignature(parts[0]+"."+parts[1], parts[2], header.Algorithm, publicKey); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Decode and parse the claims
	claimsData, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims AppleIDTokenClaims
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Unmarshal extra claims
	var extraClaims map[string]interface{}
	if err := json.Unmarshal(claimsData, &extraClaims); err == nil {
		// Remove standard claims from extra
		delete(extraClaims, "iss")
		delete(extraClaims, "sub")
		delete(extraClaims, "aud")
		delete(extraClaims, "exp")
		delete(extraClaims, "iat")
		delete(extraClaims, "auth_time")
		delete(extraClaims, "nonce")
		delete(extraClaims, "email")
		delete(extraClaims, "email_verified")
		delete(extraClaims, "is_private_email")
		delete(extraClaims, "real_user_status")
		delete(extraClaims, "transfer_sub")
		delete(extraClaims, "at_hash")
		claims.Extra = extraClaims
	}

	// Validate claims
	if err := v.validateClaims(&claims, nonce); err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	return &claims, nil
}

// validateClaims validates the token claims
func (v *AppleJWTValidator) validateClaims(claims *AppleIDTokenClaims, expectedNonce string) error {
	// Skip most validation in test mode
	if v.skipSignatureCheck {
		return nil
	}

	now := time.Now().Unix()

	// Validate issuer
	if claims.Issuer != appleIssuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", appleIssuer, claims.Issuer)
	}

	// Validate audience
	foundClientID := false
	for _, aud := range claims.Audience {
		if aud == v.clientID {
			foundClientID = true
			break
		}
	}
	if !foundClientID {
		return fmt.Errorf("invalid audience: client ID %s not found in %v", v.clientID, claims.Audience)
	}

	// Validate expiration
	if claims.ExpirationTime <= now {
		return fmt.Errorf("token has expired: exp=%d, now=%d", claims.ExpirationTime, now)
	}

	// Validate issued at (with 5 minute leeway for clock skew)
	if claims.IssuedAt > now+300 {
		return fmt.Errorf("token issued in the future: iat=%d, now=%d", claims.IssuedAt, now)
	}

	// Validate token is not too old (Apple tokens are valid for 10 minutes)
	if claims.IssuedAt < now-600 {
		return fmt.Errorf("token is too old: iat=%d, now=%d", claims.IssuedAt, now)
	}

	// Validate nonce if provided
	if expectedNonce != "" && claims.Nonce != expectedNonce {
		return fmt.Errorf("nonce mismatch: expected %s, got %s", expectedNonce, claims.Nonce)
	}

	// Validate auth_time if present
	if claims.AuthTime > 0 && claims.AuthTime > now {
		return fmt.Errorf("auth_time is in the future: auth_time=%d, now=%d", claims.AuthTime, now)
	}

	return nil
}

// getPublicKey retrieves Apple's public key by key ID
func (v *AppleJWTValidator) getPublicKey(ctx context.Context, keyID string) (interface{}, error) {
	// Check cache first
	if key := v.keysCache.get(keyID); key != nil {
		return key, nil
	}

	// Fetch keys from Apple
	keys, err := v.fetchAppleKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple keys: %w", err)
	}

	// Update cache
	if err := v.keysCache.update(keys); err != nil {
		return nil, fmt.Errorf("failed to update keys cache: %w", err)
	}

	// Get the requested key
	key := v.keysCache.get(keyID)
	if key == nil {
		return nil, fmt.Errorf("key with ID %s not found", keyID)
	}

	return key, nil
}

// fetchAppleKeys fetches Apple's public keys
func (v *AppleJWTValidator) fetchAppleKeys(ctx context.Context) ([]ApplePublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", appleKeysURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var keysResp AppleKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&keysResp); err != nil {
		return nil, err
	}

	return keysResp.Keys, nil
}

// verifySignature verifies the JWT signature
func (v *AppleJWTValidator) verifySignature(signingInput, signature string, algorithm string, publicKey interface{}) error {
	// Decode the signature
	sigBytes, err := base64URLDecode(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the signing input
	hash := sha256.Sum256([]byte(signingInput))

	switch algorithm {
	case "RS256":
		rsaKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("invalid key type for RS256")
		}
		return rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hash[:], sigBytes)

	case "ES256":
		ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("invalid key type for ES256")
		}

		// Parse ECDSA signature (r and s values)
		if len(sigBytes) != 64 {
			return fmt.Errorf("invalid ECDSA signature length: %d", len(sigBytes))
		}

		r := new(big.Int).SetBytes(sigBytes[:32])
		s := new(big.Int).SetBytes(sigBytes[32:])

		if !ecdsa.Verify(ecdsaKey, hash[:], r, s) {
			return fmt.Errorf("ECDSA signature verification failed")
		}
		return nil

	default:
		return fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

// get retrieves a key from cache
func (c *appleKeysCache) get(keyID string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if cache has expired
	if time.Since(c.lastUpdate) > c.ttl {
		return nil
	}

	return c.keys[keyID]
}

// update updates the cache with new keys
func (c *appleKeysCache) update(keys []ApplePublicKey) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	newKeys := make(map[string]interface{})

	for _, key := range keys {
		publicKey, err := parseApplePublicKey(key)
		if err != nil {
			return fmt.Errorf("failed to parse key %s: %w", key.KID, err)
		}
		newKeys[key.KID] = publicKey
	}

	c.keys = newKeys
	c.lastUpdate = time.Now()

	return nil
}

// parseApplePublicKey parses an Apple public key
func parseApplePublicKey(key ApplePublicKey) (interface{}, error) {
	switch key.KTY {
	case "RSA":
		// Parse RSA key
		nBytes, err := base64URLDecode(key.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode modulus: %w", err)
		}

		eBytes, err := base64URLDecode(key.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode exponent: %w", err)
		}

		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		return &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}, nil

	case "EC":
		// Parse ECDSA key
		xBytes, err := base64URLDecode(key.X)
		if err != nil {
			return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
		}

		yBytes, err := base64URLDecode(key.Y)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
		}

		var curve elliptic.Curve
		switch key.CRV {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported curve: %s", key.CRV)
		}

		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)

		return &ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported key type: %s", key.KTY)
	}
}

// IsEmailVerified returns whether the email is verified
func (c *AppleIDTokenClaims) IsEmailVerified() bool {
	switch v := c.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}

// IsPrivateEmailAddress returns whether the email is a private relay address
func (c *AppleIDTokenClaims) IsPrivateEmailAddress() bool {
	switch v := c.IsPrivateEmail.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}
