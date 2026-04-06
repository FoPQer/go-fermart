package models

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"`
	Balance   int    `json:"balance"`
	Withdrawn int    `json:"withdrawn"`
}
