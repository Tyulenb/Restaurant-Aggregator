package domain

import "time"

type Order struct {
	ID              string
	IdempotencyKey  string
	EstablishmentID string
	CustomerID      string
	DeliveryID      string
	Status          string
	TotalPrice      float32
	RejectionReason *string
	Items           []OrderItem
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
