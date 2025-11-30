package filevalidator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"testing"
)

func TestArchiveValidator_ValidateContent(t *testing.T) {
	validator := DefaultArchiveValidator()

	tests := []struct {
		name      string
		createZip func() ([]byte, error)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid small zip",
			createZip: func() ([]byte, error) {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				// Add a small file
				f, err := w.Create("test.txt")
				if err != nil {
					return nil, err
				}

				_, err = f.Write([]byte("Hello, World!"))
				if err != nil {
					return nil, err
				}

				err = w.Close()
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: false,
		},
		{
			name: "zip with too many files",
			createZip: func() ([]byte, error) {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				// Add more files than allowed
				for i := 0; i < validator.MaxFiles+1; i++ {
					f, err := w.Create(fmt.Sprintf("file%d.txt", i))
					if err != nil {
						return nil, err
					}
					_, err = f.Write([]byte("test"))
					if err != nil {
						return nil, err
					}
				}

				err := w.Close()
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: true,
			errorMsg:  "too many files",
		},
		{
			name: "zip with high compression ratio",
			createZip: func() ([]byte, error) {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				// Create a highly compressible file (lots of zeros)
				f, err := w.Create("zeros.bin")
				if err != nil {
					return nil, err
				}

				// Write 10MB of zeros (highly compressible)
				zeros := make([]byte, 10*1024*1024)
				_, err = f.Write(zeros)
				if err != nil {
					return nil, err
				}

				err = w.Close()
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: true,
			errorMsg:  "suspicious compression ratio",
		},
		{
			name: "zip with directory traversal",
			createZip: func() ([]byte, error) {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)

				// Add file with directory traversal path
				f, err := w.Create("../../../etc/passwd")
				if err != nil {
					return nil, err
				}

				_, err = f.Write([]byte("malicious"))
				if err != nil {
					return nil, err
				}

				err = w.Close()
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: true,
			errorMsg:  "dangerous path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.createZip()
			if err != nil {
				t.Fatalf("Failed to create test zip: %v", err)
			}

			reader := bytes.NewReader(data)
			err = validator.ValidateContent(reader, int64(len(data)))

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

func TestArchiveValidator_SupportedMIMETypes(t *testing.T) {
	validator := DefaultArchiveValidator()
	types := validator.SupportedMIMETypes()

	// Only ZIP-based formats are actually supported
	expectedTypes := []string{
		"application/zip",
		"application/x-zip-compressed",
		"application/java-archive",
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

func containsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
