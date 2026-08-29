package establishment_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

const establishmentID = "establishment-1"

func TestEstablishmentUseCase_GetMenu_BuildsActiveCategoryTree(t *testing.T) {
	ctx := t.Context()
	deletedAt := time.Now()
	description := "Fresh bread and coffee"
	selectedEstablishment := &domain.Establishment{
		ID: establishmentID, Name: "Bakery", Description: &description, Available: true,
	}
	categories := []domain.Category{
		{ID: "category-1", EstablishmentID: establishmentID, ExternalID: "bakery", Name: "Bakery"},
		{ID: "category-2", EstablishmentID: establishmentID, ExternalID: "drinks", Name: "Drinks"},
		{ID: "category-deleted", EstablishmentID: establishmentID, ExternalID: "old", Name: "Old", DeletedAt: &deletedAt},
	}
	products := []domain.Product{
		{ID: "product-1", CategoryID: "category-1", Name: "Baguette", Available: true},
		{ID: "product-2", CategoryID: "category-1", Name: "Sold out bread", Available: false},
		{ID: "product-3", CategoryID: "category-2", Name: "Coffee", Available: true},
		{ID: "product-deleted", CategoryID: "category-1", Name: "Old bread", DeletedAt: &deletedAt},
		{ID: "product-in-deleted-category", CategoryID: "category-deleted", Name: "Old product"},
		{ID: "orphan-product", CategoryID: "unknown-category", Name: "Orphan"},
	}
	useCase, productRepo, categoryRepo, establishmentRepo := newUseCase(t)
	establishmentRepo.EXPECT().SelectEstablishmentById(ctx, establishmentID).
		Return(selectedEstablishment, nil).Once()
	categoryRepo.EXPECT().SelectCategories(ctx, establishmentID).Return(categories, nil).Once()
	productRepo.EXPECT().SelectProductsByEstablishmentId(ctx, establishmentID).Return(products, nil).Once()

	menu, err := useCase.GetMenu(ctx, establishmentID)

	require.NoError(t, err)
	assert.Equal(t, *selectedEstablishment, menu.Establishment)
	require.Len(t, menu.Categories, 2)
	assert.Equal(t, domain.MenuCategory{
		ID: "category-1", ExternalID: "bakery", Name: "Bakery",
		Products: []domain.Product{products[0], products[1]},
	}, menu.Categories[0])
	assert.Equal(t, domain.MenuCategory{
		ID: "category-2", ExternalID: "drinks", Name: "Drinks",
		Products: []domain.Product{products[2]},
	}, menu.Categories[1])
}

func TestEstablishmentUseCase_GetMenu_ReturnsEmptyNonNilCategories(t *testing.T) {
	ctx := t.Context()
	selectedEstablishment := &domain.Establishment{ID: establishmentID, Name: "Bakery"}
	useCase, productRepo, categoryRepo, establishmentRepo := newUseCase(t)
	establishmentRepo.EXPECT().SelectEstablishmentById(ctx, establishmentID).
		Return(selectedEstablishment, nil).Once()
	categoryRepo.EXPECT().SelectCategories(ctx, establishmentID).Return(nil, nil).Once()
	productRepo.EXPECT().SelectProductsByEstablishmentId(ctx, establishmentID).Return(nil, nil).Once()

	menu, err := useCase.GetMenu(ctx, establishmentID)

	require.NoError(t, err)
	assert.Equal(t, *selectedEstablishment, menu.Establishment)
	assert.Empty(t, menu.Categories)
	assert.NotNil(t, menu.Categories)
}

func TestEstablishmentUseCase_GetMenu_HandlesRepositoryFailures(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name               string
		establishment      *domain.Establishment
		establishmentError error
		categoryError      error
		productError       error
		wantErr            error
		wantText           string
	}{
		{
			name: "establishment lookup error", establishmentError: repositoryErr,
			wantErr: repositoryErr, wantText: `select establishment "` + establishmentID + `"`,
		},
		{
			name: "establishment not found", wantErr: domain.ErrEstablishmentNotFound,
		},
		{
			name: "category lookup error", establishment: &domain.Establishment{ID: establishmentID},
			categoryError: repositoryErr, wantErr: repositoryErr,
			wantText: `select categories for establishment "` + establishmentID + `"`,
		},
		{
			name: "product lookup error", establishment: &domain.Establishment{ID: establishmentID},
			productError: repositoryErr, wantErr: repositoryErr,
			wantText: `select products for establishment "` + establishmentID + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			useCase, productRepo, categoryRepo, establishmentRepo := newUseCase(t)
			establishmentRepo.EXPECT().SelectEstablishmentById(ctx, establishmentID).
				Return(tt.establishment, tt.establishmentError).Once()
			if tt.establishment != nil && tt.establishmentError == nil {
				categoryRepo.EXPECT().SelectCategories(ctx, establishmentID).
					Return(nil, tt.categoryError).Once()
				if tt.categoryError == nil {
					productRepo.EXPECT().SelectProductsByEstablishmentId(ctx, establishmentID).
						Return(nil, tt.productError).Once()
				}
			}

			menu, err := useCase.GetMenu(ctx, establishmentID)

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantText != "" {
				assert.ErrorContains(t, err, tt.wantText)
			}
			assert.Equal(t, domain.Menu{}, menu)
		})
	}
}

func newUseCase(t *testing.T) (
	*establishment.EstablishmentUseCase,
	*mocks.MockProductRepository,
	*mocks.MockCategoryRepository,
	*mocks.MockEstablishmentRepository,
) {
	t.Helper()
	productRepo := mocks.NewMockProductRepository(t)
	categoryRepo := mocks.NewMockCategoryRepository(t)
	establishmentRepo := mocks.NewMockEstablishmentRepository(t)
	return establishment.NewEstablishmentUseCase(productRepo, categoryRepo, establishmentRepo),
		productRepo, categoryRepo, establishmentRepo
}
