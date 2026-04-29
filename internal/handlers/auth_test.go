package handlers

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	userRepo "FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type stubAuthUserRepo struct {
	registerFn func(ctx context.Context, username, password string) (string, error)
	loginFn    func(ctx context.Context, username, password string) (string, error)
}

func (s *stubAuthUserRepo) Register(ctx context.Context, username, password string) (string, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubAuthUserRepo) Login(ctx context.Context, username, password string) (string, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubAuthUserRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return nil, nil
}

func (s *stubAuthUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	return nil
}

func newAuthHandlerForTest(repo userRepo.Repository) *AuthHandler {
	userService := services.NewUserService(repo)
	claimsService := auth.NewClaimsService()
	cnf := &config.Config{SecretKey: []byte("test-secret")}
	return NewAuthHandler(userService, claimsService, cnf)
}

func newJSONContext(body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	return ctx, rec
}

func TestAuthHandler_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		repo          *stubAuthUserRepo
		wantStatus    int
		wantErr       bool
		wantAuthUser  string
		wantAuthExist bool
	}{
		{
			name: "success",
			body: `{"login":"alice","password":"secret"}`,
			repo: &stubAuthUserRepo{
				registerFn: func(ctx context.Context, username, password string) (string, error) {
					if username != "alice" || password != "secret" {
						t.Fatalf("unexpected credentials: %s/%s", username, password)
					}
					return "u-1", nil
				},
			},
			wantStatus:    http.StatusOK,
			wantAuthExist: true,
			wantAuthUser:  "u-1",
		},
		{
			name:          "bad request on invalid json",
			body:          `{`,
			repo:          &stubAuthUserRepo{},
			wantStatus:    http.StatusBadRequest,
			wantErr:       true,
			wantAuthExist: false,
		},
		{
			name: "conflict on existing user",
			body: `{"login":"alice","password":"secret"}`,
			repo: &stubAuthUserRepo{
				registerFn: func(ctx context.Context, username, password string) (string, error) {
					return "", &userRepo.ErrUserAlreadyExists{Username: username}
				},
			},
			wantStatus:    http.StatusConflict,
			wantErr:       true,
			wantAuthExist: false,
		},
		{
			name: "internal error on generic failure",
			body: `{"login":"alice","password":"secret"}`,
			repo: &stubAuthUserRepo{
				registerFn: func(ctx context.Context, username, password string) (string, error) {
					return "", errors.New("db down")
				},
			},
			wantStatus:    http.StatusInternalServerError,
			wantErr:       true,
			wantAuthExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAuthHandlerForTest(tt.repo)
			ctx, rec := newJSONContext(tt.body)

			err := h.Register(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got err=%v", tt.wantErr, err)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			authHeader := rec.Header().Get("Authorization")
			if tt.wantAuthExist {
				if authHeader == "" {
					t.Fatal("expected Authorization header")
				}
				const prefix = "Bearer "
				if !strings.HasPrefix(authHeader, prefix) {
					t.Fatalf("expected Authorization to start with %q, got %q", prefix, authHeader)
				}
				token := strings.TrimPrefix(authHeader, prefix)
				gotUserID, parseErr := h.claimsService.GetUserIDFromJWTString(token, h.cnf.GetSecretKey())
				if parseErr != nil {
					t.Fatalf("failed to parse jwt: %v", parseErr)
				}
				if gotUserID != tt.wantAuthUser {
					t.Fatalf("expected user id %q in jwt, got %q", tt.wantAuthUser, gotUserID)
				}
			} else if authHeader != "" {
				t.Fatalf("expected no Authorization header, got %q", authHeader)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		repo          *stubAuthUserRepo
		wantStatus    int
		wantErr       bool
		wantAuthUser  string
		wantAuthExist bool
	}{
		{
			name: "success",
			body: `{"login":"alice","password":"secret"}`,
			repo: &stubAuthUserRepo{
				loginFn: func(ctx context.Context, username, password string) (string, error) {
					if username != "alice" || password != "secret" {
						t.Fatalf("unexpected credentials: %s/%s", username, password)
					}
					return "u-1", nil
				},
			},
			wantStatus:    http.StatusOK,
			wantAuthExist: true,
			wantAuthUser:  "u-1",
		},
		{
			name:          "bad request on invalid json",
			body:          `{`,
			repo:          &stubAuthUserRepo{},
			wantStatus:    http.StatusBadRequest,
			wantErr:       true,
			wantAuthExist: false,
		},
		{
			name: "unauthorized on invalid credentials",
			body: `{"login":"alice","password":"bad"}`,
			repo: &stubAuthUserRepo{
				loginFn: func(ctx context.Context, username, password string) (string, error) {
					return "", &userRepo.ErrInvalidCredentials{}
				},
			},
			wantStatus:    http.StatusUnauthorized,
			wantErr:       true,
			wantAuthExist: false,
		},
		{
			name: "internal error on generic failure",
			body: `{"login":"alice","password":"secret"}`,
			repo: &stubAuthUserRepo{
				loginFn: func(ctx context.Context, username, password string) (string, error) {
					return "", errors.New("db down")
				},
			},
			wantStatus:    http.StatusInternalServerError,
			wantErr:       true,
			wantAuthExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAuthHandlerForTest(tt.repo)
			ctx, rec := newJSONContext(tt.body)

			err := h.Login(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got err=%v", tt.wantErr, err)
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			authHeader := rec.Header().Get("Authorization")
			if tt.wantAuthExist {
				if authHeader == "" {
					t.Fatal("expected Authorization header")
				}
				const prefix = "Bearer "
				if !strings.HasPrefix(authHeader, prefix) {
					t.Fatalf("expected Authorization to start with %q, got %q", prefix, authHeader)
				}
				token := strings.TrimPrefix(authHeader, prefix)
				gotUserID, parseErr := h.claimsService.GetUserIDFromJWTString(token, h.cnf.GetSecretKey())
				if parseErr != nil {
					t.Fatalf("failed to parse jwt: %v", parseErr)
				}
				if gotUserID != tt.wantAuthUser {
					t.Fatalf("expected user id %q in jwt, got %q", tt.wantAuthUser, gotUserID)
				}
			} else if authHeader != "" {
				t.Fatalf("expected no Authorization header, got %q", authHeader)
			}
		})
	}
}
