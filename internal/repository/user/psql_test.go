package user

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeRow implements pgx.Row for testing.
type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

// --- Register ---

func TestPsqlRepository_Register(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(m *MockpgxQuerier)
		wantID    string
		wantErr   error
	}{
		{
			name: "success",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error {
					*dest[0].(*string) = "uid-1"
					return nil
				}}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "alice", "hash").Return(row)
			},
			wantID: "uid-1",
		},
		{
			name: "username already exists",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "alice", "hash").Return(row)
			},
			wantErr: &ErrUserAlreadyExists{Username: "alice"},
		},
		{
			name: "db error",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return errors.New("db error") }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "alice", "hash").Return(row)
			},
			wantErr: errors.New("failed to register user"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mock := NewMockpgxQuerier(ctrl)
			tt.setupMock(mock)

			repo := &PsqlRepository{conn: mock}
			gotID, err := repo.Register(context.Background(), "alice", "hash")

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var errAlreadyExists *ErrUserAlreadyExists
				if errors.As(tt.wantErr, &errAlreadyExists) {
					var got *ErrUserAlreadyExists
					if !errors.As(err, &got) {
						t.Fatalf("expected ErrUserAlreadyExists, got %v", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotID != tt.wantID {
					t.Errorf("want ID %q, got %q", tt.wantID, gotID)
				}
			}
		})
	}
}

// --- Login ---

func TestPsqlRepository_Login(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(m *MockpgxQuerier)
		wantID    string
		wantErr   error
	}{
		{
			name: "success",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error {
					*dest[0].(*string) = "uid-2"
					return nil
				}}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "bob", "pass").Return(row)
			},
			wantID: "uid-2",
		},
		{
			name: "invalid credentials",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "bob", "wrong").Return(row)
			},
			wantErr: &ErrInvalidCredentials{},
		},
		{
			name: "db error",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return errors.New("db error") }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "bob", "pass").Return(row)
			},
			wantErr: errors.New("failed to login user with db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mock := NewMockpgxQuerier(ctrl)

			username, password := "bob", "pass"
			if tt.name == "invalid credentials" {
				password = "wrong"
			}
			tt.setupMock(mock)

			repo := &PsqlRepository{conn: mock}
			gotID, err := repo.Login(context.Background(), username, password)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var errInvalid *ErrInvalidCredentials
				if errors.As(tt.wantErr, &errInvalid) {
					var got *ErrInvalidCredentials
					if !errors.As(err, &got) {
						t.Fatalf("expected ErrInvalidCredentials, got %v", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotID != tt.wantID {
					t.Errorf("want ID %q, got %q", tt.wantID, gotID)
				}
			}
		})
	}
}

// --- GetUserByID ---

func TestPsqlRepository_GetUserByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(m *MockpgxQuerier)
		wantUser  *models.User
		wantErr   error
	}{
		{
			name:   "success",
			userID: "uid-3",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error {
					*dest[0].(*string) = "uid-3"
					*dest[1].(*string) = "charlie"
					*dest[2].(*float32) = 500.0
					*dest[3].(*float32) = 100.0
					return nil
				}}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "uid-3").Return(row)
			},
			wantUser: &models.User{ID: "uid-3", Username: "charlie", Balance: 500.0, SumWithdrawn: 100.0},
		},
		{
			name:   "not found",
			userID: "uid-missing",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "uid-missing").Return(row)
			},
			wantErr: &ErrUserNotFound{UserID: "uid-missing"},
		},
		{
			name:   "db error",
			userID: "uid-3",
			setupMock: func(m *MockpgxQuerier) {
				row := &fakeRow{scanFn: func(dest ...any) error { return errors.New("db error") }}
				m.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "uid-3").Return(row)
			},
			wantErr: errors.New("failed to get user by ID from db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mock := NewMockpgxQuerier(ctrl)
			tt.setupMock(mock)

			repo := &PsqlRepository{conn: mock}
			gotUser, err := repo.GetUserByID(context.Background(), tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var errNotFound *ErrUserNotFound
				if errors.As(tt.wantErr, &errNotFound) {
					var got *ErrUserNotFound
					if !errors.As(err, &got) {
						t.Fatalf("expected ErrUserNotFound, got %v", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if gotUser.ID != tt.wantUser.ID || gotUser.Username != tt.wantUser.Username {
					t.Errorf("got user %+v, want %+v", gotUser, tt.wantUser)
				}
			}
		})
	}
}

// --- UpdateUser ---

func TestPsqlRepository_UpdateUser(t *testing.T) {
	tests := []struct {
		name      string
		user      *models.User
		setupMock func(m *MockpgxQuerier)
		wantErr   error
	}{
		{
			name: "success",
			user: &models.User{ID: "uid-4", Balance: 200.0, SumWithdrawn: 50.0},
			setupMock: func(m *MockpgxQuerier) {
				tag := pgconn.NewCommandTag("UPDATE 1")
				m.EXPECT().Exec(gomock.Any(), gomock.Any(), float32(200.0), float32(50.0), "uid-4").Return(tag, nil)
			},
		},
		{
			name: "user not found (zero rows affected)",
			user: &models.User{ID: "uid-missing", Balance: 0, SumWithdrawn: 0},
			setupMock: func(m *MockpgxQuerier) {
				tag := pgconn.NewCommandTag("UPDATE 0")
				m.EXPECT().Exec(gomock.Any(), gomock.Any(), float32(0), float32(0), "uid-missing").Return(tag, nil)
			},
			wantErr: &ErrUserNotFound{UserID: "uid-missing"},
		},
		{
			name: "db error",
			user: &models.User{ID: "uid-4", Balance: 200.0, SumWithdrawn: 50.0},
			setupMock: func(m *MockpgxQuerier) {
				m.EXPECT().Exec(gomock.Any(), gomock.Any(), float32(200.0), float32(50.0), "uid-4").Return(pgconn.CommandTag{}, errors.New("db error"))
			},
			wantErr: errors.New("failed to update user in db"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mock := NewMockpgxQuerier(ctrl)
			tt.setupMock(mock)

			repo := &PsqlRepository{conn: mock}
			err := repo.UpdateUser(context.Background(), tt.user)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var errNotFound *ErrUserNotFound
				if errors.As(tt.wantErr, &errNotFound) {
					var got *ErrUserNotFound
					if !errors.As(err, &got) {
						t.Fatalf("expected ErrUserNotFound, got %v", err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
