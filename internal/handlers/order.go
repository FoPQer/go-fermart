package handlers

import (
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
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
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		log.Printf("failed to get user from context: %v", err)
		return c.String(http.StatusUnauthorized, "Failed to get user from context")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("failed to parse claims from token: %v", err)
		return c.String(http.StatusUnauthorized, "Failed to parse claims from token")
	}
	userID, ok := claims["UserID"].(string)
	if !ok {
		log.Printf("failed to parse userID from claims: %v", claims["UserID"])
		return c.String(http.StatusUnauthorized, "Failed to parse userID from claims")
	}
	loadedOrder, err := h.orderService.LoadOrder(c.Request().Context(), userID, req.OrderID)
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
	token, err := echo.ContextGet[*jwt.Token](c, "user")
	if err != nil {
		log.Printf("failed to get user from context: %v", err)
		return c.String(http.StatusUnauthorized, "Failed to get user from context")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("failed to parse claims from token: %v", err)
		return c.String(http.StatusUnauthorized, "Failed to parse claims from token")
	}
	userID, ok := claims["UserID"].(string)
	if !ok {
		log.Printf("failed to parse userID from claims: %v", claims["UserID"])
		return c.String(http.StatusUnauthorized, "Failed to parse userID from claims")
	}
	orders, err := h.orderService.GetOrders(c.Request().Context(), userID)
	if err != nil {
		log.Printf("failed to get orders for userID %s: %v", userID, err)
		return c.String(http.StatusInternalServerError, "Failed to get orders")
	}
	responses := make([]OrderResponse, len(orders))
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