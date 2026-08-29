package customer

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type OrderRepository interface {
	InsertOrder(ctx context.Context, order domain.Order) (*domain.Order, error)
	FindOrderById(ctx context.Context, orderID string) (*domain.Order, error)
	FindOrderByIdForUpdate(ctx context.Context, orderID string) (*domain.Order, error)
	FindOrderByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Order, error)

	UpdateOrderStatus(ctx context.Context, orderID string, status int) (*domain.Order, error)
	InsertOrderItem(ctx context.Context, oi domain.OrderItem) (*domain.OrderItem, error)
}

type ProductRepository interface {
	FindProductById(ctx context.Context, productID string) (*domain.Product, error)
	FindProductByIdForUpdate(ctx context.Context, productID string) (*domain.Product, error)
	DecrementProductQuantity(ctx context.Context, productID string, quantity int) error
	IncrementProductQuantity(ctx context.Context, productID string, quantity int) error
}

type EstablishmentRepository interface {
	FindEstablishmentById(ctx context.Context, establishmentId string) (*domain.Establishment, error)
}

type Transactor interface {
	WithTx(context.Context, func(context.Context) error) error
}
