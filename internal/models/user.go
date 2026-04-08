package models

import (
	"fmt"
	"time"
)

type User struct {
	ID           string  `json:"id"`
	Username     string  `json:"username"`
	Password     string  `json:"-"`
	Balance      float32 `json:"balance"`
	SumWithdrawn float32 `json:"withdrawn"`
}

func GenerateUserID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
