package domain

type OrderItem struct {
	ID                string
	ProductID         string
	ExternalProductID string
	ProductName       string
	UnitPrice         float32
	Quantity          int
	TotalSum          float32
}

type CreateOrderItem struct {
	ProductID string
	Quantity  int
}
