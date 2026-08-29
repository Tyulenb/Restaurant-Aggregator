package product_test

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/product"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/product/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

const (
	establishmentID    = "establishment-1"
	externalProductID  = "baguette"
	externalCategoryID = "bakery"
)

func TestProductUseCase_ListPartnerProducts_SelectsAndFiltersProducts(t *testing.T) {
	deletedAt := time.Now()
	products := []domain.Product{
		{ID: "product-1", ExternalID: externalProductID},
		{ID: "product-2", ExternalID: "old-baguette", DeletedAt: &deletedAt},
	}
	tests := []struct {
		name               string
		categoryExternalID string
		includeDeleted     bool
		want               []domain.Product
	}{
		{name: "all active products", want: products[:1]},
		{name: "all products including deleted", includeDeleted: true, want: products},
		{name: "active products in category", categoryExternalID: externalCategoryID, want: products[:1]},
		{
			name:               "category products including deleted",
			categoryExternalID: externalCategoryID,
			includeDeleted:     true,
			want:               products,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			repo := mocks.NewMockProductRepository(t)
			if tt.categoryExternalID == "" {
				repo.EXPECT().SelectProductsByEstablishmentId(ctx, establishmentID).
					Return(products, nil).Once()
			} else {
				repo.EXPECT().SelectProductsByEstablishmentIdAndCategoryExternalId(
					ctx, establishmentID, tt.categoryExternalID,
				).Return(products, nil).Once()
			}
			useCase := newProductUseCase(t, repo)

			actual, err := useCase.ListPartnerProducts(
				ctx, establishmentID, tt.categoryExternalID, tt.includeDeleted,
			)

			require.NoError(t, err)
			assert.Equal(t, tt.want, actual)
		})
	}
}

func TestProductUseCase_ListPartnerProducts_HandlesEmptyAndErrorResults(t *testing.T) {
	t.Run("nil result becomes empty slice", func(t *testing.T) {
		repo := mocks.NewMockProductRepository(t)
		repo.EXPECT().SelectProductsByEstablishmentId(mock.Anything, establishmentID).
			Return(nil, nil).Once()
		useCase := newProductUseCase(t, repo)

		products, err := useCase.ListPartnerProducts(t.Context(), establishmentID, "", false)

		require.NoError(t, err)
		assert.Empty(t, products)
		assert.NotNil(t, products)
	})

	t.Run("wraps repository error", func(t *testing.T) {
		repositoryErr := errors.New("database unavailable")
		repo := mocks.NewMockProductRepository(t)
		repo.EXPECT().SelectProductsByEstablishmentIdAndCategoryExternalId(
			mock.Anything, establishmentID, externalCategoryID,
		).Return(nil, repositoryErr).Once()
		useCase := newProductUseCase(t, repo)

		products, err := useCase.ListPartnerProducts(
			t.Context(), establishmentID, externalCategoryID, false,
		)

		require.ErrorIs(t, err, repositoryErr)
		assert.ErrorContains(t, err, `select products for establishment "`+establishmentID+`"`)
		assert.Nil(t, products)
	})
}

func TestProductUseCase_ListPartnerProducts_ValidatesCategoryFilter(t *testing.T) {
	repo := mocks.NewMockProductRepository(t)
	useCase := newProductUseCase(t, repo)

	products, err := useCase.ListPartnerProducts(
		t.Context(), establishmentID, strings.Repeat("я", 129), false,
	)

	require.ErrorIs(t, err, domain.ErrProductValidation)
	assert.Nil(t, products)
}

