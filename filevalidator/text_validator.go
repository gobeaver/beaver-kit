package filevalidator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// JSONValidator validates JSON files
type JSONValidator struct {
	MaxSize  int64
	MaxDepth int // Maximum nesting depth (0 = unlimited)
}

// DefaultJSONValidator creates a JSON validator with sensible defaults
func DefaultJSONValidator() *JSONValidator {
	return &JSONValidator{
		MaxSize:  50 * MB,
		MaxDepth: 100, // Prevent stack overflow from deeply nested JSON
	}
}

// ValidateContent validates JSON by attempting to decode it
func (v *JSONValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Use streaming decoder for memory efficiency
	decoder := json.NewDecoder(reader)

	// Validate by decoding into interface{}
	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("invalid JSON: %v", err))
	}

	// Check depth if configured
	if v.MaxDepth > 0 {
		depth := v.measureDepth(data, 0)
		if depth > v.MaxDepth {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("JSON nesting depth %d exceeds maximum %d", depth, v.MaxDepth))
		}
	}

	return nil
}

func (v *JSONValidator) measureDepth(data interface{}, current int) int {
	switch val := data.(type) {
	case map[string]interface{}:
		maxChild := current
		for _, child := range val {
			d := v.measureDepth(child, current+1)
			if d > maxChild {
				maxChild = d
			}
		}
		return maxChild
	case []interface{}:
		maxChild := current
		for _, child := range val {
			d := v.measureDepth(child, current+1)
			if d > maxChild {
				maxChild = d
			}
		}
		return maxChild
	default:
		return current
	}
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *JSONValidator) SupportedMIMETypes() []string {
	return []string{
		"application/json",
		"text/json",
	}
}

// XMLValidator validates XML files
type XMLValidator struct {
	MaxSize  int64
	MaxDepth int  // Maximum nesting depth
	AllowDTD bool // Allow DTD declarations (can be dangerous)
}

// DefaultXMLValidator creates an XML validator with secure defaults
func DefaultXMLValidator() *XMLValidator {
	return &XMLValidator{
		MaxSize:  50 * MB,
		MaxDepth: 100,
		AllowDTD: false, // DTD disabled by default (XXE protection)
	}
}

// ValidateContent validates XML structure
func (v *XMLValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	// Check for DTD if not allowed
	if !v.AllowDTD {
		// Read beginning to check for DTD
		buf := make([]byte, 1024)
		n, _ := io.ReadFull(reader, buf)
		buf = buf[:n]

		if bytes.Contains(buf, []byte("<!DOCTYPE")) || bytes.Contains(buf, []byte("<!ENTITY")) {
			return NewValidationError(ErrorTypeContent,
				"XML DTD/ENTITY declarations not allowed (XXE protection)")
		}

		// Reconstruct reader
		reader = io.MultiReader(bytes.NewReader(buf), reader)
	}

	// Parse XML using streaming decoder
	decoder := xml.NewDecoder(reader)
	depth := 0
	maxDepth := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("invalid XML: %v", err))
		}

		switch token.(type) {
		case xml.StartElement:
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
			if v.MaxDepth > 0 && depth > v.MaxDepth {
				return NewValidationError(ErrorTypeContent,
					fmt.Sprintf("XML nesting depth exceeds maximum %d", v.MaxDepth))
			}
		case xml.EndElement:
			depth--
		}
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *XMLValidator) SupportedMIMETypes() []string {
	return []string{
		"application/xml",
		"text/xml",
	}
}

// CSVValidator validates CSV files
type CSVValidator struct {
	MaxSize       int64
	MaxRows       int  // Maximum number of rows (0 = unlimited)
	MaxColumns    int  // Maximum number of columns (0 = unlimited)
	MaxLineLength int  // Maximum length of a single line
	Delimiter     rune // CSV delimiter (default: comma)
	RequireUTF8   bool // Require valid UTF-8 encoding
}

// DefaultCSVValidator creates a CSV validator with sensible defaults
func DefaultCSVValidator() *CSVValidator {
	return &CSVValidator{
		MaxSize:       100 * MB,
		MaxRows:       1000000, // 1 million rows
		MaxColumns:    1000,
		MaxLineLength: 1 * 1024 * 1024, // 1MB per line
		Delimiter:     ',',
		RequireUTF8:   true,
	}
}

// ValidateContent validates CSV by scanning rows
func (v *CSVValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	scanner := bufio.NewScanner(reader)

	// Set max line length
	if v.MaxLineLength > 0 {
		scanner.Buffer(make([]byte, v.MaxLineLength), v.MaxLineLength)
	}

	rowCount := 0
	var headerColumns int

	for scanner.Scan() {
		rowCount++

		if v.MaxRows > 0 && rowCount > v.MaxRows {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("CSV rows %d exceed maximum %d", rowCount, v.MaxRows))
		}

		line := scanner.Text()

		// Check UTF-8 if required
		if v.RequireUTF8 && !utf8.ValidString(line) {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("invalid UTF-8 at row %d", rowCount))
		}

		// Count columns (simple split - doesn't handle quoted fields perfectly)
		columns := v.countColumns(line)

		if rowCount == 1 {
			headerColumns = columns
		}

		if v.MaxColumns > 0 && columns > v.MaxColumns {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("CSV columns %d exceed maximum %d at row %d", columns, v.MaxColumns, rowCount))
		}
	}

	if err := scanner.Err(); err != nil {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("CSV read error: %v", err))
	}

	// Must have at least one row
	if rowCount == 0 {
		return NewValidationError(ErrorTypeContent, "empty CSV file")
	}

	_ = headerColumns // Could be used for consistency checking if needed

	return nil
}

func (v *CSVValidator) countColumns(line string) int {
	if len(line) == 0 {
		return 0
	}

	count := 1
	inQuotes := false

	for _, ch := range line {
		if ch == '"' {
			inQuotes = !inQuotes
		} else if ch == v.Delimiter && !inQuotes {
			count++
		}
	}

	return count
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *CSVValidator) SupportedMIMETypes() []string {
	return []string{
		"text/csv",
		"application/csv",
		"text/comma-separated-values",
	}
}

// PlainTextValidator validates plain text files
type PlainTextValidator struct {
	MaxSize     int64
	RequireUTF8 bool
}

// DefaultPlainTextValidator creates a text validator
func DefaultPlainTextValidator() *PlainTextValidator {
	return &PlainTextValidator{
		MaxSize:     10 * MB,
		RequireUTF8: true,
	}
}

// ValidateContent validates text encoding
func (v *PlainTextValidator) ValidateContent(reader io.Reader, size int64) error {
	if size > v.MaxSize {
		return NewValidationError(ErrorTypeContent,
			fmt.Sprintf("file size %d exceeds maximum %d", size, v.MaxSize))
	}

	if !v.RequireUTF8 {
		return nil // No validation needed
	}

	// Check UTF-8 validity by reading in chunks
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if !utf8.Valid(buf[:n]) {
				return NewValidationError(ErrorTypeContent, "invalid UTF-8 encoding")
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return NewValidationError(ErrorTypeContent,
				fmt.Sprintf("read error: %v", err))
		}
	}

	return nil
}

// SupportedMIMETypes returns MIME types this validator handles
func (v *PlainTextValidator) SupportedMIMETypes() []string {
	return []string{
		"text/plain",
	}
}
