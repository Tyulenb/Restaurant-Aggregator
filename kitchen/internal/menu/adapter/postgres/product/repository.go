package product

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/adapter/postgres/product/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/establishment"
	application "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/application/product"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/menu/domain"
)

const uniqueViolationCode = "23505"

type ProductRepository struct {
	queries *sqlc.Queries
}

var _ application.ProductRepository = (*ProductRepository)(nil)
var _ establishment.ProductRepository = (*ProductRepository)(nil)

func NewProductRepository(db sqlc.DBTX) *ProductRepository {
	return &ProductRepository{queries: sqlc.New(db)}
}

func (r *ProductRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *ProductRepository) SelectProductsByEstablishmentIdAndCategoryExternalId(
	ctx context.Context,
	establishmentID, categoryExternalID string,
) ([]domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).SelectProductsByEstablishmentIdAndCategoryExternalId(
		ctx,
		sqlc.SelectProductsByEstablishmentIdAndCategoryExternalIdParams{
			EstablishmentID: id,
			ExternalID:      categoryExternalID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("select products by category external ID: %w", err)
	}

	products, err := productsToDomain(rows)
	if err != nil {
		return nil, fmt.Errorf("convert products selected by category external ID: %w", err)
	}
	if err := r.setCategoryExternalIDs(ctx, products); err != nil {
		return nil, fmt.Errorf("resolve categories for products selected by category external ID: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) SelectProductsByEstablishmentId(
	ctx context.Context,
	establishmentID string,
) ([]domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).SelectProductsByEstablishmentId(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select products by establishment ID: %w", err)
	}

	products, err := productsToDomain(rows)
	if err != nil {
		return nil, fmt.Errorf("convert products selected by establishment ID: %w", err)
	}
	if err := r.setCategoryExternalIDs(ctx, products); err != nil {
		return nil, fmt.Errorf("resolve categories for products selected by establishment ID: %w", err)
	}

	return products, nil
}

func (r *ProductRepository) InsertProduct(ctx context.Context, product domain.Product) (*domain.Product, error) {
	params, err := insertProductParams(product)
	if err != nil {
		return nil, fmt.Errorf("prepare product for insertion: %w", err)
	}

	row, err := r.getQueries(ctx).InsertProduct(ctx, params)
	if isUniqueViolation(err) {
		return nil, fmt.Errorf("insert product: %w", domain.ErrProductConflict)
	}
	if err != nil {
		return nil, fmt.Errorf("insert product: %w", err)
	}

	insertedProduct, err := productToDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert inserted product: %w", err)
	}
	if err := r.setCategoryExternalID(ctx, &insertedProduct); err != nil {
		return nil, fmt.Errorf("resolve inserted product category: %w", err)
	}

	return &insertedProduct, nil
}

func (r *ProductRepository) SelectProductByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectProductByExternalId(ctx, sqlc.SelectProductByExternalIdParams{
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	return r.selectedProduct(ctx, row, err, "select product by external ID")
}

func (r *ProductRepository) SelectProductByExternalIdForUpdate(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).SelectProductByExternalIdForUpdate(
		ctx,
		sqlc.SelectProductByExternalIdForUpdateParams{
			EstablishmentID: id,
			ExternalID:      externalID,
		},
	)
	return r.selectedProduct(ctx, row, err, "select product by external ID for update")
}

func (r *ProductRepository) UpdateProductByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
	cmd application.CommandUpdateProduct,
) (*domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	params := sqlc.UpdateProductByExternalIdParams{
		DescriptionSet:  cmd.DescriptionSet,
		EstablishmentID: id,
		ExternalID:      externalID,
	}
	if cmd.CategoryExternalID != nil {
		params.CategoryExternalID = pgtype.Text{String: *cmd.CategoryExternalID, Valid: true}
	}
	if cmd.Name != nil {
		params.Name = pgtype.Text{String: *cmd.Name, Valid: true}
	}
	if cmd.DescriptionSet && cmd.Description != nil {
		params.Description = pgtype.Text{String: *cmd.Description, Valid: true}
	}
	if cmd.Price != nil {
		params.Price, err = numericFromFloat32(*cmd.Price)
		if err != nil {
			return nil, fmt.Errorf("convert product price: %w", err)
		}
	}
	if cmd.Available != nil {
		params.Available = pgtype.Bool{Bool: *cmd.Available, Valid: true}
	}

	row, err := r.getQueries(ctx).UpdateProductByExternalId(ctx, params)
	return r.selectedProduct(ctx, row, err, "update product by external ID")
}

