package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

type ErrInvalidToken struct {
	Err string
}

func (e *ErrInvalidToken) Error() string {
	return e.Err
}

type ErrMissingUserID struct {
	Claims *Claims
}

func (e *ErrMissingUserID) Error() string {
	return fmt.Sprintf("missing user ID in token with claims: %v", e.Claims)
}

type ClaimsService struct {
}

func NewClaimsService() *ClaimsService {
	return &ClaimsService{}
}

func (s *ClaimsService) CreateClaims(userID string) *Claims {
	return NewClaims(userID)
}

func (s *ClaimsService) BuildJWTString(claims *Claims, secretKey []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func (s *ClaimsService) GetUserIDFromClaims(claims *Claims) string {
	return claims.UserID
}

func (s *ClaimsService) GetUserIDFromJWTString(tokenString string, secretKey []byte) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return "", &ErrInvalidToken{Err: "invalid token"}
	}
	if claims.UserID == "" {
		return "", &ErrMissingUserID{Claims: claims}
	}
	return claims.UserID, nil
}