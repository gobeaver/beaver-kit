package oauth

import (
	"encoding/base64"
	"strings"
)

// base64URLDecode decodes a base64url encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}
