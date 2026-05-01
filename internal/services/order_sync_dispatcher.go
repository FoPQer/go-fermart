package services

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/order"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultOrderWorkerCount = 4
	defaultOrderQueueSize   = 128
	orderSyncTimeout        = time.Second
)

type OrderSyncDispatcher interface {
	Enqueue(order *models.Order)
	Close()
}

type WorkerPoolOrderSyncDispatcher struct {
	repo        order.Repository
	userService *UserService
	config      *config.Config
	httpClient  *http.Client
	jobs        chan *models.Order
	workers     sync.WaitGroup
	closeOnce   sync.Once
}

func NewOrderSyncDispatcher(repo order.Repository, userService *UserService, config *config.Config) OrderSyncDispatcher {
	dispatcher := &WorkerPoolOrderSyncDispatcher{
		repo:        repo,
		userService: userService,
		config:      config,
		httpClient:  &http.Client{Timeout: orderSyncTimeout},
		jobs:        make(chan *models.Order, defaultOrderQueueSize),
	}
	dispatcher.startWorkers(defaultOrderWorkerCount)
	return dispatcher
}

func (d *WorkerPoolOrderSyncDispatcher) Enqueue(order *models.Order) {
	d.jobs <- order
}

func (d *WorkerPoolOrderSyncDispatcher) Close() {
	d.closeOnce.Do(func() {
		close(d.jobs)
		d.workers.Wait()
	})
}

func (d *WorkerPoolOrderSyncDispatcher) startWorkers(count int) {
	for range count {
		d.workers.Go(func() {
			for order := range d.jobs {
				d.syncOrder(order)
			}
		})
	}
}

func (d *WorkerPoolOrderSyncDispatcher) syncOrder(order *models.Order) {
	ctx, cancel := context.WithTimeout(context.Background(), orderSyncTimeout)
	defer cancel()

	orderURL, err := d.accrualOrderURL(order.ID)
	if err != nil {
		slog.Error("failed to build order details url", "orderID", order.ID, "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, orderURL, nil)
	if err != nil {
		slog.Error("failed to build order details request", "orderID", order.ID, "error", err)
		return
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		slog.Error("failed to fetch order details", "orderID", order.ID, "error", err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var orderDetails OrderDetails
		if err := json.NewDecoder(resp.Body).Decode(&orderDetails); err != nil {
			slog.Error("failed to decode order details", "orderID", order.ID, "error", err)
			return
		}

		order.Status = orderDetails.Status
		now := time.Now()
		order.ProcessedAt = &now
		if orderDetails.Accrual > 0 {
			if d.userService == nil {
				slog.Error("failed to add funds to user", "orderID", order.ID, "error", "user service is nil")
				return
			}
			if err := d.userService.DoDeposit(ctx, order.UserID, orderDetails.Accrual); err != nil {
				slog.Error("failed to add funds to user", "orderID", order.ID, "error", err)
				return
			}
			order.Accrual = orderDetails.Accrual
		}
		if err := d.repo.UpdateOrder(ctx, order); err != nil {
			slog.Error("failed to update order", "orderID", order.ID, "error", err)
			return
		}
	case http.StatusNoContent:
		slog.Info("order is not registered", "orderID", order.ID)
	case http.StatusTooManyRequests:
		slog.Info("too many requests for order", "orderID", order.ID)
	default:
		slog.Info("unexpected status code for order", "statusCode", resp.StatusCode, "orderID", order.ID)
	}
}

func (d *WorkerPoolOrderSyncDispatcher) accrualOrderURL(orderID string) (string, error) {
	if d.config == nil {
		return "", fmt.Errorf("accrual config is nil")
	}

	baseURL := strings.TrimRight(d.config.GetAccrualAddress(), "/")
	if baseURL == "" {
		return "", fmt.Errorf("accrual address is empty")
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	return fmt.Sprintf("%s/api/orders/%s", baseURL, orderID), nil
}