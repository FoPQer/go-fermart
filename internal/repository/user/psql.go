package user

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/txutil"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PsqlRepository struct {
	conn pgxQuerier
}

func NewPgsqlRepository(conn *pgxpool.Pool) *PsqlRepository {
	return &PsqlRepository{conn: conn}
}

func (r *PsqlRepository) querier(ctx context.Context) pgxQuerier {
	if tx, ok := txutil.TxFromContext(ctx); ok {
		return tx
	}
	return r.conn
}

func (r *PsqlRepository) Register(ctx context.Context, username, password string) (string, error) {
	var userID string
	err := r.conn.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING RETURNING id", username, password).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &ErrUserAlreadyExists{Username: username}
		}
		return "", fmt.Errorf("failed to register user: %w", err)
	}

	return userID, nil
}

func (r *PsqlRepository) Login(ctx context.Context, username, password string) (string, error) {
	var userID string
	err := r.conn.QueryRow(ctx, "SELECT id FROM users WHERE username = $1 AND password_hash = $2", username, password).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &ErrInvalidCredentials{}
		}
		return "", fmt.Errorf("failed to login user with db: %w", err)
	}
	return userID, nil
}

func (r *PsqlRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := r.querier(ctx).QueryRow(ctx, "SELECT id, username, balance, sumWithdrawn FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Username, &user.Balance, &user.SumWithdrawn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &ErrUserNotFound{UserID: userID}
		}
		return nil, fmt.Errorf("failed to get user by ID from db: %w", err)
	}
	return &user, nil
}

func (r *PsqlRepository) UpdateUser(ctx context.Context, user *models.User) error {
	commandTag, err := r.querier(ctx).Exec(ctx, "UPDATE users SET balance = $1, sumWithdrawn = $2 WHERE id = $3", user.Balance, user.SumWithdrawn, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user in db: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return &ErrUserNotFound{UserID: user.ID}
	}
	return nil
}
