package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPin hashes a PIN using bcrypt (same as password but semantically different)
func HashPin(pin string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPin compares a plaintext PIN with a hashed PIN
func CheckPin(pin, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin))
	return err == nil
}