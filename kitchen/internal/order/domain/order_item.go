package domain

type OrderItem struct {
	ID                string
	OrderID           string
	ProductID         string
	ExternalProductID string
	ProductName       string
	UnitPrice         float32
	Quantity          int
	TotalSum          float32
}