func (r *ProductRepository) DeleteProductByExternalId(
	ctx context.Context,
	establishmentID, externalID string,
) (*domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}

	row, err := r.getQueries(ctx).DeleteProductByExternalId(ctx, sqlc.DeleteProductByExternalIdParams{
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	return r.selectedProduct(ctx, row, err, "delete product by external ID")
}

func (r *ProductRepository) DeleteProductByCategoryId(
	ctx context.Context,
	categoryID string,
) ([]domain.Product, error) {
	id, err := parseUUID("category ID", categoryID)
	if err != nil {
		return nil, err
	}

	rows, err := r.getQueries(ctx).DeleteProductByCategoryId(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("delete products by category ID: %w", err)
	}

	products, err := productsToDomain(rows)
	if err != nil {
		return nil, fmt.Errorf("delete products by category ID: convert product: %w", err)
	}
	if err := r.setCategoryExternalIDs(ctx, products); err != nil {
		return nil, fmt.Errorf("delete products by category ID: resolve categories: %w", err)
	}
	if len(products) == 0 {
		return nil, nil
	}

	return products, nil
}

func (r *ProductRepository) UpdateProductQuantity(
	ctx context.Context,
	establishmentID, externalID string,
	newQuantity int,
) (*domain.Product, error) {
	id, err := parseUUID("establishment ID", establishmentID)
	if err != nil {
		return nil, err
	}
	quantityValue, err := int4FromInt(newQuantity)
	if err != nil {
		return nil, fmt.Errorf("convert product quantity: %w", err)
	}

	row, err := r.getQueries(ctx).UpdateProductQuantity(ctx, sqlc.UpdateProductQuantityParams{
		Quantity:        quantityValue,
		EstablishmentID: id,
		ExternalID:      externalID,
	})
	return r.selectedProduct(ctx, row, err, "update product quantity")
}

func insertProductParams(product domain.Product) (sqlc.InsertProductParams, error) {
	id, err := parseUUID("product ID", product.ID)
	if err != nil {
		return sqlc.InsertProductParams{}, err
	}
	establishmentID, err := parseUUID("establishment ID", product.EstablishmentID)
	if err != nil {
		return sqlc.InsertProductParams{}, err
	}
	categoryID, err := parseUUID("category ID", product.CategoryID)
	if err != nil {
		return sqlc.InsertProductParams{}, err
	}
	price, err := numericFromFloat32(product.Price)
	if err != nil {
		return sqlc.InsertProductParams{}, fmt.Errorf("convert product price: %w", err)
	}
	var inventoryTypeID int32
	switch product.InventoryTypeID {
	case domain.Finite:
		inventoryTypeID = 1
	case domain.Unlimited:
		inventoryTypeID = 2
	default:
		return sqlc.InsertProductParams{}, fmt.Errorf("unsupported inventory type %d", product.InventoryTypeID)
	}

	params := sqlc.InsertProductParams{
		ID:              id,
		ExternalID:      product.ExternalID,
		EstablishmentID: establishmentID,
		CategoryID:      categoryID,
		Name:            product.Name,
		Available:       product.Available,
		InventoryTypeID: inventoryTypeID,
		Price:           price,
		CreatedAt:       pgtype.Timestamptz{Time: product.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: product.UpdatedAt, Valid: true},
	}
	if product.Description != nil {
		params.Description = pgtype.Text{String: *product.Description, Valid: true}
	}
	if product.Quantity != nil {
		params.Quantity, err = int4FromInt(*product.Quantity)
		if err != nil {
			return sqlc.InsertProductParams{}, fmt.Errorf("convert product quantity: %w", err)
		}
	}
	if product.DeletedAt != nil {
		params.DeletedAt = pgtype.Timestamptz{Time: *product.DeletedAt, Valid: true}
	}

	return params, nil
}

func (r *ProductRepository) selectedProduct(
	ctx context.Context, row sqlc.Product, err error, operation string,
) (*domain.Product, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	product, err := productToDomain(row)
	if err != nil {
		return nil, fmt.Errorf("%s: convert product: %w", operation, err)
	}
	if err := r.setCategoryExternalID(ctx, &product); err != nil {
		return nil, fmt.Errorf("%s: resolve category: %w", operation, err)
	}

	return &product, nil
}

func (r *ProductRepository) setCategoryExternalIDs(ctx context.Context, products []domain.Product) error {
	externalIDs := make(map[string]string)
	for i := range products {
		if externalID, ok := externalIDs[products[i].CategoryID]; ok {
			products[i].CategoryExternalID = externalID
			continue
		}
		if err := r.setCategoryExternalID(ctx, &products[i]); err != nil {
			return err
		}
		externalIDs[products[i].CategoryID] = products[i].CategoryExternalID
	}

	return nil
}

func (r *ProductRepository) setCategoryExternalID(ctx context.Context, product *domain.Product) error {
	categoryID, err := parseUUID("category ID", product.CategoryID)
	if err != nil {
		return err
	}
	externalID, err := r.getQueries(ctx).SelectProductCategoryExternalId(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("select category external ID: %w", err)
	}
	product.CategoryExternalID = externalID

	return nil
}

func productsToDomain(rows []sqlc.Product) ([]domain.Product, error) {
	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		product, err := productToDomain(row)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func productToDomain(row sqlc.Product) (domain.Product, error) {
	price, err := row.Price.Float64Value()
	if err != nil {
		return domain.Product{}, fmt.Errorf("convert price: %w", err)
	}

	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}
	var quantity *int
	if row.Quantity.Valid {
		value := int(row.Quantity.Int32)
		quantity = &value
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}

	return domain.Product{
		ID:              row.ID.String(),
		ExternalID:      row.ExternalID,
		EstablishmentID: row.EstablishmentID.String(),
		CategoryID:      row.CategoryID.String(),
		Name:            row.Name,
		Description:     description,
		Available:       row.Available,
		InventoryTypeID: domain.InventoryType(row.InventoryTypeID),
		Quantity:        quantity,
		Price:           float32(price.Float64),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		DeletedAt:       deletedAt,
	}, nil
}

func numericFromFloat32(value float32) (pgtype.Numeric, error) {
	var numeric pgtype.Numeric
	err := numeric.Scan(strconv.FormatFloat(float64(value), 'f', -1, 32))
	return numeric, err
}

func int4FromInt(value int) (pgtype.Int4, error) {
	if int64(value) < -1<<31 || int64(value) > 1<<31-1 {
		return pgtype.Int4{}, fmt.Errorf("value %d is outside PostgreSQL integer range", value)
	}

	return pgtype.Int4{Int32: int32(value), Valid: true}, nil
}

func parseUUID(name, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", name, err)
	}

	return id, nil
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == uniqueViolationCode
}
