package handlers

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
)

var secretKey = []byte("your-secret-key")
type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct{
	userService *services.UserService
	claimsService *auth.ClaimsService
}

func NewAuthHandler(userService *services.UserService, claimsService *auth.ClaimsService) *AuthHandler {
	return &AuthHandler{userService: userService, claimsService: claimsService}
}

func (h *AuthHandler) Register(c *echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		c.Response().WriteHeader(http.StatusBadRequest)
		return err
	}
	userID, err := h.userService.Register(c.Request().Context(), req.Username, req.Password)
	if errors.Is(err, &user.ErrUserAlreadyExists{}) {
		c.Response().WriteHeader(http.StatusConflict)
		return err
	}
	if err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	claims := h.claimsService.CreateClaims(userID)
	token, err := h.claimsService.BuildJWTString(claims, secretKey)
	if err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	c.Response().Header().Add("Authorization", fmt.Sprintf("Bearer %s", token))
	c.Response().WriteHeader(http.StatusOK)
	return nil
}

func (h *AuthHandler) Login(c *echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		c.Response().WriteHeader(http.StatusBadRequest)
		return err
	}
	userID, err := h.userService.Login(c.Request().Context(), req.Username, req.Password)
	if errors.Is(err, &user.ErrInvalidCredentials{}) {
		c.Response().WriteHeader(http.StatusUnauthorized)
		return err
	}
	if err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	claims := h.claimsService.CreateClaims(userID)
	token, err := h.claimsService.BuildJWTString(claims, secretKey)
	if err != nil {
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	c.Response().Header().Add("Authorization", fmt.Sprintf("Bearer %s", token))
	c.Response().WriteHeader(http.StatusOK)
	return nil
}