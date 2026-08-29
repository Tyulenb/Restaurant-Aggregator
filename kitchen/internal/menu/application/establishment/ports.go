package establishment

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type ProductRepository interface {
	SelectProductsByEstablishmentId(ctx context.Context, establishmentID string) ([]domain.Product, error)
}

type CategoryRepository interface {
	SelectCategories(ctx context.Context, establishmentID string) ([]domain.Category, error)
}

type EstablishmentRepository interface {
	SelectEstablishmentById(ctx context.Context, establishmentID string) (*domain.Establishment, error)
}
