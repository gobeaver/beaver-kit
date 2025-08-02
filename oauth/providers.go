package oauth

import (
	"fmt"
)

// Provider factory functions

func NewGoogleProvider(config ProviderConfig) (Provider, error) {
	provider := NewGoogle(config)
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func NewGitHubProvider(config ProviderConfig) (Provider, error) {
	provider := NewGitHub(config)
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func NewAppleProvider(config ProviderConfig) (Provider, error) {
	provider, err := NewApple(config)
	if err != nil {
		return nil, err
	}
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func NewTwitterProvider(config ProviderConfig) (Provider, error) {
	provider := NewTwitter(config)
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
	return provider, nil
}

func NewCustomProvider(config ProviderConfig) (Provider, error) {
	// Placeholder - will be implemented in providers/custom.go
	return nil, fmt.Errorf("custom provider not yet implemented")
}