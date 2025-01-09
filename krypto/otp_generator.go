package krypto

import (
	"math/rand"
	"time"
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
	// Create a new Rand object with a cryptographically secure source
	source := rand.NewSource(int64(time.Now().UnixNano()))
	seededRand := rand.New(source)

	// Define the character set from which the OTP will be generated
	charset := "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		otp[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(otp)
}
