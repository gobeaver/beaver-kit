package filevalidator

import (
	"bytes"
	"testing"
)

func TestPDFValidator_ValidateContent(t *testing.T) {
	validator := DefaultPDFValidator()

	tests := []struct {
		name      string
		data      []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid PDF header and trailer",
			data:      []byte("%PDF-1.4\n...content...%%EOF"),
			wantError: false,
		},
		{
			name:      "missing PDF header",
			data:      []byte("Not a PDF\n...content...%%EOF"),
			wantError: true,
			errorMsg:  "invalid PDF header",
		},
		{
			name:      "missing PDF trailer",
			data:      []byte("%PDF-1.4\n...content..."),
			wantError: true,
			errorMsg:  "invalid PDF trailer",
		},
		{
			name:      "valid PDF with forms",
			data:      []byte("%PDF-1.4\n/AcroForm\n%%EOF"),
			wantError: false,
		},
		{
			name:      "valid PDF with JavaScript (type validation only)",
			data:      []byte("%PDF-1.4\n/JavaScript\n%%EOF"),
			wantError: false, // We only validate type, not security
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			err := validator.ValidateContent(reader, int64(len(tt.data)))

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestPDFValidator_SupportedMIMETypes(t *testing.T) {
	validator := DefaultPDFValidator()
	types := validator.SupportedMIMETypes()

	expectedTypes := []string{
		"application/pdf",
		"application/x-pdf",
		"application/vnd.pdf",
	}

	if len(types) != len(expectedTypes) {
		t.Errorf("Expected %d MIME types, got %d", len(expectedTypes), len(types))
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, typ := range types {
			if typ == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected MIME type %s not found", expectedType)
		}
	}
}

func TestPDFValidator_MaxSize(t *testing.T) {
	validator := &PDFValidator{
		MaxSize: 100, // 100 bytes max
	}

	// Test file exceeding max size
	data := []byte("%PDF-1.4\n" + string(make([]byte, 200)) + "%%EOF")
	reader := bytes.NewReader(data)
	err := validator.ValidateContent(reader, int64(len(data)))

	if err == nil {
		t.Error("Expected error for oversized PDF, got nil")
	}
}

func TestPDFValidator_SeekableReader(t *testing.T) {
	validator := DefaultPDFValidator()

	// Test with a seekable reader (bytes.Reader)
	data := []byte("%PDF-1.7\nsome content here\n%%EOF")
	reader := bytes.NewReader(data)
	err := validator.ValidateContent(reader, int64(len(data)))

	if err != nil {
		t.Errorf("Expected no error for valid PDF, got: %v", err)
	}
}
