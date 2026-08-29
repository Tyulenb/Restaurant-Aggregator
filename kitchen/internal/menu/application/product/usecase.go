package product

import (
	"context"
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

const (
	minStockDelta = -1 << 31
	maxStockDelta = 1<<31 - 1
)

type ProductUseCase struct {
	Repo ProductRepository
}

func NewProductUseCase(repo ProductRepository) *ProductUseCase {
	return &ProductUseCase{Repo: repo}
}

func NewUseCase(repo ProductRepository) *ProductUseCase {
	return NewProductUseCase(repo)
}

func (pu *ProductUseCase) ListPartnerProducts(
	ctx context.Context, establishmentID, categoryExternalID string, includeDeleted bool,
) ([]domain.Product, error) {
	if categoryExternalID != "" && !validExternalID(categoryExternalID) {
		return nil, fmt.Errorf("%w: category external ID must contain at most 128 characters", domain.ErrProductValidation)
	}

	var (
		products []domain.Product
		err      error
	)
	if categoryExternalID == "" {
		products, err = pu.Repo.SelectProductsByEstablishmentId(ctx, establishmentID)
	} else {
		products, err = pu.Repo.SelectProductsByEstablishmentIdAndCategoryExternalId(
			ctx, establishmentID, categoryExternalID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("select products for establishment %q: %w", establishmentID, err)
	}

	result := make([]domain.Product, 0, len(products))
	for _, product := range products {
		if includeDeleted || product.DeletedAt == nil {
			result = append(result, product)
		}
	}

	return result, nil
}

func (pu *ProductUseCase) CreatePartnerProduct(
	ctx context.Context, establishmentID string, cmd CommandCreateProduct,
) (*domain.Product, error) {
	inventoryType, err := validateCreateCommand(cmd)
	if err != nil {
		return nil, err
	}

	category, err := pu.Repo.SelectCategoryByExternalId(ctx, establishmentID, cmd.CategoryExternalID)
	if err != nil {
		return nil, fmt.Errorf("select category %q: %w", cmd.CategoryExternalID, err)
	}
	if category == nil || category.DeletedAt != nil {
		return nil, domain.ErrCategoryNotFound
	}

	now := time.Now()
	newProduct := domain.Product{
		ID:              uuid.NewString(),
		ExternalID:      cmd.ExternalID,
		EstablishmentID: establishmentID,
		CategoryID:      category.ID,
		Name:            cmd.Name,
		Description:     cmd.Description,
		Available:       cmd.Available,
		InventoryTypeID: inventoryType,
		Quantity:        cmd.Quantity,
		Price:           cmd.Price,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	createdProduct, err := pu.Repo.InsertProduct(ctx, newProduct)
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}
	if createdProduct == nil {
		return nil, fmt.Errorf("insert product: repository returned a nil product")
	}

	return createdProduct, nil
}

func (pu *ProductUseCase) FindProductByExternalId(
	ctx context.Context, establishmentID, externalID string,
) (*domain.Product, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrProductValidation)
	}

	selectedProduct, err := pu.Repo.SelectProductByExternalId(ctx, establishmentID, externalID)
	if err != nil {
		return nil, fmt.Errorf("select product %q: %w", externalID, err)
	}
	if selectedProduct == nil {
		return nil, domain.ErrProductNotFound
	}

	return selectedProduct, nil
}

func (pu *ProductUseCase) UpdateProductByExternalId(
	ctx context.Context, establishmentID, externalID string, cmd CommandUpdateProduct,
) (*domain.Product, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrProductValidation)
	}
	if err := validateUpdateCommand(cmd); err != nil {
		return nil, err
	}
	if cmd.CategoryExternalID != nil {
		category, err := pu.Repo.SelectCategoryByExternalId(ctx, establishmentID, *cmd.CategoryExternalID)
		if err != nil {
			return nil, fmt.Errorf("select category %q: %w", *cmd.CategoryExternalID, err)
		}
		if category == nil || category.DeletedAt != nil {
			return nil, domain.ErrCategoryNotFound
		}
	}

	updatedProduct, err := pu.Repo.UpdateProductByExternalId(ctx, establishmentID, externalID, cmd)
	if err != nil {
		return nil, fmt.Errorf("update product %q: %w", externalID, err)
	}
	if updatedProduct == nil {
		return nil, domain.ErrProductNotFound
	}

	return updatedProduct, nil
}

