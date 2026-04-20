package models

import "time"

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
	OrderStatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Status      OrderStatus `json:"status"`
	UploadedAt  time.Time   `json:"uploaded_at"`
	ProcessedAt *time.Time  `json:"processed_at,omitempty"`
	Withdrawn   float32     `json:"withdrawn,omitempty"`
	Accrual     float32     `json:"accrual,omitempty"`
}