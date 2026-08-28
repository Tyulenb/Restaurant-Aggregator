package domain

import "time"

type OrderStatusID int

const (
	OrderCreated   OrderStatusID = 1
	OrderAccepted  OrderStatusID = 2
	OrderPreparing OrderStatusID = 3
	OrderReady     OrderStatusID = 4
	OrderCompleted OrderStatusID = 5
	OrderRejected  OrderStatusID = 6
	OrderCanceled  OrderStatusID = 7
)

type Order struct {
	ID              string
	IdempotencyKey  string
	EstablishmentID string
	CustomerID      string
	DeliveryID      string
	StatusID        OrderStatusID
	TotalPrice      float32
	RejectionReason *string
	Items           []OrderItem
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
