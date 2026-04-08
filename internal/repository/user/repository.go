package user

import (
	"FoPQer/go-fermart/internal/models"
	"context"
)

type Repository interface {
	Register(ctx context.Context, username, password string) (string, error)
	Login(ctx context.Context, username, password string) (string, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}