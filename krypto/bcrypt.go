package krypto

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptHashPassword generates a bcrypt hash from the provided password
// using a cost factor of 12. It returns the hashed password as a string.
//
// Parameters:
//
//	password: The plaintext password to be hashed.
//
// Returns:
//
//	string: The hashed password.
//	error: An error, if any, encountered during the hashing process.
func BcryptHashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// BcryptCheckPasswordHash compares a bcrypt hashed password with a plaintext password
// to check if they match.
//
// Parameters:
//   password: The plaintext password to be checked.
//   hash: The bcrypt hashed password to be compared with the plaintext password.
//
// Returns:
//   bool: A boolean indicating whether the plaintext password matches the bcrypt hash.

func BcryptCheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
