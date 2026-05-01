package services

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/user"
	"context"
	"fmt"
)

type UserService struct {
	repo user.Repository
}

func NewUserService(repo user.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, username, password string) (string, error) {
	userID, err := s.repo.Register(ctx, username, password)
	if err != nil {
		return "", fmt.Errorf("failed to register user: %w", err)
	}
	return userID, nil
}

func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	userID, err := s.repo.Login(ctx, username, password)
	if err != nil {
		return "", fmt.Errorf("failed to login user: %w", err)
	}
	return userID, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	return user, nil
}

func (s *UserService) DoDeposit(ctx context.Context, userID string, sum float32) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	user.AddBalance(sum)

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	return nil
}

func (s *UserService) DoWithdraw(ctx context.Context, userID string, sum float32) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	if user.Balance < sum {
		return models.ErrNotEnoughFunds
	}

	if err := user.Withdraw(sum); err != nil {
		return fmt.Errorf("failed to withdraw funds: %w", err)
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	return nil
}