package services

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"strings"
	"testing"
)

type stubUserRepo struct {
	registerFn    func(ctx context.Context, username, password string) (string, error)
	loginFn       func(ctx context.Context, username, password string) (string, error)
	getUserByIDFn func(ctx context.Context, userID string) (*models.User, error)
	updateUserFn  func(ctx context.Context, user *models.User) error

	updateUserCalled bool
}

func (s *stubUserRepo) Register(ctx context.Context, username, password string) (string, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubUserRepo) Login(ctx context.Context, username, password string) (string, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubUserRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	if s.getUserByIDFn != nil {
		return s.getUserByIDFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	s.updateUserCalled = true
	if s.updateUserFn != nil {
		return s.updateUserFn(ctx, user)
	}
	return nil
}

func TestUserService_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		repo          *stubUserRepo
		wantUserID    string
		wantErrSubstr string
	}{
		{
			name: "registers user successfully",
			repo: &stubUserRepo{
				registerFn: func(ctx context.Context, username, password string) (string, error) {
					if username != "alice" {
						t.Fatalf("unexpected username: %s", username)
					}
					if password != "secret" {
						t.Fatalf("unexpected password: %s", password)
					}
					return "u-1", nil
				},
			},
			wantUserID: "u-1",
		},
		{
			name: "wraps repository error",
			repo: &stubUserRepo{
				registerFn: func(ctx context.Context, username, password string) (string, error) {
					return "", errors.New("db error")
				},
			},
			wantErrSubstr: "failed to register user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.repo)

			userID, err := svc.Register(context.Background(), "alice", "secret")
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if userID != tt.wantUserID {
				t.Fatalf("expected userID %s, got %s", tt.wantUserID, userID)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		repo          *stubUserRepo
		wantUserID    string
		wantErrSubstr string
	}{
		{
			name: "logs in successfully",
			repo: &stubUserRepo{
				loginFn: func(ctx context.Context, username, password string) (string, error) {
					return "u-1", nil
				},
			},
			wantUserID: "u-1",
		},
		{
			name: "wraps repository error",
			repo: &stubUserRepo{
				loginFn: func(ctx context.Context, username, password string) (string, error) {
					return "", errors.New("unauthorized")
				},
			},
			wantErrSubstr: "failed to login user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.repo)

			userID, err := svc.Login(context.Background(), "alice", "secret")
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if userID != tt.wantUserID {
				t.Fatalf("expected userID %s, got %s", tt.wantUserID, userID)
			}
		})
	}
}

func TestUserService_GetUserInfo(t *testing.T) {
	t.Parallel()

	want := &models.User{ID: "u-1", Username: "alice", Balance: 10}
	tests := []struct {
		name          string
		repo          *stubUserRepo
		wantUser      *models.User
		wantErrSubstr string
	}{
		{
			name: "returns user info",
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					if userID != "u-1" {
						t.Fatalf("unexpected userID: %s", userID)
					}
					return want, nil
				},
			},
			wantUser: want,
		},
		{
			name: "wraps repository error",
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return nil, errors.New("not found")
				},
			},
			wantErrSubstr: "failed to get user info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.repo)

			got, err := svc.GetUserInfo(context.Background(), "u-1")
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.wantUser {
				t.Fatalf("expected %+v, got %+v", tt.wantUser, got)
			}
		})
	}
}

func TestUserService_DoDeposit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		sum               float32
		repo              *stubUserRepo
		wantErrSubstr     string
		wantUpdateCalled  bool
		verifyUpdatedUser func(t *testing.T, user *models.User)
	}{
		{
			name: "adds balance and updates user",
			sum:  5,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: "u-1", Balance: 10}, nil
				},
			},
			wantUpdateCalled: true,
			verifyUpdatedUser: func(t *testing.T, user *models.User) {
				if user.Balance != 15 {
					t.Fatalf("expected balance 15, got %.2f", user.Balance)
				}
			},
		},
		{
			name: "returns wrapped error when user lookup fails",
			sum:  5,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return nil, errors.New("db down")
				},
			},
			wantErrSubstr: "failed to get user info",
		},
		{
			name: "returns wrapped error when update fails",
			sum:  5,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: "u-1", Balance: 10}, nil
				},
				updateUserFn: func(ctx context.Context, user *models.User) error {
					return errors.New("write failed")
				},
			},
			wantUpdateCalled: true,
			wantErrSubstr:    "failed to update user balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verifyUpdatedUser != nil {
				tt.repo.updateUserFn = func(ctx context.Context, user *models.User) error {
					tt.verifyUpdatedUser(t, user)
					return nil
				}
			}

			svc := NewUserService(tt.repo)
			err := svc.DoDeposit(context.Background(), "u-1", tt.sum)

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

			if tt.repo.updateUserCalled != tt.wantUpdateCalled {
				t.Fatalf("expected UpdateUser called=%v, got %v", tt.wantUpdateCalled, tt.repo.updateUserCalled)
			}
		})
	}
}

func TestUserService_DoWithdraw(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		sum               float32
		repo              *stubUserRepo
		wantErrSubstr     string
		wantNotEnough     bool
		wantUpdateCalled  bool
		verifyUpdatedUser func(t *testing.T, user *models.User)
	}{
		{
			name: "withdraws and updates user",
			sum:  4,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: "u-1", Balance: 10}, nil
				},
			},
			wantUpdateCalled: true,
			verifyUpdatedUser: func(t *testing.T, user *models.User) {
				if user.Balance != 6 {
					t.Fatalf("expected balance 6, got %.2f", user.Balance)
				}
				if user.SumWithdrawn != 4 {
					t.Fatalf("expected withdrawn 4, got %.2f", user.SumWithdrawn)
				}
			},
		},
		{
			name: "returns domain error and does not call update when funds are insufficient",
			sum:  4,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 3}, nil
				},
			},
			wantNotEnough:    true,
			wantUpdateCalled: false,
		},
		{
			name: "returns wrapped error when user lookup fails",
			sum:  1,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return nil, errors.New("db down")
				},
			},
			wantErrSubstr: "failed to get user info",
		},
		{
			name: "returns wrapped error when update fails",
			sum:  4,
			repo: &stubUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: "u-1", Balance: 10}, nil
				},
				updateUserFn: func(ctx context.Context, user *models.User) error {
					return errors.New("write failed")
				},
			},
			wantErrSubstr:    "failed to update user balance",
			wantUpdateCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verifyUpdatedUser != nil {
				tt.repo.updateUserFn = func(ctx context.Context, user *models.User) error {
					tt.verifyUpdatedUser(t, user)
					return nil
				}
			}

			svc := NewUserService(tt.repo)
			err := svc.DoWithdraw(context.Background(), "u-1", tt.sum)

			if tt.wantNotEnough {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, models.ErrNotEnoughFunds) {
					t.Fatalf("expected ErrNotEnoughFunds, got %T", err)
				}
			} else if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected wrapped error, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.repo.updateUserCalled != tt.wantUpdateCalled {
				t.Fatalf("expected UpdateUser called=%v, got %v", tt.wantUpdateCalled, tt.repo.updateUserCalled)
			}
		})
	}
}
