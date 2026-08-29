package partner

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type OrderRepository interface {
	InsertOrder(ctx context.Context, order domain.Order) (*domain.Order, error)
	FindOrderByIdForUpdate(ctx context.Context, orderID string) (*domain.Order, error)
	FindOrderByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Order, error)

	FindOrdersByEstablishmentID(ctx context.Context, establishmentID string, status *int) ([]domain.Order, error)

	UpdateOrderStatus(ctx context.Context, orderID string, status int) (*domain.Order, error)
	RejectOrder(ctx context.Context, orderID string, reason *string) (*domain.Order, error)
}

type ProductRepository interface {
	FindProductByIdForUpdate(ctx context.Context, productID string) (*domain.Product, error)
	IncrementProductQuantity(ctx context.Context, productID string, quantity int) error
}

type Transactor interface {
	WithTx(context.Context, func(context.Context) error) error
}
