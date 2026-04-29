package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

type fakeRows struct {
	rowsData []models.Order
	index    int
	scanErr  error
	rowsErr  error
}

func (r *fakeRows) Close() {}

func (r *fakeRows) Err() error {
	return r.rowsErr
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	if r.index >= len(r.rowsData) {
		return false
	}
	r.index++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if r.index == 0 || r.index > len(r.rowsData) {
		return fmt.Errorf("scan called with invalid cursor state")
	}
	current := r.rowsData[r.index-1]

	switch len(dest) {
	case 7:
		*dest[0].(*string) = current.ID
		*dest[1].(*string) = current.UserID
		*dest[2].(*models.OrderStatus) = current.Status
		*dest[3].(*time.Time) = current.UploadedAt
		*dest[4].(*float32) = current.Accrual
		*dest[5].(*float32) = current.Withdrawn
		*dest[6].(**time.Time) = current.ProcessedAt
		return nil
	case 4:
		*dest[0].(*string) = current.ID
		*dest[1].(*string) = current.UserID
		*dest[2].(*float32) = current.Withdrawn
		*dest[3].(**time.Time) = current.ProcessedAt
		return nil
	default:
		return fmt.Errorf("unexpected scan destination length: %d", len(dest))
	}
}

func (r *fakeRows) Values() ([]any, error) {
	return nil, nil
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}

func TestPgsqlRepository_LoadOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	processedAt := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name               string
		row                pgx.Row
		wantErrContains    string
		wantAlreadyExists  bool
		wantAnotherUserErr bool
		wantCreatedOrder   bool
	}{
		{
			name: "maps no rows to another user error",
			row: &fakeRow{scanFn: func(dest ...any) error {
				return pgx.ErrNoRows
			}},
			wantAnotherUserErr: true,
		},
		{
			name: "wraps scan error",
			row: &fakeRow{scanFn: func(dest ...any) error {
				return errors.New("scan failed")
			}},
			wantErrContains: "failed to load order",
		},
		{
			name: "returns already exists for same user",
			row: &fakeRow{scanFn: func(dest ...any) error {
				*dest[0].(*string) = "79927398713"
				*dest[1].(*string) = "u-1"
				*dest[2].(*models.OrderStatus) = models.OrderStatusProcessed
				*dest[3].(*time.Time) = time.Now().UTC()
				*dest[4].(*float32) = 10
				*dest[5].(*float32) = 0
				*dest[6].(**time.Time) = &processedAt
				*dest[7].(*bool) = false
				return nil
			}},
			wantAlreadyExists: true,
			wantCreatedOrder:  true,
		},
		{
			name: "returns created order without error",
			row: &fakeRow{scanFn: func(dest ...any) error {
				*dest[0].(*string) = "79927398713"
				*dest[1].(*string) = "u-1"
				*dest[2].(*models.OrderStatus) = models.OrderStatusNew
				*dest[3].(*time.Time) = time.Now().UTC()
				*dest[4].(*float32) = 0
				*dest[5].(*float32) = 0
				*dest[6].(**time.Time) = nil
				*dest[7].(*bool) = true
				return nil
			}},
			wantCreatedOrder: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConn := NewMockpgxQuerier(ctrl)
			mockConn.EXPECT().QueryRow(gomock.Any(), gomock.Any(), "79927398713", "u-1").Return(tt.row)

			repo := &PgsqlRepository{conn: mockConn}
			order, err := repo.LoadOrder(ctx, "u-1", "79927398713")

			if tt.wantAnotherUserErr {
				var anotherUserErr *ErrOrderAlreadyExistsForAnotherUser
				if !errors.As(err, &anotherUserErr) {
					t.Fatalf("expected ErrOrderAlreadyExistsForAnotherUser, got %T", err)
				}
				return
			}

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("expected error to contain %q, got %v", tt.wantErrContains, err)
				}
				return
			}

			if tt.wantAlreadyExists {
				var existsErr *ErrOrderAlreadyExists
				if !errors.As(err, &existsErr) {
					t.Fatalf("expected ErrOrderAlreadyExists, got %T", err)
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.wantCreatedOrder {
				if order == nil {
					t.Fatal("expected non-nil order")
				}
				if order.ID != "79927398713" || order.UserID != "u-1" {
					t.Fatalf("unexpected order: %+v", order)
				}
			}
		})
	}
}

