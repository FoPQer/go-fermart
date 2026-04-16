package handlers

import (
	"FoPQer/go-fermart/internal/auth/util"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"log"
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
	orderService *services.OrderService
}

func NewBalanceHandler(userService *services.UserService, orderService *services.OrderService) *BalanceHandler {
	return &BalanceHandler{
		userService: userService,
		orderService: orderService,
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
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	var req WithdrawRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request body")
	}
	order, err := h.orderService.LoadOrder(c.Request().Context(), userID, req.Order)
	errWrongOrderIDFormat := &services.ErrWrongOrderIDFormat{}
	if errors.As(err, &errWrongOrderIDFormat) {
		log.Printf("wrong order ID format: %s", req.Order)
		return c.String(http.StatusUnprocessableEntity, "Wrong order ID format")
	} else if err != nil {
		log.Printf("failed to load orderID %s: %v", req.Order, err)
		return c.String(http.StatusInternalServerError, "Failed to load order")
	}
	err = h.userService.DoWithdraw(c.Request().Context(), userID, req.Sum)
	errNotEnoughFunds := &services.ErrNotEnoughFunds{}
	if errors.As(err, &errNotEnoughFunds) {
		return c.String(http.StatusPaymentRequired, "Insufficient balance")
	} else if err != nil {
		log.Printf("failed to process withdrawal for userID %s: %v", userID, err)
		return c.String(http.StatusInternalServerError, "Failed to process withdrawal")
	}
	order.Accrual = req.Sum
	order.Status = "PROCESSED"

	return c.String(http.StatusOK, "Order processed successfully")
}

func (h *BalanceHandler) GetWithdrawals(c *echo.Context) error {
	return c.String(http.StatusOK, "Get Withdrawals endpoint")
}