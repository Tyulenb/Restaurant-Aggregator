package adapterhttp

import "time"

type CreateOrderRequest struct {
	EstablishmentID string            `json:"establishment_id"`
	Items           []CreateOrderItem `json:"items"`
}

// CreateOrderItem identifies a product and the requested quantity.
type CreateOrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Order struct {
	ID              string      `json:"id"`
	EstablishmentID string      `json:"establishment_id"`
	CustomerID      string      `json:"customer_id"`
	DeliveryID      string      `json:"delivery_id"`
	StatusID        int         `json:"status_id"`
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
