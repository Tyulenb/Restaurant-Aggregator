package product

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	transactor "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/infrastructure/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/adapter/postgres/product/sqlc"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/customer"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/application/partner"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-tyulenb-7224f74f/internal/order/domain"
)

var _ customer.ProductRepository = (*ProductRepository)(nil)
var _ partner.ProductRepository = (*ProductRepository)(nil)

type ProductRepository struct {
	queries *sqlc.Queries
}

func NewProductRepository(db sqlc.DBTX) *ProductRepository {
	return &ProductRepository{queries: sqlc.New(db)}
}

func (r *ProductRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *ProductRepository) FindProductById(ctx context.Context, productID string) (*domain.Product, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return nil, fmt.Errorf("parse product ID: %w", err)
	}

	row, err := r.getQueries(ctx).SelectProductByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select product by ID: %w", err)
	}

	product, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert product selected by ID: %w", err)
	}

	return &product, nil
}

func (r *ProductRepository) FindProductByIdForUpdate(ctx context.Context, productID string) (*domain.Product, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return nil, fmt.Errorf("parse product ID: %w", err)
	}
	row, err := r.getQueries(ctx).SelectProductByIDForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select product by ID for update: %w", err)
	}
	product, err := toDomain(row)
	if err != nil {
		return nil, fmt.Errorf("convert product selected for update: %w", err)
	}
	return &product, nil
}

func (r *ProductRepository) DecrementProductQuantity(ctx context.Context, productID string, quantity int) error {
	id, err := uuid.Parse(productID)
	if err != nil {
		return fmt.Errorf("parse product ID: %w", err)
	}
	rows, err := r.getQueries(ctx).DecrementProductQuantity(ctx, sqlc.DecrementProductQuantityParams{
		ID: id, Quantity: pgtype.Int4{Int32: int32(quantity), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("decrement product quantity: %w", err)
	}
	if rows != 1 {
		return domain.ErrInsufficientStock
	}
	return nil
}

func (r *ProductRepository) IncrementProductQuantity(ctx context.Context, productID string, quantity int) error {
	id, err := uuid.Parse(productID)
	if err != nil {
		return fmt.Errorf("parse product ID: %w", err)
	}

	rows, err := r.getQueries(ctx).IncrementProductQuantity(ctx, sqlc.IncrementProductQuantityParams{
		ID: id, Quantity: pgtype.Int4{Int32: int32(quantity), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("increment product quantity: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("increment product quantity: expected one finite product, updated %d", rows)
	}

	return nil
}

func toDomain(row sqlc.Product) (domain.Product, error) {
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
