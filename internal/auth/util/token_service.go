package util

import (
	"fmt"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func GetUserIDFromToken(c *echo.Context) (string, error) {
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		log.Printf("failed to get user from context: %v", err)
		return "", fmt.Errorf("failed to get user from context: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("failed to parse claims from token: %v", err)
		return "", fmt.Errorf("failed to parse claims from token: %v", err)
	}
	userID, ok := claims["UserID"].(string)
	if !ok {
		log.Printf("failed to parse userID from claims: %v", claims["UserID"])
		return "", fmt.Errorf("failed to parse userID from claims: %v", claims["UserID"])
	}
	return userID, nil
}