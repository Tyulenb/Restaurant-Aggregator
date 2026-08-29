package category

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type CategoryRepository interface {
	SelectCategoriesWithDeleted(ctx context.Context, establishmentID string) ([]domain.Category, error)
	SelectCategories(ctx context.Context, establishmentID string) ([]domain.Category, error)
	SelectCategoryByExternalId(ctx context.Context, establishmentID, externalID string) (*domain.Category, error)
	InsertCategory(ctx context.Context, category domain.Category) (*domain.Category, error)

	UpdateCategoryNameByExternalId(
		ctx context.Context, establishmentID, externalID, name string,
	) (*domain.Category, error)
	UpdateCategoryDescriptionByExternalId(
		ctx context.Context, establishmentID, externalID string, description *string,
	) (*domain.Category, error)
	UpdateCategoryNameAndDescriptionByExternalId(
		ctx context.Context, establishmentID, externalID, name string, description *string,
	) (*domain.Category, error)
	DeleteCategoryByExternalId(ctx context.Context, establishmentID, externalID string) (*domain.Category, error)
}

type ProductRepository interface {
	DeleteProductByCategoryId(ctx context.Context, categoryId string) ([]domain.Product, error)
}

type Transactor interface {
	WithTx(context.Context, func(context.Context) error) error
}
