package filevalidator

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestOfficeValidator_ValidateContent(t *testing.T) {
	tests := []struct {
		name      string
		validator *OfficeValidator
		makeZip   func() []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid docx",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml": `<?xml version="1.0"?>`,
					"_rels/.rels":         `<?xml version="1.0"?>`,
					"word/document.xml":   `<document/>`,
				})
			},
			wantErr: false,
		},
		{
			name:      "valid xlsx",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml": `<?xml version="1.0"?>`,
					"_rels/.rels":         `<?xml version="1.0"?>`,
					"xl/workbook.xml":     `<workbook/>`,
				})
			},
			wantErr: false,
		},
		{
			name:      "valid pptx",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml":  `<?xml version="1.0"?>`,
					"_rels/.rels":          `<?xml version="1.0"?>`,
					"ppt/presentation.xml": `<presentation/>`,
				})
			},
			wantErr: false,
		},
		{
			name:      "missing content types",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"_rels/.rels":       `<?xml version="1.0"?>`,
					"word/document.xml": `<document/>`,
				})
			},
			wantErr: true,
			errMsg:  "missing [Content_Types].xml",
		},
		{
			name:      "missing rels",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml": `<?xml version="1.0"?>`,
					"word/document.xml":   `<document/>`,
				})
			},
			wantErr: true,
			errMsg:  "missing _rels/.rels",
		},
		{
			name:      "macros not allowed by default",
			validator: DefaultOfficeValidator(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml": `<?xml version="1.0"?>`,
					"_rels/.rels":         `<?xml version="1.0"?>`,
					"word/document.xml":   `<document/>`,
					"word/vbaProject.bin": `VBA content`,
				})
			},
			wantErr: true,
			errMsg:  "macro-enabled documents are not allowed",
		},
		{
			name: "macros allowed when enabled",
			validator: func() *OfficeValidator {
				v := DefaultOfficeValidator()
				v.AllowMacros = true
				return v
			}(),
			makeZip: func() []byte {
				return createOfficeZip(map[string]string{
					"[Content_Types].xml": `<?xml version="1.0"?>`,
					"_rels/.rels":         `<?xml version="1.0"?>`,
					"word/document.xml":   `<document/>`,
					"word/vbaProject.bin": `VBA content`,
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.makeZip()
			reader := bytes.NewReader(data)

			err := tt.validator.ValidateContent(reader, int64(len(data)))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOfficeValidator_SupportedMIMETypes(t *testing.T) {
	v := DefaultOfficeValidator()
	types := v.SupportedMIMETypes()

	expected := []string{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	for _, exp := range expected {
		found := false
		for _, typ := range types {
			if typ == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected MIME type %s not found", exp)
		}
	}

	// Check macros types are included when enabled
	v.AllowMacros = true
	types = v.SupportedMIMETypes()

	macroTypes := []string{
		"application/vnd.ms-word.document.macroEnabled.12",
		"application/vnd.ms-excel.sheet.macroEnabled.12",
	}

	for _, exp := range macroTypes {
		found := false
		for _, typ := range types {
			if typ == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected macro MIME type %s not found when macros enabled", exp)
		}
	}
}

// createOfficeZip creates a ZIP file with the given files
func createOfficeZip(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(content))
	}

	w.Close()
	return buf.Bytes()
}
