package krypto

import (
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Token string `json:"token"`
	jwt.RegisteredClaims
}

func NewHs256AccessToken(claims UserClaims) (string, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return accessToken.SignedString([]byte(os.Getenv("JWT_HS256_KEY"))) //nolint:forbidigo // legacy API, use config-based JWT functions for new code
}

func NewHs256RefreshToken(claims jwt.RegisteredClaims) (string, error) {
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return refreshToken.SignedString([]byte(os.Getenv("JWT_HS256_KEY"))) //nolint:forbidigo // legacy API, use config-based JWT functions for new code
}

func ParseHs256AccessToken(accessToken string) (*UserClaims, error) {
	parsedAccessToken, err := jwt.ParseWithClaims(accessToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_HS256_KEY")), nil //nolint:forbidigo // legacy API, use config-based JWT functions for new code
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := parsedAccessToken.Claims.(*UserClaims); ok && parsedAccessToken.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func ParseHs256RefreshToken(refreshToken string) *jwt.RegisteredClaims {
	parsedRefreshToken, _ := jwt.ParseWithClaims(refreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_HS256_KEY")), nil //nolint:forbidigo // legacy API, use config-based JWT functions for new code
	})

	return parsedRefreshToken.Claims.(*jwt.RegisteredClaims)
}
