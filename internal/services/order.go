package services

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/order"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ErrWrongOrderIDFormat struct {
	OrderID string
}

func (e *ErrWrongOrderIDFormat) Error() string {
	return fmt.Sprintf("wrong order ID format: %s", e.OrderID)
}

type OrderDetails struct {
	OrderID string  			`json:"order"`
	Status  models.OrderStatus  `json:"status"`
	Accrual float32 			`json:"accrual,omitempty"`
}

type OrderService struct {
	repo order.Repository
	userService *UserService
	config *config.Config
}

func NewOrderService(repo order.Repository, userService *UserService, config *config.Config) *OrderService {
	return &OrderService{repo: repo, userService: userService, config: config}
}

func (s *OrderService) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	if err := s.checkOrder(orderID); err != nil {
		return nil, &ErrWrongOrderIDFormat{OrderID: orderID}
	}

	order, err := s.repo.LoadOrder(ctx, userID, orderID)

	if err != nil {
		return order, fmt.Errorf("failed to load order: %w", err)
	}

	go func(order *models.Order) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		resp, err := http.Get(fmt.Sprintf("http://%s/api/orders/%s", s.config.GetAccrualAddress(), order.ID))
		if err != nil {
			fmt.Printf("failed to fetch order details: %v\n", err)
			return
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var orderDetails OrderDetails
			if err := json.NewDecoder(resp.Body).Decode(&orderDetails); err != nil {
				fmt.Printf("failed to decode order details: %v\n", err)
				return
			}

			// Update order status and accrual in the repository
			// This part is simplified and should ideally be done through a method in the repository
			order.Status = orderDetails.Status
			now := time.Now()
			order.ProcessedAt = &now
			if orderDetails.Accrual > 0 {
				if err := s.userService.DoDeposit(ctx, order.UserID, orderDetails.Accrual); err != nil {
					fmt.Printf("failed to add funds to user: %v\n", err)
					return
				}
				order.Accrual = orderDetails.Accrual
			}
			if err := s.repo.UpdateOrder(ctx, order); err != nil {
				fmt.Printf("failed to update order: %v\n", err)
				return
			}
		case http.StatusNoContent:
			fmt.Printf("Order is not registered: %s\n", order.ID)
			return 
		case http.StatusTooManyRequests:
			fmt.Printf("Too many requests for order: %s\n", order.ID)
			return
		default:
			fmt.Printf("Unexpected status code %d for order: %s\n", resp.StatusCode, order.ID)
			return
		}
	}(order)

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