func TestProductUseCase_CreatePartnerProduct_ValidatesCommand(t *testing.T) {
	zero := 0
	negative := -1
	tests := []struct {
		name   string
		mutate func(*product.CommandCreateProduct)
	}{
		{name: "empty external ID", mutate: func(cmd *product.CommandCreateProduct) { cmd.ExternalID = "" }},
		{name: "long external ID", mutate: func(cmd *product.CommandCreateProduct) { cmd.ExternalID = strings.Repeat("я", 129) }},
		{name: "empty category ID", mutate: func(cmd *product.CommandCreateProduct) { cmd.CategoryExternalID = "" }},
		{name: "long category ID", mutate: func(cmd *product.CommandCreateProduct) { cmd.CategoryExternalID = strings.Repeat("я", 129) }},
		{name: "empty name", mutate: func(cmd *product.CommandCreateProduct) { cmd.Name = "" }},
		{name: "long name", mutate: func(cmd *product.CommandCreateProduct) { cmd.Name = strings.Repeat("я", 256) }},
		{name: "negative price", mutate: func(cmd *product.CommandCreateProduct) { cmd.Price = -1 }},
		{name: "NaN price", mutate: func(cmd *product.CommandCreateProduct) { cmd.Price = float32(math.NaN()) }},
		{name: "infinite price", mutate: func(cmd *product.CommandCreateProduct) { cmd.Price = float32(math.Inf(1)) }},
		{name: "finite without quantity", mutate: func(cmd *product.CommandCreateProduct) { cmd.InventoryType = "finite" }},
		{name: "finite with negative quantity", mutate: func(cmd *product.CommandCreateProduct) { cmd.InventoryType = "finite"; cmd.Quantity = &negative }},
		{name: "untracked with quantity", mutate: func(cmd *product.CommandCreateProduct) { cmd.Quantity = &zero }},
		{name: "unlimited with quantity", mutate: func(cmd *product.CommandCreateProduct) { cmd.InventoryType = "unlimited"; cmd.Quantity = &zero }},
		{name: "undocumented unlimited type", mutate: func(cmd *product.CommandCreateProduct) { cmd.InventoryType = "unlimited" }},
		{name: "unknown inventory type", mutate: func(cmd *product.CommandCreateProduct) { cmd.InventoryType = "warehouse" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCreateCommand()
			tt.mutate(&cmd)
			repo := mocks.NewMockProductRepository(t)
			useCase := newProductUseCase(t, repo)

			created, err := useCase.CreatePartnerProduct(t.Context(), establishmentID, cmd)

			require.ErrorIs(t, err, domain.ErrProductValidation)
			assert.Nil(t, created)
		})
	}
}

func TestProductUseCase_CreatePartnerProduct_CreatesInventoryTypes(t *testing.T) {
	zero := 0
	tests := []struct {
		name          string
		inventoryType string
		quantity      *int
		wantType      domain.InventoryType
	}{
		{name: "finite", inventoryType: "finite", quantity: &zero, wantType: domain.Finite},
		{name: "untracked", inventoryType: "untracked", wantType: domain.Unlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			cmd := validCreateCommand()
			cmd.InventoryType = tt.inventoryType
			cmd.Quantity = tt.quantity
			category := &domain.Category{ID: "category-1", ExternalID: externalCategoryID}
			repo := mocks.NewMockProductRepository(t)
			categoryRepo := mocks.NewMockCategoryRepository(t)
			categoryRepo.EXPECT().SelectCategoryByExternalId(ctx, establishmentID, externalCategoryID).
				Return(category, nil).Once()
			var inserted domain.Product
			repo.EXPECT().InsertProduct(ctx, mock.AnythingOfType("domain.Product")).
				RunAndReturn(func(_ context.Context, value domain.Product) (*domain.Product, error) {
					inserted = value
					return &value, nil
				}).Once()
			useCase := product.NewProductUseCase(repo, categoryRepo, executingTransactor(t))

			before := time.Now()
			created, err := useCase.CreatePartnerProduct(ctx, establishmentID, cmd)
			after := time.Now()

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, inserted, *created)
			assert.NotEmpty(t, inserted.ID)
			assert.Equal(t, cmd.ExternalID, inserted.ExternalID)
			assert.Equal(t, establishmentID, inserted.EstablishmentID)
			assert.Equal(t, category.ID, inserted.CategoryID)
			assert.Equal(t, cmd.Name, inserted.Name)
			assert.Same(t, cmd.Description, inserted.Description)
			assert.Equal(t, cmd.Available, inserted.Available)
			assert.Equal(t, tt.wantType, inserted.InventoryTypeID)
			assert.Equal(t, tt.quantity, inserted.Quantity)
			assert.Equal(t, cmd.Price, inserted.Price)
			assert.WithinRange(t, inserted.CreatedAt, before, after)
			assert.Equal(t, inserted.CreatedAt, inserted.UpdatedAt)
		})
	}
}

func TestProductUseCase_CreatePartnerProduct_HandlesCategoryFailures(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	deletedAt := time.Now()
	tests := []struct {
		name     string
		category *domain.Category
		repoErr  error
		wantErr  error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "missing category", wantErr: domain.ErrCategoryNotFound},
		{name: "deleted category", category: &domain.Category{ID: "category-1", DeletedAt: &deletedAt}, wantErr: domain.ErrCategoryNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			categoryRepo := mocks.NewMockCategoryRepository(t)
			categoryRepo.EXPECT().SelectCategoryByExternalId(mock.Anything, establishmentID, externalCategoryID).
				Return(tt.category, tt.repoErr).Once()
			useCase := product.NewProductUseCase(repo, categoryRepo, executingTransactor(t))

			created, err := useCase.CreatePartnerProduct(t.Context(), establishmentID, validCreateCommand())

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, created)
		})
	}
}

