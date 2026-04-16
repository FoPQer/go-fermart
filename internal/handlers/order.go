package handlers

import (
	"FoPQer/go-fermart/internal/auth/util"
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
)

type LoadOrderRequest struct {
	OrderID string `validate:"required"`
}

type GetOrdersResponse struct {
	Orders []OrderResponse `json:"orders"`
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
	errAlreadyExists := &order.ErrOrderAlreadyExists{}
	errAlreadyExistsForAnotherUser := &order.ErrOrderAlreadyExistsForAnotherUser{}
	errWrongOrderIDFormat := &services.ErrWrongOrderIDFormat{}
	if errors.As(err, &errAlreadyExists) {
		return c.JSON(http.StatusOK, loadedOrder)
	} else if errors.As(err, &errAlreadyExistsForAnotherUser) {
		return c.String(http.StatusConflict, "Order already exists for another user")
	} else if errors.As(err, &errWrongOrderIDFormat) {
		log.Printf("wrong order ID format: %s", req.OrderID)
		return c.String(http.StatusUnprocessableEntity, "Wrong order ID format")
	} else if err != nil {
		log.Printf("failed to load orderID %s: %v", req.OrderID, err)
		return c.String(http.StatusInternalServerError, "Failed to load order")
	}
	return c.JSON(http.StatusAccepted, loadedOrder)
}

func (h *OrderHandler) GetOrders(c *echo.Context) error {
	userID, err := util.GetUserIDFromToken(c)
	if err != nil {
		return c.String(http.StatusUnauthorized, "Failed to get user from token")
	}
	orders, err := h.orderService.GetOrders(c.Request().Context(), userID)
	if err != nil {
		log.Printf("failed to get orders for userID %s: %v", userID, err)
		return c.String(http.StatusInternalServerError, "Failed to get orders")
	}
	responses := make([]OrderResponse, 0)
	for _, order := range orders {
		responses = append(responses, OrderResponse{
			OrderID:    order.ID,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt,
		})

	}
	return c.JSON(http.StatusOK, GetOrdersResponse{Orders: responses})
}

func (h *OrderHandler) GetOrderInfo(c *echo.Context) error {
	return c.String(http.StatusOK, "Get Order by ID endpoint")
}