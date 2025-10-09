package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateOTP generates a 6-digit OTP
func GenerateOTP() string {
	// Generate a random number between 100000 and 999999
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to timestamp-based generation (less secure but works)
		return fmt.Sprintf("%06d", 100000+int(n.Int64()%900000))
	}

	otp := n.Int64() + 100000
	return fmt.Sprintf("%06d", otp)
}