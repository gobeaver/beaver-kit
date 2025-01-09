package filevalidator

import (
	"bytes"
	"io"
	"testing"
)

func TestContentValidatorRegistry(t *testing.T) {
	registry := NewContentValidatorRegistry()

	// Test registering a validator
	mockValidator := &mockContentValidator{
		mimeTypes: []string{"application/test"},
	}

	registry.Register("application/test", mockValidator)

	// Test getting a registered validator
	validator := registry.GetValidator("application/test")
	if validator == nil {
		t.Error("Expected validator for application/test, got nil")
	}

	// Test getting unregistered validator
	validator = registry.GetValidator("application/unknown")
	if validator != nil {
		t.Error("Expected nil for unregistered MIME type, got validator")
	}
}

func TestContentValidatorRegistryValidateContent(t *testing.T) {
	registry := NewContentValidatorRegistry()

	// Register a mock validator
	mockValidator := &mockContentValidator{
		mimeTypes: []string{"application/test"},
		validateFunc: func(reader io.Reader, size int64) error {
			return nil
		},
	}

	registry.Register("application/test", mockValidator)

	// Test validation with registered type
	reader := bytes.NewReader([]byte("test content"))
	err := registry.ValidateContent("application/test", reader, 12)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test validation with unregistered type (should return nil)
	err = registry.ValidateContent("application/unknown", reader, 12)
	if err != nil {
		t.Errorf("Expected no error for unregistered type, got: %v", err)
	}
}

// Mock content validator for testing
type mockContentValidator struct {
	mimeTypes    []string
	validateFunc func(reader io.Reader, size int64) error
}

func (m *mockContentValidator) ValidateContent(reader io.Reader, size int64) error {
	if m.validateFunc != nil {
		return m.validateFunc(reader, size)
	}
	return nil
}

func (m *mockContentValidator) SupportedMIMETypes() []string {
	return m.mimeTypes
}
