package models

import (
	"fmt"
	"time"
)

type ErrNotEnoughFunds struct {
	UserID string
	Sum    float32
}

func (e *ErrNotEnoughFunds) Error() string {
	return fmt.Sprintf("user %s has insufficient funds: %.2f", e.UserID, e.Sum)
}

type User struct {
	ID           string  `json:"id"`
	Username     string  `json:"username"`
	Password     string  `json:"-"`
	Balance      float32 `json:"balance"`
	SumWithdrawn float32 `json:"withdrawn"`
}

func (u *User) AddBalance(amount float32) {
	u.Balance += amount
}

func (u *User) Withdraw(amount float32) error {
	if u.Balance < amount {
		return &ErrNotEnoughFunds{UserID: u.ID, Sum: amount}
	}
	u.Balance -= amount
	u.SumWithdrawn += amount
	return nil
}

func GenerateUserID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
