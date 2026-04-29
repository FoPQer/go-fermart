package auth

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func NewClaims(userID string) *Claims {
	return &Claims{
		UserID: userID,
	}
}