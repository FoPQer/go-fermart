package services

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"strings"
	"testing"
)

type stubOrderRepo struct {
	loadOrderFn      func(ctx context.Context, userID string, orderID string) (*models.Order, error)
	getOrdersByUser  func(ctx context.Context, userID string) ([]*models.Order, error)
	getOrderByIDFn   func(ctx context.Context, orderID string) (*models.Order, error)
	loadOrderCalled  bool
	getOrdersCalled  bool
	getOrderByIDCall bool
}

func (s *stubOrderRepo) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	s.loadOrderCalled = true
	if s.loadOrderFn != nil {
		return s.loadOrderFn(ctx, userID, orderID)
	}
	return nil, nil
}

func (s *stubOrderRepo) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	s.getOrdersCalled = true
	if s.getOrdersByUser != nil {
		return s.getOrdersByUser(ctx, userID)
	}
	return nil, nil
}

func (s *stubOrderRepo) GetOrdersWithdrawnByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	s.getOrdersCalled = true
	if s.getOrdersByUser != nil {
		return s.getOrdersByUser(ctx, userID)
	}
	return nil, nil
}

func (s *stubOrderRepo) UpdateOrder(ctx context.Context, order *models.Order) error {
	return nil
}

func TestOrderService_checkOrder(t *testing.T) {
	t.Parallel()

	svc := NewOrderService(&stubOrderRepo{}, nil)
	tests := []struct {
		name    string
		orderID string
		wantErr bool
	}{
		{name: "valid luhn number", orderID: "79927398713", wantErr: false},
		{name: "empty order id", orderID: "", wantErr: true},
		{name: "contains non digits", orderID: "12ab34", wantErr: true},
		{name: "invalid checksum", orderID: "79927398714", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.checkOrder(tt.orderID)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for orderID %q, got nil", tt.orderID)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error for orderID %q, got %v", tt.orderID, err)
			}
		})
	}
}

func TestOrderService_LoadOrder(t *testing.T) {
	t.Parallel()

	t.Run("returns wrong format error and skips repo for invalid id", func(t *testing.T) {
		repo := &stubOrderRepo{}
		svc := NewOrderService(repo, nil)

		order, err := svc.LoadOrder(context.Background(), "user-1", "invalid")
		if order != nil {
			t.Fatalf("expected nil order, got %+v", order)
		}
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var formatErr *ErrWrongOrderIDFormat
		if !errors.As(err, &formatErr) {
			t.Fatalf("expected ErrWrongOrderIDFormat, got %T", err)
		}
		if repo.loadOrderCalled {
			t.Fatal("expected repository not to be called for invalid order id")
		}
	})

	t.Run("loads order successfully", func(t *testing.T) {
		want := &models.Order{ID: "79927398713", UserID: "user-1", Status: "NEW"}
		repo := &stubOrderRepo{
			loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
				if userID != "user-1" {
					t.Fatalf("unexpected userID: %s", userID)
				}
				if orderID != "79927398713" {
					t.Fatalf("unexpected orderID: %s", orderID)
				}
				return want, nil
			},
		}
		svc := NewOrderService(repo, nil)

		got, err := svc.LoadOrder(context.Background(), "user-1", "79927398713")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != want {
			t.Fatalf("expected %+v, got %+v", want, got)
		}
	})

	t.Run("wraps repository errors", func(t *testing.T) {
		repo := &stubOrderRepo{
			loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
				return nil, errors.New("db unavailable")
			},
		}
		svc := NewOrderService(repo, nil)

		_, err := svc.LoadOrder(context.Background(), "user-1", "79927398713")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to load order") {
			t.Fatalf("expected wrapped error, got %v", err)
		}
	})
}

func TestOrderService_GetOrdersAndWithdrawals(t *testing.T) {
	t.Parallel()

	want := []*models.Order{{ID: "1", UserID: "user-1"}}
	repo := &stubOrderRepo{
		getOrdersByUser: func(ctx context.Context, userID string) ([]*models.Order, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			return want, nil
		},
	}
	svc := NewOrderService(repo, nil)

	orders, err := svc.GetOrders(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error from GetOrders, got %v", err)
	}
	if len(orders) != 1 || orders[0] != want[0] {
		t.Fatalf("unexpected orders result: %+v", orders)
	}

	withdrawals, err := svc.GetWithdrawals(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error from GetWithdrawals, got %v", err)
	}
	if len(withdrawals) != 1 || withdrawals[0] != want[0] {
		t.Fatalf("unexpected withdrawals result: %+v", withdrawals)
	}
}
