package models

type Order struct {
	ID          string  `json:"id"`
	UserID      int     `json:"user_id"`
	Status      string  `json:"status"`
	UploadedAt  string  `json:"uploaded_at"`
	ProcessedAt string  `json:"processed_at,omitempty"`
	Withdrawn   float32 `json:"withdrawn,omitempty"`
	Accrual     float32 `json:"accrual,omitempty"`
}