func TestPgsqlRepository_GetOrdersByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	uploadedAt := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name            string
		queryRows       pgx.Rows
		queryErr        error
		wantErrContains string
		wantCount       int
	}{
		{
			name:            "wraps query error",
			queryErr:        errors.New("query failed"),
			wantErrContains: "failed to get orders by user ID",
		},
		{
			name: "wraps scan error",
			queryRows: &fakeRows{
				rowsData: []models.Order{{ID: "1", UserID: "u-1", UploadedAt: uploadedAt}},
				scanErr:  errors.New("scan failed"),
			},
			wantErrContains: "failed to scan order",
		},
		{
			name: "wraps iteration error",
			queryRows: &fakeRows{
				rowsData: []models.Order{{ID: "1", UserID: "u-1", UploadedAt: uploadedAt}},
				rowsErr:  errors.New("rows err"),
			},
			wantErrContains: "rows iteration error",
		},
		{
			name: "returns orders",
			queryRows: &fakeRows{rowsData: []models.Order{{
				ID:         "1",
				UserID:     "u-1",
				Status:     models.OrderStatusProcessed,
				UploadedAt: uploadedAt,
				Accrual:    10,
				Withdrawn:  2,
			}}},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConn := NewMockpgxQuerier(ctrl)
			mockConn.EXPECT().Query(gomock.Any(), gomock.Any(), "u-1").Return(tt.queryRows, tt.queryErr)

			repo := &PgsqlRepository{conn: mockConn}
			orders, err := repo.GetOrdersByUserID(ctx, "u-1")

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("expected error to contain %q, got %v", tt.wantErrContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(orders) != tt.wantCount {
				t.Fatalf("expected %d orders, got %d", tt.wantCount, len(orders))
			}
		})
	}
}

func TestPgsqlRepository_GetOrdersWithdrawnByUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	processedAt := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name            string
		queryRows       pgx.Rows
		queryErr        error
		wantErrContains string
		wantCount       int
	}{
		{
			name:            "wraps query error",
			queryErr:        errors.New("query failed"),
			wantErrContains: "failed to get withdrawn orders by user ID",
		},
		{
			name: "wraps scan error",
			queryRows: &fakeRows{
				rowsData: []models.Order{{ID: "1", UserID: "u-1", Withdrawn: 50, ProcessedAt: &processedAt}},
				scanErr:  errors.New("scan failed"),
			},
			wantErrContains: "failed to scan order",
		},
		{
			name: "wraps iteration error",
			queryRows: &fakeRows{
				rowsData: []models.Order{{ID: "1", UserID: "u-1", Withdrawn: 50, ProcessedAt: &processedAt}},
				rowsErr:  errors.New("rows err"),
			},
			wantErrContains: "rows iteration error",
		},
		{
			name: "returns withdrawn orders",
			queryRows: &fakeRows{rowsData: []models.Order{{
				ID:          "1",
				UserID:      "u-1",
				Withdrawn:   50,
				ProcessedAt: &processedAt,
			}}},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockConn := NewMockpgxQuerier(ctrl)
			mockConn.EXPECT().Query(gomock.Any(), gomock.Any(), "u-1").Return(tt.queryRows, tt.queryErr)

			repo := &PgsqlRepository{conn: mockConn}
			orders, err := repo.GetOrdersWithdrawnByUserID(ctx, "u-1")

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("expected error to contain %q, got %v", tt.wantErrContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(orders) != tt.wantCount {
				t.Fatalf("expected %d orders, got %d", tt.wantCount, len(orders))
			}
		})
	}
}

func TestPgsqlRepository_UpdateOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	order := &models.Order{
		ID:         "79927398713",
		Status:     models.OrderStatusProcessed,
		Accrual:    100,
		Withdrawn:  10,
		ProcessedAt: nil,
	}

	t.Run("wraps exec error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := NewMockpgxQuerier(ctrl)
		mockConn.EXPECT().Exec(gomock.Any(), gomock.Any(), order.Status, order.Accrual, order.Withdrawn, order.ProcessedAt, order.ID).
			Return(pgconn.CommandTag{}, errors.New("exec failed"))

		repo := &PgsqlRepository{conn: mockConn}
		err := repo.UpdateOrder(ctx, order)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to update order") {
			t.Fatalf("expected wrapped error, got %v", err)
		}
	})

	t.Run("updates successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := NewMockpgxQuerier(ctrl)
		mockConn.EXPECT().Exec(gomock.Any(), gomock.Any(), order.Status, order.Accrual, order.Withdrawn, order.ProcessedAt, order.ID).
			Return(pgconn.CommandTag{}, nil)

		repo := &PgsqlRepository{conn: mockConn}
		err := repo.UpdateOrder(ctx, order)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
