package handlers

import (
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

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

func (h *OrderHandler) LoadOrder(c *echo.Context) error {
	return c.String(200, "Load Order endpoint")
}

func (h *OrderHandler) GetOrders(c *echo.Context) error {
	return c.String(200, "Get Orders endpoint")
}

func (h *OrderHandler) GetOrderInfo(c *echo.Context) error {
	return c.String(200, "Get Order by ID endpoint")
}