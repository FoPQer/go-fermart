package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"fmt"
	"time"
)

type ErrOrderAlreadyExists struct {
	OrderID string
}

func (e *ErrOrderAlreadyExists) Error() string {
	return fmt.Sprintf("order with ID %s already exists", e.OrderID)
}

type ErrOrderAlreadyExistsForAnotherUser struct {
	OrderID string
	UserID  int
}

func (e *ErrOrderAlreadyExistsForAnotherUser) Error() string {
	return fmt.Sprintf("order with ID %s already exists for user %d", e.OrderID, e.UserID)
}

type MemoryRepository struct {
	orders map[string]*models.Order
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders: make(map[string]*models.Order),
	}
}

func (r *MemoryRepository) LoadOrder(_ context.Context, userID int, orderID string) (*models.Order, error) {
	if order, exists := r.orders[orderID]; exists {
		if order.UserID != userID {
			return nil, &ErrOrderAlreadyExistsForAnotherUser{OrderID: orderID, UserID: userID}
		}
		return nil, &ErrOrderAlreadyExists{OrderID: orderID}
	}
	r.orders[orderID] = &models.Order{
		ID:          orderID,
		UserID:      userID,
		Status:      "REGISTERED",
		UploadedAt:  time.Now().Format(time.RFC3339),
	}
	return r.orders[orderID], nil
}

func (r *MemoryRepository) GetOrdersByUserID(_ context.Context, userID int) ([]*models.Order, error) {
	var orders []*models.Order
	for _, order := range r.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (r *MemoryRepository) GetOrderByOrderID(_ context.Context, orderID string) (*models.Order, error) {
	order, exists := r.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order with ID %s not found", orderID)
	}
	return order, nil
}