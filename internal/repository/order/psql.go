package order

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgsqlRepository struct {
	conn *pgxpool.Pool
}

func NewPgsqlRepository(conn *pgxpool.Pool) *PgsqlRepository {
	return &PgsqlRepository{conn: conn}
}

func (r *PgsqlRepository) LoadOrder(ctx context.Context, userID string, orderID string) (*models.Order, error) {
	var order models.Order
	var created bool

	err := r.conn.QueryRow(ctx,
		`WITH inserted AS (
			INSERT INTO orders (number, user_id)
			VALUES ($1, $2)
			ON CONFLICT (number) DO NOTHING
			RETURNING number, user_id, status, uploaded_at, accrual, withdrawn, processed_at, true AS created
		)
		SELECT number, user_id, status, uploaded_at, accrual, withdrawn, processed_at, created
		FROM inserted
		UNION ALL
		SELECT number, user_id, status, uploaded_at, accrual, withdrawn, processed_at, false AS created
		FROM orders
		WHERE number = $1 AND user_id = $2 AND NOT EXISTS (SELECT 1 FROM inserted)`,
		orderID, userID,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.UploadedAt, &order.Accrual, &order.Withdrawn, &order.ProcessedAt, &created)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &ErrOrderAlreadyExistsForAnotherUser{OrderID: orderID, UserID: userID}
		}
		return nil, fmt.Errorf("failed to load order: %w", err)
	}

	if !created {
		return &order, &ErrOrderAlreadyExists{OrderID: orderID}
	}

	return &order, nil
}

func (r *PgsqlRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]*models.Order, error) {
	rows, err := r.conn.Query(ctx, "SELECT number, user_id, status, uploaded_at, accrual, withdrawn, processed_at FROM orders WHERE user_id = $1", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user ID: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.UploadedAt, &order.Accrual, &order.Withdrawn, &order.ProcessedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, &order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return orders, nil
}
