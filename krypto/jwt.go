package krypto

import (
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

type GenerateJwt interface {
	GenerateJwtFile(claims jwt.Claims, filePath string) (string, error)
}

// GenerateJWTFile generates a JWT token using a private key
func GenerateJWTFile(jwtClaims jwt.Claims, filePath string) (string, error) {
	// Load private key
	privateKeyBytes, err := os.ReadFile("private.pem")
	if err != nil {
		return "", err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return "", err
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodRS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // Token expires in 24 hours
	// Add any additional claims as needed

	// Sign the token
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token using a public key
func ValidateJWT(tokenString string) (jwt.MapClaims, error) {
	// Load public key
	publicKeyBytes, err := os.ReadFile("public.pem")
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		return nil, err
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Make sure the signing method is the one we expect
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

//
//func encryptAndSignJwt(privateKey string, publicKey string, payload string) (string, error) {
//	privKeyBytes, _ := pem.Decode([]byte(privateKey))
//	if privKeyBytes == nil {
//		return "", fmt.Errorf("failed to decode private key")
//	}
//
//	privKey, err := x509.ParsePKCS1PrivateKey(privKeyBytes.Bytes)
//	if err != nil {
//		return "", err
//	}
//
//	pubKeyBytes, _ := pem.Decode([]byte(publicKey))
//	if pubKeyBytes == nil {
//		return "", fmt.Errorf("failed to decode public key")
//	}
//
//	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes.Bytes)
//	if err != nil {
//		return "", err
//	}
//
//	// Generate a random symmetric key for encryption
//	symmetricKey := make([]byte, 32)
//	_, err = rand.Read(symmetricKey)
//	if err != nil {
//		return "", err
//	}
//
//	// Create the JWT token
//	token := jwt.New(jwt.SigningMethodRS256)
//	token.Claims = jwt.MapClaims{
//		"exp": time.Now().Add(time.Hour * 1).Unix(),
//		"iat": time.Now().Unix(),
//		"sub": payload,
//	}
//
//	// Sign the token with the private key
//	signedToken, err := token.SignedString(privKey)
//	if err != nil {
//		return "", err
//	}
//
//	// Encrypt the symmetric key with the public key
//	encryptedSymmetricKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey.(*rsa.PublicKey), symmetricKey)
//	if err != nil {
//		return "", err
//	}
//
//	// Construct the final encrypted and signed JWT
//	finalJwt := signedToken + "." + string(encryptedSymmetricKey)
//
//	return finalJwt, nil
//}
//
//func verifyAndDecryptJwt(privateKey string, publicKey string, jwtString string) (string, error) {
//	privKeyBytes, _ := pem.Decode([]byte(privateKey))
//	if privKeyBytes == nil {
//		return "", fmt.Errorf("failed to decode private key")
//	}
//
//	privKey, err := x509.ParsePKCS1PrivateKey(privKeyBytes.Bytes)
//	if err != nil {
//		return "", err
//	}
//
//	pubKeyBytes, _ := pem.Decode([]byte(publicKey))
//	if pubKeyBytes == nil {
//		return "", fmt.Errorf("failed to decode public key")
//	}
//
//	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes.Bytes)
//	if err != nil {
//		return "", err
//	}
//
//	parts := strings.Split(jwtString, ".")
//	if len(parts) != 2 {
//		return "", fmt.Errorf("invalid JWT format")
//	}
//
//	// Verify the JWT signature using the public key
//	token, err := jwt.Parse(parts[0]+"."+parts[1], func(token *jwt.Token) (interface{}, error) {
//		return pubKey, nil
//	})
//	if err != nil {
//		return "", err
//	}
//
//	// Decrypt the symmetric key using the private key
//	encryptedSymmetricKey := []byte(parts[1])
//	decryptedSymmetricKey, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, encryptedSymmetricKey)
//	if err != nil {
//		return "", err
//	}
//
//	// Use the decrypted symmetric key to decrypt the payload
//	block, _ := pem.Decode([]byte(decryptedSymmetricKey))
//	if block == nil {
//		return "", fmt.Errorf("failed to decode symmetric key")
//	}
//
//	cipherText := []byte(parts[0])
//	plainText, err := aesDecrypt(cipherText, block.Bytes)
//	if err != nil {
//		return "", err
//	}
//
//	return string(plainText), nil
//}
//
//func main() {
//	privateKey := `
//		-----BEGIN RSA PRIVATE KEY-----
//		... (your private key in PEM format) ...
//		-----END RSA PRIVATE KEY-----
//	`
//
//	publicKey := `
//		-----BEGIN PUBLIC KEY-----
//		... (your public key in PEM format) ...
//		-----END PUBLIC KEY-----
//	`
//
//	payload := "your_payload_data"
//
//	// Encrypt and sign JWT
//	encryptedAndSignedJwt, err := encryptAndSignJwt(privateKey, publicKey, payload)
//	if err != nil {
//		fmt.Println("Error encrypting and signing JWT:", err)
//		return
//	}
//	fmt.Println("Encrypted and signed JWT:", encryptedAndSignedJwt)
//
//	// Verify and decrypt JWT
//	decryptedPayload, err := verifyAndDecryptJwt(privateKey, publicKey, encryptedAndSignedJwt)
//	if err != nil {
//		fmt.Println("Error verifying and decrypting JWT:", err)
//		return
//	}
//	fmt.Println("Decrypted payload:", decryptedPayload)
//}
