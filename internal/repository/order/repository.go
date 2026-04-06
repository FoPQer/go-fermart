package order

import "FoPQer/go-fermart/internal/models"

type Repository interface {
	LoadOrder(userID int, orderID string) error
	GetOrdersByUserID(userID int) ([]*models.Order, error)
	GetOrdersByOrderID(orderID string) (*models.Order, error)
}