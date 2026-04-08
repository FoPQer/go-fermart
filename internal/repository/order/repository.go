package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
)

type Repository interface {
	LoadOrder(ctx context.Context, userID int, orderID string) error
	GetOrdersByUserID(ctx context.Context, userID int) ([]*models.Order, error)
	GetOrderByOrderID(ctx context.Context, orderID string) (*models.Order, error)
}