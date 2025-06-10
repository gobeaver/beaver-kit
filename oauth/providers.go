package oauth

import (
	"fmt"
)

// Provider factory functions

func NewGoogleProvider(config ProviderConfig) (Provider, error) {
	// Google provider implementation will be added later
	return nil, fmt.Errorf("google provider not yet implemented")
}

func NewGitHubProvider(config ProviderConfig) (Provider, error) {
	provider := NewGitHub(config)
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func NewAppleProvider(config ProviderConfig) (Provider, error) {
	// Placeholder - will be implemented in providers/apple.go
	return nil, fmt.Errorf("apple provider not yet implemented")
}

func NewTwitterProvider(config ProviderConfig) (Provider, error) {
	// Placeholder - will be implemented in providers/twitter.go
	return nil, fmt.Errorf("twitter provider not yet implemented")
}

func NewCustomProvider(config ProviderConfig) (Provider, error) {
	// Placeholder - will be implemented in providers/custom.go
	return nil, fmt.Errorf("custom provider not yet implemented")
}