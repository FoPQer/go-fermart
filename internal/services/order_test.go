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
	getWithdrawalsByUser func(ctx context.Context, userID string) ([]*models.Order, error)
	updateOrderFn    func(ctx context.Context, order *models.Order) error
	loadOrderCalled  bool
	getOrdersCalled  bool
	getWithdrawalsCalled bool
	updateOrderCalled bool
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
	s.getWithdrawalsCalled = true
	if s.getWithdrawalsByUser != nil {
		return s.getWithdrawalsByUser(ctx, userID)
	}
	return nil, nil
}

func (s *stubOrderRepo) UpdateOrder(ctx context.Context, order *models.Order) error {
	s.updateOrderCalled = true
	if s.updateOrderFn != nil {
		return s.updateOrderFn(ctx, order)
	}
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

	tests := []struct {
		name               string
		orderID            string
		repo               *stubOrderRepo
		wantOrder          *models.Order
		wantErrSubstr      string
		wantWrongFormatErr bool
		wantRepoCalled     bool
	}{
		{
			name:               "returns wrong format error and skips repo for invalid id",
			orderID:            "invalid",
			repo:               &stubOrderRepo{},
			wantWrongFormatErr: true,
			wantRepoCalled:     false,
		},
		{
			name:    "loads order successfully",
			orderID: "79927398713",
			wantOrder: &models.Order{ID: "79927398713", UserID: "user-1", Status: "NEW"},
			wantRepoCalled: true,
		},
		{
			name:          "wraps repository errors",
			orderID:       "79927398713",
			repo:          &stubOrderRepo{loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) { return nil, errors.New("db unavailable") }},
			wantErrSubstr: "failed to load order",
			wantRepoCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			if repo == nil {
				repo = &stubOrderRepo{
					loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
						if userID != "user-1" {
							t.Fatalf("unexpected userID: %s", userID)
						}
						if orderID != tt.orderID {
							t.Fatalf("unexpected orderID: %s", orderID)
						}
						return tt.wantOrder, nil
					},
				}
			}

			svc := NewOrderService(repo, nil)
			got, err := svc.LoadOrder(context.Background(), "user-1", tt.orderID)

			if tt.wantWrongFormatErr {
				if got != nil {
					t.Fatalf("expected nil order, got %+v", got)
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var formatErr *ErrWrongOrderIDFormat
				if !errors.As(err, &formatErr) {
					t.Fatalf("expected ErrWrongOrderIDFormat, got %T", err)
				}
			} else if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if got != tt.wantOrder {
					t.Fatalf("expected %+v, got %+v", tt.wantOrder, got)
				}
			}

			if repo.loadOrderCalled != tt.wantRepoCalled {
				t.Fatalf("expected repository called=%v, got %v", tt.wantRepoCalled, repo.loadOrderCalled)
			}
		})
	}
}

func TestOrderService_GetOrders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		repo          *stubOrderRepo
		wantOrders    []*models.Order
		wantErrSubstr string
	}{
		{
			name: "returns orders successfully",
			repo: &stubOrderRepo{getOrdersByUser: func(ctx context.Context, userID string) ([]*models.Order, error) {
				if userID != "user-1" {
					t.Fatalf("unexpected userID: %s", userID)
				}
				return []*models.Order{{ID: "1", UserID: "user-1"}}, nil
			}},
			wantOrders: []*models.Order{{ID: "1", UserID: "user-1"}},
		},
		{
			name: "wraps get orders repository error",
			repo: &stubOrderRepo{getOrdersByUser: func(ctx context.Context, userID string) ([]*models.Order, error) {
				return nil, errors.New("db failed")
			}},
			wantErrSubstr: "failed to get orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(tt.repo, nil)
			orders, err := svc.GetOrders(context.Background(), "user-1")

			if tt.wantErrSubstr != "" {
				if orders != nil {
					t.Fatalf("expected nil orders, got %+v", orders)
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error from GetOrders, got %v", err)
			}
			if len(orders) != len(tt.wantOrders) {
				t.Fatalf("unexpected orders result: %+v", orders)
			}
			if orders[0].ID != tt.wantOrders[0].ID || orders[0].UserID != tt.wantOrders[0].UserID {
				t.Fatalf("unexpected orders result: %+v", orders)
			}
			if !tt.repo.getOrdersCalled {
				t.Fatal("expected orders repository method to be called")
			}
		})
	}
}

func TestOrderService_GetWithdrawals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		repo            *stubOrderRepo
		wantWithdrawals []*models.Order
		wantErrSubstr   string
	}{
		{
			name: "returns withdrawals successfully",
			repo: &stubOrderRepo{getWithdrawalsByUser: func(ctx context.Context, userID string) ([]*models.Order, error) {
				if userID != "user-1" {
					t.Fatalf("unexpected userID: %s", userID)
				}
				return []*models.Order{{ID: "1", UserID: "user-1"}}, nil
			}},
			wantWithdrawals: []*models.Order{{ID: "1", UserID: "user-1"}},
		},
		{
			name: "wraps get withdrawals repository error",
			repo: &stubOrderRepo{getWithdrawalsByUser: func(ctx context.Context, userID string) ([]*models.Order, error) {
				return nil, errors.New("db failed")
			}},
			wantErrSubstr: "failed to get withdrawals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(tt.repo, nil)
			withdrawals, err := svc.GetWithdrawals(context.Background(), "user-1")

			if tt.wantErrSubstr != "" {
				if withdrawals != nil {
					t.Fatalf("expected nil withdrawals, got %+v", withdrawals)
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error from GetWithdrawals, got %v", err)
			}
			if len(withdrawals) != len(tt.wantWithdrawals) {
				t.Fatalf("unexpected withdrawals result: %+v", withdrawals)
			}
			if withdrawals[0].ID != tt.wantWithdrawals[0].ID || withdrawals[0].UserID != tt.wantWithdrawals[0].UserID {
				t.Fatalf("unexpected withdrawals result: %+v", withdrawals)
			}
			if !tt.repo.getWithdrawalsCalled {
				t.Fatal("expected withdrawals repository method to be called")
			}
		})
	}
}

func TestOrderService_UpdateOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		repo          *stubOrderRepo
		wantErrSubstr string
	}{
		{
			name: "updates order successfully",
			repo: &stubOrderRepo{updateOrderFn: func(ctx context.Context, order *models.Order) error {
				if order.ID != "79927398713" {
					t.Fatalf("unexpected order id: %s", order.ID)
				}
				return nil
			}},
		},
		{
			name: "wraps repository error",
			repo: &stubOrderRepo{updateOrderFn: func(ctx context.Context, order *models.Order) error {
				return errors.New("write failed")
			}},
			wantErrSubstr: "failed to update order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(tt.repo, nil)
			err := svc.UpdateOrder(context.Background(), &models.Order{ID: "79927398713"})

			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if !tt.repo.updateOrderCalled {
				t.Fatal("expected UpdateOrder repository method to be called")
			}
		})
	}
}
