package services

import (
	"FoPQer/go-fermart/internal/models"
	"FoPQer/go-fermart/internal/repository/user"
)

type UserService struct {
	repo user.Repository
}

func NewUserService(repo user.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(username, password string) error {
	// TODO: Реализовать регистрацию пользователя
	return nil
}

func (s *UserService) Login(username, password string) (string, error) {
	// TODO: Реализовать вход пользователя
	return "", nil
}

func (s *UserService) GetUserInfo(userID int) (*models.User, error) {
	// TODO: Реализовать получение информации о пользователе
	return nil, nil
}

func (s *UserService) DoWithdraw(orderID int, sum int) error {
	// TODO: Реализовать снятие средств пользователем
	return nil
}