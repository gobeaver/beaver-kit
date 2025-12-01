package krypto

import (
	"crypto/rand"
	"math/big"
)

// GenerateOTP generates a random One-Time Password (OTP) of the specified length.
// It uses a cryptographically secure random number generator to ensure the randomness
// of the generated OTP.
//
// Parameters:
//
//	length: The length of the OTP to be generated.
//
// Returns:
//
//	string: The randomly generated OTP.
func GenerateOTP(length int) string {
	// Define the character set from which the OTP will be generated
	charset := "0123456789"
	charsetLen := big.NewInt(int64(len(charset)))
	otp := make([]byte, length)
	for i := range otp {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback should never happen with crypto/rand
			panic("crypto/rand failed: " + err.Error())
		}
		otp[i] = charset[n.Int64()]
	}
	return string(otp)
}