func (pu *ProductUseCase) DeleteProductByExternalId(
	ctx context.Context, establishmentID, externalID string,
) (*domain.Product, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrProductValidation)
	}

	deletedProduct, err := pu.Repo.DeleteProductByExternalId(ctx, establishmentID, externalID)
	if err != nil {
		return nil, fmt.Errorf("delete product %q: %w", externalID, err)
	}
	if deletedProduct == nil {
		return nil, domain.ErrProductNotFound
	}

	return deletedProduct, nil
}

func (pu *ProductUseCase) AdjustProductStock(
	ctx context.Context, establishmentID, externalID string, delta int,
) (*domain.Product, error) {
	if !validExternalID(externalID) {
		return nil, fmt.Errorf("%w: external ID must contain 1 to 128 characters", domain.ErrProductValidation)
	}
	if delta == 0 {
		return nil, fmt.Errorf("%w: delta must not be zero", domain.ErrProductValidation)
	}
	if int64(delta) < minStockDelta || int64(delta) > maxStockDelta {
		return nil, fmt.Errorf("%w: delta is outside the supported range", domain.ErrProductValidation)
	}

	product, err := pu.Repo.SelectProductByExternalId(ctx, establishmentID, externalID)
	if err != nil {
		return nil, fmt.Errorf("select product %q: %w", externalID, err)
	}
	if product == nil || product.DeletedAt != nil {
		return nil, domain.ErrProductNotFound
	}
	if product.InventoryTypeID != domain.Finite || product.Quantity == nil {
		return nil, fmt.Errorf("%w: product does not use finite inventory", domain.ErrStockAdjustmentConflict)
	}
	resultingQuantity := int64(*product.Quantity) + int64(delta)
	if resultingQuantity < 0 || resultingQuantity > maxStockDelta {
		return nil, fmt.Errorf("%w: resulting quantity is outside the supported range", domain.ErrStockAdjustmentConflict)
	}

	updatedProduct, err := pu.Repo.UpdateProductQuantity(ctx, establishmentID, externalID, delta)
	if err != nil {
		return nil, fmt.Errorf("adjust product %q stock: %w", externalID, err)
	}
	if updatedProduct == nil {
		return nil, fmt.Errorf("%w: stock changed before the adjustment was applied", domain.ErrStockAdjustmentConflict)
	}

	return updatedProduct, nil
}

func validateCreateCommand(cmd CommandCreateProduct) (domain.InventoryType, error) {
	if !validExternalID(cmd.ExternalID) || !validExternalID(cmd.CategoryExternalID) ||
		!validName(cmd.Name) || !validPrice(cmd.Price) {
		return 0, domain.ErrProductValidation
	}

	switch cmd.InventoryType {
	case "finite":
		if cmd.Quantity == nil || *cmd.Quantity < 0 {
			return 0, fmt.Errorf("%w: finite inventory requires a non-negative quantity", domain.ErrProductValidation)
		}
		return domain.Finite, nil
	case "untracked":
		if cmd.Quantity != nil {
			return 0, fmt.Errorf("%w: untracked inventory must not have a quantity", domain.ErrProductValidation)
		}
		return domain.Unlimited, nil
	default:
		return 0, fmt.Errorf("%w: unknown inventory type %q", domain.ErrProductValidation, cmd.InventoryType)
	}
}

func validateUpdateCommand(cmd CommandUpdateProduct) error {
	if cmd.CategoryExternalID == nil && cmd.Name == nil && cmd.Description == nil && !cmd.DescriptionSet &&
		cmd.Price == nil && cmd.Available == nil {
		return fmt.Errorf("%w: at least one field is required", domain.ErrProductValidation)
	}
	if cmd.CategoryExternalID != nil && !validExternalID(*cmd.CategoryExternalID) {
		return domain.ErrProductValidation
	}
	if cmd.Name != nil && !validName(*cmd.Name) {
		return domain.ErrProductValidation
	}
	if cmd.Price != nil && !validPrice(*cmd.Price) {
		return domain.ErrProductValidation
	}

	return nil
}

func validExternalID(value string) bool {
	length := utf8.RuneCountInString(value)
	return length >= 1 && length <= 128
}

func validName(value string) bool {
	length := utf8.RuneCountInString(value)
	return length >= 1 && length <= 255
}

func validPrice(value float32) bool {
	return value >= 0 && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
