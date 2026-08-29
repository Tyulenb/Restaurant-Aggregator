package category_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/category"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/category/mocks"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

const (
	establishmentID = "establishment-1"
	externalID      = "breakfast"
)

type transactionContextKey struct{}

func TestCategoryUseCase_GetCategories_SelectsRequestedVisibility(t *testing.T) {
	deletedAt := time.Now()
	tests := []struct {
		name           string
		includeDeleted bool
		categories     []domain.Category
	}{
		{
			name:       "active categories only",
			categories: []domain.Category{{ID: "category-1", ExternalID: externalID}},
		},
		{
			name:           "including deleted categories",
			includeDeleted: true,
			categories: []domain.Category{
				{ID: "category-1", ExternalID: externalID},
				{ID: "category-2", ExternalID: "lunch", DeletedAt: &deletedAt},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			repo := mocks.NewMockCategoryRepository(t)
			if tt.includeDeleted {
				repo.EXPECT().SelectCategoriesWithDeleted(ctx, establishmentID).Return(tt.categories, nil).Once()
			} else {
				repo.EXPECT().SelectCategories(ctx, establishmentID).Return(tt.categories, nil).Once()
			}
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			actual, err := useCase.GetCategories(ctx, establishmentID, tt.includeDeleted)

			require.NoError(t, err)
			assert.Equal(t, tt.categories, actual)
		})
	}
}

func TestCategoryUseCase_GetCategories_NormalizesNilResult(t *testing.T) {
	repo := mocks.NewMockCategoryRepository(t)
	repo.EXPECT().SelectCategories(mock.Anything, establishmentID).Return(nil, nil).Once()
	useCase := category.NewCategoryUseCase(repo, nil, nil)

	categories, err := useCase.GetCategories(t.Context(), establishmentID, false)

	require.NoError(t, err)
	assert.Empty(t, categories)
	assert.NotNil(t, categories)
}

func TestCategoryUseCase_GetCategories_WrapsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repo := mocks.NewMockCategoryRepository(t)
	repo.EXPECT().SelectCategoriesWithDeleted(mock.Anything, establishmentID).Return(nil, repositoryErr).Once()
	useCase := category.NewCategoryUseCase(repo, nil, nil)

	categories, err := useCase.GetCategories(t.Context(), establishmentID, true)

	require.ErrorIs(t, err, repositoryErr)
	assert.ErrorContains(t, err, `select categories for establishment "`+establishmentID+`"`)
	assert.Nil(t, categories)
}

func TestCategoryUseCase_CreateCategory_ValidatesCharacterLimits(t *testing.T) {
	tests := []struct {
		name       string
		externalID string
		category   string
	}{
		{name: "empty external ID", category: "Breakfast"},
		{name: "external ID over 128 characters", externalID: strings.Repeat("я", 129), category: "Breakfast"},
		{name: "empty name", externalID: externalID},
		{name: "name over 255 characters", externalID: externalID, category: strings.Repeat("я", 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockCategoryRepository(t)
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			created, err := useCase.CreateCategory(
				t.Context(), establishmentID, tt.externalID, tt.category, nil,
			)

			require.ErrorIs(t, err, domain.ErrCategoryValidation)
			assert.Nil(t, created)
		})
	}
}

func TestCategoryUseCase_CreateCategory_CreatesCategory(t *testing.T) {
	ctx := t.Context()
	description := "Morning meals"
	var inserted domain.Category
	repo := mocks.NewMockCategoryRepository(t)
	repo.EXPECT().InsertCategory(ctx, mock.AnythingOfType("domain.Category")).
		RunAndReturn(func(_ context.Context, value domain.Category) (*domain.Category, error) {
			inserted = value
			return &value, nil
		}).Once()
	useCase := category.NewCategoryUseCase(repo, nil, nil)

	before := time.Now()
	created, err := useCase.CreateCategory(
		ctx, establishmentID, strings.Repeat("я", 128), strings.Repeat("б", 255), &description,
	)
	after := time.Now()

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, inserted, *created)
	assert.NotEmpty(t, inserted.ID)
	assert.Equal(t, establishmentID, inserted.EstablishmentID)
	assert.Equal(t, strings.Repeat("я", 128), inserted.ExternalID)
	assert.Equal(t, strings.Repeat("б", 255), inserted.Name)
	assert.Same(t, &description, inserted.Description)
	assert.WithinRange(t, inserted.CreatedAt, before, after)
	assert.Equal(t, inserted.CreatedAt, inserted.UpdatedAt)
	assert.Nil(t, inserted.DeletedAt)
}

