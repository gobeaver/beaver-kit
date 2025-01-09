package krypto_test

import (
	"github.com/gobeaver/beaver-kit/krypto"
	"testing"
)

func TestArgon2idHashPassword(t *testing.T) {
	// Define a test password
	testPassword := "password123"

	// Hash the test password
	hashedPassword, err := krypto.Argon2idHashPassword(testPassword)
	if err != nil {
		t.Fatalf("Error hashing password: %v", err)
	}

	// Verify the hash is not empty
	if hashedPassword == "" {
		t.Fatal("Hashed password is empty")
	}

	t.Logf("Hashed password: %s", hashedPassword)
}

func TestArgon2idVerifyPassword(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		matchPassword  string
		wantMatch      bool
		wantErrOnHash  bool
		wantErrOnMatch bool
	}{
		{
			name:          "Correct password",
			password:      "secure_password_123",
			matchPassword: "secure_password_123",
			wantMatch:     true,
		},
		{
			name:          "Incorrect password",
			password:      "secure_password_123",
			matchPassword: "wrong_password",
			wantMatch:     false,
		},
		{
			name:          "Empty password",
			password:      "",
			matchPassword: "",
			wantMatch:     true,
		},
		{
			name:          "Long password",
			password:      "this_is_a_very_long_password_that_exceeds_typical_input_fields_but_should_still_work_properly_with_argon2id_hashing_algorithm",
			matchPassword: "this_is_a_very_long_password_that_exceeds_typical_input_fields_but_should_still_work_properly_with_argon2id_hashing_algorithm",
			wantMatch:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash the password
			hashedPassword, err := krypto.Argon2idHashPassword(tt.password)
			if (err != nil) != tt.wantErrOnHash {
				t.Fatalf("Argon2idHashPassword() error = %v, wantErr %v", err, tt.wantErrOnHash)
			}
			if tt.wantErrOnHash {
				return
			}

			// Verify the password
			match, err := krypto.Argon2idVerifyPassword(tt.matchPassword, hashedPassword)
			if (err != nil) != tt.wantErrOnMatch {
				t.Fatalf("Argon2idVerifyPassword() error = %v, wantErr %v", err, tt.wantErrOnMatch)
			}
			if tt.wantErrOnMatch {
				return
			}

			if match != tt.wantMatch {
				t.Errorf("Argon2idVerifyPassword() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

func TestArgon2idVerifyPassword_InvalidHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "Invalid hash format (missing parts)",
			password: "test",
			hash:     "d4096$3$6",
			wantErr:  true,
		},
		{
			name:     "Invalid memory parameter",
			password: "test",
			hash:     "invalid$3$6$YWJjZGVmZ2hpamtsbW5vcA==$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr:  true,
		},
		{
			name:     "Invalid salt encoding",
			password: "test",
			hash:     "d4096$3$6$invalid-base64$$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=",
			wantErr:  true,
		},
		{
			name:     "Invalid hash encoding",
			password: "test",
			hash:     "d4096$3$6$YWJjZGVmZ2hpamtsbW5vcA==$invalid-base64",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := krypto.Argon2idVerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Argon2idVerifyPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
