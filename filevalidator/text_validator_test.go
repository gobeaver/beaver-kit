package filevalidator

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		maxDepth int
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid simple json",
			json:    `{"name": "test", "value": 123}`,
			wantErr: false,
		},
		{
			name:    "valid array",
			json:    `[1, 2, 3, "four"]`,
			wantErr: false,
		},
		{
			name:    "valid nested",
			json:    `{"a": {"b": {"c": 1}}}`,
			wantErr: false,
		},
		{
			name:    "invalid json - syntax error",
			json:    `{"name": }`,
			wantErr: true,
			errMsg:  "invalid JSON",
		},
		{
			name:    "invalid json - unclosed",
			json:    `{"name": "test"`,
			wantErr: true,
			errMsg:  "invalid JSON",
		},
		{
			name:     "depth exceeded",
			json:     `{"a":{"b":{"c":{"d":{"e":1}}}}}`,
			maxDepth: 3,
			wantErr:  true,
			errMsg:   "nesting depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultJSONValidator()
			if tt.maxDepth > 0 {
				v.MaxDepth = tt.maxDepth
			}

			data := []byte(tt.json)
			err := v.ValidateContent(bytes.NewReader(data), int64(len(data)))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestXMLValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		allowDTD bool
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid simple xml",
			xml:     `<?xml version="1.0"?><root><item>test</item></root>`,
			wantErr: false,
		},
		{
			name:    "valid xml with attributes",
			xml:     `<root attr="value"><child/></root>`,
			wantErr: false,
		},
		{
			name:    "invalid xml - unclosed tag",
			xml:     `<root><item>`,
			wantErr: true,
			errMsg:  "invalid XML",
		},
		{
			name:    "DTD blocked by default",
			xml:     `<!DOCTYPE foo><root/>`,
			wantErr: true,
			errMsg:  "DTD/ENTITY declarations not allowed",
		},
		{
			name:     "DTD allowed when enabled",
			xml:      `<!DOCTYPE foo><root/>`,
			allowDTD: true,
			wantErr:  false,
		},
		{
			name:    "ENTITY blocked (XXE protection)",
			xml:     `<!ENTITY xxe SYSTEM "file:///etc/passwd"><root/>`,
			wantErr: true,
			errMsg:  "DTD/ENTITY declarations not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultXMLValidator()
			v.AllowDTD = tt.allowDTD

			data := []byte(tt.xml)
			err := v.ValidateContent(bytes.NewReader(data), int64(len(data)))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCSVValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name       string
		csv        string
		maxRows    int
		maxColumns int
		wantErr    bool
		errMsg     string
	}{
		{
			name:    "valid csv",
			csv:     "name,age,city\nJohn,30,NYC\nJane,25,LA",
			wantErr: false,
		},
		{
			name:    "valid csv with quotes",
			csv:     `"name","value"\n"test","hello, world"`,
			wantErr: false,
		},
		{
			name:    "empty csv",
			csv:     "",
			wantErr: true,
			errMsg:  "empty CSV",
		},
		{
			name:    "too many rows",
			csv:     "a\nb\nc\nd",
			maxRows: 2,
			wantErr: true,
			errMsg:  "rows",
		},
		{
			name:       "too many columns",
			csv:        "a,b,c,d,e",
			maxColumns: 3,
			wantErr:    true,
			errMsg:     "columns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultCSVValidator()
			if tt.maxRows > 0 {
				v.MaxRows = tt.maxRows
			}
			if tt.maxColumns > 0 {
				v.MaxColumns = tt.maxColumns
			}

			data := []byte(tt.csv)
			err := v.ValidateContent(bytes.NewReader(data), int64(len(data)))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPlainTextValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		requireUTF8 bool
		wantErr     bool
	}{
		{
			name:        "valid utf8",
			data:        []byte("Hello, 世界!"),
			requireUTF8: true,
			wantErr:     false,
		},
		{
			name:        "valid ascii",
			data:        []byte("Hello, World!"),
			requireUTF8: true,
			wantErr:     false,
		},
		{
			name:        "invalid utf8",
			data:        []byte{0xFF, 0xFE, 0x00, 0x01}, // Invalid UTF-8 sequence
			requireUTF8: true,
			wantErr:     true,
		},
		{
			name:        "invalid utf8 but not required",
			data:        []byte{0xFF, 0xFE, 0x00, 0x01},
			requireUTF8: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := DefaultPlainTextValidator()
			v.RequireUTF8 = tt.requireUTF8

			err := v.ValidateContent(bytes.NewReader(tt.data), int64(len(tt.data)))

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTextValidators_SupportedMIMETypes(t *testing.T) {
	validators := []struct {
		name      string
		validator ContentValidator
		expected  []string
	}{
		{"JSON", DefaultJSONValidator(), []string{"application/json"}},
		{"XML", DefaultXMLValidator(), []string{"application/xml", "text/xml"}},
		{"CSV", DefaultCSVValidator(), []string{"text/csv"}},
		{"PlainText", DefaultPlainTextValidator(), []string{"text/plain"}},
	}

	for _, tt := range validators {
		t.Run(tt.name, func(t *testing.T) {
			types := tt.validator.SupportedMIMETypes()
			for _, exp := range tt.expected {
				found := false
				for _, typ := range types {
					if typ == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected MIME type %s not found for %s", exp, tt.name)
				}
			}
		})
	}
}

func TestJSONValidator_MaxSize(t *testing.T) {
	v := &JSONValidator{MaxSize: 10}

	data := []byte(`{"key": "value"}`) // > 10 bytes
	err := v.ValidateContent(bytes.NewReader(data), int64(len(data)))

	if err == nil {
		t.Error("expected size error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected size error, got: %v", err)
	}
}

func TestCSVValidator_UTF8(t *testing.T) {
	v := DefaultCSVValidator()
	v.RequireUTF8 = true

	// Invalid UTF-8 in CSV
	data := []byte("name,value\n\xFF\xFE,test")
	err := v.ValidateContent(bytes.NewReader(data), int64(len(data)))

	if err == nil {
		t.Error("expected UTF-8 error, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("expected UTF-8 error, got: %v", err)
	}
}
