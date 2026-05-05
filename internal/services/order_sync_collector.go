package services

import (
	"FoPQer/go-fermart/internal/repository/order"
	"context"
	"log/slog"
	"sync"
	"time"
)

const defaultOrderScanInterval = time.Second

type OrderSyncCollector struct {
	repo       order.Repository
	dispatcher OrderSyncDispatcher
	interval   time.Duration
	done       chan struct{}
	wg         sync.WaitGroup
	closeOnce  sync.Once
}

func NewOrderSyncCollector(repo order.Repository, dispatcher OrderSyncDispatcher, interval time.Duration) *OrderSyncCollector {
	collector := &OrderSyncCollector{
		repo:       repo,
		dispatcher: dispatcher,
		interval:   interval,
		done:       make(chan struct{}),
	}
	collector.start(context.Background())
	return collector
}

func (c *OrderSyncCollector) start(ctx context.Context) {
	if c.interval <= 0 {
		c.interval = defaultOrderScanInterval
	}

	c.wg.Go(func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.collect(ctx)
			case <-c.done:
				return
			}
		}
	})
}

func (c *OrderSyncCollector) collect(ctx context.Context) {
	orders, err := c.repo.GetUnprocessedOrders(ctx)
	if err != nil {
		slog.Error("failed to collect unprocessed orders", "error", err)
		return
	}

	for _, currentOrder := range orders {
		if currentOrder == nil {
			continue
		}
		c.dispatcher.Enqueue(currentOrder)
	}
}

func (c *OrderSyncCollector) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.wg.Wait()
	})
}
