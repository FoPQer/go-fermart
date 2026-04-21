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

type ErrUserNotFound struct {
	UserID string
}

func (e *ErrUserNotFound) Error() string {
	return fmt.Sprintf("user with ID %s not found", e.UserID)
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
		Balance: 1000,
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
	return nil, &ErrUserNotFound{UserID: userID}
}

func (r *MemoryRepository) UpdateUser(_ context.Context, user *models.User) error {
	for username, u := range r.users {
		if u.ID == user.ID {
			r.users[username] = user
			return nil
		}
	}
	return &ErrUserNotFound{UserID: user.ID}
}