package models

import (
	"errors"
	"fmt"
	"time"
)

var ErrNotEnoughFunds = errors.New("not enough funds")
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
		return ErrNotEnoughFunds
	}
	u.Balance -= amount
	u.SumWithdrawn += amount
	return nil
}

func GenerateUserID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
