package user

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PsqlRepository struct {
	conn *pgxpool.Pool
}

func NewPgsqlRepository(conn *pgxpool.Pool) *PsqlRepository {
	return &PsqlRepository{conn: conn}
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
	err := r.conn.QueryRow(ctx, "SELECT id, username, balance, sumWithdrawn FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Username, &user.Balance, &user.SumWithdrawn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &ErrUserNotFound{UserID: userID}
		}
		return nil, fmt.Errorf("failed to get user by ID from db: %w", err)
	}
	return &user, nil
}
