package handlers

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	userRepo "FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/services"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func assertStatusCode(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("expected status %d, got %d", wantStatus, rec.Code)
	}
}

func assertBodyText(t *testing.T, rec *httptest.ResponseRecorder, wantBody string) {
	t.Helper()
	if rec.Body.String() != wantBody {
		t.Fatalf("expected body %q, got %q", wantBody, rec.Body.String())
	}
}

func decodeJSONResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var got T
	if decodeErr := json.Unmarshal(rec.Body.Bytes(), &got); decodeErr != nil {
		t.Fatalf("failed to decode json response: %v", decodeErr)
	}
	return got
}

type stubBalanceUserRepo struct {
	registerFn    func(ctx context.Context, username, password string) (string, error)
	loginFn       func(ctx context.Context, username, password string) (string, error)
	getUserByIDFn func(ctx context.Context, userID string) (*models.User, error)
	updateUserFn  func(ctx context.Context, user *models.User) error
}

func (s *stubBalanceUserRepo) Register(ctx context.Context, username, password string) (string, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubBalanceUserRepo) Login(ctx context.Context, username, password string) (string, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, username, password)
	}
	return "", nil
}

func (s *stubBalanceUserRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	if s.getUserByIDFn != nil {
		return s.getUserByIDFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubBalanceUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	if s.updateUserFn != nil {
		return s.updateUserFn(ctx, user)
	}
	return nil
}

type stubBalanceOrderRepo struct {
	loadOrderFn              func(ctx context.Context, userID string, orderID string) (*models.Order, error)
	getOrdersByUserIDFn      func(ctx context.Context, userID string) ([]*models.Order, error)
	getWithdrawalsByUserIDFn func(ctx context.Context, userID string) ([]*models.Order, error)
	updateOrderFn            func(ctx context.Context, order *models.Order) error

	loadOrderCalled bool
}

func (s *stubBalanceOrderRepo) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	s.loadOrderCalled = true
	if s.loadOrderFn != nil {
		return s.loadOrderFn(ctx, userID, orderID)
	}
	return nil, nil
}

func (s *stubBalanceOrderRepo) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	if s.getOrdersByUserIDFn != nil {
		return s.getOrdersByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubBalanceOrderRepo) GetOrdersWithdrawnByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	if s.getWithdrawalsByUserIDFn != nil {
		return s.getWithdrawalsByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubBalanceOrderRepo) UpdateOrder(ctx context.Context, order *models.Order) error {
	if s.updateOrderFn != nil {
		return s.updateOrderFn(ctx, order)
	}
	return nil
}

func newBalanceHandlerForTest(userRepo *stubBalanceUserRepo, orderRepo *stubBalanceOrderRepo) *BalanceHandler {
	userService := services.NewUserService(userRepo)
	orderService := services.NewOrderService(orderRepo, userService, &config.Config{AccrualAddress: "localhost:8080"})
	return NewBalanceHandler(userService, orderService)
}

func newBalanceContext(method, path, body, userID string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	if userID != "" {
		ctx.Set("user", &jwt.Token{Claims: jwt.MapClaims{"UserID": userID}})
	}
	return ctx, rec
}

func TestBalanceHandler_GetBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     string
		userRepo   *stubBalanceUserRepo
		wantStatus int
		wantBody   string
		wantJSON   *GetBalanceResponse
	}{
		{
			name:       "unauthorized when token is missing",
			userID:     "",
			userRepo:   &stubBalanceUserRepo{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Failed to get user from token",
		},
		{
			name:   "unauthorized when user not found",
			userID: "u-1",
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return nil, &userRepo.ErrUserNotFound{UserID: userID}
				},
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "User not found",
		},
		{
			name:   "internal error when user service fails",
			userID: "u-1",
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to get user info",
		},
		{
			name:   "returns balance",
			userID: "u-1",
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 42.5, SumWithdrawn: 10.25}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantJSON: &GetBalanceResponse{
				Current:   42.5,
				Withdrawn: 10.25,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBalanceHandlerForTest(tt.userRepo, &stubBalanceOrderRepo{})
			ctx, rec := newBalanceContext(http.MethodGet, "/api/user/balance", "", tt.userID)

			err := h.GetBalance(ctx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			assertStatusCode(t, rec, tt.wantStatus)
			if tt.wantJSON != nil {
				got := decodeJSONResponse[GetBalanceResponse](t, rec)
				if got != *tt.wantJSON {
					t.Fatalf("expected %+v, got %+v", *tt.wantJSON, got)
				}
			} else {
				assertBodyText(t, rec, tt.wantBody)
			}
		})
	}
}

