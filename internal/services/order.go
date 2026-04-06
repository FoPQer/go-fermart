package services

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/order"
	"fmt"
)

type OrderService struct {
	repo order.Repository
}

func NewOrderService(repo order.Repository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) LoadOrder(userID int, orderID string) error {
	if err := s.checkOrder(orderID); err != nil {
		return fmt.Errorf("failed to check order: %w", err)
	}

	if err := s.repo.LoadOrder(userID, orderID); err != nil {
		return fmt.Errorf("failed to load order: %w", err)
	}

	return nil
}

func (s *OrderService) GetOrders(userID int) ([]*models.Order, error) {
	orders, err := s.repo.GetOrdersByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	return orders, nil
}

func (s *OrderService) GetWithdrawals(userID int) ([]*models.Order, error) {
	// TODO: Реализовать получение информации о снятиях пользователя
	return nil, nil
}

func (s *OrderService) GetOrderByID(orderID string) (*models.Order, error) {
	order, err := s.repo.GetOrdersByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

func (s *OrderService) checkOrder(orderID string) error {
	// TODO: Проверка номера по алгоритму Луна
	return nil
}