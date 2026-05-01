package services

import (
	"FoPQer/go-fermart/internal/config"
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

type OrderDetails struct {
	OrderID string             `json:"order"`
	Status  models.OrderStatus `json:"status"`
	Accrual float32            `json:"accrual,omitempty"`
}

type OrderService struct {
	repo        order.Repository
	dispatcher OrderSyncDispatcher
}

func NewOrderService(repo order.Repository, userService *UserService, config *config.Config) *OrderService {
	return &OrderService{
		repo:        repo,
		dispatcher: NewOrderSyncDispatcher(repo, userService, config),
	}
}

func (s *OrderService) Close() {
	s.dispatcher.Close()
}

func (s *OrderService) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	if err := s.checkOrder(orderID); err != nil {
		return nil, &ErrWrongOrderIDFormat{OrderID: orderID}
	}

	order, err := s.repo.LoadOrder(ctx, userID, orderID)

	if err != nil {
		return order, fmt.Errorf("failed to load order: %w", err)
	}

	s.dispatchOrderSync(order)

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
	withdrawals, err := s.repo.GetOrdersWithdrawnByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}
	return withdrawals, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, order *models.Order) error {
	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
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

func (s *OrderService) dispatchOrderSync(order *models.Order) {
	if order == nil || s.isTerminalStatus(order.Status) {
		return
	}

	s.dispatcher.Enqueue(order)
}

func (s *OrderService) isTerminalStatus(status models.OrderStatus) bool {
	switch status {
	case models.OrderStatusInvalid, models.OrderStatusProcessed:
		return true
	default:
		return false
	}
}