func TestCategoryUseCase_CreateCategory_HandlesRepositoryFailures(t *testing.T) {
	repositoryErr := errors.New("external ID conflict")
	tests := []struct {
		name    string
		result  *domain.Category
		repoErr error
		wantErr error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "nil repository result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockCategoryRepository(t)
			repo.EXPECT().InsertCategory(mock.Anything, mock.Anything).Return(tt.result, tt.repoErr).Once()
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			created, err := useCase.CreateCategory(t.Context(), establishmentID, externalID, "Breakfast", nil)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.ErrorContains(t, err, "repository returned a nil category")
			}
			assert.ErrorContains(t, err, "insert category")
			assert.Nil(t, created)
		})
	}
}

func TestCategoryUseCase_GetCategoryByExternalID(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	expected := &domain.Category{ID: "category-1", EstablishmentID: establishmentID, ExternalID: externalID}
	tests := []struct {
		name       string
		externalID string
		result     *domain.Category
		repoErr    error
		wantErr    error
		callRepo   bool
	}{
		{name: "returns category", externalID: externalID, result: expected, callRepo: true},
		{name: "invalid external ID", wantErr: domain.ErrCategoryValidation},
		{name: "repository error", externalID: externalID, repoErr: repositoryErr, wantErr: repositoryErr, callRepo: true},
		{name: "category not found", externalID: externalID, wantErr: domain.ErrCategoryNotFound, callRepo: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockCategoryRepository(t)
			if tt.callRepo {
				repo.EXPECT().SelectCategoryByExternalId(mock.Anything, establishmentID, tt.externalID).
					Return(tt.result, tt.repoErr).Once()
			}
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			actual, err := useCase.GetCategoryByExternalID(t.Context(), establishmentID, tt.externalID)

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

func TestCategoryUseCase_UpdateCategoryByExternalID_ValidatesCommand(t *testing.T) {
	empty := ""
	longName := strings.Repeat("я", 256)
	tests := []struct {
		name       string
		externalID string
		command    category.CommandUpdateCategory
	}{
		{name: "invalid external ID", command: category.CommandUpdateCategory{Description: stringPointer("description"), DescriptionSet: true}},
		{name: "no fields", externalID: externalID},
		{name: "empty name", externalID: externalID, command: category.CommandUpdateCategory{Name: &empty}},
		{name: "name over 255 characters", externalID: externalID, command: category.CommandUpdateCategory{Name: &longName}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockCategoryRepository(t)
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			updated, err := useCase.UpdateCategoryByExternalID(
				t.Context(), establishmentID, tt.externalID, tt.command,
			)

			require.ErrorIs(t, err, domain.ErrCategoryValidation)
			assert.Nil(t, updated)
		})
	}
}

func TestCategoryUseCase_UpdateCategoryByExternalID_UpdatesPartialFields(t *testing.T) {
	name := "Brunch"
	description := "Served until noon"
	tests := []struct {
		name      string
		command   category.CommandUpdateCategory
		operation string
	}{
		{
			name: "name", command: category.CommandUpdateCategory{Name: &name},
			operation: "name",
		},
		{
			name: "description", command: category.CommandUpdateCategory{Description: &description, DescriptionSet: true},
			operation: "description",
		},
		{
			name: "clear description", command: category.CommandUpdateCategory{DescriptionSet: true},
			operation: "clear description",
		},
		{
			name:      "name and description",
			command:   category.CommandUpdateCategory{Name: &name, Description: &description, DescriptionSet: true},
			operation: "both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := &domain.Category{ID: "category-1", ExternalID: externalID, Name: name}
			repo := mocks.NewMockCategoryRepository(t)
			switch tt.operation {
			case "name":
				repo.EXPECT().UpdateCategoryNameByExternalId(
					mock.Anything, establishmentID, externalID, name,
				).Return(expected, nil).Once()
			case "description":
				repo.EXPECT().UpdateCategoryDescriptionByExternalId(
					mock.Anything, establishmentID, externalID, &description,
				).Return(expected, nil).Once()
			case "clear description":
				repo.EXPECT().UpdateCategoryDescriptionByExternalId(
					mock.Anything, establishmentID, externalID, (*string)(nil),
				).Return(expected, nil).Once()
			case "both":
				repo.EXPECT().UpdateCategoryNameAndDescriptionByExternalId(
					mock.Anything, establishmentID, externalID, name, &description,
				).Return(expected, nil).Once()
			}
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			updated, err := useCase.UpdateCategoryByExternalID(
				t.Context(), establishmentID, externalID, tt.command,
			)

			require.NoError(t, err)
			assert.Same(t, expected, updated)
		})
	}
}

func TestCategoryUseCase_UpdateCategoryByExternalID_HandlesRepositoryFailures(t *testing.T) {
	description := "Morning meals"
	command := category.CommandUpdateCategory{Description: &description, DescriptionSet: true}
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "category not found", wantErr: domain.ErrCategoryNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockCategoryRepository(t)
			repo.EXPECT().UpdateCategoryDescriptionByExternalId(
				mock.Anything, establishmentID, externalID, &description,
			).
				Return(nil, tt.repoErr).Once()
			useCase := category.NewCategoryUseCase(repo, nil, nil)

			updated, err := useCase.UpdateCategoryByExternalID(
				t.Context(), establishmentID, externalID, command,
			)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, updated)
		})
	}
}

