package filevalidator

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

// ImageValidator validates image file dimensions and format.
// This is TYPE validation, not security scanning.
// For malware detection, integrate with ClamAV or similar.
type ImageValidator struct {
	MaxWidth   int
	MaxHeight  int
	MaxPixels  int
	MinWidth   int
	MinHeight  int
	AllowSVG   bool
	MaxSVGSize int64
}

// DefaultImageValidator creates an image validator with sensible defaults
func DefaultImageValidator() *ImageValidator {
	return &ImageValidator{
		MaxWidth:   10000,
		MaxHeight:  10000,
		MaxPixels:  50000000, // 50 megapixels
		MinWidth:   1,
		MinHeight:  1,
		AllowSVG:   true,
		MaxSVGSize: 5 * MB,
	}
}

// ValidateContent validates an image by reading only the header.
// Uses image.DecodeConfig which only reads bytes needed for dimensions.
// Does NOT load entire image into memory.
func (v *ImageValidator) ValidateContent(reader io.Reader, size int64) error {
	// Peek at first 1KB to check for SVG
	header := make([]byte, 1024)
	n, err := io.ReadFull(reader, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return NewValidationError(ErrorTypeContent, "failed to read image header")
	}
	header = header[:n]

	// Check if it's an SVG (text-based, needs different handling)
	if v.isSVG(header) {
		return v.validateSVG(size)
	}

	// For binary images, use DecodeConfig which only reads the header
	// Reconstruct reader with the bytes we already read
	combinedReader := io.MultiReader(bytes.NewReader(header), reader)

	img, _, err := image.DecodeConfig(combinedReader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, fmt.Sprintf("cannot decode image: %v", err))
	}

	// Validate dimensions
	if img.Width > v.MaxWidth {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("image width %d exceeds maximum %d", img.Width, v.MaxWidth))
	}

	if img.Height > v.MaxHeight {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("image height %d exceeds maximum %d", img.Height, v.MaxHeight))
	}

	if img.Width < v.MinWidth {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("image width %d below minimum %d", img.Width, v.MinWidth))
	}

	if img.Height < v.MinHeight {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("image height %d below minimum %d", img.Height, v.MinHeight))
	}

	// Check total pixels (decompression bomb protection)
	totalPixels := img.Width * img.Height
	if totalPixels > v.MaxPixels {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("total pixels %d exceeds maximum %d", totalPixels, v.MaxPixels))
	}

	return nil
}

// SupportedMIMETypes returns the MIME types this validator can handle
func (v *ImageValidator) SupportedMIMETypes() []string {
	types := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/bmp",
		"image/tiff",
		"image/x-icon",
		"image/vnd.microsoft.icon",
	}

	if v.AllowSVG {
		types = append(types, "image/svg+xml")
	}

	return types
}

// isSVG checks if the data looks like an SVG file
func (v *ImageValidator) isSVG(data []byte) bool {
	return bytes.Contains(data, []byte("<?xml")) || bytes.Contains(data, []byte("<svg"))
}

// validateSVG validates SVG files.
// SVG is XML-based, so we just check size limits.
// For XSS protection, sanitize SVGs at render time, not upload time.
func (v *ImageValidator) validateSVG(size int64) error {
	if !v.AllowSVG {
		return NewValidationError(ErrorTypeContent, "SVG files are not allowed")
	}

	if size > v.MaxSVGSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("SVG file size %d exceeds maximum %d", size, v.MaxSVGSize))
	}

	// SVG validated - it's text/XML, dimensions aren't in a standard header
	// For proper SVG dimension checking, you'd need to parse the XML
	return nil
}
