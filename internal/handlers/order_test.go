package handlers

import (
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/models"
	orderRepo "FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/services"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type stubOrderHandlerUserRepo struct{}

func (s *stubOrderHandlerUserRepo) Register(ctx context.Context, username, password string) (string, error) {
	return "", nil
}

func (s *stubOrderHandlerUserRepo) Login(ctx context.Context, username, password string) (string, error) {
	return "", nil
}

func (s *stubOrderHandlerUserRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return &models.User{ID: userID, Balance: 0}, nil
}

func (s *stubOrderHandlerUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	return nil
}

type stubOrderHandlerOrderRepo struct {
	loadOrderFn         func(ctx context.Context, userID string, orderID string) (*models.Order, error)
	getOrdersByUserIDFn func(ctx context.Context, userID string) ([]*models.Order, error)
}

func (s *stubOrderHandlerOrderRepo) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	if s.loadOrderFn != nil {
		return s.loadOrderFn(ctx, userID, orderID)
	}
	return nil, nil
}

func (s *stubOrderHandlerOrderRepo) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	if s.getOrdersByUserIDFn != nil {
		return s.getOrdersByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubOrderHandlerOrderRepo) GetOrdersWithdrawnByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	return nil, nil
}

func (s *stubOrderHandlerOrderRepo) GetUnprocessedOrders(ctx context.Context) ([]*models.Order, error) {
	return nil, nil
}

func (s *stubOrderHandlerOrderRepo) UpdateOrder(ctx context.Context, order *models.Order) error {
	return nil
}

func newOrderHandlerForTest(orderRepo *stubOrderHandlerOrderRepo) *OrderHandler {
	userService := services.NewUserService(&stubOrderHandlerUserRepo{})
	orderService := services.NewOrderService(orderRepo, userService, &config.Config{AccrualAddress: "localhost:8080"}, nil)
	return NewOrderHandler(orderService)
}

func newOrderContext(method, path, body, userID string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	if userID != "" {
		ctx.Set("user", &jwt.Token{Claims: jwt.MapClaims{"UserID": userID}})
	}
	return ctx, rec
}

type failingReadCloser struct{}

func (f failingReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func (f failingReadCloser) Close() error {
	return nil
}

func TestOrderHandler_LoadOrder(t *testing.T) {
	const validOrderID = "79927398713"

	tests := []struct {
		name       string
		userID     string
		body       string
		repo       *stubOrderHandlerOrderRepo
		badBody    bool
		wantStatus int
		wantBody   string
		wantJSON   bool
	}{
		{
			name:       "bad request when body cannot be read",
			userID:     "u-1",
			repo:       &stubOrderHandlerOrderRepo{},
			badBody:    true,
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid request body",
		},
		{
			name:       "unauthorized when token is missing",
			userID:     "",
			body:       validOrderID,
			repo:       &stubOrderHandlerOrderRepo{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Failed to get user from token",
		},
		{
			name:       "unprocessable entity for wrong order format",
			userID:     "u-1",
			body:       "12345",
			repo:       &stubOrderHandlerOrderRepo{},
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Wrong order ID format",
		},
		{
			name:   "conflict when order belongs to another user",
			userID: "u-1",
			body:   validOrderID,
			repo: &stubOrderHandlerOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return nil, &orderRepo.ErrOrderAlreadyExistsForAnotherUser{OrderID: orderID, UserID: "u-2"}
				},
			},
			wantStatus: http.StatusConflict,
			wantBody:   "Order already exists for another user",
		},
		{
			name:   "returns existing order with status ok",
			userID: "u-1",
			body:   validOrderID,
			repo: &stubOrderHandlerOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID, Status: models.OrderStatusNew}, &orderRepo.ErrOrderAlreadyExists{OrderID: orderID}
				},
			},
			wantStatus: http.StatusOK,
			wantJSON:   true,
		},
		{
			name:   "internal server error on generic failure",
			userID: "u-1",
			body:   validOrderID,
			repo: &stubOrderHandlerOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   http.StatusText(http.StatusInternalServerError),
		},
		{
			name:   "accepts new order",
			userID: "u-1",
			body:   validOrderID,
			repo: &stubOrderHandlerOrderRepo{
				loadOrderFn: func(ctx context.Context, userID string, orderID string) (*models.Order, error) {
					return &models.Order{ID: orderID, UserID: userID, Status: models.OrderStatusNew}, nil
				},
			},
			wantStatus: http.StatusAccepted,
			wantJSON:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newOrderHandlerForTest(tt.repo)
			ctx, rec := newOrderContext(http.MethodPost, "/api/user/orders", tt.body, tt.userID)
			if tt.badBody {
				ctx.Request().Body = io.NopCloser(failingReadCloser{})
			}

			err := h.LoadOrder(ctx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			assertStatusCode(t, rec, tt.wantStatus)

			if tt.wantJSON {
				got := decodeJSONResponse[models.Order](t, rec)
				if got.ID != validOrderID {
					t.Fatalf("expected order id %s, got %s", validOrderID, got.ID)
				}
			} else {
				assertBodyText(t, rec, tt.wantBody)
			}
		})
	}
}

func TestOrderHandler_GetOrders(t *testing.T) {
	now := time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		userID     string
		repo       *stubOrderHandlerOrderRepo
		wantStatus int
		wantBody   string
		wantCount  int
	}{
		{
			name:       "unauthorized when token is missing",
			userID:     "",
			repo:       &stubOrderHandlerOrderRepo{},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Failed to get user from token",
		},
		{
			name:   "internal error on service failure",
			userID: "u-1",
			repo: &stubOrderHandlerOrderRepo{
				getOrdersByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   http.StatusText(http.StatusInternalServerError),
		},
		{
			name:   "returns mapped orders",
			userID: "u-1",
			repo: &stubOrderHandlerOrderRepo{
				getOrdersByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return []*models.Order{{
						ID:         "79927398713",
						Status:     models.OrderStatusProcessed,
						Accrual:    123.45,
						UploadedAt: now,
					}}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:   "returns empty list",
			userID: "u-1",
			repo: &stubOrderHandlerOrderRepo{
				getOrdersByUserIDFn: func(ctx context.Context, userID string) ([]*models.Order, error) {
					return []*models.Order{}, nil
				},
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newOrderHandlerForTest(tt.repo)
			ctx, rec := newOrderContext(http.MethodGet, "/api/user/orders", "", tt.userID)

			err := h.GetOrders(ctx)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			assertStatusCode(t, rec, tt.wantStatus)

			if tt.wantBody != "" {
				assertBodyText(t, rec, tt.wantBody)
				return
			}

			got := decodeJSONResponse[[]OrderResponse](t, rec)
			if len(got) != tt.wantCount {
				t.Fatalf("expected %d orders, got %d", tt.wantCount, len(got))
			}
			if tt.wantCount == 0 {
				return
			}

			if got[0].OrderID != "79927398713" {
				t.Fatalf("expected order id 79927398713, got %s", got[0].OrderID)
			}
			if got[0].Status != string(models.OrderStatusProcessed) {
				t.Fatalf("expected status %s, got %s", models.OrderStatusProcessed, got[0].Status)
			}
			if got[0].Accrual != 123.45 {
				t.Fatalf("expected accrual 123.45, got %.2f", got[0].Accrual)
			}
			if got[0].UploadedAt != now.Format(time.RFC3339) {
				t.Fatalf("expected uploaded_at %s, got %s", now.Format(time.RFC3339), got[0].UploadedAt)
			}
		})
	}
}
