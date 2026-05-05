package handlers

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

type RegisterRequest struct {
	Username string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct{
	userService *services.UserService
	claimsService *auth.ClaimsService
	cnf *config.Config
}

func NewAuthHandler(userService *services.UserService, claimsService *auth.ClaimsService, cnf *config.Config) *AuthHandler {
	return &AuthHandler{userService: userService, claimsService: claimsService, cnf: cnf}
}

func (h *AuthHandler) Register(c *echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		c.Response().WriteHeader(http.StatusBadRequest)
		return err
	}
	userID, err := h.userService.Register(c.Request().Context(), req.Username, req.Password)
	var alreadyExistsErr *user.ErrUserAlreadyExists
	if errors.As(err, &alreadyExistsErr) {
		c.Response().WriteHeader(http.StatusConflict)
		return err
	}
	if err != nil {
		slog.Error("failed to register user", "error", err)
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	claims := h.claimsService.CreateClaims(userID)
	token, err := h.claimsService.BuildJWTString(claims, h.cnf.GetSecretKey())
	if err != nil {
		slog.Error("failed to build JWT string", "error", err)
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
	var invalidCredsErr *user.ErrInvalidCredentials
	if errors.As(err, &invalidCredsErr) {
		c.Response().WriteHeader(http.StatusUnauthorized)
		return err
	}
	if err != nil {
		slog.Error("failed to login user", "error", err)
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	claims := h.claimsService.CreateClaims(userID)
	token, err := h.claimsService.BuildJWTString(claims, h.cnf.GetSecretKey())
	if err != nil {
		slog.Error("failed to build JWT string", "error", err)
		c.Response().WriteHeader(http.StatusInternalServerError)
		return err
	}

	c.Response().Header().Add("Authorization", fmt.Sprintf("Bearer %s", token))
	c.Response().WriteHeader(http.StatusOK)
	return nil
}