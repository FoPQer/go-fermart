package services

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/user"
	"context"
	"fmt"
)

type ErrNotEnoughFunds struct {
	UserID string
	Sum    float32
}

func (e *ErrNotEnoughFunds) Error() string {
	return fmt.Sprintf("user %s has insufficient funds: %.2f", e.UserID, e.Sum)
}

type UserService struct {
	repo user.Repository
}

func NewUserService(repo user.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, username, password string) (string, error) {
	userID, err := s.repo.Register(ctx, username, password)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	userID, err := s.repo.Login(ctx, username, password)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) DoWithdraw(ctx context.Context, userID string, sum float32) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Balance < sum {
		return &ErrNotEnoughFunds{UserID: userID, Sum: sum}
	}

	user.Balance -= sum
	user.SumWithdrawn += sum

	return nil
}