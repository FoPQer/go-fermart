package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"testing"
)

func TestMemoryRepository_LoadOrder(t *testing.T) {
	t.Run("creates new order", func(t *testing.T) {
		repo := NewMemoryRepository()

		got, err := repo.LoadOrder(context.Background(), "u-1", "79927398713")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil order")
		}
		if got.ID != "79927398713" {
			t.Fatalf("expected order id 79927398713, got %s", got.ID)
		}
		if got.UserID != "u-1" {
			t.Fatalf("expected user id u-1, got %s", got.UserID)
		}
		if got.Status != models.OrderStatusNew {
			t.Fatalf("expected status %s, got %s", models.OrderStatusNew, got.Status)
		}
		if got.UploadedAt.IsZero() {
			t.Fatal("expected UploadedAt to be set")
		}
	})

	t.Run("returns ErrOrderAlreadyExists for same user", func(t *testing.T) {
		repo := NewMemoryRepository()
		first, err := repo.LoadOrder(context.Background(), "u-1", "79927398713")
		if err != nil {
			t.Fatalf("unexpected error on initial load: %v", err)
		}

		got, err := repo.LoadOrder(context.Background(), "u-1", "79927398713")
		if got == nil {
			t.Fatal("expected non-nil order")
		}
		if got != first {
			t.Fatal("expected to get the same stored order pointer")
		}
		var alreadyExistsErr *ErrOrderAlreadyExists
		if !errors.As(err, &alreadyExistsErr) {
			t.Fatalf("expected ErrOrderAlreadyExists, got %T", err)
		}
		if alreadyExistsErr.OrderID != "79927398713" {
			t.Fatalf("expected OrderID 79927398713, got %s", alreadyExistsErr.OrderID)
		}
	})

	t.Run("returns ErrOrderAlreadyExistsForAnotherUser for another user", func(t *testing.T) {
		repo := NewMemoryRepository()
		_, err := repo.LoadOrder(context.Background(), "u-1", "79927398713")
		if err != nil {
			t.Fatalf("unexpected error on initial load: %v", err)
		}

		got, err := repo.LoadOrder(context.Background(), "u-2", "79927398713")
		if got != nil {
			t.Fatalf("expected nil order, got %+v", got)
		}
		var anotherUserErr *ErrOrderAlreadyExistsForAnotherUser
		if !errors.As(err, &anotherUserErr) {
			t.Fatalf("expected ErrOrderAlreadyExistsForAnotherUser, got %T", err)
		}
		if anotherUserErr.OrderID != "79927398713" {
			t.Fatalf("expected OrderID 79927398713, got %s", anotherUserErr.OrderID)
		}
		if anotherUserErr.UserID != "u-2" {
			t.Fatalf("expected UserID u-2, got %s", anotherUserErr.UserID)
		}
	})
}

func TestMemoryRepository_GetOrdersByUserID(t *testing.T) {
	repo := NewMemoryRepository()
	_, _ = repo.LoadOrder(context.Background(), "u-1", "79927398713")
	_, _ = repo.LoadOrder(context.Background(), "u-1", "12345678903")
	_, _ = repo.LoadOrder(context.Background(), "u-2", "4111111111111111")

	orders, err := repo.GetOrdersByUserID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	for _, o := range orders {
		if o.UserID != "u-1" {
			t.Fatalf("expected all orders for u-1, got userID %s", o.UserID)
		}
	}
}

func TestMemoryRepository_GetOrdersWithdrawnByUserID(t *testing.T) {
	repo := NewMemoryRepository()

	o1, _ := repo.LoadOrder(context.Background(), "u-1", "79927398713")
	o2, _ := repo.LoadOrder(context.Background(), "u-1", "12345678903")
	o3, _ := repo.LoadOrder(context.Background(), "u-2", "4111111111111111")

	o1.Withdrawn = 150
	if err := repo.UpdateOrder(context.Background(), o1); err != nil {
		t.Fatalf("failed to update first order: %v", err)
	}
	o2.Withdrawn = 0
	if err := repo.UpdateOrder(context.Background(), o2); err != nil {
		t.Fatalf("failed to update second order: %v", err)
	}
	o3.Withdrawn = 200
	if err := repo.UpdateOrder(context.Background(), o3); err != nil {
		t.Fatalf("failed to update third order: %v", err)
	}

	orders, err := repo.GetOrdersWithdrawnByUserID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 withdrawn order, got %d", len(orders))
	}
	if orders[0].ID != "79927398713" {
		t.Fatalf("expected order 79927398713, got %s", orders[0].ID)
	}
	if orders[0].Withdrawn != 150 {
		t.Fatalf("expected withdrawn 150, got %.2f", orders[0].Withdrawn)
	}
}

func TestMemoryRepository_UpdateOrder(t *testing.T) {
	t.Run("updates existing order", func(t *testing.T) {
		repo := NewMemoryRepository()
		order, err := repo.LoadOrder(context.Background(), "u-1", "79927398713")
		if err != nil {
			t.Fatalf("unexpected error on initial load: %v", err)
		}

		order.Status = models.OrderStatusProcessed
		order.Accrual = 321.5
		order.Withdrawn = 100
		if err := repo.UpdateOrder(context.Background(), order); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		orders, err := repo.GetOrdersByUserID(context.Background(), "u-1")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(orders) != 1 {
			t.Fatalf("expected 1 order, got %d", len(orders))
		}
		if orders[0].Status != models.OrderStatusProcessed {
			t.Fatalf("expected status %s, got %s", models.OrderStatusProcessed, orders[0].Status)
		}
		if orders[0].Accrual != 321.5 {
			t.Fatalf("expected accrual 321.5, got %.2f", orders[0].Accrual)
		}
		if orders[0].Withdrawn != 100 {
			t.Fatalf("expected withdrawn 100, got %.2f", orders[0].Withdrawn)
		}
	})

	t.Run("returns error for missing order", func(t *testing.T) {
		repo := NewMemoryRepository()

		err := repo.UpdateOrder(context.Background(), &models.Order{ID: "missing"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "order with ID missing does not exist" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
