package product

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type ProductRepository interface {
	SelectProductsByEstablishmentIdAndCategoryExternalId(
		ctx context.Context, establishmentID, categoryExternalID string,
	) ([]domain.Product, error)
	SelectProductsByEstablishmentId(
		ctx context.Context, establishmentID string,
	) ([]domain.Product, error)
	InsertProduct(ctx context.Context, product domain.Product) (*domain.Product, error)
	SelectProductByExternalId(ctx context.Context, establishmentID, externalID string) (*domain.Product, error)
	SelectProductByExternalIdForUpdate(ctx context.Context, establishmentID, externalID string) (*domain.Product, error)
	UpdateProductByExternalId(
		ctx context.Context, establishmentID, externalID string, cmd CommandUpdateProduct,
	) (*domain.Product, error)
	DeleteProductByExternalId(ctx context.Context, establishmentID, externalID string) (*domain.Product, error)
	UpdateProductQuantity(ctx context.Context, establishmentID, externalID string, quantity int) (*domain.Product, error)
}

type CategoryRepository interface {
	SelectCategoryByExternalId(ctx context.Context, establishmentID, externalID string) (*domain.Category, error)
}

type Transactor interface {
	WithTx(context.Context, func(context.Context) error) error
}