func TestCategoryUseCase_DeleteCategoryByExternalID(t *testing.T) {
	categoryErr := errors.New("category database unavailable")
	productErr := errors.New("product database unavailable")
	transactorErr := errors.New("transaction unavailable")
	deletedAt := time.Now()
	expected := &domain.Category{ID: "category-1", ExternalID: externalID, DeletedAt: &deletedAt}

	t.Run("deletes category and products with transaction context", func(t *testing.T) {
		categoryRepo := mocks.NewMockCategoryRepository(t)
		productRepo := mocks.NewMockProductRepository(t)
		tx := mocks.NewMockTransactor(t)
		txCtx := context.WithValue(t.Context(), transactionContextKey{}, "transaction")
		tx.EXPECT().WithTx(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, callback func(context.Context) error) error {
				return callback(txCtx)
			}).Once()
		categoryRepo.EXPECT().DeleteCategoryByExternalId(txCtx, establishmentID, externalID).
			Return(expected, nil).Once()
		productRepo.EXPECT().DeleteProductByCategoryId(txCtx, expected.ID).Return(nil, nil).Once()
		useCase := category.NewCategoryUseCase(categoryRepo, productRepo, tx)

		deleted, err := useCase.DeleteCategoryByExternalID(t.Context(), establishmentID, externalID)

		require.NoError(t, err)
		assert.Same(t, expected, deleted)
	})

	tests := []struct {
		name          string
		externalID    string
		category      *domain.Category
		categoryErr   error
		productErr    error
		transactorErr error
		wantErr       error
		callTx        bool
		callProducts  bool
	}{
		{name: "repeated deletion remains successful", externalID: externalID, category: expected, callTx: true, callProducts: true},
		{name: "invalid external ID", wantErr: domain.ErrCategoryValidation},
		{name: "category failure", externalID: externalID, categoryErr: categoryErr, wantErr: categoryErr, callTx: true},
		{name: "category never existed", externalID: externalID, wantErr: domain.ErrCategoryNotFound, callTx: true},
		{name: "product cascade failure", externalID: externalID, category: expected, productErr: productErr, wantErr: productErr, callTx: true, callProducts: true},
		{name: "transactor failure", externalID: externalID, transactorErr: transactorErr, wantErr: transactorErr, callTx: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryRepo := mocks.NewMockCategoryRepository(t)
			productRepo := mocks.NewMockProductRepository(t)
			tx := mocks.NewMockTransactor(t)
			if tt.callTx {
				txCall := tx.EXPECT().WithTx(mock.Anything, mock.Anything)
				if tt.transactorErr != nil {
					txCall.Return(tt.transactorErr).Once()
				} else {
					txCall.RunAndReturn(func(ctx context.Context, callback func(context.Context) error) error {
						return callback(ctx)
					}).Once()
					categoryRepo.EXPECT().DeleteCategoryByExternalId(mock.Anything, establishmentID, tt.externalID).
						Return(tt.category, tt.categoryErr).Once()
					if tt.callProducts {
						productRepo.EXPECT().DeleteProductByCategoryId(mock.Anything, expected.ID).
							Return(nil, tt.productErr).Once()
					}
				}
			}
			useCase := category.NewCategoryUseCase(categoryRepo, productRepo, tx)

			deleted, err := useCase.DeleteCategoryByExternalID(t.Context(), establishmentID, tt.externalID)

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

func stringPointer(value string) *string {
	return &value
}
