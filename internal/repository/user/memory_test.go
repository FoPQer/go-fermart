package user

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"testing"
)

func TestMemoryRepository_Register(t *testing.T) {
	t.Run("registers new user", func(t *testing.T) {
		repo := NewMemoryRepository()

		userID, err := repo.Register(context.Background(), "alice", "secret")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userID == "" {
			t.Fatal("expected non-empty userID")
		}

		stored, ok := repo.users["alice"]
		if !ok {
			t.Fatal("expected user to be stored by username")
		}
		if stored.ID != userID {
			t.Fatalf("expected stored ID %s, got %s", userID, stored.ID)
		}
		if stored.Password != "secret" {
			t.Fatalf("expected stored password secret, got %s", stored.Password)
		}
		if stored.Balance != 1000 {
			t.Fatalf("expected initial balance 1000, got %.2f", stored.Balance)
		}
	})

	t.Run("returns ErrUserAlreadyExists for duplicate username", func(t *testing.T) {
		repo := NewMemoryRepository()
		_, _ = repo.Register(context.Background(), "alice", "secret")

		_, err := repo.Register(context.Background(), "alice", "another")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var existsErr *ErrUserAlreadyExists
		if !errors.As(err, &existsErr) {
			t.Fatalf("expected ErrUserAlreadyExists, got %T", err)
		}
		if existsErr.Username != "alice" {
			t.Fatalf("expected username alice, got %s", existsErr.Username)
		}
	})
}

func TestMemoryRepository_Login(t *testing.T) {
	repo := NewMemoryRepository()
	userID, err := repo.Register(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("returns userID for valid credentials", func(t *testing.T) {
		got, loginErr := repo.Login(context.Background(), "alice", "secret")
		if loginErr != nil {
			t.Fatalf("expected nil error, got %v", loginErr)
		}
		if got != userID {
			t.Fatalf("expected userID %s, got %s", userID, got)
		}
	})

	t.Run("returns ErrInvalidCredentials for wrong password", func(t *testing.T) {
		_, loginErr := repo.Login(context.Background(), "alice", "wrong")
		if loginErr == nil {
			t.Fatal("expected error, got nil")
		}
		var invalidErr *ErrInvalidCredentials
		if !errors.As(loginErr, &invalidErr) {
			t.Fatalf("expected ErrInvalidCredentials, got %T", loginErr)
		}
	})

	t.Run("returns ErrInvalidCredentials for unknown username", func(t *testing.T) {
		_, loginErr := repo.Login(context.Background(), "bob", "secret")
		if loginErr == nil {
			t.Fatal("expected error, got nil")
		}
		var invalidErr *ErrInvalidCredentials
		if !errors.As(loginErr, &invalidErr) {
			t.Fatalf("expected ErrInvalidCredentials, got %T", loginErr)
		}
	})
}

func TestMemoryRepository_GetUserByID(t *testing.T) {
	repo := NewMemoryRepository()
	userID, err := repo.Register(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	t.Run("returns user by ID", func(t *testing.T) {
		got, getErr := repo.GetUserByID(context.Background(), userID)
		if getErr != nil {
			t.Fatalf("expected nil error, got %v", getErr)
		}
		if got == nil {
			t.Fatal("expected non-nil user")
		}
		if got.ID != userID {
			t.Fatalf("expected user ID %s, got %s", userID, got.ID)
		}
		if got.Username != "alice" {
			t.Fatalf("expected username alice, got %s", got.Username)
		}
	})

	t.Run("returns ErrUserNotFound for missing user", func(t *testing.T) {
		_, getErr := repo.GetUserByID(context.Background(), "missing")
		if getErr == nil {
			t.Fatal("expected error, got nil")
		}
		var notFoundErr *ErrUserNotFound
		if !errors.As(getErr, &notFoundErr) {
			t.Fatalf("expected ErrUserNotFound, got %T", getErr)
		}
		if notFoundErr.UserID != "missing" {
			t.Fatalf("expected missing userID, got %s", notFoundErr.UserID)
		}
	})
}

func TestMemoryRepository_UpdateUser(t *testing.T) {
	t.Run("updates existing user by ID", func(t *testing.T) {
		repo := NewMemoryRepository()
		userID, err := repo.Register(context.Background(), "alice", "secret")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		updated := &models.User{
			ID:           userID,
			Username:     "alice",
			Password:     "secret",
			Balance:      777,
			SumWithdrawn: 123,
		}
		if updateErr := repo.UpdateUser(context.Background(), updated); updateErr != nil {
			t.Fatalf("expected nil error, got %v", updateErr)
		}

		got, getErr := repo.GetUserByID(context.Background(), userID)
		if getErr != nil {
			t.Fatalf("failed to get updated user: %v", getErr)
		}
		if got.Balance != 777 {
			t.Fatalf("expected balance 777, got %.2f", got.Balance)
		}
		if got.SumWithdrawn != 123 {
			t.Fatalf("expected withdrawn 123, got %.2f", got.SumWithdrawn)
		}
	})

	t.Run("returns ErrUserNotFound for missing user", func(t *testing.T) {
		repo := NewMemoryRepository()
		updateErr := repo.UpdateUser(context.Background(), &models.User{ID: "missing"})
		if updateErr == nil {
			t.Fatal("expected error, got nil")
		}
		var notFoundErr *ErrUserNotFound
		if !errors.As(updateErr, &notFoundErr) {
			t.Fatalf("expected ErrUserNotFound, got %T", updateErr)
		}
		if notFoundErr.UserID != "missing" {
			t.Fatalf("expected missing userID, got %s", notFoundErr.UserID)
		}
	})
}
