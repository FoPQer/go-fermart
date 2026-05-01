package handlers

import (
	"FoPQer/go-fermart/internal/auth/util"
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"log/slog"
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
	ProcessedAt *time.Time  `json:"processed_at"`
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
		slog.Error("failed to get user info", "userID", userID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
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

	var errWrongOrderIDFormat *services.ErrWrongOrderIDFormat
	if err != nil {
		if errors.As(err, &errWrongOrderIDFormat) {
			slog.Info("wrong order ID format", "orderID", req.Order)
			return c.String(http.StatusUnprocessableEntity, "Wrong order ID format")
		}
		slog.Info("failed to load orderID", "orderID", req.Order, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	err = h.userService.DoWithdraw(c.Request().Context(), userID, req.Sum)

	if err != nil {
		if errors.Is(err, models.ErrNotEnoughFunds) {
			return c.String(http.StatusPaymentRequired, "Insufficient balance")
		}
		slog.Info("failed to process withdrawal for userID", "userID", userID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	order.Withdrawn = req.Sum
	if err := h.orderService.UpdateOrder(c.Request().Context(), order); err != nil {
		slog.Error("failed to update order with withdrawal info", "orderID", order.ID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	return c.JSON(http.StatusOK, order)
}

func (h *BalanceHandler) GetWithdrawals(c *echo.Context) error {
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	withdrawals, err := h.orderService.GetWithdrawals(c.Request().Context(), userID)
	if err != nil {
		slog.Error("failed to get withdrawals", "userID", userID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	responses := make([]WithdrawResponse, 0)
	for _, w := range withdrawals {
		responses = append(responses, WithdrawResponse{
			Order:       w.ID,
			Sum:         w.Withdrawn,
			ProcessedAt: w.ProcessedAt,
		})
	}
	if len(responses) > 0 {
		return c.JSON(http.StatusOK, responses)
	}
	return c.NoContent(http.StatusNoContent)
}