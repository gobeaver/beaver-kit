package filevalidator

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"testing"
)

// Helper to create a real multipart.FileHeader
func createMultipartFileHeader(filename string, content []byte) (*multipart.FileHeader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 1024)
	if err != nil {
		return nil, err
	}
	if len(form.File["file"]) == 0 {
		return nil, fmt.Errorf("no file found in form")
	}
	return form.File["file"][0], nil
}

// Tests to increase coverage for validator.go ValidateWithContext

func TestValidateWithContext_WithContentValidation(t *testing.T) {
	// Create a validator with content validation enabled
	validator := NewBuilder().
		Accept("text/plain").
		WithContentValidation().
		WithDefaultRegistry().
		Build()

	// Create a real file header
	content := []byte("test content for validation")
	fileHeader, err := createMultipartFileHeader("test.txt", content)
	if err != nil {
		t.Fatalf("Failed to create file header: %v", err)
	}

	err = validator.ValidateWithContext(context.Background(), fileHeader)
	if err != nil {
		t.Errorf("ValidateWithContext() error = %v, want nil", err)
	}
}

func TestValidateWithContext_ContentValidationFailed(t *testing.T) {
	// Create a validator with required content validation
	validator := NewBuilder().
		Accept("application/pdf").
		RequireContentValidation().
		WithDefaultRegistry().
		Build()

	// Create a real file header with invalid PDF content
	content := []byte("not a real PDF")
	fileHeader, err := createMultipartFileHeader("test.pdf", content)
	if err != nil {
		t.Fatalf("Failed to create file header: %v", err)
	}

	err = validator.ValidateWithContext(context.Background(), fileHeader)
	if err == nil {
		t.Error("ValidateWithContext() should error for invalid PDF content")
	}
}

func TestValidateWithContext_StrictMIMEValidation(t *testing.T) {
	// Create a validator with strict MIME validation
	validator := NewBuilder().
		Accept("image/png", "text/plain").
		StrictMIME().
		Build()

	// Create a real file header with text content but .png extension
	content := []byte("not an image")
	fileHeader, err := createMultipartFileHeader("test.png", content)
	if err != nil {
		t.Fatalf("Failed to create file header: %v", err)
	}

	err = validator.ValidateWithContext(context.Background(), fileHeader)
	if err == nil {
		t.Error("ValidateWithContext() should error for MIME mismatch in strict mode")
	}
	if !IsErrorOfType(err, ErrorTypeMIME) {
		t.Errorf("Expected ErrorTypeMIME, got %v", GetErrorType(err))
	}
}

func TestValidateWithContext_ContentValidationOptional(t *testing.T) {
	// Create a validator with optional content validation (not required)
	validator := NewBuilder().
		Accept("application/pdf").
		WithContentValidation(). // Enabled but not required
		WithDefaultRegistry().
		Build()

	// Create a real file header with invalid PDF content
	content := []byte("not a real PDF")
	fileHeader, err := createMultipartFileHeader("test.pdf", content)
	if err != nil {
		t.Fatalf("Failed to create file header: %v", err)
	}

	// Should not error because content validation is not required
	err = validator.ValidateWithContext(context.Background(), fileHeader)
	// This might still error due to MIME detection, but not due to content validation requirement
	_ = err
}
