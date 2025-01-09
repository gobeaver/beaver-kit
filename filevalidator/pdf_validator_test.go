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
			name:      "PDF with JavaScript",
			data:      []byte("%PDF-1.4\n/JavaScript (alert('XSS'))\n%%EOF"),
			wantError: true,
			errorMsg:  "JavaScript",
		},
		{
			name:      "PDF with embedded files",
			data:      []byte("%PDF-1.4\n/EmbeddedFiles\n%%EOF"),
			wantError: true,
			errorMsg:  "embedded files",
		},
		{
			name:      "PDF with launch action",
			data:      []byte("%PDF-1.4\n/Launch /F (cmd.exe)\n%%EOF"),
			wantError: true,
			errorMsg:  "launch actions",
		},
		{
			name:      "PDF with forms (allowed)",
			data:      []byte("%PDF-1.4\n/AcroForm\n%%EOF"),
			wantError: false, // Forms are allowed by default
		},
		{
			name:      "PDF with suspicious executable",
			data:      []byte("%PDF-1.4\ncmd.exe\n%%EOF"),
			wantError: true,
			errorMsg:  "suspicious patterns",
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

func TestPDFValidator_containsJavaScript(t *testing.T) {
	validator := DefaultPDFValidator()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "JavaScript tag",
			data:     []byte("some content /JavaScript more content"),
			expected: true,
		},
		{
			name:     "JS tag",
			data:     []byte("some content /JS more content"),
			expected: true,
		},
		{
			name:     "app.alert",
			data:     []byte("some content app.alert('test') more content"),
			expected: true,
		},
		{
			name:     "no JavaScript",
			data:     []byte("clean PDF content without scripts"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.containsJavaScript(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPDFValidator_ConfigurableRestrictions(t *testing.T) {
	// Test with JavaScript allowed
	validator := &PDFValidator{
		AllowJavaScript:    true,
		AllowEmbeddedFiles: false,
		AllowForms:         true,
		AllowActions:       false,
		MaxSize:            50 * MB,
		ValidateStructure:  true,
	}

	data := []byte("%PDF-1.4\n/JavaScript (alert('test'))\n%%EOF")
	reader := bytes.NewReader(data)
	err := validator.ValidateContent(reader, int64(len(data)))

	if err != nil {
		t.Errorf("Expected no error with JavaScript allowed, got: %v", err)
	}

	// Test with forms not allowed
	validator.AllowForms = false
	data = []byte("%PDF-1.4\n/AcroForm\n%%EOF")
	reader = bytes.NewReader(data)
	err = validator.ValidateContent(reader, int64(len(data)))

	if err == nil {
		t.Error("Expected error with forms not allowed, got nil")
	}
}
