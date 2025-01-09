package filevalidator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
)

// ImageValidator validates image files for malicious content and reasonable dimensions
type ImageValidator struct {
	MaxWidth       int
	MaxHeight      int
	MaxPixels      int
	MinWidth       int
	MinHeight      int
	ValidatePixels bool
	AllowSVG       bool
	MaxSVGSize     int64
}

// DefaultImageValidator creates an image validator with sensible defaults
func DefaultImageValidator() *ImageValidator {
	return &ImageValidator{
		MaxWidth:       10000,
		MaxHeight:      10000,
		MaxPixels:      50000000, // 50 megapixels
		MinWidth:       1,
		MinHeight:      1,
		ValidatePixels: true,
		AllowSVG:       true,
		MaxSVGSize:     5 * MB,
	}
}

// ValidateContent validates the content of an image file
func (v *ImageValidator) ValidateContent(reader io.Reader, size int64) error {
	// Read the entire content
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return NewValidationError(ErrorTypeContent, "failed to read image content")
	}

	// Check if it's an SVG
	if v.isSVG(data) {
		return v.validateSVG(data, size)
	}

	// Try to decode as a regular image
	imageReader := bytes.NewReader(data)
	img, format, err := image.DecodeConfig(imageReader)
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

	// Check total pixels
	totalPixels := img.Width * img.Height
	if totalPixels > v.MaxPixels {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("total pixels %d exceeds maximum %d", totalPixels, v.MaxPixels))
	}

	// Check for specific format vulnerabilities
	if format == "jpeg" || format == "jpg" {
		if err := v.validateJPEG(data); err != nil {
			return err
		}
	} else if format == "png" {
		if err := v.validatePNG(data); err != nil {
			return err
		}
	} else if format == "gif" {
		if err := v.validateGIF(data); err != nil {
			return err
		}
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
	// Check for SVG XML declaration or <svg tag
	if bytes.Contains(data[:min(len(data), 1024)], []byte("<?xml")) ||
		bytes.Contains(data[:min(len(data), 1024)], []byte("<svg")) {
		return true
	}
	return false
}

// validateSVG validates SVG content for potentially malicious content
func (v *ImageValidator) validateSVG(data []byte, size int64) error {
	if !v.AllowSVG {
		return NewValidationError(ErrorTypeContent, "SVG files are not allowed")
	}

	if size > v.MaxSVGSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("SVG file size %d exceeds maximum %d", size, v.MaxSVGSize))
	}

	// Check for potentially dangerous SVG content
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"onload=",
		"onerror=",
		"onclick=",
		"onmouseover=",
		"<iframe",
		"<embed",
		"<object",
		"<link",
		"@import",
		"<use",
		"<animate",
		"<set",
		"<animateMotion",
		"<animateTransform",
		"<foreignObject",
	}

	for _, pattern := range dangerousPatterns {
		if bytes.Contains(data, []byte(pattern)) {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("SVG contains potentially dangerous content: %s", pattern))
		}
	}

	return nil
}

// validateJPEG performs JPEG-specific validation
func (v *ImageValidator) validateJPEG(data []byte) error {
	// Check for valid JPEG header
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return NewValidationError(ErrorTypeContent, "invalid JPEG header")
	}

	// Check for valid JPEG footer
	if len(data) < 2 || data[len(data)-2] != 0xFF || data[len(data)-1] != 0xD9 {
		return NewValidationError(ErrorTypeContent, "invalid JPEG footer")
	}

	// Look for suspicious markers
	for i := 2; i < len(data)-1; i++ {
		if data[i] == 0xFF {
			marker := data[i+1]
			// Check for comment segments that might contain scripts
			if marker == 0xFE { // COM marker
				// Read segment length
				if i+3 < len(data) {
					length := int(data[i+2])<<8 | int(data[i+3])
					if i+length+2 <= len(data) {
						comment := data[i+4 : i+2+length]
						if v.containsMaliciousContent(comment) {
							return NewValidationError(ErrorTypeContent,
								"JPEG comment contains suspicious content")
						}
					}
				}
			}
		}
	}

	return nil
}

// validatePNG performs PNG-specific validation
func (v *ImageValidator) validatePNG(data []byte) error {
	// Check PNG signature
	pngSignature := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	if len(data) < len(pngSignature) || !bytes.Equal(data[:8], pngSignature) {
		return NewValidationError(ErrorTypeContent, "invalid PNG signature")
	}

	// Parse PNG chunks
	offset := 8
	for offset < len(data) {
		if offset+8 > len(data) {
			break
		}

		// Read chunk length and type
		length := binary.BigEndian.Uint32(data[offset : offset+4])
		chunkType := string(data[offset+4 : offset+8])

		// Check for suspicious chunk types
		if chunkType == "tEXt" || chunkType == "zTXt" || chunkType == "iTXt" {
			// Check text chunks for malicious content
			if offset+12+int(length) <= len(data) {
				textData := data[offset+8 : offset+8+int(length)]
				if v.containsMaliciousContent(textData) {
					return NewValidationError(ErrorTypeContent,
						fmt.Sprintf("PNG %s chunk contains suspicious content", chunkType))
				}
			}
		}

		// Move to next chunk
		offset += 12 + int(length) // 4 (length) + 4 (type) + length + 4 (CRC)
		if offset > len(data) {
			break
		}
	}

	return nil
}

// validateGIF performs GIF-specific validation
func (v *ImageValidator) validateGIF(data []byte) error {
	// Check GIF header
	if len(data) < 6 {
		return NewValidationError(ErrorTypeContent, "GIF file too small")
	}

	header := string(data[:6])
	if header != "GIF87a" && header != "GIF89a" {
		return NewValidationError(ErrorTypeContent, "invalid GIF header")
	}

	// Check for very large logical screen dimensions
	if len(data) >= 10 {
		width := int(data[6]) | (int(data[7]) << 8)
		height := int(data[8]) | (int(data[9]) << 8)

		if width > v.MaxWidth || height > v.MaxHeight {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("GIF logical screen dimensions too large: %dx%d", width, height))
		}
	}

	return nil
}

// containsMaliciousContent checks for potentially malicious content in image metadata
func (v *ImageValidator) containsMaliciousContent(data []byte) bool {
	maliciousPatterns := []string{
		"<script",
		"javascript:",
		"eval(",
		"document.",
		"window.",
		"alert(",
		"prompt(",
		"confirm(",
		".exe",
		".bat",
		".cmd",
		".ps1",
		".vbs",
		"data:text/html",
	}

	for _, pattern := range maliciousPatterns {
		if bytes.Contains(data, []byte(pattern)) {
			return true
		}
	}

	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
