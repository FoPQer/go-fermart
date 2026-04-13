package handlers

import (
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

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
	Accrual    float64    `json:"accrual,omitempty"`
	UploadedAt time.Time  `json:"uploaded_at"`
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
	loadedOrder, err := h.orderService.LoadOrder(c.Request().Context(), c.Get("userID").(int), req.OrderID)
	if errors.Is(err, &order.ErrOrderAlreadyExists{}) {
		return c.JSON(http.StatusOK, loadedOrder)
	} else if errors.Is(err, &order.ErrOrderAlreadyExistsForAnotherUser{}) {
		return c.String(http.StatusConflict, "Order already exists for another user")
	} else if errors.Is(err, &services.ErrWrongOrderIDFormat{}) {
		log.Printf("wrong order ID format: %s", req.OrderID)
		return c.String(http.StatusUnprocessableEntity, "Wrong order ID format")
	} else if err != nil {
		log.Printf("failed to load orderID %s: %v", req.OrderID, err)
		return c.String(http.StatusInternalServerError, "Failed to load order")
	}
	return c.JSON(http.StatusAccepted, loadedOrder)
}

func (h *OrderHandler) GetOrders(c *echo.Context) error {
	return c.String(http.StatusOK, "Get Orders endpoint")
}

func (h *OrderHandler) GetOrderInfo(c *echo.Context) error {
	return c.String(http.StatusOK, "Get Order by ID endpoint")
}