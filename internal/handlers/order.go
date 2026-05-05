package handlers

import (
	"FoPQer/go-fermart/internal/auth/util"
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

type LoadOrderRequest struct {
	OrderID string `validate:"required"`
}

type OrderResponse struct {
	OrderID    string     `json:"number"`
	Status     string     `json:"status"`
	Accrual    float32    `json:"accrual,omitempty"`
	UploadedAt string     `json:"uploaded_at"`
}

type OrderHandler struct{
	orderService *services.OrderService
}

func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) LoadOrder(c *echo.Context) error {
	rawOrderID, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid request body")
	}
	req := LoadOrderRequest{
		OrderID: string(rawOrderID),
	}
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	loadedOrder, err := h.orderService.LoadOrder(c.Request().Context(), userID, req.OrderID)
	if err == nil {
		return c.JSON(http.StatusAccepted, loadedOrder)
	}

	errAlreadyExists := &order.ErrOrderAlreadyExists{OrderID: req.OrderID}
	errAlreadyExistsForAnotherUser := &order.ErrOrderAlreadyExistsForAnotherUser{OrderID: req.OrderID, UserID: userID}
	errWrongOrderIDFormat := &services.ErrWrongOrderIDFormat{OrderID: req.OrderID}
	switch {
	case errors.As(err, &errAlreadyExists):
		return c.JSON(http.StatusOK, loadedOrder)
	case errors.As(err, &errAlreadyExistsForAnotherUser):
		return c.String(http.StatusConflict, "Order already exists for another user")
	case errors.As(err, &errWrongOrderIDFormat):
		slog.Error("wrong order ID format", "orderID", req.OrderID)
		return c.String(http.StatusUnprocessableEntity, "Wrong order ID format")
	default:
		slog.Error("failed to load order", "orderID", req.OrderID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func (h *OrderHandler) GetOrders(c *echo.Context) error {
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	orders, err := h.orderService.GetOrders(c.Request().Context(), userID)
	if err != nil {
		slog.Error("failed to get orders", "userID", userID, "error", err)
		return c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
	responses := make([]OrderResponse, 0)
	for _, order := range orders {
		responses = append(responses, OrderResponse{
			OrderID:    order.ID,
			Status:     string(order.Status),
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		})

	}
	return c.JSON(http.StatusOK, responses)
}