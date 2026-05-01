package services

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	if got := parseRetryAfter("1"); got != time.Second {
		t.Fatalf("expected 1s, got %v", got)
	}

	future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(future); got <= 0 {
		t.Fatalf("expected positive duration for http-date, got %v", got)
	}

	if got := parseRetryAfter("bad-value"); got != 0 {
		t.Fatalf("expected zero for invalid header, got %v", got)
	}
}

func TestDispatcher_429PausesNextRequest(t *testing.T) {
	t.Parallel()

	var (
		mu           sync.Mutex
		requestTimes []time.Time
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		callNum := len(requestTimes)
		mu.Unlock()

		if callNum == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	dispatcher := &WorkerPoolOrderSyncDispatcher{
		repo:       &stubOrderRepo{},
		config:     &config.Config{AccrualAddress: server.URL},
		httpClient: &http.Client{Timeout: orderSyncTimeout},
		jobs:       make(chan *models.Order, 2),
		done:       make(chan struct{}),
	}
	dispatcher.startWorkers(context.Background(), 1)
	defer dispatcher.Close()

	dispatcher.Enqueue(&models.Order{ID: "79927398713", UserID: "u-1", Status: models.OrderStatusNew})
	dispatcher.Enqueue(&models.Order{ID: "79927398714", UserID: "u-1", Status: models.OrderStatusNew})

	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		count := len(requestTimes)
		mu.Unlock()
		if count >= 2 {
			break
		}

		select {
		case <-deadline:
			t.Fatal("expected two requests to accrual service")
		case <-time.After(20 * time.Millisecond):
		}
	}

	mu.Lock()
	first := requestTimes[0]
	second := requestTimes[1]
	mu.Unlock()

	if second.Sub(first) < 900*time.Millisecond {
		t.Fatalf("expected second request to be delayed by Retry-After, got %v", second.Sub(first))
	}
}

func TestDispatcher_CloseDuringPause(t *testing.T) {
	t.Parallel()

	dispatcher := &WorkerPoolOrderSyncDispatcher{
		repo:       &stubOrderRepo{},
		config:     &config.Config{AccrualAddress: "localhost:8080"},
		httpClient: &http.Client{Timeout: orderSyncTimeout},
		jobs:       make(chan *models.Order, 1),
		done:       make(chan struct{}),
	}
	dispatcher.startWorkers(context.Background(), 1)

	atomic.StoreInt64(&dispatcher.pausedUntil, time.Now().Add(10*time.Second).UnixNano())
	dispatcher.Enqueue(&models.Order{ID: "79927398713", UserID: "u-1", Status: models.OrderStatusNew})

	started := time.Now()
	dispatcher.Close()
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("expected close to finish quickly while paused, got %v", elapsed)
	}
}
