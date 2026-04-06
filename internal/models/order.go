package models

type Order struct {
	ID         string `json:"id"`
	UserID     int    `json:"user_id"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
}