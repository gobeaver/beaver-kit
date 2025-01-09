package krypto

import (
	"strings"
	"testing"
)

func TestGenerateOTP(t *testing.T) {
	length := 6 // Example length

	// Generate OTP
	otp := GenerateOTP(length)

	// Check if the length of generated OTP matches the expected length
	if len(otp) != length {
		t.Errorf("Generated OTP length is %d, expected %d", len(otp), length)
	}

	// Check if the generated OTP contains only digits
	for _, char := range otp {
		if !strings.Contains("0123456789", string(char)) {
			t.Errorf("Generated OTP contains invalid character: %s", string(char))
		}
	}
}