func TestProductUseCase_CreatePartnerProduct_HandlesInsertFailures(t *testing.T) {
	repositoryErr := errors.New("external ID conflict")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "nil repository result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			categoryRepo := mocks.NewMockCategoryRepository(t)
			categoryRepo.EXPECT().SelectCategoryByExternalId(mock.Anything, establishmentID, externalCategoryID).
				Return(&domain.Category{ID: "category-1"}, nil).Once()
			repo.EXPECT().InsertProduct(mock.Anything, mock.Anything).Return(nil, tt.repoErr).Once()
			useCase := product.NewProductUseCase(repo, categoryRepo, executingTransactor(t))

			created, err := useCase.CreatePartnerProduct(t.Context(), establishmentID, validCreateCommand())

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.ErrorContains(t, err, "repository returned a nil product")
			}
			assert.ErrorContains(t, err, "insert product")
			assert.Nil(t, created)
		})
	}
}

func TestProductUseCase_FindProductByExternalId(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	expected := &domain.Product{ID: "product-1", ExternalID: externalProductID}
	tests := []struct {
		name    string
		result  *domain.Product
		repoErr error
		wantErr error
	}{
		{name: "returns product", result: expected},
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "product not found", wantErr: domain.ErrProductNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalId(mock.Anything, establishmentID, externalProductID).
				Return(tt.result, tt.repoErr).Once()
			useCase := newProductUseCase(t, repo)

			actual, err := useCase.FindProductByExternalId(t.Context(), establishmentID, externalProductID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, actual)
				return
			}
			require.NoError(t, err)
			assert.Same(t, expected, actual)
		})
	}
}

func TestProductUseCase_ProductOperationsValidateExternalID(t *testing.T) {
	repo := mocks.NewMockProductRepository(t)
	useCase := newProductUseCase(t, repo)
	invalidID := strings.Repeat("я", 129)
	available := true

	operations := []struct {
		name string
		call func() error
	}{
		{name: "find", call: func() error {
			_, err := useCase.FindProductByExternalId(t.Context(), establishmentID, invalidID)
			return err
		}},
		{name: "update", call: func() error {
			_, err := useCase.UpdateProductByExternalId(
				t.Context(), establishmentID, invalidID, product.CommandUpdateProduct{Available: &available},
			)
			return err
		}},
		{name: "delete", call: func() error {
			_, err := useCase.DeleteProductByExternalId(t.Context(), establishmentID, invalidID)
			return err
		}},
		{name: "stock adjustment", call: func() error {
			_, err := useCase.AdjustProductStock(t.Context(), establishmentID, invalidID, 1)
			return err
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			require.ErrorIs(t, operation.call(), domain.ErrProductValidation)
		})
	}
}

func TestProductUseCase_UpdateProductByExternalId_ValidatesCommand(t *testing.T) {
	empty := ""
	longName := strings.Repeat("я", 256)
	negativePrice := float32(-1)
	nanPrice := float32(math.NaN())
	tests := []struct {
		name    string
		command product.CommandUpdateProduct
	}{
		{name: "no fields"},
		{name: "empty category ID", command: product.CommandUpdateProduct{CategoryExternalID: &empty}},
		{name: "empty name", command: product.CommandUpdateProduct{Name: &empty}},
		{name: "long name", command: product.CommandUpdateProduct{Name: &longName}},
		{name: "negative price", command: product.CommandUpdateProduct{Price: &negativePrice}},
		{name: "NaN price", command: product.CommandUpdateProduct{Price: &nanPrice}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			useCase := newProductUseCase(t, repo)

			updated, err := useCase.UpdateProductByExternalId(
				t.Context(), establishmentID, externalProductID, tt.command,
			)

			require.ErrorIs(t, err, domain.ErrProductValidation)
			assert.Nil(t, updated)
		})
	}
}

