package handlers

import "github.com/labstack/echo/v5"

type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Register(c *echo.Context) error {
	return c.String(200, "Register endpoint")
}

func (h *AuthHandler) Login(c *echo.Context) error {
	return c.String(200, "Login endpoint")
}