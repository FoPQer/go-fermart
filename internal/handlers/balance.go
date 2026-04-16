package handlers

import (
	"FoPQer/go-fermart/internal/auth/util"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

type GetBalanceResponse struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type WithdrawRequest struct {
	Order string  `json:"order" validate:"required"`
	Sum   float32 `json:"sum"   validate:"required,gt=0"`
}

type WithdrawResponse struct {
	Order 	    string     `json:"order"`
	Sum   	    float32    `json:"sum"`
	ProcessedAt time.Time  `json:"processed_at"`
}

type GetWithdrawalsResponse struct {
	Withdrawals []WithdrawResponse `json:"withdrawals"`
}


type BalanceHandler struct{
	userService *services.UserService
}

func NewBalanceHandler(userService *services.UserService) *BalanceHandler {
	return &BalanceHandler{
		userService: userService,
	}
}

func (h *BalanceHandler) GetBalance(c *echo.Context) error {
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	errUserNotFound := &user.ErrUserNotFound{}
	user, err := h.userService.GetUserInfo(c.Request().Context(), userID)
	if errors.As(err, &errUserNotFound) {
		return c.String(http.StatusUnauthorized, "User not found")
	} else if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to get user info")
	}
	resp := GetBalanceResponse{
		Current:   user.Balance,
		Withdrawn: user.SumWithdrawn,
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *BalanceHandler) Withdraw(c *echo.Context) error {
	return c.String(http.StatusOK, "Withdraw endpoint")
}

func (h *BalanceHandler) GetWithdrawals(c *echo.Context) error {
	return c.String(http.StatusOK, "Get Withdrawals endpoint")
}