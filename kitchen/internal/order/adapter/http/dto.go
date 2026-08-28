package adapterhttp

import "time"

// CreateOrderRequest is the payload for creating a customer order.
type CreateOrderRequest struct {
	EstablishmentID string            `json:"establishment_id"`
	Items           []CreateOrderItem `json:"items"`
}

// CreateOrderItem identifies a product and the requested quantity.
type CreateOrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// Order is the customer and partner representation of an order.
type Order struct {
	ID              string      `json:"id"`
	EstablishmentID string      `json:"establishment_id"`
	CustomerID      string      `json:"customer_id"`
	DeliveryID      string      `json:"delivery_id"`
	Status          string      `json:"status"`
	TotalPrice      float32     `json:"total_price"`
	RejectionReason *string     `json:"rejection_reason"`
	Items           []OrderItem `json:"items"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// OrderItem contains the immutable product snapshot captured in an order.
type OrderItem struct {
	ID                string  `json:"id"`
	ProductID         string  `json:"product_id"`
	ExternalProductID string  `json:"external_product_id"`
	ProductName       string  `json:"product_name"`
	UnitPrice         float32 `json:"unit_price"`
	Quantity          int     `json:"quantity"`
	TotalSum          float32 `json:"total_sum"`
}

// PartnerOrderListResponse is the response returned when an establishment polls its orders.
type PartnerOrderListResponse struct {
	Items []Order `json:"items"`
}

// RejectOrderRequest is the optional payload for rejecting an order.
type RejectOrderRequest struct {
	Reason string `json:"reason"`
}

// UpdateOrderStatusRequest advances an accepted order through processing.
type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}
