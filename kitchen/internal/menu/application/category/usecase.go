package category

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

type CategoryUseCase struct {
	CategoryRepo CategoryRepository
	ProductRepo  ProductRepository
	tx           Transactor
}

func NewCategoryUseCase(catRepo CategoryRepository, prRepo ProductRepository, tx Transactor) *CategoryUseCase {
	return &CategoryUseCase{
		CategoryRepo: catRepo,
		ProductRepo:  prRepo,
		tx:           tx,
	}
}

func (uc *CategoryUseCase) GetCategories(ctx context.Context, establishmentID string, includeDeleted bool) ([]domain.Category, error) {
	var (
		categories []domain.Category
		err        error
	)
	if includeDeleted {
		categories, err = uc.CategoryRepo.SelectCategoriesWithDeleted(ctx, establishmentID)
	} else {
		categories, err = uc.CategoryRepo.SelectCategories(ctx, establishmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("select categories for establishment %q: %w", establishmentID, err)
	}
	if categories == nil {
		return []domain.Category{}, nil
	}

	return categories, nil
}

func (uc *CategoryUseCase) CreateCategory(
	ctx context.Context,
	establishmentID, externalID, name string,
	description *string,
) (*domain.Category, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrCategoryValidation)
	}
	if !validCategoryName(name) {
		return nil, fmt.Errorf("%w: name must contain 1 to 255 characters", domain.ErrCategoryValidation)
	}

	now := time.Now()
	newCategory := domain.Category{
		ID:              uuid.NewString(),
		EstablishmentID: establishmentID,
		ExternalID:      externalID,
		Name:            name,
		Description:     description,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	createdCategory, err := uc.CategoryRepo.InsertCategory(ctx, newCategory)
	if err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}
	if createdCategory == nil {
		return nil, fmt.Errorf("insert category: repository returned a nil category")
	}

	return createdCategory, nil
}

func (uc *CategoryUseCase) GetCategoryByExternalID(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Category, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrCategoryValidation)
	}

	selectedCategory, err := uc.CategoryRepo.SelectCategoryByExternalId(ctx, establishmentID, externalID)
	if err != nil {
		return nil, fmt.Errorf("select category %q: %w", externalID, err)
	}
	if selectedCategory == nil {
		return nil, domain.ErrCategoryNotFound
	}

	return selectedCategory, nil
}

func (uc *CategoryUseCase) UpdateCategoryByExternalID(
	ctx context.Context,
	establishmentID, externalID string,
	cmd CommandUpdateCategory,
) (*domain.Category, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrCategoryValidation)
	}
	if cmd.Name != nil && !validCategoryName(*cmd.Name) {
		return nil, fmt.Errorf("%w: name must contain 1 to 255 characters", domain.ErrCategoryValidation)
	}
	if cmd.Name == nil && !cmd.DescriptionSet {
		return nil, fmt.Errorf("%w: at least one field is required", domain.ErrCategoryValidation)
	}

	var (
		updatedCategory *domain.Category
		err             error
	)
	switch {
	case cmd.Name != nil && cmd.DescriptionSet:
		updatedCategory, err = uc.CategoryRepo.UpdateCategoryNameAndDescriptionByExternalId(
			ctx, establishmentID, externalID, *cmd.Name, cmd.Description,
		)
	case cmd.Name != nil:
		updatedCategory, err = uc.CategoryRepo.UpdateCategoryNameByExternalId(
			ctx, establishmentID, externalID, *cmd.Name,
		)
	case cmd.DescriptionSet:
		updatedCategory, err = uc.CategoryRepo.UpdateCategoryDescriptionByExternalId(
			ctx, establishmentID, externalID, cmd.Description,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("update category %q: %w", externalID, err)
	}
	if updatedCategory == nil {
		return nil, domain.ErrCategoryNotFound
	}

	return updatedCategory, nil
}

func (uc *CategoryUseCase) DeleteCategoryByExternalID(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Category, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrCategoryValidation)
	}
	if uc.tx == nil {
		return nil, fmt.Errorf("delete category %q: transactor is not configured", externalID)
	}

	var deletedCategory *domain.Category
	err := uc.tx.WithTx(ctx, func(txCtx context.Context) error {
		var deleteErr error
		deletedCategory, deleteErr = uc.CategoryRepo.DeleteCategoryByExternalId(txCtx, establishmentID, externalID)
		if deleteErr != nil {
			return fmt.Errorf("delete category: %w", deleteErr)
		}
		if deletedCategory == nil {
			return domain.ErrCategoryNotFound
		}
		if uc.ProductRepo == nil {
			return fmt.Errorf("delete products: product repository is not configured")
		}
		_, productDeleteErr := uc.ProductRepo.DeleteProductByCategoryId(txCtx, deletedCategory.ID)
		if productDeleteErr != nil {
			return fmt.Errorf("delete products: %w", productDeleteErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("delete category %q: %w", externalID, err)
	}

	return deletedCategory, nil
}

func validExternalID(value string) bool {
	length := utf8.RuneCountInString(value)
	return length >= 1 && length <= 128
}

func validCategoryName(value string) bool {
	length := utf8.RuneCountInString(value)
	return length >= 1 && length <= 255
}
