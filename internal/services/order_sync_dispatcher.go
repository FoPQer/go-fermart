package services

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/txutil"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	transactor  txutil.Transactor
	httpClient  *http.Client
	jobs        chan *models.Order
	done        chan struct{}
	workers     sync.WaitGroup
	closeOnce   sync.Once
	enqueueMu   sync.RWMutex
	closed      bool
	pausedUntil int64
}

func NewOrderSyncDispatcher(repo order.Repository, userService *UserService, config *config.Config, transactor txutil.Transactor) OrderSyncDispatcher {
	dispatcher := &WorkerPoolOrderSyncDispatcher{
		repo:        repo,
		userService: userService,
		config:      config,
		transactor:  transactor,
		httpClient:  &http.Client{Timeout: orderSyncTimeout},
		jobs:        make(chan *models.Order, defaultOrderQueueSize),
		done:        make(chan struct{}),
	}
	dispatcher.startWorkers(context.Background(), defaultOrderWorkerCount)
	return dispatcher
}

func (d *WorkerPoolOrderSyncDispatcher) Enqueue(order *models.Order) {
	if order == nil || isTerminalOrderStatus(order.Status) {
		return
	}

	d.enqueueMu.RLock()
	defer d.enqueueMu.RUnlock()
	if d.closed {
		return
	}

	select {
	case d.jobs <- order:
	case <-d.done:
	}
}

func (d *WorkerPoolOrderSyncDispatcher) Close() {
	d.closeOnce.Do(func() {
		d.enqueueMu.Lock()
		d.closed = true
		close(d.done)
		close(d.jobs)
		d.enqueueMu.Unlock()
		d.workers.Wait()
	})
}

func (d *WorkerPoolOrderSyncDispatcher) startWorkers(ctx context.Context, count int) {
	for range count {
		d.workers.Go(func() {
			for order := range d.jobs {
				if !d.waitIfPaused() {
					return
				}
				d.syncOrder(ctx, order)
			}
		})
	}
}

func (d *WorkerPoolOrderSyncDispatcher) waitIfPaused() bool {
	for {
		pauseUntil := atomic.LoadInt64(&d.pausedUntil)
		if pauseUntil == 0 {
			return true
		}

		pauseUntilTime := time.Unix(0, pauseUntil)
		remaining := time.Until(pauseUntilTime)
		if remaining <= 0 {
			return true
		}

		timer := time.NewTimer(remaining)
		select {
		case <-timer.C:
			timer.Stop()
		case <-d.done:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return false
		}
	}
}

func (d *WorkerPoolOrderSyncDispatcher) syncOrder(ctx context.Context, order *models.Order) {
	ctx, cancel := context.WithTimeout(ctx, orderSyncTimeout)
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
			order.Accrual = orderDetails.Accrual
		}

		if err := d.withTransaction(ctx, func(txCtx context.Context) error {
			if order.Accrual > 0 {
				if err := d.userService.DoDeposit(txCtx, order.UserID, order.Accrual); err != nil {
					return fmt.Errorf("deposit accrual: %w", err)
				}
			}
			return d.repo.UpdateOrder(txCtx, order)
		}); err != nil {
			slog.Error("failed to apply accrual", "orderID", order.ID, "error", err)
			return
		}
	case http.StatusNoContent:
		slog.Info("order is not registered", "orderID", order.ID)
	case http.StatusTooManyRequests:
		d.applyRetryAfter(resp.Header.Get("Retry-After"))
		slog.Info("too many requests for order", "orderID", order.ID)
	default:
		slog.Info("unexpected status code for order", "statusCode", resp.StatusCode, "orderID", order.ID)
	}
}

func (d *WorkerPoolOrderSyncDispatcher) applyRetryAfter(retryAfterHeader string) {
	retryAfter := parseRetryAfter(retryAfterHeader)
	if retryAfter <= 0 {
		return
	}

	newPauseUntil := time.Now().Add(retryAfter).UnixNano()
	for {
		current := atomic.LoadInt64(&d.pausedUntil)
		if current >= newPauseUntil {
			return
		}
		if atomic.CompareAndSwapInt64(&d.pausedUntil, current, newPauseUntil) {
			return
		}
	}
}

func (d *WorkerPoolOrderSyncDispatcher) withTransaction(ctx context.Context, fn func(context.Context) error) error {
	if d.transactor == nil {
		return fn(ctx)
	}
	return d.transactor.WithinTransaction(ctx, fn)
}

func isTerminalOrderStatus(status models.OrderStatus) bool {
	switch status {
	case models.OrderStatusInvalid, models.OrderStatusProcessed:
		return true
	default:
		return false
	}
}

func parseRetryAfter(retryAfterHeader string) time.Duration {
	trimmed := strings.TrimSpace(retryAfterHeader)
	if trimmed == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	retryAfterTime, err := http.ParseTime(trimmed)
	if err != nil {
		return 0
	}

	duration := time.Until(retryAfterTime)
	if duration <= 0 {
		return 0
	}

	return duration
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
