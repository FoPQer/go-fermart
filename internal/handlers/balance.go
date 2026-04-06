package handlers

import (
	"time"

	"github.com/labstack/echo/v5"
)

type GetBalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	Order string  `json:"order" validate:"required"`
	Sum   float64 `json:"sum"   validate:"required,gt=0"`
}

type WithdrawResponse struct {
	Order 	    string     `json:"order"`
	Sum   	    float64    `json:"sum"`
	ProcessedAt time.Time  `json:"processed_at"`
}

type GetWithdrawalsResponse struct {
	Withdrawals []WithdrawResponse `json:"withdrawals"`
}


type BalanceHandler struct{}

func NewBalanceHandler() *BalanceHandler {
	return &BalanceHandler{}
}

func (h *BalanceHandler) GetBalance(c *echo.Context) error {
	return c.String(200, "Get Balance endpoint")
}

func (h *BalanceHandler) Withdraw(c *echo.Context) error {
	return c.String(200, "Withdraw endpoint")
}

func (h *BalanceHandler) GetWithdrawals(c *echo.Context) error {
	return c.String(200, "Get Withdrawals endpoint")
}