func TestProductUseCase_UpdateProductByExternalId_ValidatesNewCategory(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	deletedAt := time.Now()
	tests := []struct {
		name     string
		category *domain.Category
		repoErr  error
		wantErr  error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "missing category", wantErr: domain.ErrCategoryNotFound},
		{name: "deleted category", category: &domain.Category{ID: "category-2", DeletedAt: &deletedAt}, wantErr: domain.ErrCategoryNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryID := "lunch"
			cmd := product.CommandUpdateProduct{CategoryExternalID: &categoryID}
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalId(mock.Anything, establishmentID, externalProductID).
				Return(&domain.Product{ID: "product-1"}, nil).Once()
			categoryRepo := mocks.NewMockCategoryRepository(t)
			categoryRepo.EXPECT().SelectCategoryByExternalId(mock.Anything, establishmentID, categoryID).
				Return(tt.category, tt.repoErr).Once()
			useCase := product.NewProductUseCase(repo, categoryRepo, executingTransactor(t))

			updated, err := useCase.UpdateProductByExternalId(
				t.Context(), establishmentID, externalProductID, cmd,
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, updated)
		})
	}
}

func TestProductUseCase_UpdateProductByExternalId_UpdatesMetadata(t *testing.T) {
	name := strings.Repeat("я", 255)
	categoryID := strings.Repeat("к", 128)
	tests := []struct {
		name    string
		command product.CommandUpdateProduct
	}{
		{name: "name without category lookup", command: product.CommandUpdateProduct{Name: &name}},
		{name: "clear description", command: product.CommandUpdateProduct{DescriptionSet: true}},
		{name: "move category", command: product.CommandUpdateProduct{CategoryExternalID: &categoryID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := &domain.Product{ID: "product-1", ExternalID: externalProductID}
			current := &domain.Product{ID: "product-1", ExternalID: externalProductID, Name: "Current", Price: 1}
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalId(mock.Anything, establishmentID, externalProductID).
				Return(current, nil).Once()
			categoryRepo := mocks.NewMockCategoryRepository(t)
			if tt.command.CategoryExternalID != nil {
				categoryRepo.EXPECT().SelectCategoryByExternalId(mock.Anything, establishmentID, categoryID).
					Return(&domain.Category{ID: "category-2"}, nil).Once()
			}
			repo.EXPECT().UpdateProductByExternalId(
				mock.Anything, establishmentID, externalProductID, mock.Anything,
			).Return(expected, nil).Once()
			useCase := product.NewProductUseCase(repo, categoryRepo, executingTransactor(t))

			updated, err := useCase.UpdateProductByExternalId(
				t.Context(), establishmentID, externalProductID, tt.command,
			)

			require.NoError(t, err)
			assert.Same(t, expected, updated)
		})
	}
}

func TestProductUseCase_UpdateProductByExternalId_HandlesRepositoryFailures(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	available := true
	cmd := product.CommandUpdateProduct{Available: &available}
	tests := []struct {
		name     string
		selected *domain.Product
		repoErr  error
		wantErr  error
	}{
		{name: "repository error", selected: &domain.Product{ID: "product-1"}, repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "product not found", wantErr: domain.ErrProductNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalId(mock.Anything, establishmentID, externalProductID).
				Return(tt.selected, nil).Once()
			if tt.selected != nil {
				repo.EXPECT().UpdateProductByExternalId(mock.Anything, establishmentID, externalProductID, mock.Anything).
					Return(nil, tt.repoErr).Once()
			}
			useCase := newProductUseCase(t, repo)

			updated, err := useCase.UpdateProductByExternalId(
				t.Context(), establishmentID, externalProductID, cmd,
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, updated)
		})
	}
}

func TestProductUseCase_DeleteProductByExternalId(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	expected := &domain.Product{ID: "product-1", ExternalID: externalProductID}
	tests := []struct {
		name    string
		result  *domain.Product
		repoErr error
		wantErr error
	}{
		{name: "returns deleted product", result: expected},
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "product not found", wantErr: domain.ErrProductNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().DeleteProductByExternalId(mock.Anything, establishmentID, externalProductID).
				Return(tt.result, tt.repoErr).Once()
			useCase := newProductUseCase(t, repo)

			deleted, err := useCase.DeleteProductByExternalId(t.Context(), establishmentID, externalProductID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, deleted)
				return
			}
			require.NoError(t, err)
			assert.Same(t, expected, deleted)
		})
	}
}

