package application

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

type OrderRepository interface {
	InsertOrder(ctx context.Context, order domain.Order) (*domain.Order, error)
	FindOrderById(ctx context.Context, orderID string) (*domain.Order, error)
	FindOrderByIdempotencyKey(ctx context.Context, idempotencyKey string) (*domain.Order, error)

	FindNewOrdersByEstablishmentId(ctx context.Context, establishmentID string) ([]domain.Order, error)

	UpdateOrderStatus(ctx context.Context, orderID string, status int) (*domain.Order, error)
}

type ProductRepository interface {
	FindProductById(ctx context.Context, productID string) (*domain.Product, error)
}
