package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte("psycho-test-secret-key-2024")

type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID int, email, role string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "psycho-test-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	
	return claims, nil
}