func TestBalanceHandler_Withdraw(t *testing.T) {
	t.Parallel()

	const validOrderID = "79927398713"

	tests := []struct {
		name          string
		userID        string
		body          string
		userRepo      *stubBalanceUserRepo
		orderRepo     *stubBalanceOrderRepo
		wantStatus    int
		wantBody      string
		wantLoadOrder *bool
		wantJSON      bool
	}{
		{
			name:          "unauthorized when token is missing",
			userID:        "",
			body:          `{"order":"` + validOrderID + `","sum":10}`,
			userRepo:      &stubBalanceUserRepo{},
			orderRepo:     &stubBalanceOrderRepo{},
			wantStatus:    http.StatusUnauthorized,
			wantBody:      "Failed to get user from token",
			wantLoadOrder: boolPtr(false),
		},
		{
			name:          "bad request on invalid json",
			userID:        "u-1",
			body:          `{`,
			userRepo:      &stubBalanceUserRepo{},
			orderRepo:     &stubBalanceOrderRepo{},
			wantStatus:    http.StatusBadRequest,
			wantBody:      "Invalid request body",
			wantLoadOrder: boolPtr(false),
		},
		{
			name:          "unprocessable entity for wrong order format",
			userID:        "u-1",
			body:          `{"order":"12345","sum":10}`,
			userRepo:      &stubBalanceUserRepo{},
			orderRepo:     &stubBalanceOrderRepo{},
			wantStatus:    http.StatusUnprocessableEntity,
			wantBody:      "Wrong order ID format",
			wantLoadOrder: boolPtr(false),
		},
		{
			name:   "internal error on load order failure",
			userID: "u-1",
			body:   `{"order":"` + validOrderID + `","sum":10}`,
			userRepo: &stubBalanceUserRepo{},
			orderRepo: &stubBalanceOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return nil, errors.New("load failed")
				},
			},
			wantStatus:    http.StatusInternalServerError,
			wantBody:      "Failed to load order",
			wantLoadOrder: boolPtr(true),
		},
		{
			name:   "payment required when balance is insufficient",
			userID: "u-1",
			body:   `{"order":"` + validOrderID + `","sum":10}`,
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 5}, nil
				},
			},
			orderRepo: &stubBalanceOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID}, nil
				},
			},
			wantStatus:    http.StatusPaymentRequired,
			wantBody:      "Insufficient balance",
			wantLoadOrder: boolPtr(true),
		},
		{
			name:   "internal error when withdrawal processing fails",
			userID: "u-1",
			body:   `{"order":"` + validOrderID + `","sum":10}`,
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 100}, nil
				},
				updateUserFn: func(ctx context.Context, user *models.User) error {
					return errors.New("update failed")
				},
			},
			orderRepo: &stubBalanceOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID}, nil
				},
			},
			wantStatus:    http.StatusInternalServerError,
			wantBody:      "Failed to process withdrawal",
			wantLoadOrder: boolPtr(true),
		},
		{
			name:   "internal error when order update fails",
			userID: "u-1",
			body:   `{"order":"` + validOrderID + `","sum":10}`,
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 100}, nil
				},
			},
			orderRepo: &stubBalanceOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID}, nil
				},
				updateOrderFn: func(ctx context.Context, order *models.Order) error {
					return errors.New("write failed")
				},
			},
			wantStatus:    http.StatusInternalServerError,
			wantBody:      "Failed to update order with withdrawal info",
			wantLoadOrder: boolPtr(true),
		},
		{
			name:   "returns updated order on success",
			userID: "u-1",
			body:   `{"order":"` + validOrderID + `","sum":10}`,
			userRepo: &stubBalanceUserRepo{
				getUserByIDFn: func(ctx context.Context, userID string) (*models.User, error) {
					return &models.User{ID: userID, Balance: 100}, nil
				},
			},
			orderRepo: &stubBalanceOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID}, nil
				},
			},
			wantStatus:    http.StatusOK,
			wantLoadOrder: boolPtr(true),
			wantJSON:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBalanceHandlerForTest(tt.userRepo, tt.orderRepo)
			ctx, rec := newBalanceContext(http.MethodPost, "/api/user/balance/withdraw", tt.body, tt.userID)

			err := h.Withdraw(ctx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			assertStatusCode(t, rec, tt.wantStatus)
			if tt.wantLoadOrder != nil && tt.orderRepo.loadOrderCalled != *tt.wantLoadOrder {
				t.Fatalf("expected LoadOrder called=%v, got %v", *tt.wantLoadOrder, tt.orderRepo.loadOrderCalled)
			}

			if tt.wantJSON {
				got := decodeJSONResponse[models.Order](t, rec)
				if got.Withdrawn != 10 {
					t.Fatalf("expected withdrawn 10, got %.2f", got.Withdrawn)
				}
			} else {
				assertBodyText(t, rec, tt.wantBody)
			}
		})
	}
}

func TestBalanceHandler_GetWithdrawals(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name       string
		userID     string
		orderRepo  *stubBalanceOrderRepo
		wantStatus int
		wantBody   string
		wantCount  int
	}{
		{
			name:       "unauthorized when token is missing",
			userID:     "",
			orderRepo:  &stubBalanceOrderRepo{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Failed to get user from token",
		},
		{
			name:   "internal error on service failure",
			userID: "u-1",
			orderRepo: &stubBalanceOrderRepo{
				getWithdrawalsByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Failed to get withdrawals",
		},
		{
			name:   "no content when withdrawals are empty",
			userID: "u-1",
			orderRepo: &stubBalanceOrderRepo{
				getWithdrawalsByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return []*models.Order{}, nil
				},
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "returns mapped withdrawals",
			userID: "u-1",
			orderRepo: &stubBalanceOrderRepo{
				getWithdrawalsByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return []*models.Order{{ID: "1", Withdrawn: 100, ProcessedAt: &now}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBalanceHandlerForTest(&stubBalanceUserRepo{}, tt.orderRepo)
			ctx, rec := newBalanceContext(http.MethodGet, "/api/user/withdrawals", "", tt.userID)

			err := h.GetWithdrawals(ctx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			assertStatusCode(t, rec, tt.wantStatus)

			if tt.wantCount > 0 {
				got := decodeJSONResponse[[]WithdrawResponse](t, rec)
				if len(got) != tt.wantCount {
					t.Fatalf("expected %d withdrawals, got %d", tt.wantCount, len(got))
				}
				if got[0].Order != "1" || got[0].Sum != 100 || got[0].ProcessedAt == nil {
					t.Fatalf("unexpected first withdrawal: %+v", got[0])
				}
			} else if tt.wantBody != "" {
				assertBodyText(t, rec, tt.wantBody)
			}
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}
