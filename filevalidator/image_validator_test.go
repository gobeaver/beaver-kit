package filevalidator

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestImageValidator_ValidateContent(t *testing.T) {
	validator := DefaultImageValidator()

	tests := []struct {
		name      string
		createImg func() ([]byte, error)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid PNG image",
			createImg: func() ([]byte, error) {
				// Create a small valid PNG
				img := image.NewRGBA(image.Rect(0, 0, 10, 10))
				// Fill with red
				for y := 0; y < 10; y++ {
					for x := 0; x < 10; x++ {
						img.Set(x, y, color.RGBA{255, 0, 0, 255})
					}
				}

				buf := new(bytes.Buffer)
				err := png.Encode(buf, img)
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: false,
		},
		{
			name: "image too wide",
			createImg: func() ([]byte, error) {
				// Create an image wider than the max
				img := image.NewRGBA(image.Rect(0, 0, validator.MaxWidth+1, 10))

				buf := new(bytes.Buffer)
				err := png.Encode(buf, img)
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: true,
			errorMsg:  "exceeds maximum",
		},
		{
			name: "image too tall",
			createImg: func() ([]byte, error) {
				// Create an image taller than the max
				img := image.NewRGBA(image.Rect(0, 0, 10, validator.MaxHeight+1))

				buf := new(bytes.Buffer)
				err := png.Encode(buf, img)
				if err != nil {
					return nil, err
				}

				return buf.Bytes(), nil
			},
			wantError: true,
			errorMsg:  "exceeds maximum",
		},
		{
			name: "valid SVG",
			createImg: func() ([]byte, error) {
				svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <rect width="100" height="100" fill="red"/>
</svg>`
				return []byte(svg), nil
			},
			wantError: false,
		},
		{
			name: "SVG with script (type validation only, not security)",
			createImg: func() ([]byte, error) {
				// SVG with script is valid for TYPE validation
				// Security scanning should be done separately (e.g., ClamAV)
				svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">
  <script>alert('XSS')</script>
  <rect width="100" height="100" fill="red"/>
</svg>`
				return []byte(svg), nil
			},
			wantError: false, // We only validate type, not security
		},
		{
			name: "invalid image data",
			createImg: func() ([]byte, error) {
				return []byte("not an image"), nil
			},
			wantError: true,
			errorMsg:  "cannot decode image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.createImg()
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
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

func TestImageValidator_SupportedMIMETypes(t *testing.T) {
	validator := DefaultImageValidator()
	types := validator.SupportedMIMETypes()

	// Should include SVG by default
	foundSVG := false
	for _, typ := range types {
		if typ == "image/svg+xml" {
			foundSVG = true
			break
		}
	}

	if !foundSVG {
		t.Error("Expected SVG to be supported by default")
	}

	// Test with SVG disabled
	validator.AllowSVG = false
	types = validator.SupportedMIMETypes()

	foundSVG = false
	for _, typ := range types {
		if typ == "image/svg+xml" {
			foundSVG = true
			break
		}
	}

	if foundSVG {
		t.Error("Expected SVG not to be supported when disabled")
	}
}

func TestImageValidator_isSVG(t *testing.T) {
	validator := DefaultImageValidator()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "XML declaration",
			data:     []byte(`<?xml version="1.0"?><svg></svg>`),
			expected: true,
		},
		{
			name:     "SVG tag",
			data:     []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
			expected: true,
		},
		{
			name:     "Not SVG",
			data:     []byte(`<html><body>Not SVG</body></html>`),
			expected: false,
		},
		{
			name:     "PNG header",
			data:     []byte{137, 80, 78, 71, 13, 10, 26, 10},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isSVG(tt.data)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestImageValidator_SVGSizeLimit(t *testing.T) {
	validator := &ImageValidator{
		AllowSVG:   true,
		MaxSVGSize: 100, // 100 bytes max
	}

	// Small SVG should pass
	smallSVG := []byte(`<svg width="10" height="10"></svg>`)
	reader := bytes.NewReader(smallSVG)
	err := validator.ValidateContent(reader, int64(len(smallSVG)))
	if err != nil {
		t.Errorf("Expected small SVG to pass, got: %v", err)
	}

	// Large SVG should fail
	largeSVG := []byte(`<svg width="10" height="10">` + string(make([]byte, 200)) + `</svg>`)
	reader = bytes.NewReader(largeSVG)
	err = validator.ValidateContent(reader, int64(len(largeSVG)))
	if err == nil {
		t.Error("Expected large SVG to fail, got nil")
	}
}
