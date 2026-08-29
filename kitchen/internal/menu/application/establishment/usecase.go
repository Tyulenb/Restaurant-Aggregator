package establishment

import (
	"context"
	"fmt"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type EstablishmentUseCase struct {
	ProductRepo       ProductRepository
	CategoryRepo      CategoryRepository
	EstablishmentRepo EstablishmentRepository
}

func NewEstablishmentUseCase(
	productRepo ProductRepository,
	categoryRepo CategoryRepository,
	establishmentRepo EstablishmentRepository,
) *EstablishmentUseCase {
	return &EstablishmentUseCase{
		ProductRepo:       productRepo,
		CategoryRepo:      categoryRepo,
		EstablishmentRepo: establishmentRepo,
	}
}

func (uc *EstablishmentUseCase) GetMenu(ctx context.Context, establishmentID string) (domain.Menu, error) {
	establishment, err := uc.EstablishmentRepo.SelectEstablishmentById(ctx, establishmentID)
	if err != nil {
		return domain.Menu{}, fmt.Errorf("select establishment %q: %w", establishmentID, err)
	}
	if establishment == nil {
		return domain.Menu{}, domain.ErrEstablishmentNotFound
	}

	categories, err := uc.CategoryRepo.SelectCategories(ctx, establishmentID)
	if err != nil {
		return domain.Menu{}, fmt.Errorf("select categories for establishment %q: %w", establishmentID, err)
	}

	products, err := uc.ProductRepo.SelectProductsByEstablishmentId(ctx, establishmentID)
	if err != nil {
		return domain.Menu{}, fmt.Errorf("select products for establishment %q: %w", establishmentID, err)
	}

	menuCategories := make([]domain.MenuCategory, 0, len(categories))
	categoryIndexes := make(map[string]int, len(categories))
	for _, category := range categories {
		if category.DeletedAt != nil {
			continue
		}

		categoryIndexes[category.ID] = len(menuCategories)
		menuCategories = append(menuCategories, domain.MenuCategory{
			ID:          category.ID,
			ExternalID:  category.ExternalID,
			Name:        category.Name,
			Description: category.Description,
			Products:    make([]domain.Product, 0),
		})
	}

	for _, product := range products {
		categoryIndex, categoryExists := categoryIndexes[product.CategoryID]
		if product.DeletedAt != nil || !categoryExists {
			continue
		}

		menuCategories[categoryIndex].Products = append(menuCategories[categoryIndex].Products, product)
	}

	return domain.Menu{
		Establishment: *establishment,
		Categories:    menuCategories,
	}, nil
}
