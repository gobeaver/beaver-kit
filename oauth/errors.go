package oauth

import (
	"errors"
	"fmt"
)

// Package-level errors
var (
	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrNotInitialized indicates the service hasn't been initialized
	ErrNotInitialized = errors.New("oauth service not initialized")

	// ErrProviderNotFound indicates the requested provider doesn't exist
	ErrProviderNotFound = errors.New("oauth provider not found")

	// ErrInvalidState indicates state parameter mismatch (CSRF protection)
	ErrInvalidState = errors.New("invalid state parameter")

	// ErrTokenExpired indicates the token has expired
	ErrTokenExpired = errors.New("token expired")

	// ErrNoRefreshToken indicates no refresh token is available
	ErrNoRefreshToken = errors.New("no refresh token available")

	// ErrPKCENotSupported indicates PKCE is not supported by provider
	ErrPKCENotSupported = errors.New("PKCE not supported by provider")

	// ErrSessionNotFound indicates session data not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidCode indicates invalid authorization code
	ErrInvalidCode = errors.New("invalid authorization code")

	// ErrNetworkError indicates a network error occurred
	ErrNetworkError = errors.New("network error")

	// ErrInvalidResponse indicates invalid response from provider
	ErrInvalidResponse = errors.New("invalid response from provider")

	// ErrAccessDenied indicates user denied access
	ErrAccessDenied = errors.New("access denied by user")

	// ErrUnsupportedResponseType indicates unsupported response type
	ErrUnsupportedResponseType = errors.New("unsupported response type")

	// ErrInvalidScope indicates invalid or unauthorized scope
	ErrInvalidScope = errors.New("invalid scope")

	// ErrServerError indicates provider server error
	ErrServerError = errors.New("provider server error")

	// ErrTemporarilyUnavailable indicates service temporarily unavailable
	ErrTemporarilyUnavailable = errors.New("service temporarily unavailable")
)

// Error represents a detailed OAuth error
type Error struct {
	Code        string // OAuth error code (e.g., "invalid_request")
	Description string // Human-readable error description
	URI         string // Optional URI with error details
	Provider    string // Provider where error occurred
	Err         error  // Underlying error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("oauth error [%s]: %s (%s)", e.Provider, e.Description, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("oauth error [%s]: %v", e.Provider, e.Err)
	}
	return fmt.Sprintf("oauth error [%s]: %s", e.Provider, e.Code)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// Is checks if the error matches a target error
func (e *Error) Is(target error) bool {
	if e.Err != nil {
		return errors.Is(e.Err, target)
	}
	return false
}

// NewError creates a new OAuth error
func NewError(provider, code, description string) *Error {
	return &Error{
		Provider:    provider,
		Code:        code,
		Description: description,
	}
}

// WrapError wraps an error with OAuth context
func WrapError(provider string, err error) *Error {
	return &Error{
		Provider: provider,
		Err:      err,
	}
}

// ParseError parses OAuth error from response
func ParseError(provider string, code, description, uri string) *Error {
	oauthErr := &Error{
		Provider:    provider,
		Code:        code,
		Description: description,
		URI:         uri,
	}

	// Map OAuth error codes to standard errors
	switch code {
	case "access_denied":
		oauthErr.Err = ErrAccessDenied
	case "invalid_request", "invalid_grant":
		oauthErr.Err = ErrInvalidCode
	case "invalid_scope":
		oauthErr.Err = ErrInvalidScope
	case "server_error":
		oauthErr.Err = ErrServerError
	case "temporarily_unavailable":
		oauthErr.Err = ErrTemporarilyUnavailable
	case "unsupported_response_type":
		oauthErr.Err = ErrUnsupportedResponseType
	}

	return oauthErr
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable errors
	if errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrServerError) ||
		errors.Is(err, ErrTemporarilyUnavailable) {
		return true
	}

	// Check OAuth error
	var oauthErr *Error
	if errors.As(err, &oauthErr) {
		return oauthErr.Code == "temporarily_unavailable" ||
			oauthErr.Code == "server_error"
	}

	return false
}
