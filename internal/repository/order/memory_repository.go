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
	UserID  string
}

func (e *ErrOrderAlreadyExistsForAnotherUser) Error() string {
	return fmt.Sprintf("order with ID %s already exists for user %s", e.OrderID, e.UserID)
}

type MemoryRepository struct {
	orders map[string]*models.Order
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		orders: make(map[string]*models.Order),
	}
}

func (r *MemoryRepository) LoadOrder(_ context.Context, userID string, orderID string) (*models.Order, error) {
	if order, exists := r.orders[orderID]; exists {
		if order.UserID != userID {
			return nil, &ErrOrderAlreadyExistsForAnotherUser{OrderID: orderID, UserID: userID}
		}
		return order, &ErrOrderAlreadyExists{OrderID: orderID}
	}
	
	r.orders[orderID] = &models.Order{
		ID:          orderID,
		UserID:      userID,
		Status:      "REGISTERED",
		UploadedAt:  time.Now().Format(time.RFC3339),
	}
	return r.orders[orderID], nil
}

func (r *MemoryRepository) GetOrdersByUserID(_ context.Context, userID string) ([]*models.Order, error) {
	orders := make([]*models.Order, 0)
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