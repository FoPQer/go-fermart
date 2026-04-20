package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
)

type Repository interface {
	LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error)
}