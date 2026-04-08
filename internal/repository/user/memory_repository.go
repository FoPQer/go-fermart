package user

import (
	"FoPQer/go-fermart/internal/models"
	"context"
	"fmt"
)

type ErrUserAlreadyExists struct {
	Username string
}

func (e *ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("user with username %s already exists", e.Username)
}

type ErrInvalidCredentials struct{}

func (e *ErrInvalidCredentials) Error() string {
	return "invalid username or password"
}

type MemoryRepository struct {
	users map[string]*models.User
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users: make(map[string]*models.User),
	}
}

func (r *MemoryRepository) Register(_ context.Context, username, password string) (string, error) {
	if _, exists := r.users[username]; exists {
		return "", &ErrUserAlreadyExists{Username: username}
	}
	userID := models.GenerateUserID()
	r.users[username] = &models.User{
		ID: 	  userID,
		Username: username,
		Password: password,
	}
	return userID, nil
}

func (r *MemoryRepository) Login(_ context.Context, username, password string) (string, error) {
	user, exists := r.users[username]
	if !exists || user.Password != password {
		return "", &ErrInvalidCredentials{}
	}
	return user.ID, nil
}

func (r *MemoryRepository) GetUserByID(_ context.Context, userID string) (*models.User, error) {
	for _, user := range r.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user with ID %s not found", userID)
}