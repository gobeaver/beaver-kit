package krypto

import (
	"math/rand"
	"time"
)

// RandomDelayWithRange introduces a non-deterministic delay in program execution to help mitigate
// timing-based attacks. This is particularly useful when:
//
//  1. Processing sensitive operations (login attempts, password resets, etc.) to mask
//     whether the operation was successful or failed based on response time
//  2. Preventing rate limiting bypass attempts by making the exact execution time unpredictable
//  3. Mitigating timing side-channel attacks that could leak information about the
//     system's internal state or operations
//  4. Adding jitter to API responses to prevent attackers from inferring server-side
//     processing patterns
//
// Parameters:
//
//	min: minimum delay in seconds
//	max: maximum delay in seconds
//
// Example usage:
//
//	// Add random delay between 0.5 and 2 seconds after failed login attempts
//	if loginAttemptFailed {
//	    delayRandomly(0.5, 2)
//	}
//
//	// Add jitter to API response timing
//	func handleSensitiveRequest() {
//	    delayRandomly(0.1, 0.5)
//	    // ... process request
//	}
//
// Security considerations:
//   - Ensure the random number generator is properly seeded (rand.Seed())
//   - Consider the tradeoff between security and user experience when setting delays
//   - Be aware that very short delays may not effectively mask timing differences
//   - For cryptographic operations, consider using crypto/rand instead of math/rand
func RandomDelayWithRange(minSec, maxSec float64) {
	// Convert min and max seconds to milliseconds
	minMillis := minSec * 1000
	maxMillis := maxSec * 1000

	// Generate a random number between min and max milliseconds
	// Using math/rand is acceptable here as this is for timing jitter, not security
	randomMillis := rand.Float64()*(maxMillis-minMillis) + minMillis //nolint:gosec // timing jitter doesn't require crypto strength

	// Convert milliseconds to duration and sleep
	time.Sleep(time.Duration(randomMillis) * time.Millisecond)
}
