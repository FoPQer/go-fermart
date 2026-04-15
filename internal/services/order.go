package services

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/order"
	"context"
	"fmt"
)

type ErrWrongOrderIDFormat struct {
	OrderID string
}

func (e *ErrWrongOrderIDFormat) Error() string {
	return fmt.Sprintf("wrong order ID format: %s", e.OrderID)
}

type OrderService struct {
	repo order.Repository
}

func NewOrderService(repo order.Repository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	if err := s.checkOrder(orderID); err != nil {
		return nil, &ErrWrongOrderIDFormat{OrderID: orderID}
	}

	order, err := s.repo.LoadOrder(ctx, userID, orderID)

	if err != nil {
		return nil, fmt.Errorf("failed to load order: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]*models.Order, error) {
	orders, err := s.repo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	return orders, nil
}

func (s *OrderService) GetWithdrawals(ctx context.Context, userID string) ([]*models.Order, error) {
	withdrawals, err := s.repo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}
	return withdrawals, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*models.Order, error) {
	order, err := s.repo.GetOrderByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

func (s *OrderService) checkOrder(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("order ID is empty")
	}

	sum := 0
	shouldDouble := false

	for i := len(orderID) - 1; i >= 0; i-- {
		if orderID[i] < '0' || orderID[i] > '9' {
			return fmt.Errorf("order ID must contain only digits")
		}

		digit := int(orderID[i] - '0')
		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	if sum%10 != 0 {
		return fmt.Errorf("invalid order ID")
	}

	return nil
}
