package krypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type PublicPrivatePair struct {
	PrivateKey string
	PublicKey  string
}

// GenerateRSAKeyPair generates a pair of RSA public and private keys with
// a key size of 2048 bits. It returns the public and private keys encoded
// in PEM format.
//
// Returns:
//
//	PublicPrivatePair: A struct containing the RSA public and private keys
//	error: An error, if any, encountered during the key pair generation process.
func GenerateRSAKeyPair() (PublicPrivatePair, error) {
	// Generate a private key with RSA
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return PublicPrivatePair{}, err
	}

	// Extract the public key from the private key
	publicKey := &privateKey.PublicKey

	// Encode the private key to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Encode the public key to PEM format
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(publicKey),
	})

	return PublicPrivatePair{
		PrivateKey: string(privateKeyPEM),
		PublicKey:  string(publicKeyPEM),
	}, nil

}

// ValidateRSAKeyPair validates an RSA key pair by encrypting and decrypting a message.
// It returns true if the validation succeeds, indicating that the key pair is valid.
// Otherwise, it returns false.
//
// Parameters:
//
//	keyPair: The RSA public and private keys encoded in PEM format.
//
// Returns:
//
//	bool: A boolean indicating whether the key pair is valid.
//	error: An error, if any, encountered during the validation process.
func ValidateRSAKeyPair(keyPair PublicPrivatePair) (bool, error) {
	// Parse the public key
	publicKeyBlock, _ := pem.Decode([]byte(keyPair.PublicKey))
	if publicKeyBlock == nil || publicKeyBlock.Type != "RSA PUBLIC KEY" {
		return false, errors.New("invalid public key format")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return false, err
	}

	// Parse the private key
	privateKeyBlock, _ := pem.Decode([]byte(keyPair.PrivateKey))
	if privateKeyBlock == nil || privateKeyBlock.Type != "RSA PRIVATE KEY" {
		return false, errors.New("invalid private key format")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return false, err
	}

	// Encrypt and decrypt a test message
	message := []byte("This is a test message for RSA key pair validation.")
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, message)
	if err != nil {
		return false, err
	}
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encrypted)
	if err != nil {
		return false, err
	}

	// Compare the original message with the decrypted message
	if !bytes.Equal(message, decrypted) {
		return false, errors.New("decrypted message does not match original message")
	}

	return true, nil
}

//fmt.Println(privateKey)
//
//// Save the keys to files (you can modify the filenames)
//if err := os.WriteFile("private_keey.pem", privateKeyPEM, 0600); err != nil {
//	fmt.Println(err)
//	return err
//}
//
//if err := os.WriteFile("public_keey.pem", publicKeyPEM, 0644); err != nil {
//	return err
//}
//
//return nil