func TestProductUseCase_AdjustProductStock_ValidatesAdjustment(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	deletedAt := time.Now()
	quantity := 2
	tests := []struct {
		name        string
		newQuantity int
		selected    *domain.Product
		repoErr     error
		wantErr     error
		callRepo    bool
	}{
		{name: "negative quantity", newQuantity: -1, wantErr: domain.ErrProductValidation},
		{name: "select error", newQuantity: 1, repoErr: repositoryErr, wantErr: repositoryErr, callRepo: true},
		{name: "missing product", newQuantity: 1, wantErr: domain.ErrProductNotFound, callRepo: true},
		{name: "deleted product", newQuantity: 1, selected: finiteProduct(&quantity), wantErr: domain.ErrProductNotFound, callRepo: true},
		{name: "untracked product", newQuantity: 1, selected: &domain.Product{InventoryTypeID: domain.Unlimited}, wantErr: domain.ErrStockAdjustmentConflict, callRepo: true},
		{name: "finite product without quantity", newQuantity: 1, selected: &domain.Product{InventoryTypeID: domain.Finite}, wantErr: domain.ErrStockAdjustmentConflict, callRepo: true},
	}
	tests[3].selected.DeletedAt = &deletedAt

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			if tt.callRepo {
				repo.EXPECT().SelectProductByExternalIdForUpdate(mock.Anything, establishmentID, externalProductID).
					Return(tt.selected, tt.repoErr).Once()
			}
			useCase := newProductUseCase(t, repo)

			updated, err := useCase.AdjustProductStock(
				t.Context(), establishmentID, externalProductID, tt.newQuantity,
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, updated)
		})
	}
}

func TestProductUseCase_AdjustProductStock_RejectsQuantityOutsideInt32(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("Go int cannot represent a value outside the int32 range")
	}

	repo := mocks.NewMockProductRepository(t)
	updated, err := newProductUseCase(t, repo).AdjustProductStock(
		t.Context(), establishmentID, externalProductID, int(int64(1<<31)),
	)

	require.ErrorIs(t, err, domain.ErrProductValidation)
	assert.Nil(t, updated)
}

func TestProductUseCase_AdjustProductStock_SetsAbsoluteQuantity(t *testing.T) {
	tests := []struct {
		name        string
		current     int
		newQuantity int
	}{
		{name: "replace current quantity", current: 2, newQuantity: 3},
		{name: "set quantity to zero", current: 2, newQuantity: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := finiteProduct(&tt.current)
			updated := finiteProduct(&tt.newQuantity)
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalIdForUpdate(mock.Anything, establishmentID, externalProductID).
				Return(current, nil).Once()
			repo.EXPECT().UpdateProductQuantity(mock.Anything, establishmentID, externalProductID, tt.newQuantity).
				Return(updated, nil).Once()
			useCase := newProductUseCase(t, repo)

			actual, err := useCase.AdjustProductStock(
				t.Context(), establishmentID, externalProductID, tt.newQuantity,
			)

			require.NoError(t, err)
			assert.Same(t, updated, actual)
		})
	}
}

func TestProductUseCase_AdjustProductStock_HandlesUpdateFailures(t *testing.T) {
	quantity := 2
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "concurrent stock conflict", wantErr: domain.ErrStockAdjustmentConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockProductRepository(t)
			repo.EXPECT().SelectProductByExternalIdForUpdate(mock.Anything, establishmentID, externalProductID).
				Return(finiteProduct(&quantity), nil).Once()
			repo.EXPECT().UpdateProductQuantity(mock.Anything, establishmentID, externalProductID, 1).
				Return(nil, tt.repoErr).Once()
			useCase := newProductUseCase(t, repo)

			updated, err := useCase.AdjustProductStock(t.Context(), establishmentID, externalProductID, 1)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, updated)
		})
	}
}

func validCreateCommand() product.CommandCreateProduct {
	description := "Fresh bread"
	return product.CommandCreateProduct{
		ExternalID:         externalProductID,
		CategoryExternalID: externalCategoryID,
		Name:               "Baguette",
		Description:        &description,
		Price:              199,
		Available:          true,
		InventoryType:      "untracked",
	}
}

func finiteProduct(quantity *int) *domain.Product {
	return &domain.Product{
		ID:              "product-1",
		ExternalID:      externalProductID,
		InventoryTypeID: domain.Finite,
		Quantity:        quantity,
	}
}

func newProductUseCase(t *testing.T, repo *mocks.MockProductRepository) *product.ProductUseCase {
	t.Helper()
	return product.NewProductUseCase(repo, mocks.NewMockCategoryRepository(t), executingTransactor(t))
}

func executingTransactor(t *testing.T) *mocks.MockTransactor {
	t.Helper()
	tx := mocks.NewMockTransactor(t)
	tx.EXPECT().WithTx(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, callback func(context.Context) error) error {
			return callback(ctx)
		}).Maybe()
	return tx
}
